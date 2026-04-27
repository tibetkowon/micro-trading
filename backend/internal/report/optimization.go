package report

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/micro-trading-for-agent/backend/internal/database"
	"github.com/micro-trading-for-agent/backend/internal/logger"
	"github.com/micro-trading-for-agent/backend/internal/models"
	"github.com/micro-trading-for-agent/backend/internal/trader"
)

// settingsKeyLabel is a human-readable label for each settings key shown in logs.
// Keys not listed here fall back to the raw key name — still included in the analysis.
var settingsKeyLabel = map[string]string{
	// 수익/손절
	"take_profit_pct": "목표 수익률(%)", "stop_loss_pct": "손절률(%)",
	"etf_take_profit_pct": "ETF 목표 수익률(%)", "etf_stop_loss_pct": "ETF 손절률(%)",
	"stock_take_profit_pct": "주식 목표 수익률(%)", "stock_stop_loss_pct": "주식 손절률(%)",
	"stock_tax_rate": "주식 세율",
	// 트레일링/일손실
	"trailing_trigger_pct": "트레일링 발동(%)", "trailing_stop_pct": "트레일링 손절(%)",
	"daily_max_loss_pct": "일 최대 손실(%)",
	// 주문
	"order_amount_pct": "주문 금액 비율(%)", "max_positions": "최대 포지션 수",
	// 지표 매도
	"indicator_check_interval_min": "지표 확인 주기(분)", "indicator_rsi_sell_threshold": "RSI 매도 임계값",
	"indicator_macd_bearish_sell": "MACD 데드크로스 매도",
	// 횡보
	"stagnation_threshold_pct": "횡보 임계값(%)", "stagnation_duration_min": "횡보 감지 시간(분)",
	"stagnation_partial_exit_enabled": "횡보 부분 매도 활성", "stagnation_bid_ask_sell_threshold": "횡보 호가비율 매도 임계",
	// 부분 익절
	"partial_tp_enabled": "부분 익절 활성", "partial_tp_pct": "부분 익절 수익률(%)",
	"partial_tp_ratio": "부분 익절 비율", "partial_tp_raise_stop": "부분 익절 후 손절 상향",
	// 복합 스코어링 가중치
	"scoring_bidask_weight": "호가비율 가중치", "scoring_strength_weight": "체결강도 가중치",
	"scoring_macd_weight": "MACD 가중치", "scoring_rsi_weight": "RSI 가중치",
	"scoring_vwap_weight": "VWAP 가중치",
	// 매수 정지
	"buy_pause_start": "매수 정지 시작", "buy_pause_end": "매수 정지 종료",
	// 랭킹 필터
	"ranking_price_min": "랭킹 최소가", "ranking_price_max": "랭킹 최대가",
	"ranking_top_n":               "랭킹 상위 N종목",
	"ranking_volume_min_incrrate": "거래량 최소 증가율", "ranking_strength_min": "최소 체결강도",
	"ranking_fluctuation_min_rate": "최소 등락률", "ranking_fluctuation_max_rate": "최대 등락률",
	"rank_lease_duration_min": "랭킹 유지 시간(분)",
	// 사전 필터
	"filter_rsi_max": "사전필터 RSI 상한", "filter_disparity_m5_max": "사전필터 5분봉 이격도 상한",
	"filter_high_price_diff_min": "사전필터 고점 대비 하락 하한", "filter_open_price_diff_max": "사전필터 시가 대비 상승 상한",
	// 지수 하락 차단
	"index_drop_threshold_pct": "지수 하락 임계값(%)",
	// Hard Rejection
	"hard_disparity_m5_min": "5분봉 이격도 하한", "hard_disparity_m5_max": "5분봉 이격도 상한",
	"hard_high_price_diff_max": "고가 대비 하락 상한", "hard_high_price_diff_min": "고가 대비 하락 하한",
	"hard_prev_vol_ratio_max": "직전봉 거래량 비율 상한", "hard_strength_min": "체결강도 하한",
	"hard_rsi_max": "RSI 매수 상한", "hard_open_price_diff_max": "시가 대비 상승 상한",
	"hard_macd_bearish_enabled": "MACD 베어리시 차단", "hard_high_formed_mins_max": "고점 경과 시간 상한(분)",
	// 랭킹 기준 (선호 구간)
	"vwap_diff_min": "VWAP 이격도 하한", "vwap_diff_max": "VWAP 이격도 상한",
	"rsi_buy_min": "RSI 매수 하한", "rsi_buy_max": "RSI 매수 상한(구간)",
	"bid_ask_ratio_min": "매수호가 비율 하한",
	// 종목 선정 기준
	"min_market_cap": "최소 시가총액", "min_expected_profit_pct": "최소 기대 수익률(%)",
	"min_trading_value": "최소 거래대금", "momentum_score_min": "최소 모멘텀 스코어",
}

// settingConstraints defines validation bounds for numeric settings keys (auto-apply safety).
// Keys not listed here are string-typed or boolean and pass validation without range checks.
var settingConstraints = map[string][2]float64{
	// 수익/손절
	"take_profit_pct": {0.1, 20.0}, "stop_loss_pct": {0.1, 10.0},
	"etf_take_profit_pct": {0.1, 5.0}, "etf_stop_loss_pct": {0.1, 5.0},
	"stock_take_profit_pct": {0.1, 10.0}, "stock_stop_loss_pct": {0.1, 10.0},
	"stock_tax_rate": {0.0, 0.01},
	// 트레일링/일손실
	"trailing_trigger_pct": {0.0, 10.0}, "trailing_stop_pct": {0.1, 5.0},
	"daily_max_loss_pct": {0.0, 20.0},
	// 주문
	"order_amount_pct": {10.0, 99.0}, "max_positions": {1.0, 10.0},
	// 지표 매도
	"indicator_check_interval_min": {1.0, 60.0}, "indicator_rsi_sell_threshold": {50.0, 90.0},
	// 횡보
	"stagnation_threshold_pct": {0.1, 5.0}, "stagnation_duration_min": {5.0, 120.0},
	"stagnation_bid_ask_sell_threshold": {0.5, 3.0},
	// 부분 익절
	"partial_tp_pct": {0.1, 10.0}, "partial_tp_ratio": {0.1, 0.9},
	// 스코어링 가중치
	"scoring_bidask_weight": {0.0, 100.0}, "scoring_strength_weight": {0.0, 100.0},
	"scoring_macd_weight": {0.0, 100.0}, "scoring_rsi_weight": {0.0, 100.0},
	"scoring_vwap_weight": {0.0, 100.0},
	// 랭킹 필터
	"ranking_price_min": {0.0, 1000000.0}, "ranking_price_max": {0.0, 1000000.0},
	"ranking_top_n":               {5.0, 100.0},
	"ranking_volume_min_incrrate": {0.0, 10000.0}, "ranking_strength_min": {0.0, 500.0},
	"ranking_fluctuation_min_rate": {0.0, 30.0}, "ranking_fluctuation_max_rate": {0.0, 30.0},
	"rank_lease_duration_min": {1.0, 60.0},
	// 사전 필터
	"filter_rsi_max": {50.0, 100.0}, "filter_disparity_m5_max": {0.0, 20.0},
	"filter_high_price_diff_min": {-30.0, 0.0}, "filter_open_price_diff_max": {0.0, 30.0},
	// 지수 하락
	"index_drop_threshold_pct": {-5.0, 0.0},
	// Hard Rejection
	"hard_disparity_m5_min": {-10.0, 0.0}, "hard_disparity_m5_max": {0.0, 10.0},
	"hard_high_price_diff_max": {-5.0, 0.0}, "hard_high_price_diff_min": {-20.0, -0.1},
	"hard_prev_vol_ratio_max": {0.5, 5.0}, "hard_strength_min": {50.0, 150.0},
	"hard_rsi_max": {50.0, 90.0}, "hard_open_price_diff_max": {5.0, 30.0},
	"hard_high_formed_mins_max": {0.0, 240.0},
	// 랭킹 기준
	"vwap_diff_min": {-5.0, 5.0}, "vwap_diff_max": {0.0, 10.0},
	"rsi_buy_min": {20.0, 60.0}, "rsi_buy_max": {40.0, 80.0},
	"bid_ask_ratio_min": {0.5, 3.0},
	// 종목 선정
	"min_market_cap": {0.0, 1e12}, "min_expected_profit_pct": {0.0, 5.0},
	"min_trading_value": {0.0, 1e12}, "momentum_score_min": {0.0, 100.0},
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

// skipSettingsKeys is a set of keys excluded from the AI analysis prompt.
// These are meta/config keys that are not trading parameters Claude can optimize.
var skipSettingsKeys = map[string]bool{
	"claude_model":                  true,
	"active_preset_id":              true,
	"optimization_apply_mode":       true,
	"max_claude_candidates":         true,
	"hard_watch_symbols":            true,
	"ranking_types":                 true,
	"ranking_exchanges":             true,
	"ranking_volume_blng_cls_codes": true,
	"ranking_vi_kind_code":          true,
	"sell_conditions":               true,
	"index_codes":                   true,
	"trading_days":                  true,
	"trading_start_time":            true,
	"trading_end_time":              true,
	"ranking_condition":             true,
}

// collectCurrentSettings returns all trading-relevant settings for the analysis prompt.
// Any new settings key added to the DB is automatically included unless listed in skipSettingsKeys.
func collectCurrentSettings(ctx context.Context, db *database.DB) map[string]string {
	all := db.GetAllSettings(ctx)
	result := make(map[string]string, len(all))
	for k, v := range all {
		if skipSettingsKeys[k] {
			continue
		}
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
		logger.Info("optimization: no completed trades today, running no-trade analysis", map[string]any{"date": date})
		return generateNoTradeReport(ctx, db, claude, date)
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

// filteredEntry is used to parse the filter_reason field from filtered_stocks JSON.
type filteredEntry struct {
	FilterReason string `json:"filter_reason"`
}

// generateNoTradeReport collects today's ranking/selection log summaries and calls Claude
// for filter-loosening suggestions on a 0-trade day.
func generateNoTradeReport(ctx context.Context, db *database.DB, claude *trader.ClaudeClient, date string) error {
	rankLogs, _ := db.GetTodayRankingLogs(ctx)
	selLogs, _ := db.GetTodaySelectionLogs(ctx)

	// No activity at all → market holiday or system offline
	if len(rankLogs) == 0 && len(selLogs) == 0 {
		logger.Info("optimization: no activity logs today, skipping no-trade analysis", map[string]any{"date": date})
		return nil
	}

	type noTradeSummary struct {
		Date              string         `json:"date"`
		RankingAttempts   int            `json:"ranking_attempts"`
		AvgCandidates     float64        `json:"avg_candidates_passed"`
		CandidateTrend    []int          `json:"candidates_per_cycle"`
		PassedStockCodes  []string       `json:"passed_stock_codes"`
		FilterReasons     map[string]int `json:"filter_rejection_reasons"`
		RankingErrors     []string       `json:"ranking_errors"`
		SelectionAttempts int            `json:"selection_attempts"`
		FailReasons       []string       `json:"selection_fail_reasons"`
		HardRuleStats     map[string]int `json:"hard_rule_stats"` // 누적 Hard Rule별 위반 종목 수
	}
	summary := noTradeSummary{
		Date:            date,
		RankingAttempts: len(rankLogs),
		FilterReasons:   map[string]int{},
		HardRuleStats:   map[string]int{},
	}

	// Aggregate filter rejection reasons from ranking logs
	seenCodes := map[string]bool{}
	seenErrors := map[string]bool{}
	var totalCandidates int
	for _, rl := range rankLogs {
		totalCandidates += rl.IntersectionCount
		summary.CandidateTrend = append(summary.CandidateTrend, rl.IntersectionCount)

		// 사이클별 통과 종목 코드 수집 (중복 제거)
		var stocks []struct {
			StockCode string `json:"stock_code"`
		}
		if rl.ResultStocks != "" {
			_ = json.Unmarshal([]byte(rl.ResultStocks), &stocks)
		}
		for _, s := range stocks {
			if s.StockCode != "" && !seenCodes[s.StockCode] {
				summary.PassedStockCodes = append(summary.PassedStockCodes, s.StockCode)
				seenCodes[s.StockCode] = true
			}
		}

		// 랭킹 오류 수집 (중복 제거)
		if rl.ErrorMessage != "" && !seenErrors[rl.ErrorMessage] {
			summary.RankingErrors = append(summary.RankingErrors, rl.ErrorMessage)
			seenErrors[rl.ErrorMessage] = true
		}

		var entries []filteredEntry
		if rl.FilteredStocks != "" {
			_ = json.Unmarshal([]byte(rl.FilteredStocks), &entries)
		}
		for _, e := range entries {
			reason := normalizeFilterReason(e.FilterReason)
			if reason != "" {
				summary.FilterReasons[reason]++
			}
		}
	}
	if len(rankLogs) > 0 {
		summary.AvgCandidates = float64(totalCandidates) / float64(len(rankLogs))
	}

	// Collect unique fail reasons and aggregate hard rule stats from selection logs
	seen := map[string]bool{}
	for _, sl := range selLogs {
		summary.SelectionAttempts++
		if sl.FailReason != "" && !seen[sl.FailReason] {
			summary.FailReasons = append(summary.FailReasons, sl.FailReason)
			seen[sl.FailReason] = true
		}
		// hard_rule_stats 집계 (JSON map[string]int 파싱)
		if sl.HardRuleStats != "" && sl.HardRuleStats != "{}" {
			var ruleStats map[string]int
			if err := json.Unmarshal([]byte(sl.HardRuleStats), &ruleStats); err == nil {
				for rule, cnt := range ruleStats {
					summary.HardRuleStats[rule] += cnt
				}
			}
		}
	}

	summaryJSON, _ := json.MarshalIndent(summary, "", "  ")
	currentSettings := collectCurrentSettings(ctx, db)

	logger.Info("optimization: starting no-trade Claude analysis", map[string]any{
		"date": date, "ranking_attempts": summary.RankingAttempts,
	})

	result, err := claude.AnalyzeNoTradeDay(ctx, string(summaryJSON), currentSettings)
	if err != nil {
		return fmt.Errorf("Claude no-trade analysis: %w", err)
	}

	for i := range result.Suggestions {
		result.Suggestions[i].ID = strconv.Itoa(i + 1)
		result.Suggestions[i].Status = "PENDING"
	}

	applyMode := db.GetSetting(ctx, "optimization_apply_mode")
	if applyMode == "" {
		applyMode = "all_manual"
	}

	suggestionsJSON, _ := json.Marshal(result.Suggestions)
	optReport := models.OptimizationReport{
		Date:              date,
		OverallAssessment: result.OverallAssessment,
		Suggestions:       string(suggestionsJSON),
		ApplyModeSnapshot: applyMode,
	}

	if err := db.UpsertOptimizationReport(ctx, optReport); err != nil {
		return fmt.Errorf("UpsertOptimizationReport (no-trade): %w", err)
	}

	logger.Info("optimization: no-trade report saved", map[string]any{
		"date": date, "suggestions": len(result.Suggestions),
	})

	if applyMode == "all_auto" {
		saved, err := db.GetOptimizationReportByDate(ctx, date)
		if err != nil {
			return fmt.Errorf("reload after save (no-trade): %w", err)
		}
		var suggestions []models.OptimizationSuggestion
		if err := json.Unmarshal([]byte(saved.Suggestions), &suggestions); err != nil {
			return fmt.Errorf("parse saved suggestions (no-trade): %w", err)
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
			return fmt.Errorf("persist auto-apply results (no-trade): %w", err)
		}
		logger.Info("optimization: no-trade auto-apply complete", map[string]any{
			"date": date, "applied": appliedCount,
		})
	}

	return nil
}

// normalizeFilterReason extracts the core filter category from a detailed rejection reason string.
// e.g., "RSI 과열 (85.3 >= 80.0)" → "RSI 과열"
func normalizeFilterReason(reason string) string {
	if idx := strings.Index(reason, "("); idx > 0 {
		return strings.TrimSpace(reason[:idx])
	}
	if len(reason) > 30 {
		return reason[:30]
	}
	return strings.TrimSpace(reason)
}
