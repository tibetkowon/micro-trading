package ops

import (
	"context"
	"fmt"
	"strconv"

	"github.com/micro-trading-for-agent/backend/internal/database"
	"github.com/micro-trading-for-agent/backend/internal/kis"
	"github.com/micro-trading-for-agent/backend/internal/models"
)

// AccountBalance holds the parsed account balance for dashboard display.
type AccountBalance struct {
	TotalEval          float64 `json:"total_eval"`
	WithdrawableAmount float64 `json:"withdrawable_amount"`
	OrderableAmt       float64 `json:"orderable_amt"`
	AssetChangeAmt     float64 `json:"asset_change_amt"`
	AssetChangeRate    string  `json:"asset_change_rate"`
}

// GetAccountBalance fetches account balance via TTTC8434R and saves a snapshot.
func GetAccountBalance(ctx context.Context, client *kis.Client, db *database.DB) (*AccountBalance, error) {
	summary, err := client.GetInquireBalance(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetInquireBalance: %w", err)
	}

	totalEval, _ := strconv.ParseFloat(summary.TotalEval, 64)
	withdrawable, _ := strconv.ParseFloat(summary.DepositAmt, 64)
	orderableAmt, _ := strconv.ParseFloat(summary.OrderableAmt, 64)
	assetChangeAmt, _ := strconv.ParseFloat(summary.AssetChangeAmt, 64)
	prevTotal, _ := strconv.ParseFloat(summary.PrevTotalAsset, 64)

	assetChangeRate := "-"
	if prevTotal > 0 {
		rate := assetChangeAmt / prevTotal * 100
		assetChangeRate = fmt.Sprintf("%.2f", rate)
	}

	_ = db.CreateBalance(ctx, totalEval, withdrawable)

	return &AccountBalance{
		TotalEval:          totalEval,
		WithdrawableAmount: withdrawable,
		OrderableAmt:       orderableAmt,
		AssetChangeAmt:     assetChangeAmt,
		AssetChangeRate:    assetChangeRate,
	}, nil
}

// GetLatestBalanceFromDB returns the most recent balance snapshot from the database.
func GetLatestBalanceFromDB(ctx context.Context, db *database.DB) (*models.Balance, error) {
	return db.GetLatestBalance(ctx)
}
