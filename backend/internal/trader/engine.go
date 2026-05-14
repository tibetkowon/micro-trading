// Package trader implements the autonomous trading engine (v2 — scorer based).
package trader

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"encoding/json"

	"github.com/micro-trading-for-agent/backend/internal/database"
	"github.com/micro-trading-for-agent/backend/internal/kis"
	"github.com/micro-trading-for-agent/backend/internal/logger"
	"github.com/micro-trading-for-agent/backend/internal/models"
	"github.com/micro-trading-for-agent/backend/internal/monitor"
	"github.com/micro-trading-for-agent/backend/internal/ops"
	"github.com/micro-trading-for-agent/backend/internal/scorer"
	"github.com/micro-trading-for-agent/backend/internal/stockmaster"
)

// EngineState represents the current phase of the trading engine.
type EngineState string

const (
	StateIdle             EngineState = "IDLE"
	StateSelecting        EngineState = "SELECTING"
	StateOrdering         EngineState = "ORDERING"
	StateWaitingFill      EngineState = "WAITING_FILL"
	StateMonitoring       EngineState = "MONITORING"
	StateSearching        EngineState = "SEARCHING"
	kisWSMaxSubscriptions             = 40
)

// candidateEntry holds ranking-derived data for one stock before StockInfo is fetched.
type candidateEntry struct {
	StockCode string
	StockName string
	Strength  float64 // 체결강도 from ranking API; 0 if not from strength rank
}

// Engine runs autonomous trading cycles using the rule-based scorer (v2).
type Engine struct {
	db        *database.DB
	kisClient *kis.Client
	wsClient  *kis.WebSocketClient
	mon       *monitor.Monitor
	mstStore  *stockmaster.Store

	mu                  sync.RWMutex
	state               EngineState
	haltReason          string
	soldCh              chan string
	consecutiveLosses   int
	consecutiveLossHalt bool

	hijackCh        chan string
	streamMon       *StreamMonitor
	wsSubscriptions map[string]bool
	recentlySold    sync.Map
}

// NewEngine creates a new Engine with all required dependencies.
func NewEngine(
	db *database.DB,
	kisClient *kis.Client,
	wsClient *kis.WebSocketClient,
	mon *monitor.Monitor,
	mstStore *stockmaster.Store,
) *Engine {
	e := &Engine{
		db:              db,
		kisClient:       kisClient,
		wsClient:        wsClient,
		mon:             mon,
		mstStore:        mstStore,
		state:           StateIdle,
		soldCh:          make(chan string, 16),
		hijackCh:        make(chan string, 100),
		wsSubscriptions: make(map[string]bool),
	}
	e.streamMon = NewStreamMonitor(30000000.0, 5.0, e.hijackCh)
	return e
}

// GetState returns the current engine state (thread-safe).
func (e *Engine) GetState() EngineState {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.state
}

// GetHaltReason returns the reason the last cycle was halted.
func (e *Engine) GetHaltReason() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.haltReason
}

// SoldCh returns the channel the monitor uses to signal a completed sell.
func (e *Engine) SoldCh() chan<- string {
	return e.soldCh
}

// ForceRun triggers an immediate scan cycle in a background goroutine.
func (e *Engine) ForceRun(ctx context.Context) {
	go func() {
		logger.Info("engine: ForceRun triggered", nil)
		e.tryBuy(ctx)
	}()
}

// Start launches the trading cycle goroutine and returns a stop function.
func (e *Engine) Start(ctx context.Context) (stop func()) {
	cycleCtx, cancel := context.WithCancel(ctx)
	go e.runCycle(cycleCtx)
	go e.processPriceEvents(cycleCtx)
	go e.processHijackEvents(cycleCtx)
	return func() {
		cancel()
		e.setState(StateIdle)
		logger.Info("engine: stopped", nil)
	}
}

func (e *Engine) setState(s EngineState) {
	e.mu.Lock()
	e.state = s
	e.mu.Unlock()
	logger.Info("engine: state changed", map[string]any{"state": string(s)})
}

func (e *Engine) setHaltReason(r string) {
	e.mu.Lock()
	e.haltReason = r
	e.mu.Unlock()
}

// runCycle is the main event loop.
// It runs an initial scan on startup, then rescans on sold signals or every scan_interval minutes.
func (e *Engine) runCycle(ctx context.Context) {
	logger.Info("engine: cycle started", nil)
	e.tryBuy(ctx)

	scanInterval := e.loadScanInterval(ctx)
	ticker := time.NewTicker(scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			e.setState(StateIdle)
			return
		case code := <-e.soldCh:
			logger.Info("engine: sold signal — rescanning", map[string]any{"stock_code": code})
			e.recentlySold.Store(code, time.Now())
			time.Sleep(3 * time.Second)
			e.updateConsecutiveLosses(ctx, code)
			// 매도 후 재스캔 시 설정이 바뀌었을 수 있으므로 인터벌 재적용
			newInterval := e.loadScanInterval(ctx)
			if newInterval != scanInterval {
				scanInterval = newInterval
				ticker.Reset(scanInterval)
			}
			e.tryBuy(ctx)
		case <-ticker.C:
			e.tryBuy(ctx)
		}
	}
}

// loadScanInterval reads scan_interval from settings (minutes). Falls back to 1 minute.
func (e *Engine) loadScanInterval(ctx context.Context) time.Duration {
	settings, err := e.db.GetTradingSettings(ctx)
	if err != nil || settings.ScanIntervalMin < 1 {
		return time.Minute
	}
	d := time.Duration(settings.ScanIntervalMin) * time.Minute
	logger.Info("engine: scan interval set", map[string]any{"interval_min": settings.ScanIntervalMin})
	return d
}

// tryBuy reads settings, checks available slots and daily limits, then runs a scan cycle.
func (e *Engine) tryBuy(ctx context.Context) {
	settings, err := e.db.GetTradingSettings(ctx)
	if err != nil {
		logger.Error("engine: GetTradingSettings failed", map[string]any{"error": err.Error()})
		return
	}

	// Check slots
	current := e.mon.Count()
	slots := settings.MaxPositions - current
	if slots <= 0 {
		e.setState(StateMonitoring)
		return
	}

	// Buy-pause window check
	if e.inBuyPause(settings.BuyPauseStart, settings.BuyPauseEnd) {
		logger.Info("engine: in buy-pause window — skipping scan", nil)
		e.setState(StateMonitoring)
		return
	}

	// Daily max-loss check (0 = disabled)
	if settings.DailyMaxLossPct > 0 {
		if e.dailyLossExceeded(ctx, settings.DailyMaxLossPct) {
			e.setHaltReason(fmt.Sprintf("daily loss limit %.1f%% reached", settings.DailyMaxLossPct))
			e.setState(StateMonitoring)
			return
		}
	}

	// Consecutive loss check (0 = disabled)
	if settings.MaxConsecutiveLosses > 0 {
		e.mu.RLock()
		lossHalt := e.consecutiveLossHalt || e.consecutiveLosses >= settings.MaxConsecutiveLosses
		e.mu.RUnlock()
		if lossHalt {
			e.mu.Lock()
			e.consecutiveLossHalt = true
			e.mu.Unlock()
			e.setHaltReason(fmt.Sprintf("consecutive losses %d reached limit %d — halted for today", e.consecutiveLosses, settings.MaxConsecutiveLosses))
			logger.Warn("engine: consecutive loss halt", map[string]any{
				"consecutive_losses": e.consecutiveLosses,
				"limit":              settings.MaxConsecutiveLosses,
			})
			e.setState(StateMonitoring)
			return
		}
	}

	e.setState(StateSelecting)
	e.runScanCycle(ctx, settings, slots)
}

// runScanCycle fetches ranking candidates, applies hard filters, scores,
// then places orders for the top-N stocks up to the available slots.
func (e *Engine) runScanCycle(ctx context.Context, settings database.TradingSettings, slots int) {
	candidates, err := e.fetchCandidates(ctx, settings)
	if err != nil {
		logger.Error("engine: fetchCandidates failed", map[string]any{"error": err.Error()})
		e.setState(StateSearching)
		return
	}
	if len(candidates) == 0 {
		logger.Info("engine: no candidates from rankings", nil)
		e.setState(StateSearching)
		return
	}

	// Subscribe candidates to WebSocket price feed for real-time monitoring
	if settings.StreamBypassEnabled {
		e.streamMon.UpdateConfig(settings.StreamBigTradeAmount, settings.StreamVelocityThreshold)
		newSubs := make(map[string]bool)
		streamSlots := kisWSMaxSubscriptions - e.mon.Count()
		if streamSlots < 0 {
			streamSlots = 0
		}
		for _, c := range candidates {
			if len(newSubs) >= streamSlots {
				break
			}
			newSubs[c.StockCode] = true
		}

		e.mu.Lock()
		for code := range e.wsSubscriptions {
			if !newSubs[code] && !e.mon.Has(code) { // Keep monitored ones if they exist
				e.wsClient.UnsubscribePrice(code)
			}
		}
		for code := range newSubs {
			if !e.wsSubscriptions[code] {
				e.wsClient.SubscribePrice(code)
			}
		}
		e.wsSubscriptions = newSubs
		e.mu.Unlock()
	}

	logger.AutomationInfo("engine: scan started", map[string]any{"candidates": len(candidates)})

	type scored struct {
		cinfo   scorer.CandidateInfo
		detail  scorer.ScoreDetail
		penalty float64
	}

	var passed []scored

	// Per-filter rejection counters for the scan complete summary.
	rejectCounts := map[string]int{
		"reentry_loss":          0,
		"reentry_loss_cooldown": 0,
		"reentry_below_buy":     0,
		"reentry_cooldown":      0,
		"universal_cooldown":    0,
		"universal_price_guard": 0,
		"trading_value":         0,
		"strength":              0,
		"open_price_diff":       0,
		"etn":                   0,
		"hard_filter":           0,
		"spread":                0,
	}

	for _, c := range candidates {
		// Skip stocks already being monitored
		if e.mon.Has(c.StockCode) {
			continue
		}
		if settings.UniversalCooldownMin > 0 {
			if v, ok := e.recentlySold.Load(c.StockCode); ok {
				soldAt := v.(time.Time)
				if elapsed := time.Since(soldAt); elapsed < time.Duration(settings.UniversalCooldownMin)*time.Minute {
					logger.Info("engine: pre-filter rejected (universal cooldown)",
						map[string]any{"code": c.StockCode,
							"elapsed_sec":  int(elapsed.Seconds()),
							"cooldown_min": settings.UniversalCooldownMin})
					rejectCounts["universal_cooldown"]++
					continue
				}
			}
		}

		// ── 1-패스: 재진입 검증 로직 (손절 차단 / 익절 쿨타임 및 페널티) ──────────────────────────
		penalty := 0.0
		var lastLossTrade *models.TradeReport // 손절 쿨타임 통과 후 가격 비교에 재사용
		var lastTrade *models.TradeReport
		trade, tradeErr := e.db.GetLatestCompletedTradeByStock(ctx, c.StockCode)
		if tradeErr != nil {
			logger.Warn("engine: GetLatestCompletedTradeByStock failed — reentry check skipped",
				map[string]any{"symbol": c.StockCode, "error": tradeErr.Error()})
		} else if trade != nil {
			kst, _ := time.LoadLocation("Asia/Seoul")
			today := time.Now().In(kst).Format("2006-01-02")
			if trade.Date == today {
				if trade.ProfitAmount < 0 {
					if settings.BlockReentryOnLoss {
						logger.Info("engine: pre-filter rejected (block reentry on loss)",
							map[string]any{"code": c.StockCode})
						rejectCounts["reentry_loss"]++
						continue
					}
					// 시간 기반 손절 쿨타임 (BlockReentryOnLoss=false일 때 적용)
					if settings.LossCooldownMin > 0 && trade.SoldAt != nil {
						if time.Since(*trade.SoldAt) < time.Duration(settings.LossCooldownMin)*time.Minute {
							logger.Info("engine: pre-filter rejected (loss cooldown)",
								map[string]any{"code": c.StockCode, "loss_cooldown_min": settings.LossCooldownMin,
									"elapsed_sec": int(time.Since(*trade.SoldAt).Seconds())})
							rejectCounts["reentry_loss_cooldown"]++
							continue
						}
					}
					lastLossTrade = trade // 쿨타임 통과 → 이후 가격 비교용 보존
					lastTrade = trade
				} else {
					if trade.SoldAt != nil {
						cooldown := time.Duration(settings.ReentryCooldownMin) * time.Minute
						if time.Since(*trade.SoldAt) < cooldown {
							logger.Info("engine: pre-filter rejected (reentry cooldown)",
								map[string]any{"code": c.StockCode, "cooldown_min": settings.ReentryCooldownMin})
							rejectCounts["reentry_cooldown"]++
							continue
						}
						penalty = settings.ReentryScorePenalty
						lastTrade = trade
						if penalty > 0 {
							logger.Info("engine: reentry penalty will be applied",
								map[string]any{
									"symbol":             c.StockCode,
									"penalty":            penalty,
									"adjusted_threshold": settings.MinScoreThreshold + penalty,
									"sold_at":            trade.SoldAt,
								})
						}
					}
				}
			}
		}

		// ── 1-패스: GetStockPrice(저비용) 사전 필터 ──────────────────────────
		// 차트 fetch(4~5회 API) 전에 현재가만으로 탈락 여부를 선 확인한다.
		priceResp, err := e.kisClient.GetStockPrice(ctx, c.StockCode)
		if err != nil {
			logger.Warn("engine: GetStockPrice (pre-filter) failed",
				map[string]any{"code": c.StockCode, "error": err.Error()})
			continue
		}
		// 손절 재진입 가격 비교: 현재가 < 직전 매수가이면 하락 추세로 차단
		if lastLossTrade != nil && settings.LossReentryPriceGuard {
			currentP, _ := strconv.ParseFloat(priceResp.CurrentPrice, 64)
			if currentP > 0 && lastLossTrade.BuyPrice > 0 && currentP < lastLossTrade.BuyPrice {
				logger.Info("engine: pre-filter rejected (below last buy price)",
					map[string]any{"code": c.StockCode, "current": currentP, "last_buy": lastLossTrade.BuyPrice})
				rejectCounts["reentry_below_buy"]++
				continue
			}
		}
		if settings.UniversalPriceGuard && lastTrade != nil && lastLossTrade == nil {
			currentP, _ := strconv.ParseFloat(priceResp.CurrentPrice, 64)
			if currentP > 0 && lastTrade.BuyPrice > 0 && currentP < lastTrade.BuyPrice {
				logger.Info("engine: pre-filter rejected (universal price guard)",
					map[string]any{"code": c.StockCode, "current": currentP, "last_buy": lastTrade.BuyPrice})
				rejectCounts["universal_price_guard"]++
				continue
			}
		}
		// 최소 거래대금 필터 (0 = 비활성화)
		if settings.MinTradingValue > 0 {
			p, _ := strconv.ParseFloat(priceResp.CurrentPrice, 64)
			v, _ := strconv.ParseFloat(priceResp.Volume, 64)
			if tv := p * v; tv < settings.MinTradingValue {
				logger.Info("engine: pre-filter rejected (trading value)",
					map[string]any{"code": c.StockCode, "trading_value": tv, "min": settings.MinTradingValue})
				rejectCounts["trading_value"]++
				continue
			}
		}
		// 최소 체결강도 필터 (0 = 비활성화)
		if settings.HardStrengthMin > 0 {
			strength, _ := strconv.ParseFloat(priceResp.Strength, 64)
			if strength > 0 && strength < settings.HardStrengthMin {
				logger.Info("engine: pre-filter rejected (strength)",
					map[string]any{"code": c.StockCode, "strength": strength, "min": settings.HardStrengthMin})
				rejectCounts["strength"]++
				continue
			}
		}
		// 최대 시가대비 상승률 필터 (0 = 비활성화)
		if settings.FilterOpenPriceDiffMax > 0 {
			p, _ := strconv.ParseFloat(priceResp.CurrentPrice, 64)
			o, _ := strconv.ParseFloat(priceResp.DayOpen, 64)
			if p > 0 && o > 0 {
				openDiff := (p - o) / o * 100
				if openDiff > settings.FilterOpenPriceDiffMax {
					logger.Info("engine: pre-filter rejected (open price diff)",
						map[string]any{"code": c.StockCode, "open_diff": openDiff, "max": settings.FilterOpenPriceDiffMax})
					rejectCounts["open_price_diff"]++
					continue
				}
			}
		}
		// ── 2-패스: GetStockInfoWithPrice(차트 포함) 전체 지표 ────────────────
		info, err := ops.GetStockInfoWithPrice(ctx, e.kisClient, c.StockCode, priceResp)
		if err != nil {
			logger.Warn("engine: GetStockInfoWithPrice failed",
				map[string]any{"code": c.StockCode, "error": err.Error()})
			continue
		}

		// Tag asset type from stock master
		if sm, _ := e.mstStore.GetByCode(ctx, c.StockCode); sm != nil {
			if sm.IsETN {
				logger.Info("engine: skipping ETN — account not registered for ETN trading",
					map[string]any{"code": c.StockCode, "name": c.StockName})
				rejectCounts["etn"]++
				continue
			}
			switch {
			case sm.IsDomesticEquityETF:
				info.AssetType = "ETF_DOMESTIC"
			case sm.IsETF:
				info.AssetType = "ETF"
			default:
				info.AssetType = "STOCK"
			}
		}

		// StockInfo.Strength(inquire-price 체결강도)를 우선 사용; 없으면 랭킹 API 값 fallback
		strength := info.Strength
		if strength <= 0 {
			strength = c.Strength
		}
		cinfo := scorer.CandidateInfo{
			StockCode: c.StockCode,
			StockName: c.StockName,
			Info:      info,
			Strength:  strength,
		}

		fr := scorer.ApplyHardFilter(cinfo, settings)
		if !fr.Passed {
			logger.Info("engine: hard filter rejected",
				map[string]any{"code": c.StockCode, "reason": fr.Reason})
			rejectCounts["hard_filter"]++
			continue
		}

		// Fetch order book snapshot: bid/ask ratio, micro bid/ask for scoring + spread for filter
		if settings.ScoreWeightBidAsk > 0 || settings.ScoreWeightMicroBidAsk > 0 || settings.MaxBidAskSpreadPct > 0 {
			if snap, err := e.kisClient.GetOrderBookSnapshot(ctx, c.StockCode, 0); err == nil {
				cinfo.Info.BidAskRatio = snap.BidAskRatio
				cinfo.Info.BidAskSpread = snap.SpreadPct
				cinfo.Info.MicroBidAskRatio = snap.MicroBidAskRatio
			}
		}

		// Bid-ask spread filter (0 = disabled)
		if settings.MaxBidAskSpreadPct > 0 && cinfo.Info.BidAskSpread > settings.MaxBidAskSpreadPct {
			logger.Info("engine: spread filter rejected",
				map[string]any{"code": c.StockCode, "spread_pct": cinfo.Info.BidAskSpread, "max": settings.MaxBidAskSpreadPct})
			rejectCounts["spread"]++
			continue
		}

		if info.RSI14 == 0 && info.MACDLine == 0 {
			logger.Warn("engine: chart indicators unavailable (RSI=0, MACD=0) — chart API may have failed",
				map[string]any{"code": c.StockCode})
		}
		detail := scorer.CalcScore(cinfo, settings)
		passed = append(passed, scored{cinfo, detail, penalty})
	}

	totalRejected := 0
	for _, v := range rejectCounts {
		totalRejected += v
	}

	// Sort by composite score descending
	sort.Slice(passed, func(i, j int) bool {
		return passed[i].detail.Total > passed[j].detail.Total
	})

	// Build summary for scan log
	type topStockEntry struct {
		Code          string  `json:"code"`
		Name          string  `json:"name"`
		Strength      float64 `json:"strength"`
		RSI           float64 `json:"rsi"`
		MACDBullish   bool    `json:"macd_bullish"`
		BidAsk        float64 `json:"bid_ask"`
		VWAPDiff      float64 `json:"vwap_diff"`
		VolRatio      float64 `json:"vol_ratio"`
		ProgramNetBuy float64 `json:"program_net_buy"` // 프로그램 순매수 수량 (raw)
		MicroBidAsk   float64 `json:"micro_bid_ask"`   // 미시 호가비율 (raw)
		VIDisparity   float64 `json:"vi_disparity"`    // VI 이격도 % (raw)
		Volume        string  `json:"volume"`          // 거래량 (raw)
		Total         float64 `json:"total"`
		HasChart      bool    `json:"has_chart"` // 차트 API 성공 여부
	}
	topEntries := make([]topStockEntry, 0, 5)
	for i, p := range passed {
		if i >= 5 {
			break
		}
		info := p.cinfo.Info
		topEntries = append(topEntries, topStockEntry{
			Code:          p.cinfo.StockCode,
			Name:          p.cinfo.StockName,
			Strength:      p.cinfo.Strength,
			RSI:           info.RSI14,
			MACDBullish:   info.MACDLine > info.MACDSignal,
			BidAsk:        info.BidAskRatio,
			VWAPDiff:      info.VWAPDiff,
			VolRatio:      info.VolVs3AvgRatio,
			ProgramNetBuy: info.ProgramNetBuy,
			MicroBidAsk:   info.MicroBidAskRatio,
			VIDisparity:   info.VIDisparity,
			Volume:        info.Volume,
			Total:         p.detail.Total,
			HasChart:      info.RSI14 > 0 || info.MACDLine != 0 || info.VWAP > 0,
		})
	}
	topJSON, _ := json.Marshal(topEntries)

	// Build raw data snapshot: full StockInfo + ScoreDetail for top-5 (on-demand drilldown via API)
	type stockRawEntry struct {
		Code   string             `json:"code"`
		Name   string             `json:"name"`
		Info   *ops.StockInfo     `json:"info"`
		Scores scorer.ScoreDetail `json:"scores"`
	}
	rawEntries := make([]stockRawEntry, 0, 5)
	for i, p := range passed {
		if i >= 5 {
			break
		}
		rawEntries = append(rawEntries, stockRawEntry{
			Code:   p.cinfo.StockCode,
			Name:   p.cinfo.StockName,
			Info:   p.cinfo.Info,
			Scores: p.detail,
		})
	}
	rawJSON, _ := json.Marshal(rawEntries)

	scanLog := &models.ScanLog{
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		TotalCandidates: len(candidates),
		StocksFound:     len(passed),
		TopStocks:       string(topJSON),
		StockRawData:    string(rawJSON),
	}

	scanSummaryMsg := fmt.Sprintf(
		"engine: scan complete — passed: %d/%d | reentry: %d, value: %d, strength: %d, open_diff: %d, etn: %d, hard: %d, spread: %d",
		len(passed), len(candidates),
		rejectCounts["reentry_loss"]+rejectCounts["reentry_cooldown"],
		rejectCounts["trading_value"],
		rejectCounts["strength"],
		rejectCounts["open_price_diff"],
		rejectCounts["etn"],
		rejectCounts["hard_filter"],
		rejectCounts["spread"],
	)
	logger.AutomationInfo(scanSummaryMsg, map[string]any{
		"total_candidates": len(candidates),
		"passed_filter":    len(passed),
		"rejected":         totalRejected,
		"reject_by_filter": rejectCounts,
	})

	// Place orders for top-N stocks within the score threshold.
	// Policy: attempt rank-1 first; on failure try the next candidate.
	// Stop as soon as ONE order succeeds (one slot filled per scan cycle
	// avoids stale-balance race conditions and concurrent-order errors).
	ordered := false
	var orderedCode string
	var skipReason string
	buyCount := 0

	for _, p := range passed {
		if buyCount >= slots {
			break
		}
		threshold := settings.MinScoreThreshold + p.penalty
		if threshold > 0 && p.detail.Total < threshold {
			if buyCount == 0 {
				topScore := passed[0].detail.Total
				if p.cinfo.StockCode == passed[0].cinfo.StockCode {
					skipReason = fmt.Sprintf("top score %.1f below threshold %.1f (penalty %.1f)",
						topScore, threshold, p.penalty)
				} else {
					skipReason = fmt.Sprintf("top score %.1f above threshold but all orders failed; next candidate %.1f below threshold %.1f",
						topScore, p.detail.Total, threshold)
				}
				logger.Warn("engine: score threshold not met", map[string]any{
					"top_score":     topScore,
					"failing_score": p.detail.Total,
					"failing_code":  p.cinfo.StockCode,
					"threshold":     threshold,
					"penalty":       p.penalty,
				})
			}
			break
		}

		logger.AutomationInfo(
			fmt.Sprintf("engine: placing order — %s (%s) score: %.1f", p.cinfo.StockName, p.cinfo.StockCode, p.detail.Total),
			map[string]any{"code": p.cinfo.StockCode, "name": p.cinfo.StockName, "score": p.detail.String()},
		)

		if err := e.placeAndMonitor(ctx, p.cinfo, p.detail, settings); err != nil {
			if isInsufficientCashError(err) {
				skipReason = fmt.Sprintf("주문가능금액 부족으로 주문 중단 (종목: %s, 에러: %s)",
					p.cinfo.StockCode, err.Error())
				logger.Warn("engine: insufficient cash — stopping order loop",
					map[string]any{"code": p.cinfo.StockCode, "error": err.Error()})
				break
			}
			logger.Error("engine: placeAndMonitor failed", map[string]any{
				"code":  p.cinfo.StockCode,
				"error": err.Error(),
			})
			continue
		}

		orderedCode = p.cinfo.StockCode
		ordered = true
		buyCount++
		// 1회 스캔당 1종목만 주문: 체결 후 잔고 반영 지연으로 인한
		// APBK0952(주문가능금액 초과) 방지. 다음 슬롯은 다음 스캔 사이클에서 채운다.
		break
	}

	if len(passed) == 0 {
		skipReason = fmt.Sprintf("no stocks passed hard filter (%d rejected)", totalRejected)
	}

	scanLog.Ordered = ordered
	scanLog.OrderedCode = orderedCode
	scanLog.SkipReason = skipReason
	if len(passed) > 0 {
		scanLog.ScoreStats = passed[0].detail.String()
	}
	if _, err := e.db.CreateScanLog(ctx, scanLog); err != nil {
		logger.Warn("engine: CreateScanLog failed", map[string]any{"error": err.Error()})
	}

	if e.mon.Count() >= settings.MaxPositions {
		e.setState(StateMonitoring)
	} else if ordered {
		e.setState(StateMonitoring)
	} else {
		e.setState(StateSearching)
	}
}

// fetchCandidates collects unique stock candidates from all configured ranking types.
// The RankingCondition setting controls whether stocks must appear in ALL (AND) or ANY (OR) types.
func (e *Engine) fetchCandidates(ctx context.Context, settings database.TradingSettings) ([]candidateEntry, error) {
	candidates := make(map[string]*candidateEntry)
	appearances := make(map[string]map[string]bool) // code → set of ranking type names present

	priceMin := settings.RankingPriceMin
	priceMax := settings.RankingPriceMax
	topN := settings.RankingTopN
	if topN <= 0 {
		topN = 30
	}
	excludeCls := settings.RankingExcludeCls
	if excludeCls == "" {
		excludeCls = "1111111111"
	}

	// Determine exchange code for ranking APIs.
	// Multiple or no exchange → "0000" (all); single exchange → specific code.
	exchangeCode := exchangeInputCode(settings.RankingExchanges)

	logger.Info("engine: fetchCandidates params", map[string]any{
		"exclude_cls": excludeCls,
		"price_min":   priceMin,
		"price_max":   priceMax,
		"top_n":       topN,
		"exchanges":   settings.RankingExchanges,
	})

	for _, rankType := range settings.RankingTypes {
		name := strings.ToLower(strings.TrimSpace(rankType))

		switch name {
		case "volume":
			items, err := ops.GetVolumeRank(ctx, e.kisClient, "J", exchangeCode, "1", priceMin, priceMax, excludeCls)
			if err != nil {
				logger.Warn("engine: GetVolumeRank failed", map[string]any{"error": err.Error()})
				continue
			}
			for i, item := range items {
				if i >= topN {
					break
				}
				addCandidate(candidates, appearances, item.StockCode, item.StockName, 0, name)
			}

		case "strength":
			items, err := ops.GetStrengthRank(ctx, e.kisClient, exchangeCode, priceMin, priceMax, excludeCls)
			if err != nil {
				logger.Warn("engine: GetStrengthRank failed", map[string]any{"error": err.Error()})
				continue
			}
			for i, item := range items {
				if i >= topN {
					break
				}
				strength, _ := strconv.ParseFloat(item.Strength, 64)
				addCandidate(candidates, appearances, item.StockCode, item.StockName, strength, name)
			}

		case "fluctuation":
			items, err := ops.GetFluctuationRank(ctx, e.kisClient, exchangeCode, priceMin, priceMax, excludeCls)
			if err != nil {
				logger.Warn("engine: GetFluctuationRank failed", map[string]any{"error": err.Error()})
				continue
			}
			for i, item := range items {
				if i >= topN {
					break
				}
				addCandidate(candidates, appearances, item.StockCode, item.StockName, 0, name)
			}
		}
	}

	// Always add hard_watch_symbols (강제 감시 종목)
	for _, code := range settings.HardWatchSymbols {
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}
		if _, ok := candidates[code]; !ok {
			candidates[code] = &candidateEntry{StockCode: code}
			appearances[code] = make(map[string]bool)
		}
		for _, rt := range settings.RankingTypes {
			appearances[code][strings.ToLower(strings.TrimSpace(rt))] = true
		}
	}

	// Apply AND/OR condition
	isAND := strings.ToUpper(strings.TrimSpace(settings.RankingCondition)) == "AND"

	var result []candidateEntry
	for code, entry := range candidates {
		if isAND && len(settings.RankingTypes) > 1 {
			allPresent := true
			for _, rt := range settings.RankingTypes {
				if !appearances[code][strings.ToLower(strings.TrimSpace(rt))] {
					allPresent = false
					break
				}
			}
			if !allPresent {
				continue
			}
		}
		result = append(result, *entry)
	}

	// Post-filter using stock master to compensate for KIS API not fully honouring
	// fid_trgt_exls_cls_code for certain stock types (ETF, 우선주).
	result = e.postFilterByExcludeCls(ctx, result, excludeCls)

	return result, nil
}

// postFilterByExcludeCls removes candidates whose stock type matches an excluded
// position in the excludeCls string, using local stock master data as the source
// of truth instead of relying solely on the KIS API filter parameter.
func (e *Engine) postFilterByExcludeCls(ctx context.Context, candidates []candidateEntry, excludeCls string) []candidateEntry {
	if len(excludeCls) < 10 {
		return candidates
	}
	excludeETF := excludeCls[8] == '1'
	excludePref := excludeCls[6] == '1'
	if !excludeETF && !excludePref {
		return candidates
	}

	filtered := candidates[:0]
	for _, entry := range candidates {
		sm, _ := e.mstStore.GetByCode(ctx, entry.StockCode)
		if sm != nil && excludeETF && sm.IsETF {
			logger.Info("engine: post-filter excluded ETF",
				map[string]any{"code": entry.StockCode, "name": entry.StockName})
			continue
		}
		if excludePref && isPriorityStock(entry.StockCode) {
			logger.Info("engine: post-filter excluded preferred stock",
				map[string]any{"code": entry.StockCode, "name": entry.StockName})
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

// isPriorityStock returns true for preferred stocks (우선주).
// Korean preferred stock codes are 6 digits with the 5th digit (index 4) being '5'.
// e.g., 005935 = Samsung 우선주 vs 005930 = Samsung 보통주.
func isPriorityStock(code string) bool {
	return len(code) == 6 && code[4] == '5'
}

// addCandidate upserts a candidate into the map, keeping the highest strength seen.
func addCandidate(
	candidates map[string]*candidateEntry,
	appearances map[string]map[string]bool,
	code, name string,
	strength float64,
	rankType string,
) {
	if _, ok := candidates[code]; !ok {
		candidates[code] = &candidateEntry{StockCode: code, StockName: name, Strength: strength}
		appearances[code] = make(map[string]bool)
	} else if strength > candidates[code].Strength {
		candidates[code].Strength = strength
	}
	appearances[code][rankType] = true
}

// isInsufficientCashError reports whether the error is a KIS "insufficient funds" rejection (APBK0952).
func isInsufficientCashError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "APBK0952") || strings.HasPrefix(strings.ToLower(s), "insufficient cash")
}

// placeAndMonitor places a buy order, waits for fill, then registers the position.
func (e *Engine) placeAndMonitor(
	ctx context.Context,
	c scorer.CandidateInfo,
	detail scorer.ScoreDetail,
	settings database.TradingSettings,
) error {
	e.setState(StateOrdering)

	price, _ := strconv.ParseFloat(c.Info.CurrentPrice, 64)
	if price <= 0 {
		return fmt.Errorf("invalid current price for %s: %q", c.StockCode, c.Info.CurrentPrice)
	}

	// TTTC8908R(매수가능조회) ord_psbl_cash: 대기 주문을 포함한 실시간 주문가능금액.
	// inquire-balance의 prvs_rcdl_excc_amt(D+2 예수금)는 당일 체결을 즉시 반영하지
	// 않아 연속 주문 시 APBK0952(주문가능금액 초과) 오류가 발생하므로 이 값을 사용한다.
	feasibility, err := ops.CheckOrderFeasibility(ctx, e.kisClient, c.StockCode)
	if err != nil {
		return fmt.Errorf("CheckOrderFeasibility: %w", err)
	}
	orderAmt := feasibility.AvailableCash * settings.OrderAmountPct / 100
	qty := int(math.Floor(orderAmt / price))
	if qty <= 0 {
		return fmt.Errorf("insufficient cash %.0f KRW for %s at %.0f",
			feasibility.AvailableCash, c.StockCode, price)
	}

	// 잔고 스냅샷은 별도로 저장 (대시보드 및 PnL 계산용).
	if _, err2 := ops.GetAccountBalance(ctx, e.kisClient, e.db); err2 != nil {
		logger.Warn("engine: balance snapshot failed (non-critical)", map[string]any{"error": err2.Error()})
	}

	// Select take-profit / stop-loss by asset type
	takePct := settings.StockTakeProfitPct
	stopPct := settings.StockStopLossPct
	if c.Info.AssetType == "ETF" || c.Info.AssetType == "ETF_DOMESTIC" {
		takePct = settings.ETFTakeProfitPct
		stopPct = settings.ETFStopLossPct
	}

	// Determine order type and price from buy_order_type setting.
	// "limit"(default): 현재가 지정가  "ask1": 매도1호가 지정가  "ask2": 매도2호가 지정가  "market": 순수 시장가
	orderDivn := "00" // 지정가
	orderPrice := price
	switch settings.BuyOrderType {
	case "market":
		orderDivn = "01"
		orderPrice = 0
	case "ask1", "ask2":
		snap, snapErr := e.kisClient.GetOrderBookSnapshot(ctx, c.StockCode, price)
		if snapErr != nil {
			logger.Warn("engine: GetOrderBookSnapshot for ask-price failed — falling back to limit", map[string]any{
				"code": c.StockCode, "error": snapErr.Error(),
			})
		} else {
			askPrice := snap.AskP1
			if settings.BuyOrderType == "ask2" && snap.AskP2 > 0 {
				askPrice = snap.AskP2
			}
			if askPrice > 0 {
				orderPrice = askPrice
			}
		}
	}

	result, err := ops.PlaceOrder(ctx, e.kisClient, e.db, ops.PlaceOrderRequest{
		StockCode: c.StockCode,
		StockName: c.StockName,
		OrderType: models.OrderTypeBuy,
		Qty:       qty,
		Price:     orderPrice,
		OrderDivn: orderDivn,
		TargetPct: takePct,
		StopPct:   stopPct,
	})
	if err != nil {
		return fmt.Errorf("PlaceOrder: %w", err)
	}

	logger.Info("engine: buy order placed", map[string]any{
		"code":       c.StockCode,
		"name":       c.StockName,
		"qty":        qty,
		"price":      orderPrice,
		"order_type": settings.BuyOrderType,
		"order_divn": orderDivn,
		"order_id":   result.OrderID,
		"kis_id":     result.KISOrderID,
	})

	// Wait for fill via WebSocket or polling fallback
	e.setState(StateWaitingFill)
	filledPrice, err := e.waitForFill(ctx, result.KISOrderID, result.OrderID, price)
	if err != nil {
		logger.Warn("engine: fill wait failed — cancelling order", map[string]any{
			"code":  c.StockCode,
			"error": err.Error(),
		})
		if cancelErr := e.cancelOrder(ctx, result.OrderID); cancelErr != nil {
			logger.Error("engine: cancel failed", map[string]any{"error": cancelErr.Error()})
		}
		return fmt.Errorf("waitForFill: %w", err)
	}

	targetPrice := filledPrice * (1 + takePct/100)
	stopPrice := filledPrice * (1 - stopPct/100)

	entry := monitor.MonitoredEntry{
		StockCode:          c.StockCode,
		StockName:          c.StockName,
		FilledPrice:        filledPrice,
		TargetPrice:        targetPrice,
		StopPrice:          stopPrice,
		OrderID:            result.OrderID,
		Qty:                qty,
		Market:             "KR",
		AssetType:          c.Info.AssetType,
		SoldCh:             e.soldCh,
		TrailingTriggerPct: settings.TrailingTriggerPct,
		TrailingStopPct:    settings.TrailingStopPct,
		SellOnUpperLimit:   settings.SellOnUpperLimit,
		TrailingMode:       settings.TrailingMode,
		TickTrail: monitor.TickTrailState{
			Tier0StopLossTicks: settings.TickTier0StopLossTicks,
			Tier1TriggerPct:    settings.TickTier1TriggerPct,
			Tier1TrailTicks:    settings.TickTier1TrailTicks,
			Tier2TriggerPct:    settings.TickTier2TriggerPct,
			Tier2TrailTicks:    settings.TickTier2TrailTicks,
		},
	}

	// 상한가 가격 조회 (설정 활성화 시)
	if settings.SellOnUpperLimit {
		if priceResp, err := e.kisClient.GetStockPrice(ctx, c.StockCode); err == nil {
			if up, err2 := strconv.ParseFloat(priceResp.UpperLimitPrice, 64); err2 == nil && up > 0 {
				entry.UpperLimitPrice = up
			}
		}
	}

	if err := e.mon.Register(ctx, entry); err != nil {
		return fmt.Errorf("monitor.Register: %w", err)
	}

	kst, _ := time.LoadLocation("Asia/Seoul")
	indicatorsJSON, _ := json.Marshal(models.BuyIndicatorsSnapshot{
		RSI:           c.Info.RSI14,
		MACDBullish:   c.Info.MACDLine > c.Info.MACDSignal,
		VWAPDisparity: c.Info.VWAPDiff,
		Strength:      c.Strength,
		BidAskRatio:   c.Info.BidAskRatio,
		TotalScore:    detail.Total,
	})
	report := &models.TradeReport{
		Date:          time.Now().In(kst).Format("2006-01-02"),
		StockCode:     c.StockCode,
		StockName:     c.StockName,
		BuyOrderID:    result.OrderID,
		BuyPrice:      filledPrice,
		BuyQty:        qty,
		BuyAmount:     filledPrice * float64(qty),
		BuyReason:     detail.String(),
		BuyIndicators: string(indicatorsJSON),
	}
	if _, err := e.db.CreateTradeReport(ctx, report); err != nil {
		logger.Error("engine: CreateTradeReport failed", map[string]any{
			"code":  c.StockCode,
			"error": err.Error(),
		})
	}

	logger.Info("engine: position registered", map[string]any{
		"code":         c.StockCode,
		"filled_price": filledPrice,
		"target_price": targetPrice,
		"stop_price":   stopPrice,
	})

	e.setState(StateMonitoring)
	return nil
}

// waitForFill waits for the buy order to be filled.
// Primary: listens on wsClient.ExecCh (H0STCNI0 실시간체결통보), up to 5 minutes.
// Fallback (ws nil): polls TTTC0081R every 5 s for up to 5 minutes.
func (e *Engine) waitForFill(ctx context.Context, kisOrderID string, orderID int64, expectedPrice float64) (float64, error) {
	const fillTimeout = 5 * time.Minute

	if e.wsClient == nil {
		return e.pollFillStatus(ctx, kisOrderID, expectedPrice, 5*time.Second, fillTimeout)
	}

	timeout := time.After(fillTimeout)
	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-timeout:
			return 0, fmt.Errorf("fill timeout (%s) for KIS order %s", fillTimeout, kisOrderID)
		case ev := <-e.wsClient.ExecCh:
			if ev.KISOrderID != kisOrderID || ev.SellBuyDiv != "02" {
				continue
			}
			logger.Info("engine: fill confirmed via WebSocket", map[string]any{
				"kis_order_id": kisOrderID,
				"filled_price": ev.FilledPrice,
			})
			if ev.FilledPrice > 0 {
				return ev.FilledPrice, nil
			}
			return expectedPrice, nil
		}
	}
}

// pollFillStatus polls TTTC0081R at the given interval until the order is fully
// filled or the timeout expires.
func (e *Engine) pollFillStatus(ctx context.Context, kisOrderID string, expectedPrice float64, interval, timeout time.Duration) (float64, error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return 0, fmt.Errorf("fill poll timeout (%s) for KIS order %s", timeout, kisOrderID)
			}
			item, err := e.kisClient.GetOrderFillStatus(ctx, kisOrderID)
			if err != nil {
				logger.Warn("engine: fill poll error", map[string]any{
					"kis_order_id": kisOrderID, "error": err.Error(),
				})
				continue
			}
			if item == nil {
				continue
			}
			ordQty, _ := strconv.Atoi(item.OrdQty)
			ccldQty, _ := strconv.Atoi(item.TotCcldQty)
			if ordQty > 0 && ccldQty >= ordQty {
				filledPrice, _ := strconv.ParseFloat(item.AvgPrvs, 64)
				if filledPrice <= 0 {
					filledPrice = expectedPrice
				}
				logger.Info("engine: fill confirmed via poll", map[string]any{
					"kis_order_id": kisOrderID,
					"filled_price": filledPrice,
					"filled_qty":   ccldQty,
				})
				if err := e.db.UpdateOrderFilled(ctx, kisOrderID, models.OrderStatusFilled, filledPrice); err != nil {
					logger.Warn("engine: UpdateOrderFilled failed (non-critical)", map[string]any{"error": err.Error()})
				}
				return filledPrice, nil
			}
		}
	}
}

func (e *Engine) cancelOrder(ctx context.Context, orderID int64) error {
	_, err := ops.CancelOrder(ctx, e.kisClient, e.db, orderID)
	return err
}

// inBuyPause reports whether the current KST time falls within the configured pause window.
func (e *Engine) inBuyPause(start, end string) bool {
	if start == "" || end == "" {
		return false
	}
	kst, _ := time.LoadLocation("Asia/Seoul")
	now := time.Now().In(kst)
	hhmm := now.Hour()*100 + now.Minute()
	startHHMM := parseHHMM(start, 0)
	endHHMM := parseHHMM(end, 0)
	if startHHMM == 0 || endHHMM == 0 || startHHMM >= endHHMM {
		return false
	}
	return hhmm >= startHHMM && hhmm < endHHMM
}

// resetConsecutiveLosses resets the counter at engine start (called by main.go scheduler).
func (e *Engine) ResetConsecutiveLosses() {
	e.mu.Lock()
	e.consecutiveLosses = 0
	e.consecutiveLossHalt = false
	e.mu.Unlock()
}

// updateConsecutiveLosses checks the latest trade result for a stock and updates the counter.
func (e *Engine) updateConsecutiveLosses(ctx context.Context, stockCode string) {
	report, err := e.db.GetLatestCompletedTradeByStock(ctx, stockCode)
	if err != nil || report == nil {
		return
	}
	settings, err := e.db.GetTradingSettings(ctx)
	if err != nil || settings.MaxConsecutiveLosses <= 0 {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if report.ProfitAmount < 0 {
		e.consecutiveLosses++
		logger.Warn("engine: consecutive loss recorded", map[string]any{
			"stock_code":         stockCode,
			"profit_amount":      report.ProfitAmount,
			"consecutive_losses": e.consecutiveLosses,
		})
	} else if settings.ConsecutiveLossResetOnProfit {
		e.consecutiveLosses = 0
		logger.Info("engine: consecutive loss counter reset (profit)", map[string]any{"stock_code": stockCode})
	}
}

// dailyLossExceeded checks whether today's realised P&L has hit the max-loss limit.
func (e *Engine) dailyLossExceeded(ctx context.Context, maxLossPct float64) bool {
	entries, err := e.db.GetDailyPnL(ctx, 1)
	if err != nil || len(entries) == 0 {
		return false // on error or no data, allow trading
	}
	todayPnL := entries[0].ProfitAmount
	if todayPnL >= 0 {
		return false
	}
	bal, err := e.db.GetLatestBalance(ctx)
	if err != nil || bal == nil || bal.TotalEval <= 0 {
		return false
	}
	lossPct := math.Abs(todayPnL) / bal.TotalEval * 100
	return lossPct >= maxLossPct
}

// parseHHMM converts "HH:MM" to an integer (e.g. "09:15" → 915). Returns def on error.
func parseHHMM(s string, def int) int {
	if s == "" {
		return def
	}
	t, err := time.Parse("15:04", s)
	if err != nil {
		return def
	}
	return t.Hour()*100 + t.Minute()
}

// exchangeInputCode maps a list of exchange names to the KIS API code.
// KOSPI only → "0001", KOSDAQ only → "1001", otherwise → "0000" (all).
func exchangeInputCode(exchanges []string) string {
	if len(exchanges) == 1 {
		switch strings.ToUpper(strings.TrimSpace(exchanges[0])) {
		case "KOSPI":
			return "0001"
		case "KOSDAQ":
			return "1001"
		}
	}
	return "0000"
}

// ─── WebSocket Bypass / Hijack Logic ────────────────────────────────────────

func (e *Engine) processPriceEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-e.wsClient.PriceCh:
			if e.streamMon != nil {
				e.streamMon.AddTick(ev)
			}
		}
	}
}

func (e *Engine) processHijackEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case stockCode := <-e.hijackCh:
			go e.tryBuyHijack(ctx, stockCode)
		}
	}
}

func (e *Engine) tryBuyHijack(ctx context.Context, stockCode string) {
	settings, err := e.db.GetTradingSettings(ctx)
	if err != nil || !settings.StreamBypassEnabled {
		return
	}

	// 1. Check slots and conditions
	if settings.MaxPositions-e.mon.Count() <= 0 {
		return
	}
	if e.inBuyPause(settings.BuyPauseStart, settings.BuyPauseEnd) {
		return
	}
	if settings.DailyMaxLossPct > 0 && e.dailyLossExceeded(ctx, settings.DailyMaxLossPct) {
		return
	}

	logger.Info("engine: hijack triggered", map[string]any{"code": stockCode})

	// 2. Fetch data & check hard filters
	priceResp, err := e.kisClient.GetStockPrice(ctx, stockCode)
	if err != nil {
		logger.Warn("engine: hijack GetStockPrice failed", map[string]any{"code": stockCode, "error": err.Error()})
		return
	}
	info, err := ops.GetStockInfoWithPrice(ctx, e.kisClient, stockCode, priceResp)
	if err != nil {
		logger.Warn("engine: hijack GetStockInfoWithPrice failed", map[string]any{"code": stockCode, "error": err.Error()})
		return
	}

	cinfo := scorer.CandidateInfo{
		StockCode: stockCode,
		StockName: stockCode, // Fallback
		Info:      info,
		Strength:  info.Strength,
	}

	if sm, _ := e.mstStore.GetByCode(ctx, stockCode); sm != nil {
		cinfo.StockName = sm.StockName
		if sm.IsETN {
			logger.Info("engine: hijack rejected (ETN)", map[string]any{"code": stockCode})
			return
		}
		switch {
		case sm.IsDomesticEquityETF:
			info.AssetType = "ETF_DOMESTIC"
		case sm.IsETF:
			info.AssetType = "ETF"
		default:
			info.AssetType = "STOCK"
		}
	}

	// Must pass hard filter
	fr := scorer.ApplyHardFilter(cinfo, settings)
	if !fr.Passed {
		logger.Info("engine: hijack hard filter rejected", map[string]any{"code": stockCode, "reason": fr.Reason})
		return
	}

	// 3. Force Buy (Bypass Scoring)
	// Assign artificial 100 score for bypassed stock.
	detail := scorer.ScoreDetail{Total: 100.0}

	logger.Info("engine: executing hijack buy", map[string]any{"code": stockCode, "name": cinfo.StockName})
	if err := e.placeAndMonitor(ctx, cinfo, detail, settings); err != nil {
		logger.Error("engine: hijack placeAndMonitor failed", map[string]any{"code": stockCode, "error": err.Error()})
	}
}
