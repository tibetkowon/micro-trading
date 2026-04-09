package report

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/micro-trading-for-agent/backend/internal/database"
	"github.com/micro-trading-for-agent/backend/internal/models"
)

// tradeSummaryItem is a compact summary of one trade for BestTrade/WorstTrade/TradeSummary JSON fields.
type tradeSummaryItem struct {
	ID           int64   `json:"id"`
	StockCode    string  `json:"stock_code"`
	StockName    string  `json:"stock_name"`
	BuyPrice     float64 `json:"buy_price"`
	SellPrice    float64 `json:"sell_price"`
	Qty          int     `json:"qty"`
	ProfitAmount float64 `json:"profit_amount"`
	ProfitPct    float64 `json:"profit_pct"`
	BuyReason    string  `json:"buy_reason"`
	SellReason   string  `json:"sell_reason"`
}

func toSummary(t models.TradeReport) tradeSummaryItem {
	return tradeSummaryItem{
		ID:           t.ID,
		StockCode:    t.StockCode,
		StockName:    t.StockName,
		BuyPrice:     t.BuyPrice,
		SellPrice:    t.SellPrice,
		Qty:          t.BuyQty,
		ProfitAmount: t.ProfitAmount,
		ProfitPct:    t.ProfitPct,
		BuyReason:    t.BuyReason,
		SellReason:   t.SellReason,
	}
}

// GenerateDailyReport aggregates all completed (sold) trade_reports for the given date
// and upserts a daily_reports record.
// date: "YYYY-MM-DD". Empty string defaults to today KST.
func GenerateDailyReport(ctx context.Context, db *database.DB, date string) error {
	if date == "" {
		kst, _ := time.LoadLocation("Asia/Seoul")
		date = time.Now().In(kst).Format("2006-01-02")
	}

	// sold_at 날짜 기준으로 집계 (매수일 기준 시 전날 매수+오늘 매도 거래가 누락됨)
	completed, err := db.GetCompletedTradesBySoldDate(ctx, date)
	if err != nil {
		return fmt.Errorf("GetCompletedTradesBySoldDate: %w", err)
	}

	total := len(completed)
	var winning, losing int
	var totalProfit, totalProfitPct float64
	var bestIdx, worstIdx int = -1, -1

	for i, t := range completed {
		totalProfit += t.ProfitAmount
		totalProfitPct += t.ProfitPct
		if t.ProfitAmount > 0 {
			winning++
		} else if t.ProfitAmount < 0 {
			losing++
		}
		if bestIdx == -1 || t.ProfitPct > completed[bestIdx].ProfitPct {
			bestIdx = i
		}
		if worstIdx == -1 || t.ProfitPct < completed[worstIdx].ProfitPct {
			worstIdx = i
		}
	}

	avgProfitPct := 0.0
	if total > 0 {
		avgProfitPct = totalProfitPct / float64(total)
	}

	bestJSON := "null"
	if bestIdx >= 0 {
		b, _ := json.Marshal(toSummary(completed[bestIdx]))
		bestJSON = string(b)
	}
	worstJSON := "null"
	if worstIdx >= 0 {
		b, _ := json.Marshal(toSummary(completed[worstIdx]))
		worstJSON = string(b)
	}
	summaryItems := make([]tradeSummaryItem, len(completed))
	for i, t := range completed {
		summaryItems[i] = toSummary(t)
	}
	summaryJSON, _ := json.Marshal(summaryItems)

	dr := models.DailyReport{
		Date:              date,
		TotalTrades:       total,
		WinningTrades:     winning,
		LosingTrades:      losing,
		TotalProfitAmount: totalProfit,
		AvgProfitPct:      avgProfitPct,
		BestTrade:         bestJSON,
		WorstTrade:        worstJSON,
		TradeSummary:      string(summaryJSON),
	}

	return db.InsertOrUpdateDailyReport(ctx, dr)
}
