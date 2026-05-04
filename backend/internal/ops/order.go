package ops

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/micro-trading-for-agent/backend/internal/database"
	"github.com/micro-trading-for-agent/backend/internal/kis"
	"github.com/micro-trading-for-agent/backend/internal/models"
)

// CancelOrderResult is returned after a successful order cancellation.
type CancelOrderResult struct {
	OrderID    int64
	KISOrderID string
}

// PlaceOrderRequest contains the parameters for a new order.
type PlaceOrderRequest struct {
	StockCode string
	StockName string
	OrderType models.OrderType
	Qty       int
	Price     float64
	OrderDivn string  // "00"=지정가, "01"=시장가
	TargetPct float64
	StopPct   float64
}

// OrderFeasibility is returned by CheckOrderFeasibility.
type OrderFeasibility struct {
	OrderableQty  int
	AvailableCash float64
}

// CheckOrderFeasibility calls TTTC8908R for a specific stock.
func CheckOrderFeasibility(ctx context.Context, client *kis.Client, stockCode string) (*OrderFeasibility, error) {
	resp, err := client.GetAvailableOrder(ctx, stockCode)
	if err != nil {
		return nil, fmt.Errorf("GetAvailableOrder(%s): %w", stockCode, err)
	}
	qty, _ := strconv.Atoi(resp.OrderableQty)
	cash, _ := strconv.ParseFloat(resp.AvailableCash, 64)
	return &OrderFeasibility{OrderableQty: qty, AvailableCash: cash}, nil
}

// PlaceOrderResult is returned after successfully submitting an order.
type PlaceOrderResult struct {
	OrderID    int64
	KISOrderID string
	Status     models.OrderStatus
}

// PlaceOrder submits a buy or sell order to KIS and records it in the DB.
func PlaceOrder(ctx context.Context, client *kis.Client, db *database.DB, req PlaceOrderRequest) (*PlaceOrderResult, error) {
	kisReq := kis.OrderRequest{
		StockCode: req.StockCode,
		OrderDivn: req.OrderDivn,
		Qty:       fmt.Sprintf("%d", req.Qty),
		Price:     fmt.Sprintf("%.0f", req.Price),
	}

	var (
		kisResp *kis.OrderResponse
		err     error
	)
	switch req.OrderType {
	case models.OrderTypeBuy:
		kisResp, err = client.PlaceBuyOrder(ctx, kisReq)
	case models.OrderTypeSell:
		kisResp, err = client.PlaceSellOrder(ctx, kisReq)
	default:
		return nil, fmt.Errorf("unknown order type: %s", req.OrderType)
	}

	status := models.OrderStatusPending
	kisOrderID := ""
	if err != nil {
		status = models.OrderStatusFailed
	} else {
		kisOrderID = kisResp.KISOrderID
	}

	o := &models.Order{
		StockCode:  req.StockCode,
		StockName:  req.StockName,
		OrderType:  req.OrderType,
		Qty:        req.Qty,
		Price:      req.Price,
		Status:     status,
		KISOrderID: kisOrderID,
		Source:     models.OrderSourceAgent,
		TargetPct:  req.TargetPct,
		StopPct:    req.StopPct,
		CreatedAt:  time.Now().UTC(),
	}
	orderID, dbErr := db.CreateOrder(ctx, o)
	if dbErr != nil {
		return nil, fmt.Errorf("persist order: %w", dbErr)
	}

	if err != nil {
		return nil, fmt.Errorf("PlaceOrder KIS error: %w", err)
	}

	return &PlaceOrderResult{
		OrderID:    orderID,
		KISOrderID: kisOrderID,
		Status:     status,
	}, nil
}

// CancelOrder cancels a pending KIS order identified by its local DB id.
func CancelOrder(ctx context.Context, client *kis.Client, db *database.DB, orderID int64) (*CancelOrderResult, error) {
	o, err := db.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("order %d not found: %w", orderID, err)
	}
	if o.KISOrderID == "" {
		return nil, fmt.Errorf("order %d has no KIS order ID (may have failed on submission)", orderID)
	}
	if o.Status == models.OrderStatusCancelled {
		return nil, fmt.Errorf("order %d is already cancelled", orderID)
	}
	if o.Status == models.OrderStatusFilled {
		return nil, fmt.Errorf("order %d is already filled and cannot be cancelled", orderID)
	}

	cancellable, err := client.GetCancellableOrders(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetCancellableOrders: %w", err)
	}

	var found *kis.CancellableOrderItem
	for i := range cancellable {
		if cancellable[i].Odno == o.KISOrderID {
			found = &cancellable[i]
			break
		}
	}
	if found == nil {
		return nil, fmt.Errorf("order %s not found in KIS cancellable list", o.KISOrderID)
	}

	psblQty, _ := strconv.Atoi(found.PsblQty)
	if psblQty <= 0 {
		return nil, fmt.Errorf("order %s has no cancellable quantity", o.KISOrderID)
	}

	cancelResp, err := client.CancelKISOrder(ctx, found.OrdGnoBrno, o.KISOrderID, found.OrdDvsnCd, found.OrdUnpr)
	if err != nil {
		return nil, fmt.Errorf("CancelKISOrder: %w", err)
	}

	if dbErr := db.UpdateOrderStatus(ctx, orderID, models.OrderStatusCancelled); dbErr != nil {
		return &CancelOrderResult{OrderID: orderID, KISOrderID: cancelResp.KISOrderID},
			fmt.Errorf("KIS cancel succeeded but DB update failed: %w", dbErr)
	}

	return &CancelOrderResult{
		OrderID:    orderID,
		KISOrderID: cancelResp.KISOrderID,
	}, nil
}
