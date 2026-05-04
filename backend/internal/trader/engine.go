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
	StateIdle        EngineState = "IDLE"
	StateSelecting   EngineState = "SELECTING"
	StateOrdering    EngineState = "ORDERING"
	StateWaitingFill EngineState = "WAITING_FILL"
	StateMonitoring  EngineState = "MONITORING"
	StateSearching   EngineState = "SEARCHING"
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

	mu         sync.RWMutex
	state      EngineState
	haltReason string
	soldCh     chan string
}

// NewEngine creates a new Engine with all required dependencies.
func NewEngine(
	db *database.DB,
	kisClient *kis.Client,
	wsClient *kis.WebSocketClient,
	mon *monitor.Monitor,
	mstStore *stockmaster.Store,
) *Engine {
	return &Engine{
		db:        db,
		kisClient: kisClient,
		wsClient:  wsClient,
		mon:       mon,
		mstStore:  mstStore,
		state:     StateIdle,
		soldCh:    make(chan string, 16),
	}
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
// It runs an initial scan on startup, then rescans on sold signals or every 60 seconds.
func (e *Engine) runCycle(ctx context.Context) {
	logger.Info("engine: cycle started", nil)
	e.tryBuy(ctx)

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			e.setState(StateIdle)
			return
		case code := <-e.soldCh:
			logger.Info("engine: sold signal — rescanning", map[string]any{"stock_code": code})
			time.Sleep(3 * time.Second)
			e.tryBuy(ctx)
		case <-ticker.C:
			e.tryBuy(ctx)
		}
	}
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

	type scored struct {
		cinfo  scorer.CandidateInfo
		detail scorer.ScoreDetail
	}

	var passed []scored
	rejectedCount := 0

	for _, c := range candidates {
		// Skip stocks already being monitored
		if e.mon.Has(c.StockCode) {
			continue
		}

		info, err := ops.GetStockInfo(ctx, e.kisClient, c.StockCode)
		if err != nil {
			logger.Warn("engine: GetStockInfo failed",
				map[string]any{"code": c.StockCode, "error": err.Error()})
			continue
		}

		// Tag asset type from stock master
		if sm, _ := e.mstStore.GetByCode(ctx, c.StockCode); sm != nil {
			switch {
			case sm.IsDomesticEquityETF:
				info.AssetType = "ETF_DOMESTIC"
			case sm.IsETF:
				info.AssetType = "ETF"
			default:
				info.AssetType = "STOCK"
			}
		}

		cinfo := scorer.CandidateInfo{
			StockCode: c.StockCode,
			StockName: c.StockName,
			Info:      info,
			Strength:  c.Strength,
		}

		fr := scorer.ApplyHardFilter(cinfo, settings)
		if !fr.Passed {
			logger.Info("engine: hard filter rejected",
				map[string]any{"code": c.StockCode, "reason": fr.Reason})
			rejectedCount++
			continue
		}

		// Fetch bid/ask ratio for scoring (best-effort; 0 = neutral score if unavailable)
		if settings.ScoreWeightBidAsk > 0 {
			if ratio, err := e.kisClient.GetBidAskRatio(ctx, c.StockCode); err == nil {
				cinfo.Info.BidAskRatio = ratio
			}
		}

		detail := scorer.CalcScore(cinfo, settings)
		passed = append(passed, scored{cinfo, detail})
	}

	// Sort by composite score descending
	sort.Slice(passed, func(i, j int) bool {
		return passed[i].detail.Total > passed[j].detail.Total
	})

	// Build summary for scan log
	var topNames []string
	for i, p := range passed {
		if i >= 3 {
			break
		}
		topNames = append(topNames, fmt.Sprintf("%s(%.1f)", p.cinfo.StockCode, p.detail.Total))
	}

	scanLog := &models.ScanLog{
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		StocksFound: len(passed),
		TopStocks:   strings.Join(topNames, ", "),
	}

	logger.Info("engine: scan complete", map[string]any{
		"total_candidates": len(candidates),
		"passed_filter":    len(passed),
		"rejected":         rejectedCount,
	})

	// Place orders for top-N stocks within the score threshold
	ordered := false
	var orderedCode string
	var skipReason string
	buyCount := 0

	for _, p := range passed {
		if buyCount >= slots {
			break
		}
		if settings.MinScoreThreshold > 0 && p.detail.Total < settings.MinScoreThreshold {
			if buyCount == 0 {
				skipReason = fmt.Sprintf("top score %.1f below threshold %.1f",
					p.detail.Total, settings.MinScoreThreshold)
			}
			break
		}

		logger.Info("engine: placing order", map[string]any{
			"code":  p.cinfo.StockCode,
			"name":  p.cinfo.StockName,
			"score": p.detail.String(),
		})

		if err := e.placeAndMonitor(ctx, p.cinfo, p.detail, settings); err != nil {
			logger.Error("engine: placeAndMonitor failed", map[string]any{
				"code":  p.cinfo.StockCode,
				"error": err.Error(),
			})
			continue
		}

		orderedCode = p.cinfo.StockCode
		ordered = true
		buyCount++
	}

	if len(passed) == 0 {
		skipReason = fmt.Sprintf("no stocks passed hard filter (%d rejected)", rejectedCount)
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
		topN = 20
	}

	// Determine exchange code for ranking APIs.
	// Multiple or no exchange → "0000" (all); single exchange → specific code.
	exchangeCode := exchangeInputCode(settings.RankingExchanges)

	for _, rankType := range settings.RankingTypes {
		name := strings.ToLower(strings.TrimSpace(rankType))

		switch name {
		case "volume":
			items, err := ops.GetVolumeRank(ctx, e.kisClient, "J", exchangeCode, "1", priceMin, priceMax, "")
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
			items, err := ops.GetStrengthRank(ctx, e.kisClient, exchangeCode, priceMin, priceMax, "")
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
			items, err := ops.GetFluctuationRank(ctx, e.kisClient, exchangeCode, priceMin, priceMax, "")
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

	return result, nil
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

// placeAndMonitor places a buy order, waits for fill, then registers the position.
func (e *Engine) placeAndMonitor(
	ctx context.Context,
	c scorer.CandidateInfo,
	detail scorer.ScoreDetail,
	settings database.TradingSettings,
) error {
	e.setState(StateOrdering)

	// Fetch available cash
	bal, err := ops.GetAccountBalance(ctx, e.kisClient, e.db)
	if err != nil {
		return fmt.Errorf("GetAccountBalance: %w", err)
	}

	price, _ := strconv.ParseFloat(c.Info.CurrentPrice, 64)
	if price <= 0 {
		return fmt.Errorf("invalid current price for %s: %q", c.StockCode, c.Info.CurrentPrice)
	}

	orderAmt := bal.WithdrawableAmount * settings.OrderAmountPct / 100
	qty := int(math.Floor(orderAmt / price))
	if qty <= 0 {
		return fmt.Errorf("insufficient cash %.0f KRW for %s at %.0f",
			bal.WithdrawableAmount, c.StockCode, price)
	}

	// Select take-profit / stop-loss by asset type
	takePct := settings.StockTakeProfitPct
	stopPct := settings.StockStopLossPct
	if c.Info.AssetType == "ETF" || c.Info.AssetType == "ETF_DOMESTIC" {
		takePct = settings.ETFTakeProfitPct
		stopPct = settings.ETFStopLossPct
	}

	result, err := ops.PlaceOrder(ctx, e.kisClient, e.db, ops.PlaceOrderRequest{
		StockCode: c.StockCode,
		StockName: c.StockName,
		OrderType: models.OrderTypeBuy,
		Qty:       qty,
		Price:     price,
		OrderDivn: "00", // 지정가
		TargetPct: takePct,
		StopPct:   stopPct,
	})
	if err != nil {
		return fmt.Errorf("PlaceOrder: %w", err)
	}

	logger.Info("engine: buy order placed", map[string]any{
		"code":     c.StockCode,
		"name":     c.StockName,
		"qty":      qty,
		"price":    price,
		"order_id": result.OrderID,
		"kis_id":   result.KISOrderID,
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
		Market:             "KR",
		AssetType:          c.Info.AssetType,
		SoldCh:             e.soldCh,
		TrailingTriggerPct: settings.TrailingTriggerPct,
		TrailingStopPct:    settings.TrailingStopPct,
	}

	if err := e.mon.Register(ctx, entry); err != nil {
		return fmt.Errorf("monitor.Register: %w", err)
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

// waitForFill waits up to 5 minutes for the buy order to be filled via WebSocket.
// Falls back to a 30-second DB poll when wsClient is unavailable.
func (e *Engine) waitForFill(ctx context.Context, kisOrderID string, orderID int64, expectedPrice float64) (float64, error) {
	if e.wsClient == nil {
		// Polling fallback: assume market order fills within 30 s
		time.Sleep(30 * time.Second)
		o, err := e.db.GetOrderByID(ctx, orderID)
		if err != nil || o == nil {
			return 0, fmt.Errorf("order %d not found in DB", orderID)
		}
		if o.Status == models.OrderStatusFilled {
			if o.FilledPrice > 0 {
				return o.FilledPrice, nil
			}
			return expectedPrice, nil
		}
		return 0, fmt.Errorf("order not filled within timeout (status: %s)", o.Status)
	}

	timeout := time.After(5 * time.Minute)
	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-timeout:
			return 0, fmt.Errorf("fill timeout (5m) for KIS order %s", kisOrderID)
		case ev := <-e.wsClient.ExecCh:
			if ev.KISOrderID != kisOrderID || ev.SellBuyDiv != "02" {
				continue // not our buy order
			}
			if ev.FilledPrice > 0 {
				return ev.FilledPrice, nil
			}
			return expectedPrice, nil
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
