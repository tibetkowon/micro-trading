package report

import (
	"testing"
	"time"

	"github.com/micro-trading-for-agent/backend/internal/models"
)

func filterDailyReportsByRange(reports []models.DailyReport, from, to string) []models.DailyReport {
	var out []models.DailyReport
	for _, r := range reports {
		if r.Date >= from && r.Date <= to {
			out = append(out, r)
		}
	}
	return out
}

func TestFilterDailyReportsByRange(t *testing.T) {
	reports := []models.DailyReport{
		{Date: "2026-05-10", TotalTrades: 3},
		{Date: "2026-05-11", TotalTrades: 5},
		{Date: "2026-05-12", TotalTrades: 2},
		{Date: "2026-05-13", TotalTrades: 4},
	}
	got := filterDailyReportsByRange(reports, "2026-05-11", "2026-05-12")
	if len(got) != 2 {
		t.Fatalf("want 2 reports, got %d", len(got))
	}
	if got[0].Date != "2026-05-11" || got[1].Date != "2026-05-12" {
		t.Errorf("unexpected dates: %v %v", got[0].Date, got[1].Date)
	}
}

func TestBuildExportSummary(t *testing.T) {
	now := time.Now()
	trades := []models.TradeReport{
		{ProfitAmount: 10000, ProfitPct: 2.0, SoldAt: ptr(now)},
		{ProfitAmount: -5000, ProfitPct: -1.0, SoldAt: ptr(now)},
		{ProfitAmount: 3000, ProfitPct: 1.5, SoldAt: ptr(now)},
	}
	sum := buildExportSummary(trades, "2026-05-11", "2026-05-13")
	if sum.TotalTrades != 3 {
		t.Errorf("want TotalTrades=3, got %d", sum.TotalTrades)
	}
	if sum.WinningTrades != 2 {
		t.Errorf("want WinningTrades=2, got %d", sum.WinningTrades)
	}
	if sum.TotalProfitAmount != 8000 {
		t.Errorf("want TotalProfitAmount=8000, got %f", sum.TotalProfitAmount)
	}
}

func ptr[T any](v T) *T { return &v }
