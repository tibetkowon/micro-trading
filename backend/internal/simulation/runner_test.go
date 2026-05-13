package simulation_test

import (
	"testing"
	"time"

	"github.com/micro-trading-for-agent/backend/internal/simulation"
)

func TestComputeActualPnlUsesSubset(t *testing.T) {
	items := []simulation.TradeCandles{
		{ProfitPct: 2.0},
		{ProfitPct: -1.0},
	}
	got := simulation.ComputeActualPnl(items)
	want := 1.0
	if got != want {
		t.Errorf("ComputeActualPnl = %f, want %f", got, want)
	}
}

func TestComputeActualPnlEmpty(t *testing.T) {
	got := simulation.ComputeActualPnl(nil)
	if got != 0 {
		t.Errorf("ComputeActualPnl(nil) = %f, want 0", got)
	}
}

func TestFilterAndConvertBars_DeduplicatesAndFilters(t *testing.T) {
	kst, _ := time.LoadLocation("Asia/Seoul")
	kisDate := "20260513"
	buyKST := time.Date(2026, 5, 13, 9, 15, 0, 0, kst)

	bars := []simulation.RawBar{
		{Time: "091500", High: "10100", Low: "10000", Close: "10050"},
		{Time: "091500", High: "10100", Low: "10000", Close: "10050"},
		{Time: "091600", High: "10200", Low: "10050", Close: "10150"},
		{Time: "090000", High: "9900", Low: "9800", Close: "9850"},
	}

	candles, err := simulation.FilterAndConvertBars(bars, kisDate, buyKST, kst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candles) != 2 {
		t.Errorf("want 2 candles (dedup + filter), got %d", len(candles))
	}
}
