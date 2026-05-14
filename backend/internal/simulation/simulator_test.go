package simulation_test

import (
	"math"
	"testing"

	"github.com/micro-trading-for-agent/backend/internal/simulation"
)

func TestSimulateTrade_HitsTarget(t *testing.T) {
	candles := []simulation.MinuteCandle{
		{High: 10050, Low: 9980, Close: 10050},
		{High: 10350, Low: 10100, Close: 10300},
	}
	params := simulation.SimParams{
		TakeProfitPct:      3.0,
		StopLossPct:        2.0,
		TrailingTriggerPct: 0,
		TrailingStopPct:    0,
	}
	result := simulation.SimulateTrade(10000, candles, params)
	if result.ExitReason != "target" {
		t.Errorf("want exit reason 'target', got %q", result.ExitReason)
	}
	if result.ExitPrice != 10300 {
		t.Errorf("want exit price 10300, got %f", result.ExitPrice)
	}
}

func TestSimulateTrade_CommissionDeducted(t *testing.T) {
	candles := []simulation.MinuteCandle{
		{High: 10300, Low: 10050, Close: 10300},
	}
	params := simulation.SimParams{TakeProfitPct: 3.0, StopLossPct: 2.0}
	result := simulation.SimulateTrade(10000, candles, params)
	if result.ExitReason != "target" {
		t.Fatalf("want 'target', got %q", result.ExitReason)
	}
	want := 2.75
	if math.Abs(result.PnlPct-want) > 0.000001 {
		t.Errorf("want PnlPct %.2f (commission deducted), got %.2f", want, result.PnlPct)
	}
}

func TestSimulateTrade_HitsStop(t *testing.T) {
	candles := []simulation.MinuteCandle{
		{High: 10020, Low: 9980, Close: 9990},
		{High: 9900, Low: 9750, Close: 9800},
	}
	params := simulation.SimParams{
		TakeProfitPct: 3.0,
		StopLossPct:   2.0,
	}
	result := simulation.SimulateTrade(10000, candles, params)
	if result.ExitReason != "stop" {
		t.Errorf("want exit reason 'stop', got %q", result.ExitReason)
	}
	if result.ExitPrice != 9800 {
		t.Errorf("want exit price 9800, got %f", result.ExitPrice)
	}
}

func TestSimulateTrade_TrailingStop(t *testing.T) {
	candles := []simulation.MinuteCandle{
		{High: 10200, Low: 10100, Close: 10150},
		{High: 10300, Low: 10150, Close: 10250},
		{High: 10250, Low: 10200, Close: 10220},
	}
	params := simulation.SimParams{
		TakeProfitPct:      5.0,
		StopLossPct:        2.0,
		TrailingTriggerPct: 1.5,
		TrailingStopPct:    0.7,
	}
	result := simulation.SimulateTrade(10000, candles, params)
	if result.ExitReason != "trailing" {
		t.Errorf("want exit reason 'trailing', got %q", result.ExitReason)
	}
}

func TestSimulateTrade_NoExit(t *testing.T) {
	candles := []simulation.MinuteCandle{
		{High: 10020, Low: 9990, Close: 10010},
		{High: 10030, Low: 10000, Close: 10020},
	}
	params := simulation.SimParams{TakeProfitPct: 5.0, StopLossPct: 2.0}
	result := simulation.SimulateTrade(10000, candles, params)
	if result.ExitReason != "end_of_data" {
		t.Errorf("want exit reason 'end_of_data', got %q", result.ExitReason)
	}
	if result.ExitPrice != 10020 {
		t.Errorf("want last close 10020, got %f", result.ExitPrice)
	}
}
