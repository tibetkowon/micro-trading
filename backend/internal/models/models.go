package models

import "time"

// Setting stores key-value configuration pairs (e.g., KIS credentials).
type Setting struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
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
	ID          int64       `json:"id"`
	StockCode   string      `json:"stock_code"`
	StockName   string      `json:"stock_name"`
	OrderType   OrderType   `json:"order_type"`
	Qty         int         `json:"qty"`
	Price       float64     `json:"price"`
	FilledPrice float64     `json:"filled_price"`
	Status      OrderStatus `json:"status"`
	KISOrderID  string      `json:"kis_order_id"`
	Source      OrderSource `json:"source"`
	TargetPct   float64     `json:"target_pct"`
	StopPct     float64     `json:"stop_pct"`
	SellReason  string      `json:"sell_reason"`
	CreatedAt   time.Time   `json:"created_at"`
	ExpireAt    time.Time   `json:"expire_at"`
}

// MonitoredPosition is a buy position being watched for target/stop price hits.
type MonitoredPosition struct {
	ID           int64     `json:"id"`
	StockCode    string    `json:"stock_code"`
	StockName    string    `json:"stock_name"`
	FilledPrice  float64   `json:"filled_price"`
	CurrentPrice float64   `json:"current_price"`
	TargetPrice  float64   `json:"target_price"`
	StopPrice    float64   `json:"stop_price"`
	OrderID      int64     `json:"order_id"`
	RemainingQty int       `json:"remaining_qty"` // 추가
	CreatedAt    time.Time `json:"created_at"`
}

// Balance is a point-in-time snapshot of the account balance.
type Balance struct {
	ID              int64     `json:"id"`
	TotalEval       float64   `json:"total_eval"`
	AvailableAmount float64   `json:"available_amount"`
	RecordedAt      time.Time `json:"recorded_at"`
	ExpireAt        time.Time `json:"expire_at"`
}

// ServiceLog records service-level events for central display.
type ServiceLog struct {
	ID        int64     `json:"id"`
	Source    string    `json:"source"` // TRADER / MONITOR / SYSTEM
	Level     string    `json:"level"`  // ERROR / WARN / INFO
	Message   string    `json:"message"`
	Detail    string    `json:"detail"`
	Timestamp time.Time `json:"timestamp"`
	ExpireAt  time.Time `json:"expire_at"`
}

// KISAPILog records every KIS API error response for audit and debugging.
type KISAPILog struct {
	ID             int64     `json:"id"`
	Endpoint       string    `json:"endpoint"`
	ErrorCode      string    `json:"error_code"`
	ErrorMsg       string    `json:"error_message"`
	RawResponse    string    `json:"raw_response"`
	RequestContext string    `json:"request_context"`
	Timestamp      time.Time `json:"timestamp"`
	ExpireAt       time.Time `json:"expire_at"`
}

// ScanLog records each scanner cycle: rankings → hard filter → score → order decision.
type ScanLog struct {
	ID                 int64     `json:"id"`
	Timestamp          string    `json:"timestamp"`
	TotalCandidates    int       `json:"total_candidates"`     // 랭킹 API에서 가져온 전체 후보 수
	StocksFound        int       `json:"stocks_found"`         // 하드 필터 통과 후 후보 수
	TopStocks          string    `json:"top_stocks"`           // JSON: 상위 점수 종목 목록
	StockRawData       string    `json:"stock_raw_data"`       // JSON: 점수 산정 원본 데이터 (StockInfo 전체 + ScoreDetail)
	BelowThresholdData string    `json:"below_threshold_data"` // JSON: VirtualCandidate array
	Ordered            bool      `json:"ordered"`
	OrderedCode        string    `json:"ordered_code"`
	SkipReason         string    `json:"skip_reason"`
	ScoreStats         string    `json:"score_stats"` // JSON: 점수 통계
	ExpireAt           time.Time `json:"expire_at"`
}

// VirtualCandidate is a hard-filter-passed stock rejected only by min_score_threshold.
// Stored in ScanLog for counterfactual simulation of looser entry filters.
type VirtualCandidate struct {
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	Score      float64 `json:"score"`
	Penalty    float64 `json:"penalty"`
	EntryPrice float64 `json:"entry_price"`
	BuyTime    string  `json:"buy_time"`
}

// TradeReport records each trade lifecycle (buy → sell) with indicators and reasoning.
type TradeReport struct {
	ID             int64      `json:"id"`
	Date           string     `json:"date"` // YYYY-MM-DD
	StockCode      string     `json:"stock_code"`
	StockName      string     `json:"stock_name"`
	BuyOrderID     int64      `json:"buy_order_id"`
	SellOrderID    int64      `json:"sell_order_id"`
	ScanLogID      int64      `json:"scan_log_id"` // 연결된 ScanLog.ID (0=없음)
	BuyPrice       float64    `json:"buy_price"`
	BuyQty         int        `json:"buy_qty"`
	BuyAmount      float64    `json:"buy_amount"`
	BuyReason      string     `json:"buy_reason"`
	BuyIndicators  string     `json:"buy_indicators"` // JSON snapshot
	SellPrice      float64    `json:"sell_price"`
	SellQty        int        `json:"sell_qty"`
	SellAmount     float64    `json:"sell_amount"`
	SellReason     string     `json:"sell_reason"`
	SellIndicators string     `json:"sell_indicators"`
	ProfitAmount   float64    `json:"profit_amount"`
	ProfitPct      float64    `json:"profit_pct"`
	CreatedAt      time.Time  `json:"created_at"`
	SoldAt         *time.Time `json:"sold_at"`
	ExpireAt       time.Time  `json:"expire_at"`
}

// DailyReport summarizes all completed trades for a single trading day.
type DailyReport struct {
	ID                int64     `json:"id"`
	Date              string    `json:"date"` // YYYY-MM-DD (unique)
	TotalTrades       int       `json:"total_trades"`
	WinningTrades     int       `json:"winning_trades"`
	LosingTrades      int       `json:"losing_trades"`
	TotalProfitAmount float64   `json:"total_profit_amount"`
	AvgProfitPct      float64   `json:"avg_profit_pct"`
	BestTrade         string    `json:"best_trade"`    // JSON summary
	WorstTrade        string    `json:"worst_trade"`   // JSON summary
	TradeSummary      string    `json:"trade_summary"` // JSON array
	CreatedAt         time.Time `json:"created_at"`
	ExpireAt          time.Time `json:"expire_at"`
}

// SimulationResult stores post-market scenario simulation results.
type SimulationResult struct {
	ID              int64     `json:"id"`
	Date            string    `json:"date"`
	ScenariosJSON   string    `json:"scenarios_json"`
	RecommendedJSON string    `json:"recommended_json"`
	CreatedAt       time.Time `json:"created_at"`
	ExpireAt        time.Time `json:"expire_at"`
}

// ScoreComponents stores per-indicator normalized scores (0-100) at buy time.
// Defined here (not in scorer) to avoid import cycles.
type ScoreComponents struct {
	Strength    float64 `json:"strength"`
	RSI         float64 `json:"rsi"`
	MACD        float64 `json:"macd"`
	BidAsk      float64 `json:"bid_ask"`
	VWAP        float64 `json:"vwap"`
	Volume      float64 `json:"volume"`
	ProgramBuy  float64 `json:"program_buy"`
	MicroBidAsk float64 `json:"micro_bid_ask"`
	VIDisparity float64 `json:"vi_disparity"`
}

// BuyIndicatorsSnapshot holds the raw indicator values at the time of a buy order.
// Stored as JSON in TradeReport.BuyIndicators for frontend consumption.
type BuyIndicatorsSnapshot struct {
	RSI             float64          `json:"rsi"`
	MACDBullish     bool             `json:"macd_bullish"`
	VWAPDisparity   float64          `json:"vwap_disparity"`
	Strength        float64          `json:"strength"`
	BidAskRatio     float64          `json:"bid_ask_ratio"` // 0 when bid-ask fetch is skipped (score weight = 0 and spread filter disabled)
	TotalScore      float64          `json:"total_score"`
	Penalty         float64          `json:"penalty"`
	ScoreComponents *ScoreComponents `json:"score_components,omitempty"`
}

// Token stores the KIS OAuth access token and its validity window.
type Token struct {
	ID          int64     `json:"id"`
	AccessToken string    `json:"access_token"`
	IssuedAt    time.Time `json:"issued_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	ExpireAt    time.Time `json:"expire_at"`
}
