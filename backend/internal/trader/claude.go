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
) ([]StockCandidate, error) {
	if len(rankings) == 0 {
		return nil, fmt.Errorf("ranking list is empty")
	}

	rankJSON, _ := json.Marshal(rankings)

	prompt := fmt.Sprintf(`You are an elite Korean day-trader known for avoiding Bull Traps and finding pullback(눌림목) entries.

## Hard Rejection Rules — skip if ANY apply:
1. disparity_m5 > 3%%  → over-extended from 5-min MA, skip
2. high_price_diff > -0.5%%  → basically at today's peak, skip
3. ma5 < ma20  → downtrend, skip

## Ranking Criteria (for survivors):
- Best entry: high_price_diff between -1%% and -3%% (pulled back from high, ready to bounce)
- Volume quality: net_buy_qty > 0 + strength increasing = accumulation signal
- MACD: macd_line > macd_signal preferred (upward momentum)
- Avoid: open_price_diff > 10%% (stocks that already ran too far today)

Ranking data (JSON):
%s
Available cash: %.0f KRW

Respond with ONLY a valid JSON array — no explanation, no markdown, no extra text.
If no stock passes, respond with exactly: []
Best entry first:
[{"stock_code":"6-digit","reason":"고점(X원) 대비 -Y%% 눌림, 5분봉 MA 지지, 순매수 우세로 반등 기대"},...]`,
		string(rankJSON), availableCash)

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
