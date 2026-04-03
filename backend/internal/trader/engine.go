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
	"github.com/micro-trading-for-agent/backend/internal/mst"
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
	db        *database.DB
	kisClient *kis.Client
	wsClient  *kis.WebSocketClient
	mon       *monitor.Monitor
	claude    *ClaudeClient
	mstStore  *mst.Store // nil-safe: asset type tagging is skipped when nil

	mu         sync.RWMutex
	state      EngineState
	haltReason string      // 마지막 사이클 중지 사유 (성공 시 초기화)
	soldCh     chan string // receives stock_code when monitor executes a sell
	stopCh     chan struct{}

	leaseMu     sync.Mutex
	leaseExpiry map[string]time.Time // stockCode → lease 만료 시각 (순위에서 사라져도 유지)
}

// NewEngine creates a new Engine with all required dependencies.
// claude may be nil if ANTHROPIC_API_KEY is not configured (engine will log an error and sleep).
// mstStore may be nil; asset type tagging is skipped when nil.
func NewEngine(
	db *database.DB,
	kisClient *kis.Client,
	wsClient *kis.WebSocketClient,
	mon *monitor.Monitor,
	claude *ClaudeClient,
	mstStore *mst.Store,
) *Engine {
	return &Engine{
		db:          db,
		kisClient:   kisClient,
		wsClient:    wsClient,
		mon:         mon,
		claude:      claude,
		mstStore:    mstStore,
		state:       StateIdle,
		soldCh:      make(chan string, 16),
		stopCh:      make(chan struct{}),
		leaseExpiry: make(map[string]time.Time),
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
		rankings[i].TradingValue = info.TradingValue
		// MST 기반 자산 타입 태깅
		assetType := e.resolveAssetType(ctx, r.StockCode)
		rankings[i].AssetType = assetType
		// 세금보정: ETF_DOMESTIC/ETF는 비과세, STOCK은 거래세 적용
		stockTaxRate := settings.StockTaxRate
		if stockTaxRate <= 0 {
			stockTaxRate = 0.002
		}
		switch assetType {
		case "ETF_DOMESTIC", "ETF":
			rankings[i].ApplicableTaxRate = 0.0
		default:
			rankings[i].ApplicableTaxRate = stockTaxRate
		}
	}
	// 시가총액 필터 (MST 상장주식수 × 현재가)
	if settings.MinMarketCap > 0 {
		var passed []RankItem
		for _, item := range rankings {
			sm, _ := e.mstStore.GetByCode(ctx, item.StockCode)
			if sm == nil || sm.ListedShares <= 0 {
				// MST 미등록이거나 ListedShares 미파싱 → 필터 통과 (보수적)
				passed = append(passed, item)
				continue
			}
			price, _ := strconv.ParseFloat(item.CurrentPrice, 64)
			marketCapEok := float64(sm.ListedShares) * price / 1e8
			if marketCapEok >= settings.MinMarketCap {
				passed = append(passed, item)
			} else {
				allFilteredOut = append(allFilteredOut, filteredStockEntry{
					StockCode:    item.StockCode,
					StockName:    item.StockName,
					FilterReason: fmt.Sprintf("시가총액 미달 (%.0f억 < %.0f억)", marketCapEok, settings.MinMarketCap),
				})
			}
		}
		rankings = passed
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
	// db.go already applies defaults when key is absent; trust the settings values as-is.
	rules.IndexDropThreshold = settings.IndexDropThresholdPct
	rules.HardDisparityM5Min = settings.HardDisparityM5Min
	rules.HardDisparityM5Max = settings.HardDisparityM5Max
	rules.HardHighPriceDiffMax = settings.HardHighPriceDiffMax
	rules.HardHighPriceDiffMin = settings.HardHighPriceDiffMin
	rules.HardPrevVolRatioMax = settings.HardPrevVolRatioMax
	rules.HardStrengthMin = settings.HardStrengthMin
	rules.HardRSIMax = settings.HardRSIMax
	rules.HardOpenPriceDiffMax = settings.HardOpenPriceDiffMax
	rules.VWAPDiffMin = settings.VWAPDiffMin
	rules.VWAPDiffMax = settings.VWAPDiffMax
	rules.RSIBuyMin = settings.RSIBuyMin
	rules.RSIBuyMax = settings.RSIBuyMax
	rules.BidAskRatioMin = settings.BidAskRatioMin
	rules.MarketIndexDrop = marketIndexDrop
	rules.MinExpectedProfitPct = settings.MinExpectedProfitPct
	rules.StockTaxRate = settings.StockTaxRate
	if rules.StockTaxRate <= 0 {
		rules.StockTaxRate = 0.002
	}

	// Ask Claude to rank all viable candidates (single API call).
	// excludedCodes are already filtered server-side above; pass nil here.
	candidates, err := e.claude.SelectStocks(ctx, rankings, availableCash, nil, rules)
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

		orderAssetType := e.resolveAssetType(ctx, code)
		orderTargetPct, orderStopPct := e.resolveProfitLoss(orderAssetType, settings)
		res, perr := agent.PlaceOrder(ctx, e.kisClient, e.db, agent.PlaceOrderRequest{
			StockCode: code,
			StockName: candidateName,
			OrderType: models.OrderTypeBuy,
			Qty:       qty,
			Price:     0,
			OrderDivn: "01",
			TargetPct: orderTargetPct,
			StopPct:   orderStopPct,
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

	// Resolve take-profit and stop-loss based on asset type.
	assetType := e.resolveAssetType(ctx, stockCode)
	takeProfitPct, stopLossPct := e.resolveProfitLoss(assetType, settings)

	// Register with monitor.
	entry := monitor.MonitoredEntry{
		StockCode:          stockCode,
		StockName:          stockName,
		FilledPrice:        filledPrice,
		TargetPrice:        filledPrice * (1 + takeProfitPct/100),
		StopPrice:          filledPrice * (1 - stopLossPct/100),
		OrderID:            result.OrderID,
		AssetType:          assetType,
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
		   AND order_type = 'BUY' AND status = 'PENDING' AND market = ?`, today, "KR",
	).Scan(&count)
	return count
}

// getTodayTradedCodes returns stock codes that have been traded today from DB.
func (e *Engine) getTodayTradedCodes(ctx context.Context) []string {
	kst, _ := time.LoadLocation("Asia/Seoul")
	today := time.Now().In(kst).Format("2006-01-02")

	rows, err := e.db.QueryContext(ctx,
		`SELECT DISTINCT stock_code FROM orders
		 WHERE date(created_at) = date(?) AND source = 'AGENT' AND market = ?`, today, "KR")
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

		case "fluctuation":
			// 거래소별 복수 호출 후 dedup
			rawByCodeFlt := make(map[string]kis.FluctuationRankItem)
			for _, exch := range exchanges {
				items, err := e.kisClient.GetFluctuationRank(ctx, exch, "", "", excludeCls)
				if err != nil {
					logger.Warn("engine: GetFluctuationRank failed", map[string]any{"exchange": exch, "error": err.Error()})
					continue
				}
				for _, item := range items {
					if _, exists := rawByCodeFlt[item.StockCode]; !exists {
						rawByCodeFlt[item.StockCode] = item
					}
				}
			}
			dedupedFlt := make([]kis.FluctuationRankItem, 0, len(rawByCodeFlt))
			for _, item := range rawByCodeFlt {
				dedupedFlt = append(dedupedFlt, item)
			}
			count := 0
			for _, item := range dedupedFlt {
				if !withinPriceRange(item.CurrentPrice) {
					continue
				}
				// 등락률 범위 필터
				if settings.RankingFluctuationMinRate > 0 || settings.RankingFluctuationMaxRate > 0 {
					rate, _ := strconv.ParseFloat(item.ChangeRate, 64)
					if settings.RankingFluctuationMinRate > 0 && rate < settings.RankingFluctuationMinRate {
						continue
					}
					if settings.RankingFluctuationMaxRate > 0 && rate > settings.RankingFluctuationMaxRate {
						continue
					}
				}
				byType[rt][item.StockCode] = RankItem{
					DataRank: item.DataRank, StockCode: item.StockCode,
					StockName: item.StockName, CurrentPrice: item.CurrentPrice,
					Volume: item.Volume, VolIncrRate: item.ChangeRate,
				}
				count++
				if settings.RankingTopN > 0 && count >= settings.RankingTopN {
					break
				}
			}

		case "vi_status":
			kst, _ := time.LoadLocation("Asia/Seoul")
			dateStr := time.Now().In(kst).Format("20060102")
			viItems, err := agent.GetVIStatus(ctx, e.kisClient, dateStr)
			if err != nil {
				logger.Warn("engine: GetVIStatus failed", map[string]any{"error": err.Error()})
				continue
			}
			count := 0
			for _, item := range viItems {
				if item.ViCnclHour == "" {
					continue // 미해제 건 제외 — 해제 직후 반등 전략
				}
				// VI 종류 필터: "1"=정적, "2"=동적, ""=전체
				if settings.RankingVIKindCode != "" && item.ViKindCode != settings.RankingVIKindCode {
					continue
				}
				if !withinPriceRange(item.CurrentPrice) {
					continue
				}
				byType[rt][item.StockCode] = RankItem{
					StockCode: item.StockCode, StockName: item.StockName,
					CurrentPrice: item.CurrentPrice, RankingType: "vi_status",
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
			}
			result = append(result, merged)
		}
	}

	// Lease TTL: 순위에서 사라진 종목도 lease 만료 전이면 유지
	if settings.RankLeaseDurationMin > 0 {
		now := time.Now()
		leaseDur := time.Duration(settings.RankLeaseDurationMin) * time.Minute

		// 현재 결과에 있는 종목은 lease 갱신
		e.leaseMu.Lock()
		for _, item := range result {
			e.leaseExpiry[item.StockCode] = now.Add(leaseDur)
		}
		// lease 만료 안 된 종목 중 현재 결과에 없는 것 추가
		resultCodes := make(map[string]bool)
		for _, item := range result {
			resultCodes[item.StockCode] = true
		}
		for code, exp := range e.leaseExpiry {
			if !now.Before(exp) {
				delete(e.leaseExpiry, code) // 만료된 lease 정리
				continue
			}
			if !resultCodes[code] {
				result = append(result, RankItem{
					StockCode:   code,
					RankingType: "lease",
				})
			}
		}
		e.leaseMu.Unlock()
	}

	// Hard Watch Symbols: 순위와 무관하게 항상 후보에 포함
	if len(settings.HardWatchSymbols) > 0 {
		resultCodes := make(map[string]bool)
		for _, item := range result {
			resultCodes[item.StockCode] = true
		}
		for _, code := range settings.HardWatchSymbols {
			if !resultCodes[code] {
				result = append(result, RankItem{
					StockCode:   code,
					RankingType: "hard_watch",
				})
			}
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

// resolveProfitLoss returns the take-profit and stop-loss percentages for the given asset type.
// ETF_DOMESTIC uses etf_take_profit_pct/etf_stop_loss_pct; ETF uses the same ETF values;
// STOCK uses stock_take_profit_pct/stock_stop_loss_pct.
// Falls back to legacy TakeProfitPct/StopLossPct when the type-specific value is 0.
func (e *Engine) resolveProfitLoss(assetType string, s database.TradingSettings) (takePct, stopPct float64) {
	switch assetType {
	case "ETF_DOMESTIC", "ETF":
		takePct = s.ETFTakeProfitPct
		stopPct = s.ETFStopLossPct
	default:
		takePct = s.StockTakeProfitPct
		stopPct = s.StockStopLossPct
	}
	if takePct == 0 {
		takePct = s.TakeProfitPct
	}
	if stopPct == 0 {
		stopPct = s.StopLossPct
	}
	return takePct, stopPct
}

// resolveAssetType looks up the stock in stock_masters and returns one of:
// "ETF_DOMESTIC" (국내주식형 ETF, 비과세), "ETF" (기타 ETF, 과세), "STOCK" (일반 주식).
// Returns "STOCK" when mstStore is nil or the stock is not in the table.
func (e *Engine) resolveAssetType(ctx context.Context, stockCode string) string {
	if e.mstStore == nil {
		return "STOCK"
	}
	m, err := e.mstStore.GetByCode(ctx, stockCode)
	if err != nil || m == nil {
		return "STOCK"
	}
	if m.IsETF {
		if m.IsDomesticEquityETF {
			return "ETF_DOMESTIC"
		}
		return "ETF"
	}
	return "STOCK"
}
