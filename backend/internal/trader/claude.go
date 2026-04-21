package trader

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/micro-trading-for-agent/backend/internal/agent"
	"github.com/micro-trading-for-agent/backend/internal/logger"
	"github.com/micro-trading-for-agent/backend/internal/models"
)

// isOverloadedError returns true when the Anthropic API responds with a 529 overloaded error.
func isOverloadedError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "529") || strings.Contains(s, "overloaded_error")
}

// callWithRetry calls fn up to maxAttempts times, retrying on 529 overloaded errors
// with exponential backoff (2s, 4s, 8s …).
func callWithRetry(ctx context.Context, maxAttempts int, fn func() error) error {
	var err error
	for i := 0; i < maxAttempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		if !isOverloadedError(err) {
			return err
		}
		wait := time.Duration(1<<uint(i+1)) * time.Second // 2s, 4s, 8s
		logger.Warn("claude: overloaded, retrying", map[string]any{
			"attempt": i + 1,
			"wait":    wait.String(),
		})
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
	return err
}

// RankItem is a unified representation of a stock from any ranking API,
// enriched with technical indicators from GetStockInfo.
type RankItem struct {
	DataRank     string `json:"data_rank"`
	StockCode    string `json:"stock_code"`
	StockName    string `json:"stock_name"`
	CurrentPrice string `json:"current_price"`
	Volume       string `json:"volume"`
	RankingType  string `json:"ranking_type"`            // e.g. "volume+strength"
	VolIncrRate  string `json:"vol_incr_rate,omitempty"` // 거래량 증가율 % (volume)
	Strength     string `json:"strength,omitempty"`      // 체결강도 % (strength)
	AssetType    string `json:"asset_type,omitempty"`    // "STOCK" | "ETF" | "ETF_DOMESTIC"
	// 당일 OHLC
	DayOpen string `json:"day_open,omitempty"`
	DayHigh string `json:"day_high,omitempty"`
	DayLow  string `json:"day_low,omitempty"`
	// 파생 지표 (서버 계산)
	HighPriceDiff float64 `json:"high_price_diff,omitempty"` // (현재가-고가)/고가×100 (음수=눌림)
	OpenPriceDiff float64 `json:"open_price_diff,omitempty"` // (현재가-시가)/시가×100 (당일 상승률)
	DisparityM5   float64 `json:"disparity_m5,omitempty"`    // 5분봉 MA5 이격도
	// Technical indicators from GetStockInfo
	MA5        float64 `json:"ma5,omitempty"`
	MA20       float64 `json:"ma20,omitempty"`
	RSI14      float64 `json:"rsi14,omitempty"`
	MACDLine   float64 `json:"macd_line,omitempty"`
	MACDSignal float64 `json:"macd_signal,omitempty"`
	MACDHisto  float64 `json:"macd_histogram,omitempty"` // MACD Histogram (Line − Signal)
	// 신규 기술 지표 (Phase 2)
	VWAP            float64 `json:"vwap,omitempty"`              // 당일 VWAP
	VWAPDiff        float64 `json:"vwap_diff,omitempty"`         // (현재가-VWAP)/VWAP×100
	M5MA10          float64 `json:"m5_ma10,omitempty"`           // 5분봉 MA10
	PrevVolumeRatio float64 `json:"prev_volume_ratio,omitempty"` // 직전봉 대비 거래량 비율
	BidAskRatio     float64 `json:"bid_ask_ratio,omitempty"`     // 총 매수잔량/총 매도잔량 (0=데이터 없음)
	MomentumScore   float64 `json:"momentum_score,omitempty"`    // 복합 모멘텀 스코어 (0~100). bid_ask·체결강도·거래량 감소 가중합산
	// 세금보정 필드
	TradingValue      float64 `json:"trading_value,omitempty"`       // 당일 거래대금 (원)
	ApplicableTaxRate float64 `json:"applicable_tax_rate,omitempty"` // 0.0=ETF비과세, 0.002=주식
	// 데이터 품질 개선 필드 (1~5순위)
	RecentCandles     []agent.CandleSnap `json:"recent_candles,omitempty"`       // 최근 5개 5분봉 (구→신), dir: U/D/=
	HighFormedMinsAgo int                `json:"high_formed_mins_ago,omitempty"` // 당일 고점 형성 후 경과 시간(분)
	VolTrend3         float64            `json:"vol_trend_3,omitempty"`          // 최근 3봉 거래량 기울기 (-1~1, 음수=감소)
	VolAtHigh         int64              `json:"vol_at_high,omitempty"`          // 고점 형성 봉 거래량
	NearBidAskRatio   float64            `json:"near_bid_ask_ratio,omitempty"`   // 현재가 ±2% 범위 내 매수/매도 비율
	TopAskWall        float64            `json:"top_ask_wall,omitempty"`         // 가장 큰 매도 벽 위치 (현재가 대비 %)
	TopAskWallSize    int64              `json:"top_ask_wall_size,omitempty"`    // 가장 큰 매도 벽 잔량
	VolVs3AvgRatio        float64 `json:"vol_vs_3avg_ratio,omitempty"`         // 현재봉 거래량 / 직전 3봉 평균 거래량 (거래량 회복 비율)
	RelativeStrengthVsMkt float64 `json:"relative_strength_vs_market,omitempty"` // 개별 종목 등락률 - 시장 지수 등락률
}

// TradingRules holds parameterized hard rejection and ranking criteria for Claude prompts.
type TradingRules struct {
	// 런타임 시장 상태
	MarketIndexDrop float64 // 현재 지수 등락률 (%) — 음수=하락
	// 하드 거부 기준값
	HardDisparityM5Min     float64
	HardDisparityM5Max     float64
	HardHighPriceDiffMax   float64
	HardHighPriceDiffMin   float64
	HardPrevVolRatioMax    float64
	HardStrengthMin        float64
	HardRSIMax             float64
	HardOpenPriceDiffMax   float64
	HardMACDBearishEnabled bool    // true이면 macd_line < macd_signal 진입 차단
	HardHighFormedMinsMax  float64 // 고점 형성 후 경과 시간 상한(분). 0=비활성
	HardVolVs3AvgRatioMin  float64 // 거래량 회복 비율 하한. 0=비활성
	HardRelativeStrengthMin float64 // 시장 대비 상대강도 하한(%). 0=비활성
	// 매수 구간 기준값
	VWAPDiffMin    float64
	VWAPDiffMax    float64
	RSIBuyMin      float64
	RSIBuyMax      float64
	BidAskRatioMin float64
	// 지수 하락 임계값
	IndexDropThreshold float64 // 기본 -1.0
	// 세금보정 기준값
	MinExpectedProfitPct float64 // 주식 세후 최소 기대수익 (%). 0=미사용
	StockTaxRate         float64 // 주식 세율 (기본 0.002)
}

// DefaultTradingRules returns safe default values (matches the hard-coded prompt values).
func DefaultTradingRules() TradingRules {
	return TradingRules{
		HardDisparityM5Min:   -1.5,
		HardDisparityM5Max:   3.0,
		HardHighPriceDiffMax: -0.5,
		HardHighPriceDiffMin: -5.0,
		HardPrevVolRatioMax:  1.2,
		HardStrengthMin:      100.0,
		HardRSIMax:           70.0,
		HardOpenPriceDiffMax: 15.0,
		VWAPDiffMin:          0.0,
		VWAPDiffMax:          1.5,
		RSIBuyMin:            40.0,
		RSIBuyMax:            60.0,
		BidAskRatioMin:       1.2,
		IndexDropThreshold:   -1.0,
	}
}

// ClaudeClient wraps the Anthropic API for trading decisions.
type ClaudeClient struct {
	client anthropic.Client
	model  string
}

// NewClaudeClient creates a ClaudeClient with the given API key and model.
func NewClaudeClient(apiKey, model string) *ClaudeClient {
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	return &ClaudeClient{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
		model:  model,
	}
}

// StockCandidate is one entry in Claude's ranked selection list.
type StockCandidate struct {
	StockCode string `json:"stock_code"`
	Reason    string `json:"reason"`
}

// SelectStocks asks Claude to rank all viable candidates from the ranking list.
// Already-traded stocks are filtered server-side before this call.
// Returns an ordered slice — index 0 is the top pick. Engine tries them in order.
func (c *ClaudeClient) SelectStocks(
	ctx context.Context,
	rankings []RankItem,
	availableCash float64,
	_ []string, // excludedCodes: filtered server-side, kept for API compatibility
	rules TradingRules,
) ([]StockCandidate, error) {
	if len(rankings) == 0 {
		return nil, fmt.Errorf("ranking list is empty")
	}

	rankJSON, _ := json.Marshal(rankings)

	// 4순위: 장 시간대 컨텍스트
	now := time.Now()
	hour, min := now.Hour(), now.Minute()
	totalMin := hour*60 + min
	sessionPhase := "MID"
	sessionNote := ""
	switch {
	case totalMin < 9*60+15:
		sessionPhase = "PRE"
		sessionNote = "장 초반(09:15 이전): 변동성 극대, 진입 자제"
	case totalMin < 10*60:
		sessionPhase = "OPEN"
		sessionNote = "개장 초반(09:15~10:00): 급등 후 첫 눌림 위험 높음, 거래량 확인 필수"
	case totalMin < 14*60:
		sessionPhase = "MID"
		sessionNote = "장 중반(10:00~14:00): 가장 안정적인 눌림목 구간"
	default:
		sessionPhase = "CLOSE"
		sessionNote = "장 마감(14:00~): 15:15 청산 고려, 목표가 달성 가능 종목만 선택"
	}
	_ = sessionPhase

	marketIndexNote := ""
	if rules.MarketIndexDrop != 0 {
		marketIndexNote = fmt.Sprintf("Current market index: %.2f%% (시가 대비 등락률)\n", rules.MarketIndexDrop)
	}

	// Hard Rejection Rule 9·10 조건부 생성
	macdBearishRule := ""
	if rules.HardMACDBearishEnabled {
		macdBearishRule = "9. macd_line < macd_signal  → 진입 시점 MACD 이미 하락 교차(bearish), 역방향 진입 차단, skip\n"
	}
	highFormedRule := ""
	if rules.HardHighFormedMinsMax > 0 {
		highFormedRule = fmt.Sprintf("10. high_formed_mins_ago > %.0f  → 고점 형성 후 너무 오래 경과(모멘텀 소진), skip\n", rules.HardHighFormedMinsMax)
	}
	volVs3AvgRule := ""
	if rules.HardVolVs3AvgRatioMin > 0 {
		volVs3AvgRule = fmt.Sprintf("11. vol_vs_3avg_ratio < %.2f  → 거래량 미회복(반등 동력 없음), skip\n", rules.HardVolVs3AvgRatioMin)
	}
	relStrengthRule := ""
	if rules.HardRelativeStrengthMin != 0 {
		relStrengthRule = fmt.Sprintf("12. relative_strength_vs_market < %.1f%%  → 시장 대비 약세(상대강도 부족), skip\n", rules.HardRelativeStrengthMin)
	}
	taxNote := ""
	if rules.MinExpectedProfitPct > 0 {
		stockTaxRate := rules.StockTaxRate
		if stockTaxRate <= 0 {
			stockTaxRate = 0.002
		}
		taxNote = fmt.Sprintf(`
## Tax-Adjusted Return Rule (세금보정):
- ETF_DOMESTIC: applicable_tax_rate=0.0 → 비과세, net_expected = target_pct
- ETF: applicable_tax_rate=0.0 → 비과세, net_expected = target_pct
- STOCK: applicable_tax_rate=%.3f → net_expected = target_pct - %.1f%%
- If asset_type == "STOCK" AND net_expected < %.1f%% → SKIP (세후 기대수익 불충분)
- 동일 신호 강도면 ETF_DOMESTIC > ETF > STOCK 우선`, stockTaxRate, stockTaxRate*100, rules.MinExpectedProfitPct)
	}
	prompt := fmt.Sprintf(`You are an elite Korean day-trader known for strict risk management, avoiding Bull Traps, and finding high-probability pullback(눌림목) entries.
DO NOT explain your reasoning process. Output ONLY the final JSON array.

## Session Context (장 시간대)
Current session phase: %s — %s

%s## Hard Rejection Rules — skip if ANY apply (Kill-switch & Defense):
1. market_index_drop < %.1f%%  → 전체 시장 급락 중(투매 장세), skip all
2. disparity_m5 > %.1f%% OR disparity_m5 < %.1f%%  → 5분봉 MA에서 너무 멀거나 지지선 하향 돌파(칼날 하락), skip
3. high_price_diff > %.1f%%  → 당일 고점 부근(추격 매수 위험), skip
4. high_price_diff < %.1f%% AND prev_volume_ratio > %.1f  → 대량 거래량 동반 하락(추세 이탈), skip
5. ma5 < ma20  → 5분봉 단기 역배열(하락 추세), skip
6. strength < %.0f  → 매수 체결 우위 아님(매수세 소멸), skip
7. rsi14 > %.0f  → 단기 과매수 상태에서 꺾이는 중, skip
8. open_price_diff > %.0f%%  → 이미 너무 많이 오른 종목(설거지 위험), skip
%s%s%s%s%s
## Ranking Criteria (for survivors):
- 진정한 눌림목(True Pullback): vwap_diff between %.1f%% ~ %.1f%% (VWAP 지지선 부근 반등 대기); if vwap_diff is 0, use high_price_diff -1%% ~ -3%%
- 건전한 거래량: 하락 시 prev_volume_ratio < 0.8 (거래량 감소) 및 net_buy_qty > 0 (순매수 우세)
- 수급/모멘텀: bid_ask_ratio > %.1f (매수 호가 우세), macd_line > macd_signal
- 최적 매수 구간: rsi14 between %.0f ~ %.0f (반등 구간, 과매수 아님)
- MA 배열: ma5 > ma20 > m5_ma10 순서면 강세 배열 가산점

## New Data Fields Guide (신규 필드 해석):
- recent_candles: 최근 5개 5분봉 (구→신 순서). dir="D"+volume 감소 → 건강한 눌림목. dir="D"+volume 증가 → 추세 하락 위험.
- high_formed_mins_ago: 고점 형성 후 경과 시간(분). 5분 미만=아직 하락 중(진입 위험), 15~45분=눌림 안정화 구간, 60분 이상=추세 전환 의심.
- vol_trend_3: 최근 3봉 거래량 기울기(-1~1). 음수=거래량 감소(건강한 눌림), 양수=거래량 증가(상승 or 매도 가속 주의).
- vol_at_high: 고점 봉 거래량. (vol_at_high - current candle volume) 이 클수록 거래량 감소 눌림목.
- near_bid_ask_ratio: 현재가 ±2%% 이내 실질 매수/매도압력 비율. bid_ask_ratio보다 신뢰도 높음.
- top_ask_wall: 가장 큰 매도 벽의 현재가 대비 위치(%%양수=위). 목표가보다 낮은 위치에 큰 벽 존재 시 진입 자제.
- top_ask_wall_size: 매도 벽 잔량. 클수록 돌파 어려움.
- vol_vs_3avg_ratio: 현재봉 거래량 / 직전 3봉 평균 거래량. 1.0 이상=거래량 회복 중(반등 동력), 0.5 미만=거래량 소진 상태.
- relative_strength_vs_market: 개별 종목 등락률 - 시장 지수 등락률. 양수=시장 대비 강세, 음수=시장 대비 약세.

Ranking data (JSON format; vwap_diff=0 means VWAP unavailable):
%s
Available cash: %.0f KRW

Respond with ONLY a valid JSON array — no explanation, no markdown, no extra text.
If no stock passes, respond with exactly: []
Best entry first:
[{"stock_code":"6-digit","reason":"고점(X원) 대비 -Y%% 눌림, VWAP 지지선 근처, 거래량 감소 눌림목(vol_trend_3 음수), 체결강도 Z%% 매수우세로 반등 기대"}]`,
		sessionPhase, sessionNote,
		marketIndexNote,
		rules.IndexDropThreshold,
		rules.HardDisparityM5Max, rules.HardDisparityM5Min,
		rules.HardHighPriceDiffMax,
		rules.HardHighPriceDiffMin, rules.HardPrevVolRatioMax,
		rules.HardStrengthMin,
		rules.HardRSIMax,
		rules.HardOpenPriceDiffMax,
		taxNote,
		macdBearishRule,
		highFormedRule,
		volVs3AvgRule,
		relStrengthRule,
		rules.VWAPDiffMin, rules.VWAPDiffMax,
		rules.BidAskRatioMin,
		rules.RSIBuyMin, rules.RSIBuyMax,
		string(rankJSON), availableCash)

	var msg *anthropic.Message
	if retryErr := callWithRetry(ctx, 3, func() error {
		var e error
		msg, e = c.client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     anthropic.Model(c.model),
			MaxTokens: 4096,
			Messages: []anthropic.MessageParam{
				anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
			},
		})
		return e
	}); retryErr != nil {
		return nil, fmt.Errorf("claude SelectStocks API error: %w", retryErr)
	}

	if len(msg.Content) == 0 {
		return nil, fmt.Errorf("claude returned empty response")
	}

	raw := strings.TrimSpace(msg.Content[0].AsText().Text)

	// Extract JSON array portion: find first '[' and its matching ']'
	start := strings.Index(raw, "[")
	if start == -1 {
		return nil, fmt.Errorf("claude response has no JSON array (raw: %s)", raw)
	}
	depth, end := 0, -1
	for i := start; i < len(raw); i++ {
		switch raw[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				end = i
			}
		}
		if end != -1 {
			break
		}
	}
	if end == -1 {
		return nil, fmt.Errorf("claude response JSON array not closed (raw: %s)", raw)
	}
	raw = raw[start : end+1]

	var candidates []StockCandidate
	if err := json.Unmarshal([]byte(raw), &candidates); err != nil {
		return nil, fmt.Errorf("claude response parse error: %w (raw: %s)", err, raw)
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("claude: 조건에 맞는 종목 없음 (Hard Rejection Rule 적용)")
	}

	return candidates, nil
}

// ─────────────────────────────────────────────────────────
// Daily Report Analysis
// ─────────────────────────────────────────────────────────

// analysisResponse is the expected JSON structure from Claude for daily report analysis.
type analysisResponse struct {
	OverallAssessment string                          `json:"overall_assessment"`
	Suggestions       []models.OptimizationSuggestion `json:"suggestions"`
}

// allowedSettingsKeys is the whitelist of settings keys Claude is allowed to suggest changes for.
var allowedSettingsKeys = map[string]bool{
	// 거래 설정
	"take_profit_pct": true, "stop_loss_pct": true,
	"etf_take_profit_pct": true, "etf_stop_loss_pct": true,
	"stock_take_profit_pct": true, "stock_stop_loss_pct": true,
	"order_amount_pct": true, "max_positions": true,
	"indicator_check_interval_min": true, "indicator_rsi_sell_threshold": true,
	"stagnation_threshold_pct": true, "stagnation_duration_min": true,
	"trailing_trigger_pct": true, "trailing_stop_pct": true,
	"daily_max_loss_pct": true, "buy_pause_start": true, "buy_pause_end": true,
	// 프롬프트 파라미터 (TradingRules)
	"hard_disparity_m5_min": true, "hard_disparity_m5_max": true,
	"hard_high_price_diff_max": true, "hard_high_price_diff_min": true,
	"hard_prev_vol_ratio_max": true, "hard_strength_min": true,
	"hard_rsi_max": true, "hard_open_price_diff_max": true,
	"vwap_diff_min": true, "vwap_diff_max": true,
	"rsi_buy_min": true, "rsi_buy_max": true,
	"bid_ask_ratio_min": true, "index_drop_threshold_pct": true,
	"min_expected_profit_pct":         true,
	"momentum_score_min":              true,
	"stagnation_partial_exit_enabled": true, "stagnation_bid_ask_sell_threshold": true,
	"hard_vol_vs_3avg_ratio_min":  true,
	"hard_relative_strength_min":  true,
}

// AnalyzeDailyReport sends the daily report and current settings to Claude for optimization suggestions.
// currentSettings should contain all relevant settings key-value pairs.
func (c *ClaudeClient) AnalyzeDailyReport(
	ctx context.Context,
	dr models.DailyReport,
	currentSettings map[string]string,
) (*analysisResponse, error) {
	settingsJSON, _ := json.MarshalIndent(currentSettings, "", "  ")

	prompt := fmt.Sprintf(`You are an expert algorithmic trading analyst reviewing a Korean day-trading system's daily performance.
Analyze the trading results and current parameter settings, then suggest specific improvements.

## Today's Trading Report (%s)
- Total Trades: %d
- Winning: %d / Losing: %d
- Total Profit/Loss: %.0f KRW
- Average Return: %.2f%%
- Best Trade: %s
- Worst Trade: %s
- All Trades Summary: %s

## Current System Settings
%s

## Your Task
Based on the trading results above, provide concrete optimization suggestions.

Guidelines:
- For "settings" category: suggest changes to specific settings keys from the Current System Settings above.
  Only suggest keys that are in the provided settings. Be conservative — suggest at most 3 settings changes.
- For "feature" category: suggest new indicators, filters, or system capabilities that could improve performance.
  Limit to at most 2 feature requests.
- Each suggestion MUST include a clear "comment" explaining the specific evidence from today's trades.
- If the day was profitable with no clear issues, you may return fewer suggestions or none.

Respond with ONLY valid JSON — no markdown, no explanation:
{
  "overall_assessment": "2-3 sentence summary of today's performance and key observations",
  "suggestions": [
    {
      "id": "1",
      "category": "settings",
      "key": "settings_key_name",
      "name": "",
      "type": "",
      "current_value": "current value string",
      "suggested_value": "new value string",
      "comment": "specific evidence from today's trades explaining this suggestion"
    },
    {
      "id": "2",
      "category": "feature",
      "key": "",
      "name": "Feature Name",
      "type": "indicator",
      "current_value": "",
      "suggested_value": "",
      "comment": "specific evidence from today's trades explaining why this feature would help"
    }
  ]
}`,
		dr.Date,
		dr.TotalTrades, dr.WinningTrades, dr.LosingTrades,
		dr.TotalProfitAmount, dr.AvgProfitPct,
		dr.BestTrade, dr.WorstTrade, dr.TradeSummary,
		string(settingsJSON),
	)

	var msg *anthropic.Message
	if retryErr := callWithRetry(ctx, 3, func() error {
		var e error
		msg, e = c.client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     anthropic.Model(c.model),
			MaxTokens: 2048,
			Messages: []anthropic.MessageParam{
				anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
			},
		})
		return e
	}); retryErr != nil {
		return nil, fmt.Errorf("claude AnalyzeDailyReport API error: %w", retryErr)
	}
	if len(msg.Content) == 0 {
		return nil, fmt.Errorf("claude returned empty response")
	}

	raw := strings.TrimSpace(msg.Content[0].AsText().Text)
	// Extract JSON object
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("claude analysis response has no JSON object (raw: %s)", raw)
	}
	raw = raw[start : end+1]

	var result analysisResponse
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("claude analysis response parse error: %w (raw: %s)", err, raw)
	}

	// Filter: remove settings suggestions with unknown keys
	filtered := result.Suggestions[:0]
	for _, s := range result.Suggestions {
		if s.Category == "settings" && !allowedSettingsKeys[s.Key] {
			continue
		}
		filtered = append(filtered, s)
	}
	result.Suggestions = filtered

	return &result, nil
}
