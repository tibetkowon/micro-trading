package simulation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/micro-trading-for-agent/backend/internal/database"
	"github.com/micro-trading-for-agent/backend/internal/kis"
	"github.com/micro-trading-for-agent/backend/internal/models"
)

// ScenarioSummary is the aggregate result for one simulated scenario.
type ScenarioSummary struct {
	Label             string    `json:"label"`
	Params            SimParams `json:"params"`
	TotalPnlPct       float64   `json:"total_pnl_pct"`
	AvgPnlPct         float64   `json:"avg_pnl_pct"`
	WinRatePct        float64   `json:"win_rate_pct"`
	AvgHoldingMinutes float64   `json:"avg_holding_minutes"`
	TradeCount        int       `json:"trade_count"`
	DeltaVsActualPct  float64   `json:"delta_vs_actual_pct"`
	ProfitFactor      float64   `json:"profit_factor"`
	MaxDrawdownPct    float64   `json:"max_drawdown_pct"`
}

// RecommendedSettings is the best-performing parameter set and explanation.
type RecommendedSettings struct {
	Label           string    `json:"label"`
	Params          SimParams `json:"params"`
	Reason          string    `json:"reason"`
	ExpectedGainPct float64   `json:"expected_gain_pct"`
}

type tradeCandles struct {
	trade   models.TradeReport
	candles []MinuteCandle
}

// RunDailySimulation simulates all completed trades for date (YYYY-MM-DD) across scenarios.
func RunDailySimulation(ctx context.Context, db *database.DB, kisClient *kis.Client, date string) error {
	trades, err := db.GetCompletedTradesBySoldDate(ctx, date)
	if err != nil {
		return fmt.Errorf("fetch trades: %w", err)
	}
	if len(trades) == 0 {
		log.Printf("[simulation] no trades on %s, skipping", date)
		return nil
	}

	settings, err := db.GetTradingSettings(ctx)
	if err != nil {
		return fmt.Errorf("fetch settings: %w", err)
	}

	base := SimParams{
		TakeProfitPct:        settings.TakeProfitPct,
		StopLossPct:          settings.StopLossPct,
		TrailingTriggerPct:   settings.TrailingTriggerPct,
		TrailingStopPct:      settings.TrailingStopPct,
		MinScoreThreshold:    settings.MinScoreThreshold,
		UniversalCooldownMin: settings.UniversalCooldownMin,
		WeightStrength:       settings.ScoreWeightStrength,
		WeightRSI:            settings.ScoreWeightRSI,
		WeightMACD:           settings.ScoreWeightMACD,
		WeightBidAsk:         settings.ScoreWeightBidAsk,
		WeightVWAP:           settings.ScoreWeightVWAP,
		WeightVolume:         settings.ScoreWeightVolume,
		WeightProgramBuy:     settings.ScoreWeightProgramBuy,
		WeightMicroBidAsk:    settings.ScoreWeightMicroBidAsk,
		WeightVIDisparity:    settings.ScoreWeightVIDisparity,
	}
	scenarios := GenerateScenarios(base)

	kisDate := strings.ReplaceAll(date, "-", "")
	prepared := make([]tradeCandles, 0, len(trades))
	for _, trade := range trades {
		candles, err := fetchHoldingCandles(ctx, kisClient, trade, kisDate)
		if err != nil || len(candles) == 0 {
			continue
		}
		prepared = append(prepared, tradeCandles{trade: trade, candles: candles})
	}
	if len(prepared) == 0 {
		return fmt.Errorf("no candle data available")
	}
	sortPreparedTrades(prepared)

	var actualGrossPnl float64
	for _, p := range prepared {
		actualGrossPnl += p.trade.ProfitPct
	}
	actualNetPnl := actualGrossPnl - commissionPct*float64(len(prepared))

	summaries := runScenariosParallel(prepared, scenarios, actualNetPnl)

	recommended := pickBestScenario(summaries, base)

	scenariosJSON, _ := json.Marshal(summaries)
	recommendedJSON, _ := json.Marshal(recommended)

	result := models.SimulationResult{
		Date:            date,
		ScenariosJSON:   string(scenariosJSON),
		RecommendedJSON: string(recommendedJSON),
	}
	if err := db.UpsertSimulationResult(ctx, result); err != nil {
		return fmt.Errorf("save simulation result: %w", err)
	}
	log.Printf("[simulation] completed for %s: %d scenarios, best=%q", date, len(summaries), recommended.Label)
	return nil
}

// runScenariosParallel runs scenarios concurrently with a worker pool capped by GOMAXPROCS.
func runScenariosParallel(
	prepared []tradeCandles,
	scenarios []Scenario,
	actualNetPnl float64,
) []ScenarioSummary {
	if len(scenarios) == 0 {
		return nil
	}

	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		workers = 1
	}
	if workers > 8 {
		workers = 8
	}
	if workers > len(scenarios) {
		workers = len(scenarios)
	}

	type job struct {
		index    int
		scenario Scenario
	}
	type result struct {
		index   int
		summary ScenarioSummary
		err     error
	}

	jobs := make(chan job)
	results := make(chan result, len(scenarios))

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				summary, err := runScenarioForTrades(prepared, j.scenario)
				if err == nil {
					summary.DeltaVsActualPct = roundPct(summary.TotalPnlPct - actualNetPnl)
				}
				results <- result{index: j.index, summary: summary, err: err}
			}
		}()
	}

	go func() {
		for i, scenario := range scenarios {
			jobs <- job{index: i, scenario: scenario}
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	byIndex := make([]ScenarioSummary, len(scenarios))
	ok := make([]bool, len(scenarios))
	for r := range results {
		if r.err != nil {
			log.Printf("[simulation] scenario %q skipped: %v", scenarios[r.index].Label, r.err)
			continue
		}
		byIndex[r.index] = r.summary
		ok[r.index] = true
	}

	summaries := make([]ScenarioSummary, 0, len(scenarios))
	for i, summary := range byIndex {
		if ok[i] {
			summaries = append(summaries, summary)
		}
	}
	return summaries
}

func sortPreparedTrades(prepared []tradeCandles) {
	sort.Slice(prepared, func(i, j int) bool {
		return prepared[i].trade.CreatedAt.Before(prepared[j].trade.CreatedAt)
	})
}

func runScenarioForTrades(prepared []tradeCandles, scenario Scenario) (ScenarioSummary, error) {
	var totalPnl, totalHold float64
	var wins, count int
	tradeResults := make([]SimTradeResult, 0, len(prepared))
	pnlSeq := make([]float64, 0, len(prepared))
	lastSoldAt := make(map[string]time.Time)

	for _, item := range prepared {
		if scenario.Params.MinScoreThreshold > 0 && item.trade.BuyIndicators != "" {
			var snap models.BuyIndicatorsSnapshot
			if err := json.Unmarshal([]byte(item.trade.BuyIndicators), &snap); err == nil {
				effectiveScore := recomputeScore(snap, scenario.Params)
				if effectiveScore < scenario.Params.MinScoreThreshold {
					continue
				}
			}
		}

		if scenario.Params.UniversalCooldownMin > 0 {
			if last, ok := lastSoldAt[item.trade.StockCode]; ok {
				cooldown := time.Duration(scenario.Params.UniversalCooldownMin) * time.Minute
				if item.trade.CreatedAt.Sub(last) < cooldown {
					continue
				}
			}
		}

		result := SimulateTrade(item.trade.BuyPrice, item.candles, scenario.Params)
		tradeResults = append(tradeResults, result)
		pnlSeq = append(pnlSeq, result.PnlPct)
		totalPnl += result.PnlPct
		totalHold += float64(result.HoldingCandles)
		count++
		if result.PnlPct > 0 {
			wins++
		}
		if scenario.Params.UniversalCooldownMin > 0 {
			simulatedExitTime := item.trade.CreatedAt.Add(time.Duration(result.HoldingCandles) * time.Minute)
			lastSoldAt[item.trade.StockCode] = simulatedExitTime
		}
	}
	if count == 0 {
		return ScenarioSummary{}, fmt.Errorf("no candle data available")
	}
	return ScenarioSummary{
		Label:             scenario.Label,
		Params:            scenario.Params,
		TotalPnlPct:       roundPct(totalPnl),
		AvgPnlPct:         roundPct(totalPnl / float64(count)),
		WinRatePct:        roundPct(float64(wins) / float64(count) * 100),
		AvgHoldingMinutes: roundPct(totalHold / float64(count)),
		TradeCount:        count,
		ProfitFactor:      calcProfitFactor(tradeResults),
		MaxDrawdownPct:    calcMaxDrawdown(pnlSeq),
	}, nil
}

// fetchHoldingCandles fetches 1-minute candles for a trade's holding period.
func fetchHoldingCandles(ctx context.Context, kisClient *kis.Client, trade models.TradeReport, kisDate string) ([]MinuteCandle, error) {
	if trade.SoldAt == nil {
		return nil, fmt.Errorf("trade not closed")
	}
	kst, _ := time.LoadLocation("Asia/Seoul")
	sellKST := trade.SoldAt.In(kst)
	buyKST := trade.CreatedAt.In(kst)

	inputHour := sellKST.Format("150405")
	bars, err := kisClient.GetDayMinuteChart(ctx, trade.StockCode, kisDate, inputHour)
	if err != nil {
		return nil, err
	}

	for i, j := 0, len(bars)-1; i < j; i, j = i+1, j-1 {
		bars[i], bars[j] = bars[j], bars[i]
	}

	var candles []MinuteCandle
	for _, b := range bars {
		barTime, err := time.ParseInLocation("20060102 150405", kisDate+" "+b.Time, kst)
		if err != nil {
			continue
		}
		if barTime.Before(buyKST) {
			continue
		}
		high, _ := strconv.ParseFloat(b.High, 64)
		low, _ := strconv.ParseFloat(b.Low, 64)
		closePrice, _ := strconv.ParseFloat(b.Close, 64)
		candles = append(candles, MinuteCandle{High: high, Low: low, Close: closePrice})
	}
	return candles, nil
}

func pickBestScenario(summaries []ScenarioSummary, base SimParams) RecommendedSettings {
	if len(summaries) == 0 {
		return RecommendedSettings{Label: "현재 설정", Params: base, Reason: "시뮬레이션 데이터 없음"}
	}
	best := summaries[0]
	for _, s := range summaries[1:] {
		if s.TotalPnlPct > best.TotalPnlPct {
			best = s
		}
	}
	return RecommendedSettings{
		Label:           best.Label,
		Params:          best.Params,
		Reason:          fmt.Sprintf("당일 거래 시뮬레이션 기준 총 손익 최대 (%.2f%%)", best.TotalPnlPct),
		ExpectedGainPct: best.DeltaVsActualPct,
	}
}

// calcMaxDrawdown returns the peak-to-trough maximum drawdown from
// a sequential per-trade PnL series (cumulative equity curve).
func calcMaxDrawdown(pnls []float64) float64 {
	var cum, peak, mdd float64
	for _, p := range pnls {
		cum += p
		if cum > peak {
			peak = cum
		}
		if d := peak - cum; d > mdd {
			mdd = d
		}
	}
	return roundPct(mdd)
}

// calcProfitFactor returns grossWin / abs(grossLoss). Returns 0 if no losses.
func calcProfitFactor(results []SimTradeResult) float64 {
	var win, loss float64
	for _, r := range results {
		if r.PnlPct > 0 {
			win += r.PnlPct
		} else {
			loss += r.PnlPct
		}
	}
	if loss == 0 {
		return 0
	}
	return roundPct(win / math.Abs(loss))
}

// recomputeScore re-weights stored per-indicator scores with scenario weights.
// Falls back to stored TotalScore when ScoreComponents is nil or all weights are zero.
func recomputeScore(snap models.BuyIndicatorsSnapshot, p SimParams) float64 {
	if snap.ScoreComponents == nil {
		return snap.TotalScore
	}
	totalW := float64(p.WeightStrength + p.WeightRSI + p.WeightMACD + p.WeightBidAsk +
		p.WeightVWAP + p.WeightVolume + p.WeightProgramBuy +
		p.WeightMicroBidAsk + p.WeightVIDisparity)
	if totalW == 0 {
		return snap.TotalScore
	}
	sc := snap.ScoreComponents
	raw := sc.Strength*float64(p.WeightStrength) +
		sc.RSI*float64(p.WeightRSI) +
		sc.MACD*float64(p.WeightMACD) +
		sc.BidAsk*float64(p.WeightBidAsk) +
		sc.VWAP*float64(p.WeightVWAP) +
		sc.Volume*float64(p.WeightVolume) +
		sc.ProgramBuy*float64(p.WeightProgramBuy) +
		sc.MicroBidAsk*float64(p.WeightMicroBidAsk) +
		sc.VIDisparity*float64(p.WeightVIDisparity)
	return math.Round(raw/totalW*10) / 10
}
