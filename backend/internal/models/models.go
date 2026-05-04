package models

import "time"

// Setting stores key-value configuration pairs (e.g., KIS credentials).
type Setting struct {
	Key       string    `json:"key" firestore:"key"`
	Value     string    `json:"value" firestore:"value"`
	UpdatedAt time.Time `json:"updated_at" firestore:"updated_at"`
}

// OrderType represents buy or sell.
type OrderType string

const (
	OrderTypeBuy  OrderType = "BUY"
	OrderTypeSell OrderType = "SELL"
)

// OrderSource distinguishes agent-placed orders from manually detected ones.
type OrderSource string

const (
	OrderSourceAgent  OrderSource = "AGENT"
	OrderSourceManual OrderSource = "MANUAL"
)

// OrderStatus tracks the lifecycle of an order.
type OrderStatus string

const (
	OrderStatusPending         OrderStatus = "PENDING"
	OrderStatusFilled          OrderStatus = "FILLED"
	OrderStatusPartiallyFilled OrderStatus = "PARTIALLY_FILLED"
	OrderStatusCancelled       OrderStatus = "CANCELLED"
	OrderStatusFailed          OrderStatus = "FAILED"
)

// Order represents a single stock trade order (KR market only).
type Order struct {
	ID          int64       `json:"id" firestore:"id"`
	StockCode   string      `json:"stock_code" firestore:"stock_code"`
	StockName   string      `json:"stock_name" firestore:"stock_name"`
	OrderType   OrderType   `json:"order_type" firestore:"order_type"`
	Qty         int         `json:"qty" firestore:"qty"`
	Price       float64     `json:"price" firestore:"price"`
	FilledPrice float64     `json:"filled_price" firestore:"filled_price"`
	Status      OrderStatus `json:"status" firestore:"status"`
	KISOrderID  string      `json:"kis_order_id" firestore:"kis_order_id"`
	Source      OrderSource `json:"source" firestore:"source"`
	TargetPct   float64     `json:"target_pct" firestore:"target_pct"`
	StopPct     float64     `json:"stop_pct" firestore:"stop_pct"`
	SellReason  string      `json:"sell_reason" firestore:"sell_reason"`
	CreatedAt   time.Time   `json:"created_at" firestore:"created_at"`
}

// MonitoredPosition is a buy position being watched for target/stop price hits.
type MonitoredPosition struct {
	ID          int64     `json:"id" firestore:"id"`
	StockCode   string    `json:"stock_code" firestore:"stock_code"`
	StockName   string    `json:"stock_name" firestore:"stock_name"`
	FilledPrice float64   `json:"filled_price" firestore:"filled_price"`
	TargetPrice float64   `json:"target_price" firestore:"target_price"`
	StopPrice   float64   `json:"stop_price" firestore:"stop_price"`
	OrderID     int64     `json:"order_id" firestore:"order_id"`
	CreatedAt   time.Time `json:"created_at" firestore:"created_at"`
}

// Balance is a point-in-time snapshot of the account balance.
type Balance struct {
	ID              int64     `json:"id" firestore:"id"`
	TotalEval       float64   `json:"total_eval" firestore:"total_eval"`
	AvailableAmount float64   `json:"available_amount" firestore:"available_amount"`
	RecordedAt      time.Time `json:"recorded_at" firestore:"recorded_at"`
}

// ServiceLog records service-level events for central display.
type ServiceLog struct {
	ID        int64     `json:"id" firestore:"id"`
	Source    string    `json:"source" firestore:"source"` // TRADER / MONITOR / SYSTEM
	Level     string    `json:"level" firestore:"level"`   // ERROR / WARN / INFO
	Message   string    `json:"message" firestore:"message"`
	Detail    string    `json:"detail" firestore:"detail"`
	Timestamp time.Time `json:"timestamp" firestore:"timestamp"`
}

// KISAPILog records every KIS API error response for audit and debugging.
type KISAPILog struct {
	ID          int64     `json:"id" firestore:"id"`
	Endpoint    string    `json:"endpoint" firestore:"endpoint"`
	ErrorCode   string    `json:"error_code" firestore:"error_code"`
	ErrorMsg    string    `json:"error_message" firestore:"error_message"`
	RawResponse string    `json:"raw_response" firestore:"raw_response"`
	Timestamp   time.Time `json:"timestamp" firestore:"timestamp"`
}

// ScanLog records each scanner cycle: rankings → hard filter → score → order decision.
type ScanLog struct {
	ID              int64  `json:"id" firestore:"id"`
	Timestamp       string `json:"timestamp" firestore:"timestamp"`
	TotalCandidates int    `json:"total_candidates" firestore:"total_candidates"` // 랭킹 API에서 가져온 전체 후보 수
	StocksFound     int    `json:"stocks_found" firestore:"stocks_found"`         // 하드 필터 통과 후 후보 수
	TopStocks       string `json:"top_stocks" firestore:"top_stocks"`             // JSON: 상위 점수 종목 목록
	Ordered         bool   `json:"ordered" firestore:"ordered"`
	OrderedCode     string `json:"ordered_code" firestore:"ordered_code"`
	SkipReason      string `json:"skip_reason" firestore:"skip_reason"`
	ScoreStats      string `json:"score_stats" firestore:"score_stats"` // JSON: 점수 통계
}

// TradeReport records each trade lifecycle (buy → sell) with indicators and reasoning.
type TradeReport struct {
	ID             int64      `json:"id" firestore:"id"`
	Date           string     `json:"date" firestore:"date"` // YYYY-MM-DD
	StockCode      string     `json:"stock_code" firestore:"stock_code"`
	StockName      string     `json:"stock_name" firestore:"stock_name"`
	BuyOrderID     int64      `json:"buy_order_id" firestore:"buy_order_id"`
	SellOrderID    int64      `json:"sell_order_id" firestore:"sell_order_id"`
	ScanLogID      int64      `json:"scan_log_id" firestore:"scan_log_id"` // 연결된 ScanLog.ID (0=없음)
	BuyPrice       float64    `json:"buy_price" firestore:"buy_price"`
	BuyQty         int        `json:"buy_qty" firestore:"buy_qty"`
	BuyAmount      float64    `json:"buy_amount" firestore:"buy_amount"`
	BuyReason      string     `json:"buy_reason" firestore:"buy_reason"`
	BuyIndicators  string     `json:"buy_indicators" firestore:"buy_indicators"` // JSON snapshot
	SellPrice      float64    `json:"sell_price" firestore:"sell_price"`
	SellQty        int        `json:"sell_qty" firestore:"sell_qty"`
	SellAmount     float64    `json:"sell_amount" firestore:"sell_amount"`
	SellReason     string     `json:"sell_reason" firestore:"sell_reason"`
	SellIndicators string     `json:"sell_indicators" firestore:"sell_indicators"`
	ProfitAmount   float64    `json:"profit_amount" firestore:"profit_amount"`
	ProfitPct      float64    `json:"profit_pct" firestore:"profit_pct"`
	CreatedAt      time.Time  `json:"created_at" firestore:"created_at"`
	SoldAt         *time.Time `json:"sold_at" firestore:"sold_at"`
}

// DailyReport summarizes all completed trades for a single trading day.
type DailyReport struct {
	ID                int64     `json:"id" firestore:"id"`
	Date              string    `json:"date" firestore:"date"` // YYYY-MM-DD (unique)
	TotalTrades       int       `json:"total_trades" firestore:"total_trades"`
	WinningTrades     int       `json:"winning_trades" firestore:"winning_trades"`
	LosingTrades      int       `json:"losing_trades" firestore:"losing_trades"`
	TotalProfitAmount float64   `json:"total_profit_amount" firestore:"total_profit_amount"`
	AvgProfitPct      float64   `json:"avg_profit_pct" firestore:"avg_profit_pct"`
	BestTrade         string    `json:"best_trade" firestore:"best_trade"`       // JSON summary
	WorstTrade        string    `json:"worst_trade" firestore:"worst_trade"`     // JSON summary
	TradeSummary      string    `json:"trade_summary" firestore:"trade_summary"` // JSON array
	CreatedAt         time.Time `json:"created_at" firestore:"created_at"`
}

// Token stores the KIS OAuth access token and its validity window.
type Token struct {
	ID          int64     `json:"id" firestore:"id"`
	AccessToken string    `json:"access_token" firestore:"access_token"`
	IssuedAt    time.Time `json:"issued_at" firestore:"issued_at"`
	ExpiresAt   time.Time `json:"expires_at" firestore:"expires_at"`
}
