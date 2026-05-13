package simulation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
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
}

// RecommendedSettings is the best-performing parameter set and explanation.
type RecommendedSettings struct {
	Label           string    `json:"label"`
	Params          SimParams `json:"params"`
	Reason          string    `json:"reason"`
	ExpectedGainPct float64   `json:"expected_gain_pct"`
}

// TradeCandles pairs a completed trade with its fetched minute candles.
type TradeCandles struct {
	trade     models.TradeReport
	candles   []MinuteCandle
	ProfitPct float64
}

// RawBar contains the KIS chart bar fields needed for candle conversion.
type RawBar struct {
	Time  string
	High  string
	Low   string
	Close string
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
		TakeProfitPct:      settings.TakeProfitPct,
		StopLossPct:        settings.StopLossPct,
		TrailingTriggerPct: settings.TrailingTriggerPct,
		TrailingStopPct:    settings.TrailingStopPct,
	}
	scenarios := GenerateScenarios(base)

	kisDate := strings.ReplaceAll(date, "-", "")
	prepared := make([]TradeCandles, 0, len(trades))
	for _, trade := range trades {
		candles, err := fetchHoldingCandles(ctx, kisClient, trade, kisDate)
		if err != nil || len(candles) == 0 {
			continue
		}
		prepared = append(prepared, TradeCandles{
			trade:     trade,
			candles:   candles,
			ProfitPct: trade.ProfitPct,
		})
	}
	if len(prepared) == 0 {
		return fmt.Errorf("no candle data available")
	}

	actualTotalPnl := ComputeActualPnl(prepared)

	summaries := make([]ScenarioSummary, 0, len(scenarios))
	for _, scenario := range scenarios {
		summary, err := runScenarioForTrades(prepared, scenario)
		if err != nil {
			log.Printf("[simulation] scenario %q skipped: %v", scenario.Label, err)
			continue
		}
		summary.DeltaVsActualPct = roundPct(summary.TotalPnlPct - actualTotalPnl)
		summaries = append(summaries, summary)
	}

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

func runScenarioForTrades(prepared []TradeCandles, scenario Scenario) (ScenarioSummary, error) {
	var totalPnl, totalHold float64
	var wins, count int

	for _, item := range prepared {
		result := SimulateTrade(item.trade.BuyPrice, item.candles, scenario.Params)
		totalPnl += result.PnlPct
		totalHold += float64(result.HoldingCandles)
		count++
		if result.PnlPct > 0 {
			wins++
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
	}, nil
}

// fetchHoldingCandles fetches all 1-minute candles covering a trade's holding period.
func fetchHoldingCandles(ctx context.Context, kisClient *kis.Client, trade models.TradeReport, kisDate string) ([]MinuteCandle, error) {
	if trade.SoldAt == nil {
		return nil, fmt.Errorf("trade not closed")
	}
	kst, _ := time.LoadLocation("Asia/Seoul")
	sellKST := trade.SoldAt.In(kst)
	buyKST := trade.CreatedAt.In(kst)

	var allRaw []RawBar
	seen := make(map[string]bool)
	cursor := sellKST

	for {
		bars, err := kisClient.GetDayMinuteChart(ctx, trade.StockCode, kisDate, cursor.Format("150405"))
		if err != nil {
			return nil, err
		}
		if len(bars) == 0 {
			break
		}

		var oldestTime time.Time
		for _, b := range bars {
			if seen[b.Time] {
				continue
			}
			seen[b.Time] = true
			allRaw = append(allRaw, RawBar{
				Time:  b.Time,
				High:  b.High,
				Low:   b.Low,
				Close: b.Close,
			})
			t, err := time.ParseInLocation("20060102 150405", kisDate+" "+b.Time, kst)
			if err == nil && (oldestTime.IsZero() || t.Before(oldestTime)) {
				oldestTime = t
			}
		}

		if oldestTime.IsZero() || !oldestTime.After(buyKST) {
			break
		}
		cursor = oldestTime.Add(-1 * time.Minute)
		if cursor.Before(buyKST) {
			break
		}
	}

	return FilterAndConvertBars(allRaw, kisDate, buyKST, kst)
}

// FilterAndConvertBars deduplicates bars by time, filters bars before buyKST,
// and converts KIS string prices into chronological MinuteCandle values.
func FilterAndConvertBars(bars []RawBar, kisDate string, buyKST time.Time, kst *time.Location) ([]MinuteCandle, error) {
	seen := make(map[string]bool, len(bars))
	filtered := make([]RawBar, 0, len(bars))
	for _, b := range bars {
		if seen[b.Time] {
			continue
		}
		seen[b.Time] = true
		filtered = append(filtered, b)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Time < filtered[j].Time
	})

	candles := make([]MinuteCandle, 0, len(filtered))
	for _, b := range filtered {
		barTime, err := time.ParseInLocation("20060102 150405", kisDate+" "+b.Time, kst)
		if err != nil {
			continue
		}
		if barTime.Before(buyKST) {
			continue
		}
		high, err := strconv.ParseFloat(b.High, 64)
		if err != nil {
			return nil, fmt.Errorf("parse high %q: %w", b.High, err)
		}
		low, err := strconv.ParseFloat(b.Low, 64)
		if err != nil {
			return nil, fmt.Errorf("parse low %q: %w", b.Low, err)
		}
		closePrice, err := strconv.ParseFloat(b.Close, 64)
		if err != nil {
			return nil, fmt.Errorf("parse close %q: %w", b.Close, err)
		}
		candles = append(candles, MinuteCandle{High: high, Low: low, Close: closePrice})
	}
	return candles, nil
}

// ComputeActualPnl returns the sum of actual ProfitPct for the prepared subset.
func ComputeActualPnl(items []TradeCandles) float64 {
	var total float64
	for _, item := range items {
		total += item.ProfitPct
	}
	return total
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
