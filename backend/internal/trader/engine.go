package trader

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/micro-trading-for-agent/backend/internal/agent"
	"github.com/micro-trading-for-agent/backend/internal/database"
	"github.com/micro-trading-for-agent/backend/internal/kis"
	"github.com/micro-trading-for-agent/backend/internal/logger"
	"github.com/micro-trading-for-agent/backend/internal/models"
	"github.com/micro-trading-for-agent/backend/internal/monitor"
)

// filteredStockEntry records a single stock removed by the hard filter with the reason.
type filteredStockEntry struct {
	StockCode     string  `json:"stock_code"`
	StockName     string  `json:"stock_name"`
	FilterReason  string  `json:"filter_reason"`
	RSI14         float64 `json:"rsi14,omitempty"`
	DisparityM5   float64 `json:"disparity_m5,omitempty"`
	HighPriceDiff float64 `json:"high_price_diff,omitempty"`
	OpenPriceDiff float64 `json:"open_price_diff,omitempty"`
	MA5           float64 `json:"ma5,omitempty"`
	MA20          float64 `json:"ma20,omitempty"`
}

// EngineState represents the current phase of the trading engine.
type EngineState string

const (
	StateIdle        EngineState = "IDLE"
	StateSelecting   EngineState = "SELECTING"
	StateOrdering    EngineState = "ORDERING"
	StateWaitingFill EngineState = "WAITING_FILL"
	StateMonitoring  EngineState = "MONITORING"
	StateSearching   EngineState = "SEARCHING" // 포지션 없음 — 매수 종목 탐색 중
)

// Engine runs autonomous trading cycles: select → order → monitor → repeat.
type Engine struct {
	market    string // "KR" or "US"
	db        *database.DB
	kisClient *kis.Client
	wsClient  *kis.WebSocketClient
	mon       *monitor.Monitor
	claude    *ClaudeClient

	mu         sync.RWMutex
	state      EngineState
	haltReason string      // 마지막 사이클 중지 사유 (성공 시 초기화)
	soldCh     chan string // receives stock_code when monitor executes a sell
	stopCh     chan struct{}
}

// NewEngine creates a new Engine with all required dependencies.
// claude may be nil if ANTHROPIC_API_KEY is not configured (engine will log an error and sleep).
// market: "KR" (default) or "US".
func NewEngine(
	db *database.DB,
	kisClient *kis.Client,
	wsClient *kis.WebSocketClient,
	mon *monitor.Monitor,
	claude *ClaudeClient,
	market string,
) *Engine {
	if market == "" {
		market = "KR"
	}
	return &Engine{
		market:    market,
		db:        db,
		kisClient: kisClient,
		wsClient:  wsClient,
		mon:       mon,
		claude:    claude,
		state:     StateIdle,
		soldCh:    make(chan string, 16),
		stopCh:    make(chan struct{}),
	}
}

// GetState returns the current engine state (thread-safe).
func (e *Engine) GetState() EngineState {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.state
}

// GetHaltReason returns the reason the last cycle was halted (thread-safe).
// Returns empty string if the last cycle succeeded or if no cycle has run yet.
func (e *Engine) GetHaltReason() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.haltReason
}

// ForceRun triggers a single selectAndBuy cycle in a background goroutine,
// bypassing the normal cycle wait and skipping schedule/market-condition checks
// (trading days, buy pause period, index drop filter). Useful for manual intervention.
func (e *Engine) ForceRun(ctx context.Context) {
	go func() {
		settings, err := e.db.GetTradingSettings(ctx)
		if err != nil {
			logger.Error("engine: ForceRun GetTradingSettings failed", map[string]any{"error": err.Error()})
			return
		}
		if err := e.selectAndBuy(ctx, settings, true); err != nil {
			logger.Error("engine: ForceRun selectAndBuy failed", map[string]any{"error": err.Error()})
			e.mu.Lock()
			e.haltReason = err.Error()
			e.mu.Unlock()
		} else {
			e.mu.Lock()
			e.haltReason = ""
			e.mu.Unlock()
		}
	}()
}

func (e *Engine) setState(s EngineState) {
	e.mu.Lock()
	e.state = s
	e.mu.Unlock()
	logger.Info("engine: state changed", map[string]any{"state": string(s)})
}

// Start launches the trading cycle goroutine and returns a stop function.
func (e *Engine) Start(ctx context.Context) (stop func()) {
	cycleCtx, cancel := context.WithCancel(ctx)
	e.stopCh = make(chan struct{})

	go e.runCycle(cycleCtx)

	return func() {
		cancel()
		e.setState(StateIdle)
		logger.Info("engine: stopped", nil)
	}
}

// SoldCh returns the channel that should be sent to when a monitored position is sold.
// Pass this as SoldCh when registering MonitoredEntry objects.
func (e *Engine) SoldCh() chan<- string {
	return e.soldCh
}

// retryBackoff returns wait duration based on consecutive failure count.
// 1st: 30s, 2nd: 1m, 3rd: 3m, 4th+: 5m
func retryBackoff(failures int) time.Duration {
	switch failures {
	case 1:
		return 30 * time.Second
	case 2:
		return 1 * time.Minute
	case 3:
		return 3 * time.Minute
	default:
		return 5 * time.Minute
	}
}

func (e *Engine) runCycle(ctx context.Context) {
	consecutiveFailures := 0

	for {
		select {
		case <-ctx.Done():
			e.setState(StateIdle)
			return
		default:
		}

		settings, err := e.db.GetTradingSettings(ctx)
		if err != nil {
			logger.Error("engine: GetTradingSettings failed", map[string]any{"error": err.Error()})
			select {
			case <-ctx.Done():
				return
			case <-time.After(30 * time.Second):
			}
			continue
		}

		// 포지션 수 = 모니터링 중인 포지션 + 오늘 접수한 미체결(PENDING) 매수 주문.
		// 재시작 시 PENDING 주문이 남아있는 경우 이중 주문을 방지한다.
		currentCount := e.mon.Count() + e.countPendingOrders(ctx)
		if currentCount >= settings.MaxPositions {
			consecutiveFailures = 0 // 포지션 보유 중 = 정상 상태
			e.setState(StateMonitoring)
			select {
			case <-ctx.Done():
				e.setState(StateIdle)
				return
			case code := <-e.soldCh:
				logger.Info("engine: sold signal received, resuming cycle",
					map[string]any{"stock_code": code})
			case <-time.After(30 * time.Second):
			}
			continue
		}

		// 포지션 여유 있음 — 매수 종목 탐색 단계
		e.setState(StateSearching)

		if e.claude == nil {
			logger.Error("engine: claude client not configured (ANTHROPIC_API_KEY missing)", nil)
			select {
			case <-ctx.Done():
				return
			case <-time.After(60 * time.Second):
			}
			continue
		}

		if err := e.selectAndBuy(ctx, settings, false); err != nil {
			consecutiveFailures++
			wait := retryBackoff(consecutiveFailures)
			logger.Error("engine: selectAndBuy failed",
				map[string]any{"error": err.Error(), "failures": consecutiveFailures, "retry_in": wait.String()})
			e.mu.Lock()
			e.haltReason = err.Error()
			e.mu.Unlock()
			e.setState(StateSearching)
			select {
			case <-ctx.Done():
				return
			case <-time.After(wait):
			}
		} else {
			e.mu.Lock()
			e.haltReason = ""
			e.mu.Unlock()
			consecutiveFailures = 0
		}
	}
}

// selectAndBuy runs one stock-selection and buy cycle.
// force=true skips schedule/market-condition gates (trading days, buy pause, index drop)
// so the user can manually trigger a cycle regardless of current restrictions.
func (e *Engine) selectAndBuy(ctx context.Context, settings database.TradingSettings, force bool) error {
	if e.market == "US" {
		return e.selectAndBuyUS(ctx, settings, force)
	}

	// 요일 체크 (강제 실행 시 건너뜀)
	if !force && len(settings.TradingDays) > 0 {
		kst, _ := time.LoadLocation("Asia/Seoul")
		today := int(time.Now().In(kst).Weekday())
		allowed := false
		for _, d := range settings.TradingDays {
			if d == today {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("오늘은 거래 제외 요일 (weekday=%d)", today)
		}
	}

	e.setState(StateSelecting)

	// 매수 중단 시간대 체크 (강제 실행 시 건너뜀)
	if !force && settings.BuyPauseStart != "" && settings.BuyPauseEnd != "" {
		now := time.Now().Format("15:04")
		if now >= settings.BuyPauseStart && now < settings.BuyPauseEnd {
			e.setState(StateMonitoring)
			return fmt.Errorf("매수 중단 시간대 (%s~%s)", settings.BuyPauseStart, settings.BuyPauseEnd)
		}
	}

	// 지수 필터: 지수가 시가 대비 설정값 이상 하락 시 매수 중단 (강제 실행 시 건너뜀)
	indexDropThreshold := settings.IndexDropThresholdPct
	if indexDropThreshold == 0 {
		indexDropThreshold = -1.0
	}
	var marketIndexDrop float64
	for _, code := range settings.IndexCodes {
		if idx, idxErr := e.kisClient.GetIndexPrice(ctx, code); idxErr == nil {
			open, _ := strconv.ParseFloat(idx.DayOpen, 64)
			cur, _ := strconv.ParseFloat(idx.CurrentPrice, 64)
			if open > 0 && cur > 0 {
				drop := (cur - open) / open * 100
				if drop < marketIndexDrop {
					marketIndexDrop = drop // 가장 많이 하락한 지수 기준
				}
				if !force && drop <= indexDropThreshold {
					e.setState(StateMonitoring)
					return fmt.Errorf("지수 %.1f%% 이상 하락 (지수:%s %.2f%%↓), 매수 일시 중단", indexDropThreshold, code, drop)
				}
			}
		}
	}

	// Build today's exclusion list from DB orders.
	excludedCodes := e.getTodayTradedCodes(ctx)

	// Fetch rankings based on configured types.
	rankings, rankingLogID, err := e.getRankings(ctx, settings)
	if err != nil {
		e.setState(StateMonitoring)
		return fmt.Errorf("getRankings: %w", err)
	}
	if len(rankings) == 0 {
		e.setState(StateMonitoring)
		return fmt.Errorf("no ranking results after intersection filter")
	}

	// allFilteredOut 누적: 모든 필터 단계에서 제거된 종목을 기록 (ranked_stocks → filtered_stocks DB 기록용)
	var allFilteredOut []filteredStockEntry

	// Filter out already-traded stocks server-side before indicator fetch and Claude call.
	if len(excludedCodes) > 0 {
		excludeSet := make(map[string]bool, len(excludedCodes))
		for _, code := range excludedCodes {
			excludeSet[code] = true
		}
		filtered := make([]RankItem, 0, len(rankings))
		for _, r := range rankings {
			if excludeSet[r.StockCode] {
				allFilteredOut = append(allFilteredOut, filteredStockEntry{
					StockCode: r.StockCode, StockName: r.StockName,
					FilterReason: "오늘 이미 거래된 종목",
				})
			} else {
				filtered = append(filtered, r)
			}
		}
		if len(filtered) < len(rankings) {
			logger.Info("engine: excluded already-traded stocks from candidates",
				map[string]any{
					"before":         len(rankings),
					"after":          len(filtered),
					"excluded_codes": excludedCodes,
				})
		}
		rankings = filtered
	}
	if len(rankings) == 0 {
		e.setState(StateMonitoring)
		return fmt.Errorf("no ranking results after excluding already-traded stocks")
	}

	// Enrich each candidate with technical indicators (MA5/MA20/RSI/MACD/VWAP/M5MA10/PrevVolumeRatio).
	for i, r := range rankings {
		info, err := agent.GetStockInfo(ctx, e.kisClient, r.StockCode)
		if err != nil {
			logger.Warn("engine: GetStockInfo failed, skipping indicators",
				map[string]any{"stock_code": r.StockCode, "error": err.Error()})
			continue
		}
		rankings[i].MA5 = info.MA5
		rankings[i].MA20 = info.MA20
		rankings[i].RSI14 = info.RSI14
		rankings[i].MACDLine = info.MACDLine
		rankings[i].MACDSignal = info.MACDSignal
		rankings[i].DayOpen = info.DayOpen
		rankings[i].DayHigh = info.DayHigh
		rankings[i].DayLow = info.DayLow
		rankings[i].HighPriceDiff = info.HighPriceDiff
		rankings[i].OpenPriceDiff = info.OpenPriceDiff
		rankings[i].DisparityM5 = info.DisparityM5
		rankings[i].VWAP = info.VWAP
		rankings[i].VWAPDiff = info.VWAPDiff
		rankings[i].M5MA10 = info.M5MA10
		rankings[i].PrevVolumeRatio = info.PrevVolumeRatio
		rankings[i].BidAskRatio = info.BidAskRatio
	}
	// 거래대금 하한선 필터
	if settings.MinTradingValue > 0 {
		var passed []RankItem
		for _, item := range rankings {
			info, err := agent.GetStockInfo(ctx, e.kisClient, item.StockCode)
			if err != nil || info.TradingValue >= settings.MinTradingValue {
				passed = append(passed, item)
			} else {
				allFilteredOut = append(allFilteredOut, filteredStockEntry{
					StockCode:    item.StockCode,
					StockName:    item.StockName,
					FilterReason: fmt.Sprintf("거래대금 미달 (%.0f억 < %.0f억)", info.TradingValue/1e8, settings.MinTradingValue/1e8),
				})
			}
		}
		rankings = passed
	}

	// Get available cash.
	summary, err := e.kisClient.GetInquireBalance(ctx)
	if err != nil {
		e.setState(StateMonitoring)
		return fmt.Errorf("GetInquireBalance: %w", err)
	}
	availableCash, _ := strconv.ParseFloat(summary.DepositAmt, 64)
	if availableCash <= 0 {
		e.setState(StateMonitoring)
		return fmt.Errorf("no available cash")
	}

	// 일일 최대 손실 한도 체크 (KR — KRW 기준)
	if settings.DailyMaxLossPct > 0 {
		pnl := e.db.GetTodayRealizedPnLByMarket(ctx, "KR")
		if pnl < 0 {
			totalEval, _ := strconv.ParseFloat(summary.TotalEval, 64)
			if totalEval <= 0 {
				totalEval = availableCash
			}
			lossLimit := totalEval * settings.DailyMaxLossPct / 100
			if -pnl >= lossLimit {
				e.setState(StateMonitoring)
				msg := fmt.Sprintf("일일 최대 손실 한도 도달 (%.0f원 손실 >= 한도 %.0f원)", -pnl, lossLimit)
				e.db.InsertServiceLog(ctx, "TRADER", "ERROR", msg, "")
				return fmt.Errorf("%s", msg)
			}
		}
	}

	// Filter out stocks whose current price exceeds available cash (can't buy even 1 share).
	{
		var passed []RankItem
		for _, item := range rankings {
			price, _ := strconv.ParseFloat(item.CurrentPrice, 64)
			if price > 0 && price <= availableCash {
				passed = append(passed, item)
			} else if price > availableCash {
				allFilteredOut = append(allFilteredOut, filteredStockEntry{
					StockCode:    item.StockCode,
					StockName:    item.StockName,
					FilterReason: fmt.Sprintf("현금 부족 (주가 %.0f원 > 가용 %.0f원)", price, availableCash),
				})
			}
		}
		rankings = passed
	}
	if len(rankings) == 0 {
		e.setState(StateMonitoring)
		return fmt.Errorf("no affordable stocks after price filter (cash: %.0f)", availableCash)
	}

	// 하드 필터: LLM 전달 전 과열/이격 과대 종목 제거 (제거된 종목은 ranking log에 기록)
	{
		filterRsiMax := settings.FilterRsiMax
		if filterRsiMax == 0 {
			filterRsiMax = 80
		}
		filterDisparityM5Max := settings.FilterDisparityM5Max
		if filterDisparityM5Max == 0 {
			filterDisparityM5Max = 3.0
		}
		filterHighPriceDiffMin := settings.FilterHighPriceDiffMin
		if filterHighPriceDiffMin == 0 {
			filterHighPriceDiffMin = -5.0
		}
		filterOpenPriceDiffMax := settings.FilterOpenPriceDiffMax
		if filterOpenPriceDiffMax == 0 {
			filterOpenPriceDiffMax = 20.0
		}
		var passed []RankItem
		for _, item := range rankings {
			if item.RSI14 > 0 && item.RSI14 >= filterRsiMax {
				logger.Info("engine: hard filter RSI", map[string]any{"stock_code": item.StockCode, "rsi": item.RSI14, "threshold": filterRsiMax})
				allFilteredOut = append(allFilteredOut, filteredStockEntry{StockCode: item.StockCode, StockName: item.StockName, FilterReason: fmt.Sprintf("RSI 과열 (%.1f >= %.1f)", item.RSI14, filterRsiMax), RSI14: item.RSI14})
				continue
			}
			if item.DisparityM5 > filterDisparityM5Max {
				logger.Info("engine: hard filter disparity_m5", map[string]any{"stock_code": item.StockCode, "disparity_m5": item.DisparityM5, "threshold": filterDisparityM5Max})
				allFilteredOut = append(allFilteredOut, filteredStockEntry{StockCode: item.StockCode, StockName: item.StockName, FilterReason: fmt.Sprintf("5분봉 이격도 초과 (%.1f > %.1f)", item.DisparityM5, filterDisparityM5Max), DisparityM5: item.DisparityM5})
				continue
			}
			if item.HighPriceDiff != 0 && item.HighPriceDiff < filterHighPriceDiffMin {
				logger.Info("engine: hard filter high_price_diff", map[string]any{"stock_code": item.StockCode, "high_price_diff": item.HighPriceDiff, "threshold": filterHighPriceDiffMin})
				allFilteredOut = append(allFilteredOut, filteredStockEntry{StockCode: item.StockCode, StockName: item.StockName, FilterReason: fmt.Sprintf("고가 대비 과다 하락 (%.1f%% < %.1f%%)", item.HighPriceDiff, filterHighPriceDiffMin), HighPriceDiff: item.HighPriceDiff})
				continue
			}
			if item.OpenPriceDiff > filterOpenPriceDiffMax {
				logger.Info("engine: hard filter open_price_diff", map[string]any{"stock_code": item.StockCode, "open_price_diff": item.OpenPriceDiff, "threshold": filterOpenPriceDiffMax})
				allFilteredOut = append(allFilteredOut, filteredStockEntry{StockCode: item.StockCode, StockName: item.StockName, FilterReason: fmt.Sprintf("시가 대비 과다 상승 (%.1f%% > %.1f%%)", item.OpenPriceDiff, filterOpenPriceDiffMax), OpenPriceDiff: item.OpenPriceDiff})
				continue
			}
			passed = append(passed, item)
		}
		rankings = passed
		// 모든 필터 단계 제거 종목을 DB에 기록 (하드필터 + 이전 단계 누적)
		if rankingLogID > 0 && len(allFilteredOut) > 0 {
			filteredJSON, _ := json.Marshal(allFilteredOut)
			e.db.ExecContext(ctx, `UPDATE trader_ranking_logs SET filtered_stocks=? WHERE id=?`, string(filteredJSON), rankingLogID) //nolint:errcheck
		}
	}
	if len(rankings) == 0 {
		const failMsg = "no stocks passed hard filter (RSI/disparity/high-price/open-price)"
		e.db.ExecContext(ctx, //nolint:errcheck
			`INSERT INTO trader_selection_logs (sent_count, candidates, llm_result, fail_reason, market, ranking_log_id) VALUES (?,?,?,?,?,?)`,
			0, "[]", "", failMsg, "KR", rankingLogID)
		e.setState(StateMonitoring)
		return fmt.Errorf("%s", failMsg)
	}

	// Persist selection log to DB (Claude 호출 전 INSERT — 실패해도 로그 남김).
	var selectionLogID int64
	{
		candidatesJSON, _ := json.Marshal(rankings)
		res, dbErr := e.db.ExecContext(ctx,
			`INSERT INTO trader_selection_logs (sent_count, candidates, llm_result, market, ranking_log_id) VALUES (?,?,?,?,?)`,
			len(rankings), string(candidatesJSON), "", "KR", rankingLogID)
		if dbErr == nil {
			selectionLogID, _ = res.LastInsertId()
		}
	}

	// Build TradingRules from settings and pass to Claude prompt.
	rules := DefaultTradingRules()
	rules.IndexDropThreshold = settings.IndexDropThresholdPct
	if rules.IndexDropThreshold == 0 {
		rules.IndexDropThreshold = -1.0
	}
	rules.HardDisparityM5Min = settings.HardDisparityM5Min
	if rules.HardDisparityM5Min == 0 {
		rules.HardDisparityM5Min = -1.5
	}
	rules.HardDisparityM5Max = settings.HardDisparityM5Max
	if rules.HardDisparityM5Max == 0 {
		rules.HardDisparityM5Max = 3.0
	}
	rules.HardHighPriceDiffMax = settings.HardHighPriceDiffMax
	if rules.HardHighPriceDiffMax == 0 {
		rules.HardHighPriceDiffMax = -0.5
	}
	rules.HardHighPriceDiffMin = settings.HardHighPriceDiffMin
	if rules.HardHighPriceDiffMin == 0 {
		rules.HardHighPriceDiffMin = -5.0
	}
	rules.HardPrevVolRatioMax = settings.HardPrevVolRatioMax
	if rules.HardPrevVolRatioMax == 0 {
		rules.HardPrevVolRatioMax = 1.2
	}
	rules.HardStrengthMin = settings.HardStrengthMin
	if rules.HardStrengthMin == 0 {
		rules.HardStrengthMin = 100.0
	}
	rules.HardRSIMax = settings.HardRSIMax
	if rules.HardRSIMax == 0 {
		rules.HardRSIMax = 70.0
	}
	rules.HardOpenPriceDiffMax = settings.HardOpenPriceDiffMax
	if rules.HardOpenPriceDiffMax == 0 {
		rules.HardOpenPriceDiffMax = 15.0
	}
	rules.VWAPDiffMin = settings.VWAPDiffMin
	rules.VWAPDiffMax = settings.VWAPDiffMax
	if rules.VWAPDiffMax == 0 {
		rules.VWAPDiffMax = 1.5
	}
	rules.RSIBuyMin = settings.RSIBuyMin
	if rules.RSIBuyMin == 0 {
		rules.RSIBuyMin = 40.0
	}
	rules.RSIBuyMax = settings.RSIBuyMax
	if rules.RSIBuyMax == 0 {
		rules.RSIBuyMax = 60.0
	}
	rules.BidAskRatioMin = settings.BidAskRatioMin
	rules.MarketIndexDrop = marketIndexDrop

	// Ask Claude to rank all viable candidates (single API call).
	// excludedCodes are already filtered server-side above; pass nil here.
	candidates, err := e.claude.SelectStocks(ctx, rankings, availableCash, nil, "KR", rules)
	if err != nil {
		if selectionLogID > 0 {
			e.db.ExecContext(ctx, //nolint:errcheck
				`UPDATE trader_selection_logs SET fail_reason=? WHERE id=?`,
				"LLM 오류: "+err.Error(), selectionLogID)
		}
		e.setState(StateMonitoring)
		return fmt.Errorf("SelectStocks: %w", err)
	}
	logger.Info("engine: Claude ranked candidates",
		map[string]any{"count": len(candidates)})

	// LLM 응답 저장.
	if selectionLogID > 0 {
		llmResultJSON, _ := json.Marshal(candidates)
		e.db.ExecContext(ctx, //nolint:errcheck
			`UPDATE trader_selection_logs SET llm_result=? WHERE id=?`,
			string(llmResultJSON), selectionLogID)
	}

	// Try candidates in order until one succeeds.
	var (
		stockCode   string
		filledPrice float64
		filledQty   int
		result      *agent.PlaceOrderResult
	)
	for i, candidate := range candidates {
		code := candidate.StockCode
		logger.Info("engine: trying candidate",
			map[string]any{"rank": i + 1, "stock_code": code, "reason": candidate.Reason})

		e.setState(StateOrdering)

		feasibility, ferr := agent.CheckOrderFeasibility(ctx, e.kisClient, code)
		if ferr != nil || feasibility.OrderableQty <= 0 {
			logger.Warn("engine: candidate not orderable, skipping",
				map[string]any{"stock_code": code})
			continue
		}

		qty := int(float64(feasibility.OrderableQty) * settings.OrderAmountPct / 100)
		if qty <= 0 {
			qty = 1
		}

		// 종목명을 rankings에서 조회하여 주문 시 함께 저장
		candidateName := code
		for _, r := range rankings {
			if r.StockCode == code {
				candidateName = r.StockName
				break
			}
		}

		res, perr := agent.PlaceOrder(ctx, e.kisClient, e.db, agent.PlaceOrderRequest{
			StockCode: code,
			StockName: candidateName,
			OrderType: models.OrderTypeBuy,
			Qty:       qty,
			Price:     0,
			OrderDivn: "01",
			TargetPct: settings.TakeProfitPct,
			StopPct:   settings.StopLossPct,
		})
		if perr != nil {
			logger.Warn("engine: PlaceOrder failed, skipping",
				map[string]any{"stock_code": code, "error": perr.Error()})
			continue
		}

		e.setState(StateWaitingFill)

		fillCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		fp, fq, filled := e.waitForFill(fillCtx, res.KISOrderID)
		cancel()

		if !filled {
			if _, cancelErr := agent.CancelOrder(ctx, e.kisClient, e.db, res.OrderID); cancelErr != nil {
				logger.Warn("engine: cancel order failed after fill timeout",
					map[string]any{"order_id": res.OrderID, "error": cancelErr.Error()})
			}
			logger.Warn("engine: fill timeout, trying next candidate",
				map[string]any{"stock_code": code})
			e.db.InsertServiceLog(ctx, "TRADER", "ERROR", "미체결 타임아웃 (5분)", fmt.Sprintf("stock_code=%s order_id=%d", code, res.OrderID))
			continue
		}

		// Success.
		stockCode = code
		filledPrice = fp
		filledQty = fq
		result = res
		break
	}

	if result == nil {
		if selectionLogID > 0 {
			e.db.ExecContext(ctx, //nolint:errcheck
				`UPDATE trader_selection_logs SET fail_reason=? WHERE id=?`,
				fmt.Sprintf("모든 후보 %d개 주문 실패", len(candidates)), selectionLogID)
		}
		e.setState(StateMonitoring)
		return fmt.Errorf("all %d candidates failed", len(candidates))
	}

	logger.Info("engine: order filled",
		map[string]any{
			"stock_code":   stockCode,
			"filled_price": filledPrice,
			"filled_qty":   filledQty,
		})

	// Update selection log with the winning candidate's code and reason.
	if selectionLogID > 0 {
		chosenReason := ""
		for _, cand := range candidates {
			if cand.StockCode == stockCode {
				chosenReason = cand.Reason
				break
			}
		}
		e.db.ExecContext(ctx, //nolint:errcheck
			`UPDATE trader_selection_logs SET selected_code=?, selected_reason=? WHERE id=?`,
			stockCode, chosenReason, selectionLogID)
	}

	// Determine stock name from ranking list.
	stockName := stockCode
	for _, r := range rankings {
		if r.StockCode == stockCode {
			stockName = r.StockName
			break
		}
	}

	// Update DB with fill (stock_name도 함께 갱신).
	e.db.ExecContext(ctx, //nolint:errcheck
		`UPDATE orders SET filled_price = ?, status = ?, stock_name = CASE WHEN stock_name = '' THEN ? ELSE stock_name END WHERE id = ?`,
		filledPrice, string(models.OrderStatusFilled), stockName, result.OrderID)

	// Register with monitor.
	entry := monitor.MonitoredEntry{
		StockCode:          stockCode,
		StockName:          stockName,
		FilledPrice:        filledPrice,
		TargetPrice:        filledPrice * (1 + settings.TakeProfitPct/100),
		StopPrice:          filledPrice * (1 - settings.StopLossPct/100),
		OrderID:            result.OrderID,
		SoldCh:             e.soldCh,
		TrailingTriggerPct: settings.TrailingTriggerPct,
		TrailingStopPct:    settings.TrailingStopPct,
	}
	if regErr := e.mon.Register(ctx, entry); regErr != nil {
		logger.Error("engine: Register position failed",
			map[string]any{"error": regErr.Error()})
	}

	e.setState(StateMonitoring)
	return nil
}

// waitForFill waits for a buy order to be filled.
// - WebSocket 연결 시: 실시간 ExecCh 이벤트 사용 (즉시 감지).
// - WebSocket 미연결 시: KIS API 폴링 폴백 (10초 간격).
// Returns (filledPrice, filledQty, true) on fill, or (0, 0, false) on timeout.
func (e *Engine) waitForFill(ctx context.Context, kisOrderID string) (float64, int, bool) {
	// WebSocket이 연결된 경우 실시간 체결 이벤트 대기.
	if e.wsClient != nil && e.wsClient.IsConnected() {
		for {
			select {
			case <-ctx.Done():
				return 0, 0, false
			case ev, ok := <-e.wsClient.ExecCh:
				if !ok {
					return 0, 0, false
				}
				// Match: same KIS order ID, filled (CntgYN=="2"), buy side (SellBuyDiv=="02").
				if ev.KISOrderID == kisOrderID && ev.CntgYN == "2" && ev.SellBuyDiv == "02" {
					return ev.FilledPrice, ev.FilledQty, true
				}
			}
		}
	}

	// WebSocket 미연결 — KIS API 폴링으로 체결 여부 확인 (10초 간격).
	// 이 경로는 서버가 장중에 시작되어 WebSocket이 아직 연결 안 됐을 때 사용됨.
	logger.Info("waitForFill: WebSocket not connected, falling back to KIS API polling",
		map[string]any{"kis_order_id": kisOrderID})

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return 0, 0, false
		case <-ticker.C:
			if fp, fq, ok := e.pollOrderFill(ctx, kisOrderID); ok {
				return fp, fq, true
			}
		}
	}
}

// pollOrderFill checks whether a buy order has been filled using KIS REST API.
// 1) GetCancellableOrders: 아직 미체결이면 false 반환.
// 2) GetOrderHistory: 체결 내역에서 체결가/수량 조회.
func (e *Engine) pollOrderFill(ctx context.Context, kisOrderID string) (float64, int, bool) {
	// 미체결 주문 목록 조회 — 해당 주문이 있으면 아직 체결 안 됨.
	pending, err := e.kisClient.GetCancellableOrders(ctx)
	if err != nil {
		logger.Warn("pollOrderFill: GetCancellableOrders failed",
			map[string]any{"error": err.Error()})
		return 0, 0, false
	}
	for _, o := range pending {
		if o.Odno == kisOrderID {
			return 0, 0, false // 아직 미체결
		}
	}

	// 미체결 목록에 없음 → 체결됐거나 취소됨. 체결 내역에서 상세 조회.
	kst, _ := time.LoadLocation("Asia/Seoul")
	today := time.Now().In(kst).Format("20060102")
	history, err := e.kisClient.GetOrderHistory(ctx, today, today)
	if err != nil {
		logger.Warn("pollOrderFill: GetOrderHistory failed",
			map[string]any{"error": err.Error()})
		return 0, 0, false
	}
	for _, rec := range history {
		odno, _ := rec["odno"].(string)
		if odno != kisOrderID {
			continue
		}
		// 매수 체결만 처리 (sll_buy_dvsn_cd: "02"=매수).
		sllBuy, _ := rec["sll_buy_dvsn_cd"].(string)
		if sllBuy != "02" {
			continue
		}
		qtyStr, _ := rec["tot_ccld_qty"].(string)
		priceStr, _ := rec["avg_prvs"].(string)
		var qty int
		var price float64
		fmt.Sscanf(qtyStr, "%d", &qty)
		fmt.Sscanf(priceStr, "%f", &price)
		if qty > 0 && price > 0 {
			logger.Info("pollOrderFill: fill confirmed via KIS API",
				map[string]any{"kis_order_id": kisOrderID, "price": price, "qty": qty})
			return price, qty, true
		}
	}
	// 아직 체결 내역에 반영 안 됨 (전파 지연) — 다음 폴링까지 대기.
	return 0, 0, false
}

// countPendingOrders returns the number of today's AGENT buy orders still in PENDING status.
// Used to prevent double-ordering on server restart: if an order was placed but the server
// restarted before the fill, mon.Count() would be 0 but the order is still active in KIS.
func (e *Engine) countPendingOrders(ctx context.Context) int {
	kst, _ := time.LoadLocation("Asia/Seoul")
	today := time.Now().In(kst).Format("2006-01-02")

	var count int
	e.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM orders
		 WHERE date(created_at) = date(?) AND source = 'AGENT'
		   AND order_type = 'BUY' AND status = 'PENDING' AND market = ?`, today, e.market,
	).Scan(&count)
	return count
}

// getTodayTradedCodes returns stock codes that have been traded today from DB.
func (e *Engine) getTodayTradedCodes(ctx context.Context) []string {
	kst, _ := time.LoadLocation("Asia/Seoul")
	today := time.Now().In(kst).Format("2006-01-02")

	rows, err := e.db.QueryContext(ctx,
		`SELECT DISTINCT stock_code FROM orders
		 WHERE date(created_at) = date(?) AND source = 'AGENT' AND market = ?`, today, e.market)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var codes []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err == nil {
			codes = append(codes, code)
		}
	}
	return codes
}

// getRankings calls each configured ranking API, applies per-type filters,
// then returns only stocks that passed ALL enabled ranking types (AND logic).
// Fields from each ranking type are merged into a single RankItem per stock.
// The second return value is the ranking log ID (0 if log insert failed).
func (e *Engine) getRankings(ctx context.Context, settings database.TradingSettings) ([]RankItem, int64, error) {
	if e.market == "US" {
		return e.getRankingsUS(ctx, settings)
	}
	excludeCls := e.db.GetSetting(ctx, "ranking_excl_cls")
	priceMin := settings.RankingPriceMin
	priceMax := settings.RankingPriceMax

	exchanges := settings.RankingExchanges
	if len(exchanges) == 0 {
		exchanges = []string{"0001", "1001"}
	}
	blngClsCodes := settings.RankingVolumeBlngClsCodes
	if len(blngClsCodes) == 0 {
		blngClsCodes = []string{"0", "1", "2", "3", "4"}
	}

	// Price range is enforced client-side after each API call (not passed to API params)
	// to maximise the pool of results fetched before filtering.
	priceMinF, _ := strconv.ParseFloat(priceMin, 64)
	priceMaxF, _ := strconv.ParseFloat(priceMax, 64)
	withinPriceRange := func(currentPrice string) bool {
		p, err := strconv.ParseFloat(currentPrice, 64)
		if err != nil || p <= 0 {
			return false
		}
		if priceMinF > 0 && p < priceMinF {
			return false
		}
		if priceMaxF > 0 && p > priceMaxF {
			return false
		}
		return true
	}

	// byType holds filtered results per ranking type: stockCode → RankItem
	byType := make(map[string]map[string]RankItem) // rankingType → code → item

	for _, rt := range settings.RankingTypes {
		byType[rt] = make(map[string]RankItem)

		switch rt {
		case "volume":
			// exchanges × blng_cls_codes 조합으로 복수 호출 후 dedup
			rawByCode := make(map[string]kis.VolumeRankItem)
			for _, exch := range exchanges {
				for _, blngCls := range blngClsCodes {
					items, err := e.kisClient.GetVolumeRank(ctx, "J", exch, blngCls, "", "", excludeCls)
					if err != nil {
						logger.Warn("engine: GetVolumeRank failed", map[string]any{"exchange": exch, "blng_cls": blngCls, "error": err.Error()})
						continue
					}
					for _, item := range items {
						if _, exists := rawByCode[item.StockCode]; !exists {
							rawByCode[item.StockCode] = item
						}
					}
				}
			}
			dedupedVol := make([]kis.VolumeRankItem, 0, len(rawByCode))
			for _, item := range rawByCode {
				dedupedVol = append(dedupedVol, item)
			}
			count := 0
			for _, item := range dedupedVol {
				if !withinPriceRange(item.CurrentPrice) {
					continue
				}
				if settings.RankingVolumeMinIncrRate > 0 {
					rate, _ := strconv.ParseFloat(item.VolIncrRate, 64)
					if rate < settings.RankingVolumeMinIncrRate {
						continue
					}
				}
				byType[rt][item.StockCode] = RankItem{
					DataRank: item.DataRank, StockCode: item.StockCode,
					StockName: item.StockName, CurrentPrice: item.CurrentPrice,
					Volume: item.Volume, VolIncrRate: item.VolIncrRate,
				}
				count++
				if settings.RankingTopN > 0 && count >= settings.RankingTopN {
					break
				}
			}

		case "strength":
			// 거래소별 복수 호출 후 dedup
			rawByCodeStr := make(map[string]kis.StrengthRankItem)
			for _, exch := range exchanges {
				items, err := e.kisClient.GetStrengthRank(ctx, exch, "", "", excludeCls)
				if err != nil {
					logger.Warn("engine: GetStrengthRank failed", map[string]any{"exchange": exch, "error": err.Error()})
					continue
				}
				for _, item := range items {
					if _, exists := rawByCodeStr[item.StockCode]; !exists {
						rawByCodeStr[item.StockCode] = item
					}
				}
			}
			dedupedStr := make([]kis.StrengthRankItem, 0, len(rawByCodeStr))
			for _, item := range rawByCodeStr {
				dedupedStr = append(dedupedStr, item)
			}
			count := 0
			for _, item := range dedupedStr {
				if !withinPriceRange(item.CurrentPrice) {
					continue
				}
				if settings.RankingStrengthMin > 0 {
					str, _ := strconv.ParseFloat(item.Strength, 64)
					if str < settings.RankingStrengthMin {
						continue
					}
				}
				byType[rt][item.StockCode] = RankItem{
					DataRank: item.DataRank, StockCode: item.StockCode,
					StockName: item.StockName, CurrentPrice: item.CurrentPrice,
					Volume: item.Volume, Strength: item.Strength,
				}
				count++
				if settings.RankingTopN > 0 && count >= settings.RankingTopN {
					break
				}
			}

		case "exec_count":
			// 거래소별 복수 호출 후 dedup
			rawByCodeExec := make(map[string]kis.ExecCountRankItem)
			for _, exch := range exchanges {
				items, err := e.kisClient.GetExecCountRank(ctx, exch, "0", "", "", excludeCls)
				if err != nil {
					logger.Warn("engine: GetExecCountRank failed", map[string]any{"exchange": exch, "error": err.Error()})
					continue
				}
				for _, item := range items {
					if _, exists := rawByCodeExec[item.StockCode]; !exists {
						rawByCodeExec[item.StockCode] = item
					}
				}
			}
			dedupedExec := make([]kis.ExecCountRankItem, 0, len(rawByCodeExec))
			for _, item := range rawByCodeExec {
				dedupedExec = append(dedupedExec, item)
			}
			count := 0
			for _, item := range dedupedExec {
				if !withinPriceRange(item.CurrentPrice) {
					continue
				}
				if settings.RankingExecCountNetBuyOnly {
					netBuy, _ := strconv.ParseFloat(item.NetBuyQty, 64)
					if netBuy <= 0 {
						continue
					}
				}
				byType[rt][item.StockCode] = RankItem{
					DataRank: item.DataRank, StockCode: item.StockCode,
					StockName: item.StockName, CurrentPrice: item.CurrentPrice,
					Volume: item.Volume, NetBuyQty: item.NetBuyQty,
				}
				count++
				if settings.RankingTopN > 0 && count >= settings.RankingTopN {
					break
				}
			}

		case "disparity":
			// 거래소별 복수 호출 후 dedup
			rawByCodeDisp := make(map[string]kis.DisparityRankItem)
			for _, exch := range exchanges {
				items, err := e.kisClient.GetDisparityRank(ctx, exch, "20", "0", "", "", excludeCls)
				if err != nil {
					logger.Warn("engine: GetDisparityRank failed", map[string]any{"exchange": exch, "error": err.Error()})
					continue
				}
				for _, item := range items {
					if _, exists := rawByCodeDisp[item.StockCode]; !exists {
						rawByCodeDisp[item.StockCode] = item
					}
				}
			}
			dedupedDisp := make([]kis.DisparityRankItem, 0, len(rawByCodeDisp))
			for _, item := range rawByCodeDisp {
				dedupedDisp = append(dedupedDisp, item)
			}
			count := 0
			for _, item := range dedupedDisp {
				if !withinPriceRange(item.CurrentPrice) {
					continue
				}
				if settings.RankingDisparityD20Min > 0 || settings.RankingDisparityD20Max > 0 {
					d20, _ := strconv.ParseFloat(item.D20, 64)
					if settings.RankingDisparityD20Min > 0 && d20 < settings.RankingDisparityD20Min {
						continue
					}
					if settings.RankingDisparityD20Max > 0 && d20 > settings.RankingDisparityD20Max {
						continue
					}
				}
				byType[rt][item.StockCode] = RankItem{
					DataRank: item.DataRank, StockCode: item.StockCode,
					StockName: item.StockName, CurrentPrice: item.CurrentPrice,
					Volume: item.Volume, DisparityD20: item.D20,
				}
				count++
				if settings.RankingTopN > 0 && count >= settings.RankingTopN {
					break
				}
			}
		}
	}

	if len(byType) == 0 {
		typesJSON, _ := json.Marshal(settings.RankingTypes)
		_, _ = e.db.InsertRankingLog(ctx, models.TraderRankingLog{
			RankingTypes:      string(typesJSON),
			PriceMin:          settings.RankingPriceMin,
			PriceMax:          settings.RankingPriceMax,
			VolumeCount:       -1,
			StrengthCount:     -1,
			ExecCountCount:    -1,
			DisparityCount:    -1,
			RankingCondition:  settings.RankingCondition,
			IntersectionCount: 0,
			ResultStocks:      "[]",
			ErrorMessage:      "no ranking types configured",
			Market:            "KR",
		})
		return nil, 0, fmt.Errorf("no ranking types configured")
	}

	var result []RankItem

	if settings.RankingCondition == "OR" {
		// OR union: collect stocks appearing in any ranking type.
		seen := map[string]RankItem{}
		for _, m := range byType {
			for code, item := range m {
				if existing, exists := seen[code]; !exists {
					item.RankingType = strings.Join(settings.RankingTypes, "|")
					seen[code] = item
				} else {
					if item.VolIncrRate != "" {
						existing.VolIncrRate = item.VolIncrRate
					}
					if item.Strength != "" {
						existing.Strength = item.Strength
					}
					if item.NetBuyQty != "" {
						existing.NetBuyQty = item.NetBuyQty
					}
					if item.DisparityD20 != "" {
						existing.DisparityD20 = item.DisparityD20
					}
					seen[code] = existing
				}
			}
		}
		for _, item := range seen {
			result = append(result, item)
		}
	} else {
		// AND intersection: keep only stocks present in every enabled ranking type.
		// Use the first type as seed, then filter against the rest.
		var seedType string
		for k := range byType {
			seedType = k
			break
		}

		for code, base := range byType[seedType] {
			inAll := true
			for rt, m := range byType {
				if rt == seedType {
					continue
				}
				if _, ok := m[code]; !ok {
					inAll = false
					break
				}
			}
			if !inAll {
				continue
			}

			// Merge fields from all ranking types into one RankItem.
			merged := base
			merged.RankingType = strings.Join(settings.RankingTypes, "+")
			for rt, m := range byType {
				if rt == seedType {
					continue
				}
				other := m[code]
				if other.VolIncrRate != "" {
					merged.VolIncrRate = other.VolIncrRate
				}
				if other.Strength != "" {
					merged.Strength = other.Strength
				}
				if other.NetBuyQty != "" {
					merged.NetBuyQty = other.NetBuyQty
				}
				if other.DisparityD20 != "" {
					merged.DisparityD20 = other.DisparityD20
				}
			}
			result = append(result, merged)
		}
	}

	logger.Info("engine: rankings result", map[string]any{
		"types": settings.RankingTypes,
		"count": len(result),
	})

	// countFor returns the filtered result count for a type, or -1 if not enabled.
	countFor := func(rt string) int {
		m, ok := byType[rt]
		if !ok {
			return -1
		}
		return len(m)
	}
	typesJSON, _ := json.Marshal(settings.RankingTypes)
	resultStocksJSON, _ := json.Marshal(result)
	rankingLogID, logErr := e.db.InsertRankingLog(ctx, models.TraderRankingLog{
		RankingTypes:      string(typesJSON),
		PriceMin:          settings.RankingPriceMin,
		PriceMax:          settings.RankingPriceMax,
		VolumeCount:       countFor("volume"),
		StrengthCount:     countFor("strength"),
		ExecCountCount:    countFor("exec_count"),
		DisparityCount:    countFor("disparity"),
		RankingCondition:  settings.RankingCondition,
		IntersectionCount: len(result),
		ResultStocks:      string(resultStocksJSON),
		Market:            "KR",
	})
	if logErr != nil {
		logger.Warn("engine: InsertRankingLog failed", map[string]any{"error": logErr.Error()})
	}

	return result, rankingLogID, nil
}

// getRankingsUS fetches overseas volume ranking for US market.
// The second return value is the ranking log ID (0 if log insert failed).
func (e *Engine) getRankingsUS(ctx context.Context, settings database.TradingSettings) ([]RankItem, int64, error) {
	exchanges := settings.USRankingExchanges
	if len(exchanges) == 0 {
		exchanges = []string{"NAS"}
	}
	prc1 := settings.USRankingPriceMin
	prc2 := settings.USRankingPriceMax
	volRang := settings.USRankingVolRang
	if volRang == "" {
		volRang = "0"
	}
	topN := settings.USRankingTopN
	if topN == 0 {
		topN = 20
	}

	// 각 거래소별로 조회 후 합산 (중복 종목은 첫 번째 유지)
	var allItems []RankItem
	seen := make(map[string]bool)
	for _, excd := range exchanges {
		items, err := e.kisClient.GetOverseasVolumeRank(ctx, excd, prc1, prc2, volRang)
		if err != nil {
			logger.Warn("engine US: GetOverseasVolumeRank failed for exchange",
				map[string]any{"exchange": excd, "error": err.Error()})
			continue
		}
		for i, item := range items {
			if topN > 0 && i >= topN {
				break
			}
			if seen[item.Symb] {
				continue
			}
			seen[item.Symb] = true
			allItems = append(allItems, RankItem{
				DataRank:     item.Rank,
				StockCode:    item.Symb,
				StockName:    item.Ename,
				CurrentPrice: item.Last,
				Volume:       item.TVol,
				RankingType:  "us_volume",
				Exchange:     excd,
			})
		}
	}
	if len(allItems) == 0 {
		return nil, 0, fmt.Errorf("GetOverseasVolumeRank: no results from any exchange")
	}
	result := allItems

	typesJSON, _ := json.Marshal(settings.USRankingTypes)
	rankingLogID, logErr := e.db.InsertRankingLog(ctx, models.TraderRankingLog{
		RankingTypes:      string(typesJSON),
		PriceMin:          prc1,
		PriceMax:          prc2,
		VolumeCount:       len(result),
		IntersectionCount: len(result),
		Market:            "US",
	})
	if logErr != nil {
		logger.Warn("engine US: InsertRankingLog failed", map[string]any{"error": logErr.Error()})
	}

	return result, rankingLogID, nil
}

// excdToExchCode converts WebSocket EXCD to order OVRS_EXCG_CD.
func excdToExchCode(excd string) string {
	m := map[string]string{
		"NAS": "NASD", "NYS": "NYSE", "AMS": "AMEX",
		"HKS": "SEHK", "SHS": "SHAA", "SZS": "SZAA", "TSE": "TKSE",
	}
	if code, ok := m[excd]; ok {
		return code
	}
	return excd
}

// selectAndBuyUS is the US market version of selectAndBuy.
// Uses overseas KIS APIs for ranking, price, and order placement.
// selectAndBuyUS runs one US stock-selection and buy cycle.
// force=true skips schedule/market-condition gates (trading days, buy pause).
func (e *Engine) selectAndBuyUS(ctx context.Context, settings database.TradingSettings, force bool) error {
	// 요일 체크 (강제 실행 시 건너뜀)
	if !force && len(settings.TradingDays) > 0 {
		kst, _ := time.LoadLocation("Asia/Seoul")
		today := int(time.Now().In(kst).Weekday())
		allowed := false
		for _, d := range settings.TradingDays {
			if d == today {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("오늘은 거래 제외 요일 (weekday=%d)", today)
		}
	}

	e.setState(StateSelecting)

	// 매수 중단 시간대 체크 (강제 실행 시 건너뜀)
	if !force && settings.BuyPauseStart != "" && settings.BuyPauseEnd != "" {
		now := time.Now().UTC().Format("15:04") // US market hours in UTC
		if now >= settings.BuyPauseStart && now < settings.BuyPauseEnd {
			e.setState(StateMonitoring)
			return fmt.Errorf("매수 중단 시간대 (%s~%s)", settings.BuyPauseStart, settings.BuyPauseEnd)
		}
	}

	// 미장 최대 포지션 수 체크
	{
		usMaxPos := settings.USMaxPositions
		if usMaxPos <= 0 {
			usMaxPos = 1
		}
		currentCount := e.mon.Count() + e.countPendingOrders(ctx)
		if currentCount >= usMaxPos {
			e.setState(StateMonitoring)
			return fmt.Errorf("미장 최대 포지션 수 도달 (%d/%d)", currentCount, usMaxPos)
		}
	}

	excludedCodes := e.getTodayTradedCodes(ctx)

	rankings, rankingLogID, err := e.getRankingsUS(ctx, settings)
	if err != nil {
		e.setState(StateMonitoring)
		return fmt.Errorf("getRankings US: %w", err)
	}
	if len(rankings) == 0 {
		e.setState(StateMonitoring)
		return fmt.Errorf("no US ranking results")
	}

	// allFilteredOut 누적: 모든 필터 단계에서 제거된 종목을 기록 (US 엔진)
	var allFilteredOutUS []filteredStockEntry

	if len(excludedCodes) > 0 {
		excludeSet := make(map[string]bool, len(excludedCodes))
		for _, code := range excludedCodes {
			excludeSet[code] = true
		}
		filtered := make([]RankItem, 0, len(rankings))
		for _, r := range rankings {
			if excludeSet[r.StockCode] {
				allFilteredOutUS = append(allFilteredOutUS, filteredStockEntry{
					StockCode: r.StockCode, StockName: r.StockName,
					FilterReason: "오늘 이미 거래된 종목",
				})
			} else {
				filtered = append(filtered, r)
			}
		}
		rankings = filtered
	}
	if len(rankings) == 0 {
		e.setState(StateMonitoring)
		return fmt.Errorf("no US ranking results after excluding already-traded stocks")
	}

	// 첫 번째 종목의 거래소를 available USD 조회에 사용
	firstExcd := "NAS"
	if len(rankings) > 0 && rankings[0].Exchange != "" {
		firstExcd = rankings[0].Exchange
	}
	firstExchCode := excdToExchCode(firstExcd)

	// Get available USD amount (use first candidate's price for estimate)
	availableUSD := float64(10000) // fallback default
	if len(rankings) > 0 {
		priceResp, priceErr := e.kisClient.GetOverseasPrice(ctx, firstExcd, rankings[0].StockCode)
		if priceErr == nil && priceResp.Last != "" {
			availStr, avErr := e.kisClient.GetOverseasAvailableOrder(ctx, firstExchCode, rankings[0].StockCode, priceResp.Last)
			if avErr == nil {
				if v, pErr := strconv.ParseFloat(availStr, 64); pErr == nil && v > 0 {
					availableUSD = v
				}
			}
		}
	}

	// 일일 최대 손실 한도 체크 (US — USD 기준)
	{
		usDailyMaxLoss := settings.USDailyMaxLossPct
		if usDailyMaxLoss > 0 {
			pnl := e.db.GetTodayRealizedPnLByMarket(ctx, "US")
			if pnl < 0 {
				lossLimit := availableUSD * usDailyMaxLoss / 100
				if -pnl >= lossLimit {
					e.setState(StateMonitoring)
					msg := fmt.Sprintf("미장 일일 최대 손실 한도 도달 ($%.2f 손실 >= 한도 $%.2f)", -pnl, lossLimit)
					e.db.InsertServiceLog(ctx, "TRADER", "ERROR", msg, "")
					return fmt.Errorf("%s", msg)
				}
			}
		}
	}

	// Enrich each candidate with technical indicators via 5-minute bars (MA5/MA20/RSI/MACD/DisparityM5).
	for i, r := range rankings {
		stockExcd := r.Exchange
		if stockExcd == "" {
			stockExcd = firstExcd
		}
		info, enrichErr := agent.GetOverseasStockInfo(ctx, e.kisClient, stockExcd, r.StockCode)
		if enrichErr != nil {
			logger.Warn("engine US: GetOverseasStockInfo failed, skipping indicators",
				map[string]any{"stock_code": r.StockCode, "error": enrichErr.Error()})
			continue
		}
		rankings[i].MA5 = info.MA5
		rankings[i].MA20 = info.MA20
		rankings[i].RSI14 = info.RSI14
		rankings[i].MACDLine = info.MACDLine
		rankings[i].MACDSignal = info.MACDSignal
		rankings[i].DayOpen = info.DayOpen
		rankings[i].DayHigh = info.DayHigh
		rankings[i].DayLow = info.DayLow
		rankings[i].HighPriceDiff = info.HighPriceDiff
		rankings[i].OpenPriceDiff = info.OpenPriceDiff
		rankings[i].DisparityM5 = info.DisparityM5
		rankings[i].VWAP = info.VWAP
		rankings[i].VWAPDiff = info.VWAPDiff
		rankings[i].M5MA10 = info.M5MA10
		rankings[i].PrevVolumeRatio = info.PrevVolumeRatio
		rankings[i].BidAskRatio = info.BidAskRatio
	}

	// 최소 거래대금 필터 (USD)
	usMinTV := settings.USMinTradingValue
	if usMinTV > 0 {
		var passed []RankItem
		for _, item := range rankings {
			price, _ := strconv.ParseFloat(item.CurrentPrice, 64)
			tvol, _ := strconv.ParseFloat(item.Volume, 64)
			tradingValueUSD := price * tvol
			if price <= 0 || tradingValueUSD >= usMinTV {
				passed = append(passed, item)
			} else {
				allFilteredOutUS = append(allFilteredOutUS, filteredStockEntry{
					StockCode:    item.StockCode,
					StockName:    item.StockName,
					FilterReason: fmt.Sprintf("거래대금 미달 ($%.0f < $%.0f)", tradingValueUSD, usMinTV),
				})
			}
		}
		rankings = passed
	}
	if len(rankings) == 0 {
		e.setState(StateMonitoring)
		return fmt.Errorf("no US stocks passed min trading value filter")
	}

	// 하드 필터: LLM 전달 전 부적격 종목 제거 (제거된 종목은 ranking log에 기록)
	{
		filterRsiMax := settings.USFilterRsiMax
		if filterRsiMax == 0 {
			filterRsiMax = 80
		}
		filterDisparityM5Max := settings.USFilterDisparityM5Max
		if filterDisparityM5Max == 0 {
			filterDisparityM5Max = 3.0
		}
		filterHighPriceDiffMin := settings.USFilterHighPriceDiffMin
		if filterHighPriceDiffMin == 0 {
			filterHighPriceDiffMin = -5.0
		}
		filterOpenPriceDiffMax := settings.USFilterOpenPriceDiffMax
		if filterOpenPriceDiffMax == 0 {
			filterOpenPriceDiffMax = 20.0
		}
		var passed []RankItem
		for _, item := range rankings {
			if item.HighPriceDiff != 0 && item.HighPriceDiff < filterHighPriceDiffMin {
				logger.Info("engine US: hard filter high_price_diff", map[string]any{"stock_code": item.StockCode, "high_price_diff": item.HighPriceDiff, "threshold": filterHighPriceDiffMin})
				allFilteredOutUS = append(allFilteredOutUS, filteredStockEntry{StockCode: item.StockCode, StockName: item.StockName, FilterReason: fmt.Sprintf("고가 대비 과다 하락 (%.1f%% < %.1f%%)", item.HighPriceDiff, filterHighPriceDiffMin), HighPriceDiff: item.HighPriceDiff})
				continue
			}
			if item.MA5 > 0 && item.MA20 > 0 && item.MA5 < item.MA20 {
				logger.Info("engine US: hard filter MA trend", map[string]any{"stock_code": item.StockCode, "ma5": item.MA5, "ma20": item.MA20})
				allFilteredOutUS = append(allFilteredOutUS, filteredStockEntry{StockCode: item.StockCode, StockName: item.StockName, FilterReason: fmt.Sprintf("MA 하락추세 (MA5=%.2f < MA20=%.2f)", item.MA5, item.MA20), MA5: item.MA5, MA20: item.MA20})
				continue
			}
			if item.RSI14 > 0 && item.RSI14 >= filterRsiMax {
				logger.Info("engine US: hard filter RSI", map[string]any{"stock_code": item.StockCode, "rsi": item.RSI14, "threshold": filterRsiMax})
				allFilteredOutUS = append(allFilteredOutUS, filteredStockEntry{StockCode: item.StockCode, StockName: item.StockName, FilterReason: fmt.Sprintf("RSI 과열 (%.1f >= %.1f)", item.RSI14, filterRsiMax), RSI14: item.RSI14})
				continue
			}
			if item.DisparityM5 > filterDisparityM5Max {
				logger.Info("engine US: hard filter disparity_m5", map[string]any{"stock_code": item.StockCode, "disparity_m5": item.DisparityM5, "threshold": filterDisparityM5Max})
				allFilteredOutUS = append(allFilteredOutUS, filteredStockEntry{StockCode: item.StockCode, StockName: item.StockName, FilterReason: fmt.Sprintf("5분봉 이격도 초과 (%.1f > %.1f)", item.DisparityM5, filterDisparityM5Max), DisparityM5: item.DisparityM5})
				continue
			}
			if item.OpenPriceDiff > filterOpenPriceDiffMax {
				logger.Info("engine US: hard filter open_price_diff", map[string]any{"stock_code": item.StockCode, "open_price_diff": item.OpenPriceDiff, "threshold": filterOpenPriceDiffMax})
				allFilteredOutUS = append(allFilteredOutUS, filteredStockEntry{StockCode: item.StockCode, StockName: item.StockName, FilterReason: fmt.Sprintf("시가 대비 과다 상승 (%.1f%% > %.1f%%)", item.OpenPriceDiff, filterOpenPriceDiffMax), OpenPriceDiff: item.OpenPriceDiff})
				continue
			}
			passed = append(passed, item)
		}
		rankings = passed
		// 모든 필터 단계 제거 종목을 DB에 기록 (하드필터 + 이전 단계 누적)
		if rankingLogID > 0 && len(allFilteredOutUS) > 0 {
			filteredJSON, _ := json.Marshal(allFilteredOutUS)
			e.db.ExecContext(ctx, `UPDATE trader_ranking_logs SET filtered_stocks=? WHERE id=?`, string(filteredJSON), rankingLogID) //nolint:errcheck
		}
	}
	if len(rankings) == 0 {
		const failMsg = "no US stocks passed hard filter (high-price/MA trend/RSI/disparity/open-price)"
		e.db.ExecContext(ctx, //nolint:errcheck
			`INSERT INTO trader_selection_logs (sent_count, candidates, llm_result, fail_reason, market, ranking_log_id) VALUES (?,?,?,?,?,?)`,
			0, "[]", "", failMsg, "US", rankingLogID)
		e.setState(StateMonitoring)
		return fmt.Errorf("%s", failMsg)
	}

	// Persist selection log
	var selectionLogID int64
	{
		candidatesJSON, _ := json.Marshal(rankings)
		res, dbErr := e.db.ExecContext(ctx,
			`INSERT INTO trader_selection_logs (sent_count, candidates, llm_result, market, ranking_log_id) VALUES (?,?,?,?,?)`,
			len(rankings), string(candidatesJSON), "", "US", rankingLogID)
		if dbErr == nil {
			selectionLogID, _ = res.LastInsertId()
		}
	}

	usRules := DefaultTradingRules()
	usRules.HardDisparityM5Max = settings.USHardDisparityM5Max
	if usRules.HardDisparityM5Max == 0 {
		usRules.HardDisparityM5Max = 3.0
	}
	usRules.HardRSIMax = settings.HardRSIMax
	if usRules.HardRSIMax == 0 {
		usRules.HardRSIMax = 70.0
	}
	usRules.HardOpenPriceDiffMax = settings.USHardOpenPriceDiffMax
	if usRules.HardOpenPriceDiffMax == 0 {
		usRules.HardOpenPriceDiffMax = 15.0
	}
	usRules.RSIBuyMin = settings.RSIBuyMin
	if usRules.RSIBuyMin == 0 {
		usRules.RSIBuyMin = 40.0
	}
	usRules.RSIBuyMax = settings.RSIBuyMax
	if usRules.RSIBuyMax == 0 {
		usRules.RSIBuyMax = 60.0
	}

	candidates, err := e.claude.SelectStocks(ctx, rankings, availableUSD, nil, "US", usRules)
	if err != nil {
		if selectionLogID > 0 {
			e.db.ExecContext(ctx, //nolint:errcheck
				`UPDATE trader_selection_logs SET fail_reason=? WHERE id=?`,
				"LLM 오류: "+err.Error(), selectionLogID)
		}
		e.setState(StateMonitoring)
		return fmt.Errorf("SelectStocks US: %w", err)
	}

	if selectionLogID > 0 {
		llmResultJSON, _ := json.Marshal(candidates)
		e.db.ExecContext(ctx, //nolint:errcheck
			`UPDATE trader_selection_logs SET llm_result=? WHERE id=?`,
			string(llmResultJSON), selectionLogID)
	}

	var (
		stockCode   string
		filledPrice float64
		filledQty   int
		orderID     int64
		kisOrderID  string
	)

	for i, candidate := range candidates {
		code := candidate.StockCode
		logger.Info("engine US: trying candidate",
			map[string]any{"rank": i + 1, "stock_code": code, "reason": candidate.Reason})

		e.setState(StateOrdering)

		// 해당 종목의 거래소 코드 결정
		candExcd := firstExcd
		for _, r := range rankings {
			if r.StockCode == code && r.Exchange != "" {
				candExcd = r.Exchange
				break
			}
		}
		candExchCode := excdToExchCode(candExcd)

		// Get current price
		priceResp, priceErr := e.kisClient.GetOverseasPrice(ctx, candExcd, code)
		if priceErr != nil {
			logger.Warn("engine US: GetOverseasPrice failed, skipping",
				map[string]any{"stock_code": code, "error": priceErr.Error()})
			continue
		}
		currentPrice, pErr := strconv.ParseFloat(priceResp.Last, 64)
		if pErr != nil || currentPrice <= 0 {
			logger.Warn("engine US: invalid price, skipping", map[string]any{"stock_code": code})
			continue
		}

		// Get available USD for this stock
		availStr, _ := e.kisClient.GetOverseasAvailableOrder(ctx, candExchCode, code, priceResp.Last)
		availUSD, _ := strconv.ParseFloat(availStr, 64)
		if availUSD <= 0 {
			availUSD = availableUSD
		}

		usOrderAmt := settings.USOrderAmountPct
		if usOrderAmt <= 0 {
			usOrderAmt = 95
		}
		orderAmt := availUSD * usOrderAmt / 100
		qty := int(orderAmt / currentPrice)
		if qty <= 0 {
			qty = 1
		}

		// Find stock name from rankings
		candidateName := code
		for _, r := range rankings {
			if r.StockCode == code {
				candidateName = r.StockName
				if candidateName == "" {
					candidateName = code
				}
				break
			}
		}

		// Place buy order
		usResp, usErr := e.kisClient.PlaceOverseasBuyOrder(ctx, candExchCode, code, qty, currentPrice)
		if usErr != nil {
			logger.Warn("engine US: PlaceOverseasBuyOrder failed, skipping",
				map[string]any{"stock_code": code, "error": usErr.Error()})
			continue
		}

		// Insert order to DB
		usTakeProfit := settings.USTakeProfitPct
		if usTakeProfit <= 0 {
			usTakeProfit = 3.0
		}
		usStopLoss := settings.USStopLossPct
		if usStopLoss <= 0 {
			usStopLoss = 2.0
		}
		dbRes, dbErr := e.db.ExecContext(ctx,
			`INSERT INTO orders (stock_code, stock_name, order_type, qty, price, status, kis_order_id, source, target_pct, stop_pct, market, created_at)
			 VALUES (?, ?, 'BUY', ?, ?, 'PENDING', ?, 'AGENT', ?, ?, 'US', ?)`,
			code, candidateName, qty, currentPrice, usResp.KISOrderID,
			usTakeProfit, usStopLoss, time.Now().UTC())
		if dbErr != nil {
			logger.Warn("engine US: DB insert failed", map[string]any{"error": dbErr.Error()})
		}
		var dbOrderID int64
		if dbErr == nil {
			dbOrderID, _ = dbRes.LastInsertId()
		}

		e.setState(StateWaitingFill)

		fillCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		fp, fq, filled := e.waitForFill(fillCtx, usResp.KISOrderID)
		cancel()

		if !filled {
			logger.Warn("engine US: fill timeout, trying next candidate",
				map[string]any{"stock_code": code})
			e.db.InsertServiceLog(ctx, "TRADER", "ERROR", "미장 미체결 타임아웃 (5분)", fmt.Sprintf("stock_code=%s order_id=%d", code, dbOrderID))
			e.db.ExecContext(ctx, //nolint:errcheck
				`UPDATE orders SET status = 'FAILED' WHERE id = ?`, dbOrderID)
			continue
		}

		stockCode = code
		filledPrice = fp
		filledQty = fq
		orderID = dbOrderID
		kisOrderID = usResp.KISOrderID
		break
	}

	if stockCode == "" {
		if selectionLogID > 0 {
			e.db.ExecContext(ctx, //nolint:errcheck
				`UPDATE trader_selection_logs SET fail_reason=? WHERE id=?`,
				fmt.Sprintf("모든 US 후보 %d개 주문 실패", len(candidates)), selectionLogID)
		}
		e.setState(StateMonitoring)
		return fmt.Errorf("all US candidates failed")
	}

	_ = kisOrderID

	logger.Info("engine US: order filled",
		map[string]any{"stock_code": stockCode, "filled_price": filledPrice, "filled_qty": filledQty})

	if selectionLogID > 0 {
		chosenReason := ""
		for _, cand := range candidates {
			if cand.StockCode == stockCode {
				chosenReason = cand.Reason
				break
			}
		}
		e.db.ExecContext(ctx, //nolint:errcheck
			`UPDATE trader_selection_logs SET selected_code=?, selected_reason=? WHERE id=?`,
			stockCode, chosenReason, selectionLogID)
	}

	stockName := stockCode
	for _, r := range rankings {
		if r.StockCode == stockCode {
			stockName = r.StockName
			if stockName == "" {
				stockName = stockCode
			}
			break
		}
	}

	// Update order fill in DB
	e.db.ExecContext(ctx, //nolint:errcheck
		`UPDATE orders SET filled_price = ?, status = ? WHERE id = ?`,
		filledPrice, string(models.OrderStatusFilled), orderID)

	// 체결된 종목의 거래소 코드 결정
	filledExcd := firstExcd
	for _, r := range rankings {
		if r.StockCode == stockCode && r.Exchange != "" {
			filledExcd = r.Exchange
			break
		}
	}
	filledExchCode := excdToExchCode(filledExcd)

	usTakeProfitFinal := settings.USTakeProfitPct
	if usTakeProfitFinal <= 0 {
		usTakeProfitFinal = 3.0
	}
	usStopLossFinal := settings.USStopLossPct
	if usStopLossFinal <= 0 {
		usStopLossFinal = 2.0
	}

	// Register with monitor (트레일링 스탑 파라미터 포함)
	entry := monitor.MonitoredEntry{
		StockCode:          stockCode,
		StockName:          stockName,
		FilledPrice:        filledPrice,
		TargetPrice:        filledPrice * (1 + usTakeProfitFinal/100),
		StopPrice:          filledPrice * (1 - usStopLossFinal/100),
		OrderID:            orderID,
		Market:             "US",
		ExchCode:           filledExchCode,
		SoldCh:             e.soldCh,
		TrailingTriggerPct: settings.TrailingTriggerPct,
		TrailingStopPct:    settings.TrailingStopPct,
	}
	if regErr := e.mon.Register(ctx, entry); regErr != nil {
		logger.Error("engine US: Register position failed", map[string]any{"error": regErr.Error()})
	}

	e.setState(StateMonitoring)
	return nil
}
