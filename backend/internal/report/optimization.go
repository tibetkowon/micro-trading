package report

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/micro-trading-for-agent/backend/internal/database"
	"github.com/micro-trading-for-agent/backend/internal/logger"
	"github.com/micro-trading-for-agent/backend/internal/models"
	"github.com/micro-trading-for-agent/backend/internal/trader"
)

// settingsKeyLabel is a human-readable label for each settings key shown in logs.
var settingsKeyLabel = map[string]string{
	"take_profit_pct": "목표 수익률(%)", "stop_loss_pct": "손절률(%)",
	"etf_take_profit_pct": "ETF 목표 수익률(%)", "etf_stop_loss_pct": "ETF 손절률(%)",
	"stock_take_profit_pct": "주식 목표 수익률(%)", "stock_stop_loss_pct": "주식 손절률(%)",
	"order_amount_pct": "주문 금액 비율(%)", "max_positions": "최대 포지션 수",
	"indicator_check_interval_min": "지표 확인 주기(분)", "indicator_rsi_sell_threshold": "RSI 매도 임계값",
	"stagnation_threshold_pct": "횡보 임계값(%)", "stagnation_duration_min": "횡보 감지 시간(분)",
	"trailing_trigger_pct": "트레일링 발동(%)", "trailing_stop_pct": "트레일링 손절(%)",
	"daily_max_loss_pct":    "일 최대 손실(%)",
	"hard_disparity_m5_min": "5분봉 이격도 하한", "hard_disparity_m5_max": "5분봉 이격도 상한",
	"hard_high_price_diff_max": "고가 대비 하락 상한", "hard_high_price_diff_min": "고가 대비 하락 하한",
	"hard_prev_vol_ratio_max": "직전봉 거래량 비율 상한", "hard_strength_min": "체결강도 하한",
	"hard_rsi_max": "RSI 매수 상한", "hard_open_price_diff_max": "시가 대비 상승 상한",
	"hard_macd_bearish_enabled": "MACD 베어리시 차단", "hard_high_formed_mins_max": "고점 경과 시간 상한(분)",
	"vwap_diff_min": "VWAP 이격도 하한", "vwap_diff_max": "VWAP 이격도 상한",
	"rsi_buy_min": "RSI 매수 하한", "rsi_buy_max": "RSI 매수 상한(구간)",
	"bid_ask_ratio_min": "매수호가 비율 하한", "index_drop_threshold_pct": "지수 하락 임계값(%)",
	"min_expected_profit_pct": "최소 기대 수익률(%)",
}

// settingConstraints defines validation bounds for numeric settings keys (auto-apply safety).
// Keys not listed here are string-typed and pass validation without range checks.
var settingConstraints = map[string][2]float64{
	"take_profit_pct": {0.1, 20.0}, "stop_loss_pct": {0.1, 10.0},
	"etf_take_profit_pct": {0.1, 5.0}, "etf_stop_loss_pct": {0.1, 5.0},
	"stock_take_profit_pct": {0.1, 10.0}, "stock_stop_loss_pct": {0.1, 10.0},
	"order_amount_pct": {10.0, 99.0}, "max_positions": {1.0, 10.0},
	"indicator_check_interval_min": {1.0, 60.0}, "indicator_rsi_sell_threshold": {50.0, 90.0},
	"stagnation_threshold_pct": {0.1, 5.0}, "stagnation_duration_min": {5.0, 120.0},
	"trailing_trigger_pct": {0.0, 10.0}, "trailing_stop_pct": {0.1, 5.0},
	"daily_max_loss_pct":    {0.0, 20.0},
	"hard_disparity_m5_min": {-10.0, 0.0}, "hard_disparity_m5_max": {0.0, 10.0},
	"hard_high_price_diff_max": {-5.0, 0.0}, "hard_high_price_diff_min": {-20.0, -0.1},
	"hard_prev_vol_ratio_max": {0.5, 5.0}, "hard_strength_min": {50.0, 150.0},
	"hard_rsi_max": {50.0, 90.0}, "hard_open_price_diff_max": {5.0, 30.0},
	"hard_high_formed_mins_max": {0.0, 240.0},
	"vwap_diff_min": {-5.0, 5.0}, "vwap_diff_max": {0.0, 10.0},
	"rsi_buy_min": {20.0, 60.0}, "rsi_buy_max": {40.0, 80.0},
	"bid_ask_ratio_min": {0.5, 3.0}, "index_drop_threshold_pct": {-5.0, 0.0},
	"min_expected_profit_pct": {0.0, 5.0},
}

// validateSettingValue checks if the suggested value is within allowed range.
// Returns an error if the value is out of bounds; string-typed keys always pass.
func validateSettingValue(key, suggested string) error {
	bounds, ok := settingConstraints[key]
	if !ok {
		return nil // string key — no numeric bounds
	}
	val, err := strconv.ParseFloat(suggested, 64)
	if err != nil {
		return fmt.Errorf("key %s: suggested value %q is not a number", key, suggested)
	}
	if val < bounds[0] || val > bounds[1] {
		return fmt.Errorf("key %s: value %.4f out of allowed range [%.4f, %.4f]", key, val, bounds[0], bounds[1])
	}
	return nil
}

// collectCurrentSettings returns relevant settings key-value pairs for the analysis prompt.
func collectCurrentSettings(ctx context.Context, db *database.DB) map[string]string {
	keys := []string{
		"take_profit_pct", "stop_loss_pct",
		"etf_take_profit_pct", "etf_stop_loss_pct",
		"stock_take_profit_pct", "stock_stop_loss_pct",
		"order_amount_pct", "max_positions",
		"indicator_check_interval_min", "indicator_rsi_sell_threshold",
		"stagnation_threshold_pct", "stagnation_duration_min",
		"trailing_trigger_pct", "trailing_stop_pct",
		"daily_max_loss_pct", "buy_pause_start", "buy_pause_end",
		"hard_disparity_m5_min", "hard_disparity_m5_max",
		"hard_high_price_diff_max", "hard_high_price_diff_min",
		"hard_prev_vol_ratio_max", "hard_strength_min",
		"hard_rsi_max", "hard_open_price_diff_max",
		"hard_macd_bearish_enabled", "hard_high_formed_mins_max",
		"vwap_diff_min", "vwap_diff_max",
		"rsi_buy_min", "rsi_buy_max",
		"bid_ask_ratio_min", "index_drop_threshold_pct",
		"min_expected_profit_pct",
	}
	result := make(map[string]string, len(keys))
	for _, k := range keys {
		v := db.GetSetting(ctx, k)
		label := settingsKeyLabel[k]
		if label == "" {
			label = k
		}
		result[label+"("+k+")"] = v
	}
	return result
}

// applySuggestion applies a single suggestion and updates its status.
// For "settings" category: writes the new value to the DB automatically.
// For "feature" category: marks as APPLIED (manual implementation acknowledged by user).
func applySuggestion(ctx context.Context, db *database.DB, s *models.OptimizationSuggestion) error {
	if s.Category == "feature" {
		s.Status = "APPLIED"
		return nil
	}
	if s.Category != "settings" {
		return fmt.Errorf("unsupported suggestion category: %s", s.Category)
	}
	if err := validateSettingValue(s.Key, s.SuggestedValue); err != nil {
		return err
	}
	if err := db.SetSetting(ctx, s.Key, s.SuggestedValue); err != nil {
		return fmt.Errorf("SetSetting(%s): %w", s.Key, err)
	}
	s.Status = "APPLIED"
	return nil
}

// ApplySuggestionByID finds the suggestion with the given ID in the report for the given date,
// applies it, and persists the updated suggestions back to the DB.
func ApplySuggestionByID(ctx context.Context, db *database.DB, date, suggestionID string) error {
	r, err := db.GetOptimizationReportByDate(ctx, date)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("optimization report not found for date %s", date)
		}
		return err
	}

	var suggestions []models.OptimizationSuggestion
	if err := json.Unmarshal([]byte(r.Suggestions), &suggestions); err != nil {
		return fmt.Errorf("parse suggestions: %w", err)
	}

	found := false
	for i := range suggestions {
		if suggestions[i].ID == suggestionID {
			found = true
			if err := applySuggestion(ctx, db, &suggestions[i]); err != nil {
				return err
			}
			break
		}
	}
	if !found {
		return fmt.Errorf("suggestion %s not found in report for date %s", suggestionID, date)
	}

	updated, _ := json.Marshal(suggestions)
	r.Suggestions = string(updated)
	return db.UpsertOptimizationReport(ctx, *r)
}

// RejectSuggestionByID marks a suggestion as REJECTED without applying it.
func RejectSuggestionByID(ctx context.Context, db *database.DB, date, suggestionID string) error {
	r, err := db.GetOptimizationReportByDate(ctx, date)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("optimization report not found for date %s", date)
		}
		return err
	}

	var suggestions []models.OptimizationSuggestion
	if err := json.Unmarshal([]byte(r.Suggestions), &suggestions); err != nil {
		return fmt.Errorf("parse suggestions: %w", err)
	}

	found := false
	for i := range suggestions {
		if suggestions[i].ID == suggestionID {
			found = true
			suggestions[i].Status = "REJECTED"
			break
		}
	}
	if !found {
		return fmt.Errorf("suggestion %s not found in report for date %s", suggestionID, date)
	}

	updated, _ := json.Marshal(suggestions)
	r.Suggestions = string(updated)
	return db.UpsertOptimizationReport(ctx, *r)
}

// GenerateOptimizationSuggestions runs the full analysis pipeline:
// 1. Load daily report for the given date (generate if missing)
// 2. Call Claude for analysis
// 3. Persist the optimization report
// 4. Auto-apply suggestions based on optimization_apply_mode
//
// date: "YYYY-MM-DD". Empty string defaults to today KST.
func GenerateOptimizationSuggestions(ctx context.Context, db *database.DB, claude *trader.ClaudeClient, date string) error {
	if claude == nil {
		logger.Info("optimization: ANTHROPIC_API_KEY not configured, skipping analysis", nil)
		return nil
	}

	if date == "" {
		kst, _ := time.LoadLocation("Asia/Seoul")
		date = time.Now().In(kst).Format("2006-01-02")
	}

	// Load daily report — generate first if missing
	reports, err := db.GetDailyReports(ctx, date, date, 1)
	if err != nil {
		return fmt.Errorf("GetDailyReports: %w", err)
	}
	if len(reports) == 0 {
		if err := GenerateDailyReport(ctx, db, date); err != nil {
			return fmt.Errorf("GenerateDailyReport: %w", err)
		}
		reports, err = db.GetDailyReports(ctx, date, date, 1)
		if err != nil || len(reports) == 0 {
			return fmt.Errorf("daily report unavailable after generation")
		}
	}
	dr := reports[0]

	if dr.TotalTrades == 0 {
		logger.Info("optimization: no completed trades today, skipping analysis", map[string]any{"date": date})
		return nil
	}

	applyMode := db.GetSetting(ctx, "optimization_apply_mode")
	if applyMode == "" {
		applyMode = "all_manual"
	}

	currentSettings := collectCurrentSettings(ctx, db)

	logger.Info("optimization: starting Claude analysis", map[string]any{
		"date": date, "total_trades": dr.TotalTrades, "apply_mode": applyMode,
	})

	result, err := claude.AnalyzeDailyReport(ctx, dr, currentSettings)
	if err != nil {
		return fmt.Errorf("Claude analysis: %w", err)
	}

	// Assign sequential IDs and set initial status
	for i := range result.Suggestions {
		result.Suggestions[i].ID = strconv.Itoa(i + 1)
		result.Suggestions[i].Status = "PENDING"
	}

	suggestionsJSON, _ := json.Marshal(result.Suggestions)

	optReport := models.OptimizationReport{
		Date:              date,
		OverallAssessment: result.OverallAssessment,
		Suggestions:       string(suggestionsJSON),
		ApplyModeSnapshot: applyMode,
	}

	if err := db.UpsertOptimizationReport(ctx, optReport); err != nil {
		return fmt.Errorf("UpsertOptimizationReport: %w", err)
	}

	logger.Info("optimization: report saved", map[string]any{
		"date": date, "suggestions": len(result.Suggestions),
	})

	// Auto-apply settings suggestions if mode is all_auto
	if applyMode == "all_auto" {
		// Reload from DB to get consistent state
		saved, err := db.GetOptimizationReportByDate(ctx, date)
		if err != nil {
			return fmt.Errorf("reload after save: %w", err)
		}
		var suggestions []models.OptimizationSuggestion
		if err := json.Unmarshal([]byte(saved.Suggestions), &suggestions); err != nil {
			return fmt.Errorf("parse saved suggestions: %w", err)
		}

		appliedCount := 0
		for i := range suggestions {
			if suggestions[i].Category != "settings" {
				continue
			}
			if err := applySuggestion(ctx, db, &suggestions[i]); err != nil {
				logger.Warn("optimization: auto-apply skipped", map[string]any{
					"key": suggestions[i].Key, "error": err.Error(),
				})
			} else {
				appliedCount++
				logger.Info("optimization: auto-applied setting", map[string]any{
					"key": suggestions[i].Key, "value": suggestions[i].SuggestedValue,
				})
			}
		}

		updated, _ := json.Marshal(suggestions)
		saved.Suggestions = string(updated)
		if err := db.UpsertOptimizationReport(ctx, *saved); err != nil {
			return fmt.Errorf("persist auto-apply results: %w", err)
		}
		logger.Info("optimization: auto-apply complete", map[string]any{
			"date": date, "applied": appliedCount,
		})
	}

	return nil
}
