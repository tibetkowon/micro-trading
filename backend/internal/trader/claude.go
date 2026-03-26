package trader

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// RankItem is a unified representation of a stock from any ranking API,
// enriched with technical indicators from GetStockInfo.
type RankItem struct {
	DataRank     string `json:"data_rank"`
	StockCode    string `json:"stock_code"`
	StockName    string `json:"stock_name"`
	CurrentPrice string `json:"current_price"`
	Volume       string `json:"volume"`
	RankingType  string `json:"ranking_type"`            // e.g. "volume+strength"
	Exchange     string `json:"exchange,omitempty"`      // 미장 거래소 코드 (NAS/NYS/AMS)
	VolIncrRate  string `json:"vol_incr_rate,omitempty"` // 거래량 증가율 % (volume)
	Strength     string `json:"strength,omitempty"`      // 체결강도 % (strength)
	NetBuyQty    string `json:"net_buy_qty,omitempty"`   // 순매수체결량 (exec_count)
	DisparityD20 string `json:"disparity_d20,omitempty"` // 20일 이격도 (disparity)
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
	// 신규 기술 지표 (Phase 2)
	VWAP            float64 `json:"vwap,omitempty"`              // 당일 VWAP
	VWAPDiff        float64 `json:"vwap_diff,omitempty"`         // (현재가-VWAP)/VWAP×100
	M5MA10          float64 `json:"m5_ma10,omitempty"`           // 5분봉 MA10
	PrevVolumeRatio float64 `json:"prev_volume_ratio,omitempty"` // 직전봉 대비 거래량 비율
	BidAskRatio     float64 `json:"bid_ask_ratio,omitempty"`     // 총 매수잔량/총 매도잔량 (0=데이터 없음)
}

// TradingRules holds parameterized hard rejection and ranking criteria for Claude prompts.
type TradingRules struct {
	// 런타임 시장 상태
	MarketIndexDrop float64 // 현재 지수 등락률 (%) — 음수=하락
	// 하드 거부 기준값
	HardDisparityM5Min   float64
	HardDisparityM5Max   float64
	HardHighPriceDiffMax float64
	HardHighPriceDiffMin float64
	HardPrevVolRatioMax  float64
	HardStrengthMin      float64
	HardRSIMax           float64
	HardOpenPriceDiffMax float64
	// 매수 구간 기준값
	VWAPDiffMin    float64
	VWAPDiffMax    float64
	RSIBuyMin      float64
	RSIBuyMax      float64
	BidAskRatioMin float64
	// 지수 하락 임계값
	IndexDropThreshold float64 // 기본 -1.0
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
// market: "KR" (default) or "US" — selects the appropriate prompt and context.
// Returns an ordered slice — index 0 is the top pick. Engine tries them in order.
func (c *ClaudeClient) SelectStocks(
	ctx context.Context,
	rankings []RankItem,
	availableCash float64,
	_ []string, // excludedCodes: filtered server-side, kept for API compatibility
	market string,
	rules TradingRules,
) ([]StockCandidate, error) {
	if len(rankings) == 0 {
		return nil, fmt.Errorf("ranking list is empty")
	}

	rankJSON, _ := json.Marshal(rankings)

	var prompt string
	if market == "US" {
		prompt = fmt.Sprintf(`You are an elite US day-trader focused on NASDAQ/NYSE/AMEX stocks, known for avoiding Bull Traps and finding momentum entries.

## Hard Rejection Rules — skip if ANY apply:
1. high_price_diff < -5%%  → dropped more than 5%% from today's high, avoid (selling pressure)
2. ma5 < ma20  → downtrend, skip
3. disparity_m5 > %.1f%%  → over-extended from 5-min MA, skip
4. rsi14 >= %.0f  → overbought, skip
5. open_price_diff > %.0f%%  → already at extreme daily high, skip

## Ranking Criteria (for survivors):
- Best entry: high_price_diff between -0.5%% and -3%% (slight pullback from high, ready to bounce)
- MA trend: ma5 > ma20 confirms uptrend — prefer larger gap
- MACD: macd_line > macd_signal preferred (upward momentum)
- RSI: %.0f–%.0f is the ideal buy zone (not overbought, has momentum)
- Disparity: disparity_m5 between 0%% and 2%% (near 5-min MA support, not overextended)
- Volume confirmation: higher volume relative to average indicates institutional interest
- Prioritize consolidation/pullback: open_price_diff between 2%% and 10%% (healthy gap-up, consolidating)
- Best entry zone: stocks near 5-min MA support after a small pullback, not at daily peak
- Avoid: open_price_diff > 15%% (already overextended today, late entry risk)

Ranking data (JSON):
%s
Available cash: %.2f USD

Respond with ONLY a valid JSON array — no explanation, no markdown, no extra text.
If no stock passes, respond with exactly: []
Best entry first:
[{"stock_code":"TICKER","reason":"Pulled back -Y%% from high, MA5 > MA20, RSI=X (buy zone), MACD bullish, consolidating near 5min MA"},...]`,
			rules.HardDisparityM5Max,
			rules.HardRSIMax,
			rules.HardOpenPriceDiffMax,
			rules.RSIBuyMin, rules.RSIBuyMax,
			string(rankJSON), availableCash)
	} else {
		marketIndexNote := ""
		if rules.MarketIndexDrop != 0 {
			marketIndexNote = fmt.Sprintf("Current market index: %.2f%% (시가 대비 등락률)\n", rules.MarketIndexDrop)
		}
		prompt = fmt.Sprintf(`You are an elite Korean day-trader known for strict risk management, avoiding Bull Traps, and finding high-probability pullback(눌림목) entries.

%s## Hard Rejection Rules — skip if ANY apply (Kill-switch & Defense):
1. market_index_drop < %.1f%%  → 전체 시장 급락 중(투매 장세), skip all
2. disparity_m5 > %.1f%% OR disparity_m5 < %.1f%%  → 5분봉 MA에서 너무 멀거나 지지선 하향 돌파(칼날 하락), skip
3. high_price_diff > %.1f%%  → 당일 고점 부근(추격 매수 위험), skip
4. high_price_diff < %.1f%% AND prev_volume_ratio > %.1f  → 대량 거래량 동반 하락(추세 이탈), skip
5. ma5 < ma20  → 5분봉 단기 역배열(하락 추세), skip
6. strength < %.0f  → 매수 체결 우위 아님(매수세 소멸), skip
7. rsi14 > %.0f  → 단기 과매수 상태에서 꺾이는 중, skip
8. open_price_diff > %.0f%%  → 이미 너무 많이 오른 종목(설거지 위험), skip

## Ranking Criteria (for survivors):
- 진정한 눌림목(True Pullback): vwap_diff between %.1f%% ~ %.1f%% (VWAP 지지선 부근 반등 대기); if vwap_diff is 0, use high_price_diff -1%% ~ -3%%
- 건전한 거래량: 하락 시 prev_volume_ratio < 0.8 (거래량 감소) 및 net_buy_qty > 0 (순매수 우세)
- 수급/모멘텀: bid_ask_ratio > %.1f (매수 호가 우세), macd_line > macd_signal
- 최적 매수 구간: rsi14 between %.0f ~ %.0f (반등 구간, 과매수 아님)
- MA 배열: ma5 > ma20 > m5_ma10 순서면 강세 배열 가산점

Ranking data (JSON format; vwap_diff=0 means VWAP unavailable):
%s
Available cash: %.0f KRW

Respond with ONLY a valid JSON array — no explanation, no markdown, no extra text.
If no stock passes, respond with exactly: []
Best entry first:
[{"stock_code":"6-digit","reason":"고점(X원) 대비 -Y%% 눌림, VWAP 지지선 근처, 거래량 감소 눌림목, 체결강도 Z%% 매수우세로 반등 기대"}]`,
			marketIndexNote,
			rules.IndexDropThreshold,
			rules.HardDisparityM5Max, rules.HardDisparityM5Min,
			rules.HardHighPriceDiffMax,
			rules.HardHighPriceDiffMin, rules.HardPrevVolRatioMax,
			rules.HardStrengthMin,
			rules.HardRSIMax,
			rules.HardOpenPriceDiffMax,
			rules.VWAPDiffMin, rules.VWAPDiffMax,
			rules.BidAskRatioMin,
			rules.RSIBuyMin, rules.RSIBuyMax,
			string(rankJSON), availableCash)
	}

	msg, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: 2048,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("claude SelectStocks API error: %w", err)
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
