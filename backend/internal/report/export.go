package report

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/micro-trading-for-agent/backend/internal/database"
	"github.com/micro-trading-for-agent/backend/internal/models"
)

// ExportReport is a structured period report for LLM analysis.
type ExportReport struct {
	GeneratedAt     string                 `json:"generated_at"`
	Period          ExportPeriod           `json:"period"`
	Summary         ExportSummary          `json:"summary"`
	DailyReports    []ExportDailyEntry     `json:"daily_reports"`
	Trades          []ExportTradeEntry     `json:"trades"`
	CurrentSettings map[string]interface{} `json:"current_settings"`
	SettingsGuide   []SettingsFieldInfo    `json:"settings_guide"`
}

type ExportPeriod struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type ExportSummary struct {
	TotalTrades       int     `json:"total_trades"`
	WinningTrades     int     `json:"winning_trades"`
	LosingTrades      int     `json:"losing_trades"`
	WinRatePct        float64 `json:"win_rate_pct"`
	TotalProfitAmount float64 `json:"total_profit_amount_krw"`
	AvgProfitPct      float64 `json:"avg_profit_pct"`
}

type ExportDailyEntry struct {
	Date          string  `json:"date"`
	TotalTrades   int     `json:"total_trades"`
	WinningTrades int     `json:"winning_trades"`
	LosingTrades  int     `json:"losing_trades"`
	WinRatePct    float64 `json:"win_rate_pct"`
	TotalPnl      float64 `json:"total_pnl_krw"`
	AvgPnlPct     float64 `json:"avg_pnl_pct"`
}

type ExportTradeEntry struct {
	Date            string                 `json:"date"`
	StockCode       string                 `json:"stock_code"`
	StockName       string                 `json:"stock_name"`
	BuyPrice        float64                `json:"buy_price"`
	SellPrice       float64                `json:"sell_price"`
	Qty             int                    `json:"qty"`
	ProfitAmountKRW float64                `json:"profit_amount_krw"`
	ProfitPct       float64                `json:"profit_pct"`
	SellReason      string                 `json:"sell_reason"`
	BuyIndicators   map[string]interface{} `json:"buy_indicators"`
	HoldingMinutes  float64                `json:"holding_minutes"`
}

// SettingsFieldInfo describes a trading setting so LLMs can interpret it.
type SettingsFieldInfo struct {
	Key          string      `json:"key"`
	Description  string      `json:"description"`
	Type         string      `json:"type"`
	CurrentValue interface{} `json:"current_value"`
	MinValue     interface{} `json:"min_value,omitempty"`
	MaxValue     interface{} `json:"max_value,omitempty"`
}

// GenerateExportReport aggregates period reports, trades, and current settings.
// from and to must be formatted as YYYY-MM-DD.
func GenerateExportReport(ctx context.Context, db *database.DB, from, to string) (*ExportReport, error) {
	if from > to {
		return nil, fmt.Errorf("from(%s) must be <= to(%s)", from, to)
	}

	dailyReports, err := db.ListDailyReportsByDateRange(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("fetch daily reports: %w", err)
	}

	var allTrades []models.TradeReport
	for d := from; d <= to; d = nextDate(d) {
		trades, err := db.GetCompletedTradesBySoldDate(ctx, d)
		if err != nil {
			continue
		}
		allTrades = append(allTrades, trades...)
	}

	settings, err := db.GetTradingSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch settings: %w", err)
	}

	return &ExportReport{
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
		Period:          ExportPeriod{From: from, To: to},
		Summary:         buildExportSummary(allTrades, from, to),
		DailyReports:    buildDailyEntries(dailyReports),
		Trades:          buildTradeEntries(allTrades),
		CurrentSettings: buildSettingsMap(&settings),
		SettingsGuide:   buildSettingsGuide(&settings),
	}, nil
}

func buildExportSummary(trades []models.TradeReport, from, to string) ExportSummary {
	var winning, total int
	var totalPnl, totalPnlPct float64
	for _, t := range trades {
		total++
		if t.ProfitAmount > 0 {
			winning++
		}
		totalPnl += t.ProfitAmount
		totalPnlPct += t.ProfitPct
	}
	var winRate, avgPnlPct float64
	if total > 0 {
		winRate = float64(winning) / float64(total) * 100
		avgPnlPct = totalPnlPct / float64(total)
	}
	return ExportSummary{
		TotalTrades:       total,
		WinningTrades:     winning,
		LosingTrades:      total - winning,
		WinRatePct:        round2(winRate),
		TotalProfitAmount: totalPnl,
		AvgProfitPct:      round2(avgPnlPct),
	}
}

func buildDailyEntries(reports []models.DailyReport) []ExportDailyEntry {
	out := make([]ExportDailyEntry, 0, len(reports))
	for _, r := range reports {
		var wr float64
		if r.TotalTrades > 0 {
			wr = float64(r.WinningTrades) / float64(r.TotalTrades) * 100
		}
		out = append(out, ExportDailyEntry{
			Date:          r.Date,
			TotalTrades:   r.TotalTrades,
			WinningTrades: r.WinningTrades,
			LosingTrades:  r.LosingTrades,
			WinRatePct:    round2(wr),
			TotalPnl:      r.TotalProfitAmount,
			AvgPnlPct:     round2(r.AvgProfitPct),
		})
	}
	return out
}

func buildTradeEntries(trades []models.TradeReport) []ExportTradeEntry {
	out := make([]ExportTradeEntry, 0, len(trades))
	for _, t := range trades {
		var holdMin float64
		if t.SoldAt != nil && !t.CreatedAt.IsZero() {
			holdMin = t.SoldAt.Sub(t.CreatedAt).Minutes()
		}
		var indicators map[string]interface{}
		if t.BuyIndicators != "" {
			_ = json.Unmarshal([]byte(t.BuyIndicators), &indicators)
		}
		out = append(out, ExportTradeEntry{
			Date:            t.Date,
			StockCode:       t.StockCode,
			StockName:       t.StockName,
			BuyPrice:        t.BuyPrice,
			SellPrice:       t.SellPrice,
			Qty:             t.BuyQty,
			ProfitAmountKRW: t.ProfitAmount,
			ProfitPct:       t.ProfitPct,
			SellReason:      t.SellReason,
			BuyIndicators:   indicators,
			HoldingMinutes:  round2(holdMin),
		})
	}
	return out
}

func buildSettingsMap(s *database.TradingSettings) map[string]interface{} {
	if s == nil {
		return nil
	}
	return map[string]interface{}{
		"take_profit_pct":           s.TakeProfitPct,
		"stop_loss_pct":             s.StopLossPct,
		"max_positions":             s.MaxPositions,
		"order_amount_pct":          s.OrderAmountPct,
		"trailing_trigger_pct":      s.TrailingTriggerPct,
		"trailing_stop_pct":         s.TrailingStopPct,
		"stagnation_threshold_pct":  s.StagnationThresholdPct,
		"stagnation_duration_min":   s.StagnationDurationMin,
		"min_score_threshold":       s.MinScoreThreshold,
		"hard_rsi_max":              s.HardRSIMax,
		"hard_strength_min":         s.HardStrengthMin,
		"max_consecutive_losses":    s.MaxConsecutiveLosses,
		"daily_max_loss_pct":        s.DailyMaxLossPct,
		"score_weight_strength":     s.ScoreWeightStrength,
		"score_weight_rsi":          s.ScoreWeightRSI,
		"score_weight_macd":         s.ScoreWeightMACD,
		"score_weight_bidask":       s.ScoreWeightBidAsk,
		"score_weight_vwap":         s.ScoreWeightVWAP,
		"score_weight_volume":       s.ScoreWeightVolume,
		"score_weight_program_buy":  s.ScoreWeightProgramBuy,
		"score_weight_micro_bidask": s.ScoreWeightMicroBidAsk,
		"score_weight_vi_disparity": s.ScoreWeightVIDisparity,
		"ranking_top_n":             s.RankingTopN,
		"ranking_condition":         s.RankingCondition,
		"sell_on_upper_limit":       s.SellOnUpperLimit,
		"block_reentry_on_loss":     s.BlockReentryOnLoss,
		"reentry_cooldown_min":      s.ReentryCooldownMin,
		"reentry_score_penalty":     s.ReentryScorePenalty,
	}
}

func buildSettingsGuide(s *database.TradingSettings) []SettingsFieldInfo {
	type fieldDef struct {
		key, desc, typ string
		val            interface{}
		min, max       interface{}
	}
	var cur database.TradingSettings
	if s != nil {
		cur = *s
	}
	defs := []fieldDef{
		{"take_profit_pct", "목표 수익률: 이 비율 이상 수익 시 자동 매도", "float", cur.TakeProfitPct, 0.1, 20.0},
		{"stop_loss_pct", "손절 기준: 이 비율 이상 손실 시 자동 매도", "float", cur.StopLossPct, 0.1, 10.0},
		{"trailing_trigger_pct", "트레일링 스탑 활성화 수익률 (0=비활성)", "float", cur.TrailingTriggerPct, 0.0, 5.0},
		{"trailing_stop_pct", "트레일링 스탑 허용 하락폭 (%)", "float", cur.TrailingStopPct, 0.1, 3.0},
		{"stagnation_threshold_pct", "횡보 판단 가격변동 기준 (0=비활성)", "float", cur.StagnationThresholdPct, 0.0, 1.0},
		{"stagnation_duration_min", "횡보로 인정하는 최소 지속시간 (분)", "int", cur.StagnationDurationMin, 1, 30},
		{"min_score_threshold", "매수 진입 최소 종합 점수 (0~100)", "float", cur.MinScoreThreshold, 0.0, 100.0},
		{"hard_rsi_max", "매수 허용 RSI 최댓값 (초과 시 거부)", "float", cur.HardRSIMax, 50.0, 90.0},
		{"hard_strength_min", "매수 허용 체결강도 최솟값", "float", cur.HardStrengthMin, 80.0, 300.0},
		{"max_positions", "동시 보유 최대 포지션 수", "int", cur.MaxPositions, 1, 10},
		{"order_amount_pct", "주문당 사용 가용현금 비율 (%)", "float", cur.OrderAmountPct, 10.0, 100.0},
		{"max_consecutive_losses", "연속 손실 허용 횟수 (0=비활성)", "int", cur.MaxConsecutiveLosses, 0, 20},
		{"daily_max_loss_pct", "일일 최대 손실률 (0=비활성)", "float", cur.DailyMaxLossPct, 0.0, 10.0},
		{"score_weight_strength", "점수: 체결강도 가중치 (%)", "int", cur.ScoreWeightStrength, 0, 100},
		{"score_weight_rsi", "점수: RSI 가중치 (%)", "int", cur.ScoreWeightRSI, 0, 100},
		{"score_weight_macd", "점수: MACD 가중치 (%)", "int", cur.ScoreWeightMACD, 0, 100},
		{"score_weight_bidask", "점수: 매수/매도 잔량비 가중치 (%)", "int", cur.ScoreWeightBidAsk, 0, 100},
		{"score_weight_vwap", "점수: VWAP 괴리율 가중치 (%)", "int", cur.ScoreWeightVWAP, 0, 100},
		{"score_weight_volume", "점수: 거래량 증가율 가중치 (%)", "int", cur.ScoreWeightVolume, 0, 100},
		{"score_weight_program_buy", "점수: 프로그램 매수 가중치 (%)", "int", cur.ScoreWeightProgramBuy, 0, 100},
		{"score_weight_micro_bidask", "점수: 미시 호가 잔량비 가중치 (%)", "int", cur.ScoreWeightMicroBidAsk, 0, 100},
		{"score_weight_vi_disparity", "점수: VI 괴리율 가중치 (%)", "int", cur.ScoreWeightVIDisparity, 0, 100},
	}
	out := make([]SettingsFieldInfo, 0, len(defs))
	for _, d := range defs {
		out = append(out, SettingsFieldInfo{
			Key:          d.key,
			Description:  d.desc,
			Type:         d.typ,
			CurrentValue: d.val,
			MinValue:     d.min,
			MaxValue:     d.max,
		})
	}
	return out
}

func nextDate(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return t.AddDate(0, 0, 1).Format("2006-01-02")
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
