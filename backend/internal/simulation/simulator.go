package simulation

import "math"

// commissionPct is the combined buy+sell commission and slippage (0.25%).
const commissionPct = 0.25

// MinuteCandle is one 1-minute OHLC candle.
type MinuteCandle struct {
	High  float64
	Low   float64
	Close float64
}

// SimParams is the parameter set used by the simulator.
type SimParams struct {
	TakeProfitPct        float64 `json:"take_profit_pct"`
	StopLossPct          float64 `json:"stop_loss_pct"`
	TrailingTriggerPct   float64 `json:"trailing_trigger_pct"`
	TrailingStopPct      float64 `json:"trailing_stop_pct"`
	StagnationPct        float64 `json:"stagnation_pct"`
	MinScoreThreshold    float64 `json:"min_score_threshold"`
	UniversalCooldownMin int     `json:"universal_cooldown_min"`
	WeightStrength       int     `json:"weight_strength"`
	WeightRSI            int     `json:"weight_rsi"`
	WeightMACD           int     `json:"weight_macd"`
	WeightBidAsk         int     `json:"weight_bid_ask"`
	WeightVWAP           int     `json:"weight_vwap"`
	WeightVolume         int     `json:"weight_volume"`
	WeightProgramBuy     int     `json:"weight_program_buy"`
	WeightMicroBidAsk    int     `json:"weight_micro_bid_ask"`
	WeightVIDisparity    int     `json:"weight_vi_disparity"`
}

// SimTradeResult is the simulated exit result for one trade.
type SimTradeResult struct {
	EntryPrice     float64
	ExitPrice      float64
	ExitReason     string
	PnlPct         float64
	HoldingCandles int
}

// SimulateTrade replays candles in ascending time order and exits at the first matched condition.
func SimulateTrade(entryPrice float64, candles []MinuteCandle, params SimParams) SimTradeResult {
	targetPrice := entryPrice * (1 + params.TakeProfitPct/100)
	stopPrice := entryPrice * (1 - params.StopLossPct/100)

	trailingActive := false
	trailingPeak := entryPrice

	for i, c := range candles {
		if trailingActive && c.High > trailingPeak {
			trailingPeak = c.High
		}

		if c.High >= targetPrice {
			return SimTradeResult{
				EntryPrice:     entryPrice,
				ExitPrice:      targetPrice,
				ExitReason:     "target",
				PnlPct:         pnlPct(entryPrice, targetPrice) - commissionPct,
				HoldingCandles: i + 1,
			}
		}

		if c.Low <= stopPrice {
			return SimTradeResult{
				EntryPrice:     entryPrice,
				ExitPrice:      stopPrice,
				ExitReason:     "stop",
				PnlPct:         pnlPct(entryPrice, stopPrice) - commissionPct,
				HoldingCandles: i + 1,
			}
		}

		if !trailingActive && params.TrailingTriggerPct > 0 {
			gain := (c.High/entryPrice - 1) * 100
			if gain >= params.TrailingTriggerPct {
				trailingActive = true
				trailingPeak = c.High
			}
		}

		if trailingActive && params.TrailingStopPct > 0 {
			trailStop := trailingPeak * (1 - params.TrailingStopPct/100)
			if c.Low <= trailStop {
				exitPrice := math.Max(c.Low, trailStop)
				return SimTradeResult{
					EntryPrice:     entryPrice,
					ExitPrice:      exitPrice,
					ExitReason:     "trailing",
					PnlPct:         pnlPct(entryPrice, exitPrice) - commissionPct,
					HoldingCandles: i + 1,
				}
			}
		}
	}

	lastClose := entryPrice
	if len(candles) > 0 {
		lastClose = candles[len(candles)-1].Close
	}
	return SimTradeResult{
		EntryPrice:     entryPrice,
		ExitPrice:      lastClose,
		ExitReason:     "end_of_data",
		PnlPct:         pnlPct(entryPrice, lastClose) - commissionPct,
		HoldingCandles: len(candles),
	}
}

func pnlPct(entry, exit float64) float64 {
	if entry == 0 {
		return 0
	}
	return (exit/entry - 1) * 100
}
