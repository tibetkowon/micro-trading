package simulation

import (
	"testing"
	"time"

	"github.com/micro-trading-for-agent/backend/internal/models"
)

func TestCalcMaxDrawdown_Basic(t *testing.T) {
	pnls := []float64{3.0, -2.0, 1.0, -4.0}
	got := calcMaxDrawdown(pnls)
	if got != 5.0 {
		t.Errorf("want MDD 5.0, got %f", got)
	}
}

func TestCalcMaxDrawdown_AllPositive(t *testing.T) {
	pnls := []float64{1.0, 2.0, 3.0}
	got := calcMaxDrawdown(pnls)
	if got != 0.0 {
		t.Errorf("want MDD 0.0 (all positive), got %f", got)
	}
}

func TestCalcProfitFactor_Basic(t *testing.T) {
	results := []SimTradeResult{
		{PnlPct: 3.0},
		{PnlPct: -1.0},
		{PnlPct: 2.0},
		{PnlPct: -0.5},
	}
	got := calcProfitFactor(results)
	want := 3.33
	if got != want {
		t.Errorf("want ProfitFactor %.2f, got %.2f", want, got)
	}
}

func TestSortPreparedTrades_ByCreatedAt(t *testing.T) {
	now := time.Now()
	prepared := []tradeCandles{
		{trade: models.TradeReport{StockCode: "A", CreatedAt: now.Add(2 * time.Minute)}},
		{trade: models.TradeReport{StockCode: "B", CreatedAt: now}},
		{trade: models.TradeReport{StockCode: "C", CreatedAt: now.Add(time.Minute)}},
	}

	sortPreparedTrades(prepared)

	got := []string{
		prepared[0].trade.StockCode,
		prepared[1].trade.StockCode,
		prepared[2].trade.StockCode,
	}
	want := []string{"B", "C", "A"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("want sorted stock order %v, got %v", want, got)
		}
	}
}

func TestRunScenarioForTrades_OrderDependentMDDIsStable(t *testing.T) {
	entry := 10000.0
	winCandles := []MinuteCandle{{High: 10300, Low: 9990, Close: 10300}}
	lossCandles := []MinuteCandle{{High: 10020, Low: 9790, Close: 9800}}

	now := time.Now()
	earlier := now.Add(-10 * time.Minute)
	later := now

	prepared := []tradeCandles{
		{trade: models.TradeReport{StockCode: "A", BuyPrice: entry, CreatedAt: later}, candles: winCandles},
		{trade: models.TradeReport{StockCode: "B", BuyPrice: entry, CreatedAt: earlier}, candles: lossCandles},
	}
	sortPreparedTrades(prepared)

	scenario := Scenario{Label: "test", Params: SimParams{TakeProfitPct: 3.0, StopLossPct: 2.0}}
	summary, err := runScenarioForTrades(prepared, scenario)
	if err != nil {
		t.Fatal(err)
	}
	if summary.MaxDrawdownPct != 2.25 {
		t.Errorf("want MaxDrawdownPct 2.25, got %.2f", summary.MaxDrawdownPct)
	}
}

func TestRunScenarioForTrades_CooldownUsesSimulatedExitTime(t *testing.T) {
	entry := 10000.0
	winCandle := []MinuteCandle{{High: 10300, Low: 9990, Close: 10300}}
	laterCandle := []MinuteCandle{{High: 10300, Low: 9990, Close: 10300}}

	now := time.Now()
	soldAt := now.Add(-30 * time.Minute)
	reentryTime := now.Add(2 * time.Minute)

	prepared := []tradeCandles{
		{
			trade: models.TradeReport{
				StockCode: "A",
				BuyPrice:  entry,
				CreatedAt: now,
				SoldAt:    &soldAt,
			},
			candles: winCandle,
		},
		{
			trade: models.TradeReport{
				StockCode: "A",
				BuyPrice:  entry,
				CreatedAt: reentryTime,
			},
			candles: laterCandle,
		},
	}

	scenario := Scenario{
		Label: "cooldown-test",
		Params: SimParams{
			TakeProfitPct:        3.0,
			StopLossPct:          2.0,
			UniversalCooldownMin: 5,
		},
	}
	summary, err := runScenarioForTrades(prepared, scenario)
	if err != nil {
		t.Fatal(err)
	}
	if summary.TradeCount != 1 {
		t.Errorf("want TradeCount 1 (second skipped by cooldown), got %d", summary.TradeCount)
	}
}

func TestRunScenariosParallel_DeltaUsesActualNetPnl(t *testing.T) {
	entry := 10000.0
	prepared := []tradeCandles{
		{
			trade:   models.TradeReport{StockCode: "A", BuyPrice: entry, CreatedAt: time.Now()},
			candles: []MinuteCandle{{High: 10200, Low: 9990, Close: 10200}},
		},
		{
			trade:   models.TradeReport{StockCode: "B", BuyPrice: entry, CreatedAt: time.Now().Add(time.Minute)},
			candles: []MinuteCandle{{High: 10200, Low: 9990, Close: 10200}},
		},
	}
	scenarios := []Scenario{{
		Label:  "delta-test",
		Params: SimParams{TakeProfitPct: 2.0, StopLossPct: 2.0},
	}}
	actualGrossPnl := 4.0
	actualNetPnl := actualGrossPnl - commissionPct*float64(len(prepared))

	summaries := runScenariosParallel(prepared, scenarios, actualNetPnl)
	if len(summaries) != 1 {
		t.Fatalf("want 1 summary, got %d", len(summaries))
	}
	if summaries[0].DeltaVsActualPct != 0 {
		t.Errorf("want zero delta against actual net PnL, got %.2f", summaries[0].DeltaVsActualPct)
	}
}

func TestActualNetPnlBias(t *testing.T) {
	grossPnls := []float64{2.0, 2.0}
	n := float64(len(grossPnls))
	var gross float64
	for _, p := range grossPnls {
		gross += p
	}
	net := gross - commissionPct*n
	want := 3.5
	if net != want {
		t.Errorf("want actualNetPnl %.2f, got %.2f", want, net)
	}
}
