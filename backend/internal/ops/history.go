package ops

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/micro-trading-for-agent/backend/internal/database"
	"github.com/micro-trading-for-agent/backend/internal/kis"
	"github.com/micro-trading-for-agent/backend/internal/logger"
	"github.com/micro-trading-for-agent/backend/internal/models"
)

// GetOrderHistory returns KIS execution history and syncs it to the local DB.
// For KIS orders not yet in DB (manually placed), a new record is inserted with source=MANUAL.
func GetOrderHistory(ctx context.Context, client *kis.Client, db *database.DB, startDate, endDate string) ([]map[string]any, error) {
	history, err := client.GetOrderHistory(ctx, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("GetOrderHistory: %w", err)
	}

	for _, h := range history {
		kisOrderID, _ := h["odno"].(string)
		if kisOrderID == "" {
			continue
		}

		stockCode, _ := h["pdno"].(string)
		stockName, _ := h["prdt_name"].(string)
		ccldQty, _ := h["tot_ccld_qty"].(string)
		ordQty, _ := h["ord_qty"].(string)
		avgPrvs, _ := h["avg_prvs"].(string)
		ordUnpr, _ := h["ord_unpr"].(string)
		sllBuy, _ := h["sll_buy_dvsn_cd"].(string)
		ordDt, _ := h["ord_dt"].(string)
		ordTmd, _ := h["ord_tmd"].(string)
		cancYn, _ := h["cncl_yn"].(string)

		existing, _ := db.GetOrderByKISID(ctx, kisOrderID)

		if existing != nil {
			// Update stock name if empty
			if stockName != "" && existing.StockName == "" {
				_ = db.UpdateOrderStockName(ctx, kisOrderID, stockName)
			}

			if ccldQty == "" || ccldQty == "0" {
				continue
			}

			newStatus := models.OrderStatusFilled
			if ordQty != "" && ordQty != ccldQty {
				newStatus = models.OrderStatusPartiallyFilled
			}
			if existing.Status == models.OrderStatusPending || existing.Status == models.OrderStatusPartiallyFilled {
				filledPrice, _ := strconv.ParseFloat(avgPrvs, 64)
				_ = db.UpdateOrderFilled(ctx, kisOrderID, newStatus, filledPrice)
			}
			continue
		}

		// New record: manual trade detected
		if stockCode == "" || cancYn == "Y" {
			continue
		}

		orderType := models.OrderTypeBuy
		if sllBuy == "01" {
			orderType = models.OrderTypeSell
		}

		var status models.OrderStatus
		switch {
		case ccldQty == "" || ccldQty == "0":
			status = models.OrderStatusPending
		case ordQty != "" && ordQty == ccldQty:
			status = models.OrderStatusFilled
		default:
			status = models.OrderStatusPartiallyFilled
		}

		price, _ := strconv.ParseFloat(ordUnpr, 64)
		filledPrice, _ := strconv.ParseFloat(avgPrvs, 64)
		qty, _ := strconv.Atoi(ordQty)
		if qty <= 0 {
			continue
		}

		orderedAt := time.Now()
		if ordDt != "" && ordTmd != "" {
			if t, err := time.ParseInLocation("20060102150405", ordDt+ordTmd, time.Local); err == nil {
				orderedAt = t
			}
		} else if ordDt != "" {
			if t, err := time.ParseInLocation("20060102", ordDt, time.Local); err == nil {
				orderedAt = t
			}
		}

		o := &models.Order{
			StockCode:   stockCode,
			StockName:   stockName,
			OrderType:   orderType,
			Qty:         qty,
			Price:       price,
			FilledPrice: filledPrice,
			Status:      status,
			KISOrderID:  kisOrderID,
			Source:      models.OrderSourceManual,
			CreatedAt:   orderedAt,
		}
		if err := db.UpsertManualOrder(ctx, o); err != nil {
			logger.Warn("manual trade insert failed", map[string]any{
				"kis_order_id": kisOrderID,
				"error":        err.Error(),
			})
		} else {
			logger.Info("manual trade imported", map[string]any{
				"kis_order_id": kisOrderID,
				"stock_code":   stockCode,
				"order_type":   string(orderType),
				"status":       string(status),
			})
		}
	}

	return history, nil
}

// StartOrderSyncScheduler runs GetOrderHistory every interval in a background goroutine.
func StartOrderSyncScheduler(ctx context.Context, client *kis.Client, db *database.DB, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				open, err := IsMarketOpen(ctx, client)
				if err != nil {
					logger.Warn("market status check failed — skipping sync", map[string]any{"error": err.Error()})
					continue
				}
				if !open {
					logger.Info("market closed — skipping order sync", nil)
					continue
				}
				today := time.Now().Format("20060102")
				if _, err := GetOrderHistory(ctx, client, db, today, today); err != nil {
					logger.Warn("order sync failed", map[string]any{"error": err.Error()})
				} else {
					logger.Info("order sync completed", nil)
				}
			}
		}
	}()
}

// GetLocalOrderHistory returns paginated orders from the local database.
func GetLocalOrderHistory(ctx context.Context, db *database.DB, since time.Time, limit, offset int) ([]models.Order, error) {
	return db.ListOrdersSince(ctx, since, limit, offset)
}
