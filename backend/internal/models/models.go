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

// Order represents a single stock trade order.
type Order struct {
	ID          int64       `json:"id"`
	StockCode   string      `json:"stock_code"`
	StockName   string      `json:"stock_name"` // 종목명 (KIS 히스토리 동기화 시 채워짐)
	OrderType   OrderType   `json:"order_type"`
	Qty         int         `json:"qty"`
	Price       float64     `json:"price"`
	FilledPrice float64     `json:"filled_price"` // 체결가 (체결 후 avg_prvs 기준)
	Status      OrderStatus `json:"status"`
	KISOrderID  string      `json:"kis_order_id"`
	Source      OrderSource `json:"source"`      // AGENT: 에이전트 주문 / MANUAL: 수동 거래 감지
	TargetPct   float64     `json:"target_pct"`  // 목표 수익률 (%)
	StopPct     float64     `json:"stop_pct"`    // 손절 비율 (%)
	SellReason  string      `json:"sell_reason"` // 매도 사유 (자동 매도 시만 값 있음)
	Market      string      `json:"market"`      // "KR" | "US"
	CreatedAt   time.Time   `json:"created_at"`
}

// MonitoredPosition is a buy position being watched for target/stop price hits.
type MonitoredPosition struct {
	ID          int64     `json:"id"`
	StockCode   string    `json:"stock_code"`
	StockName   string    `json:"stock_name"`
	FilledPrice float64   `json:"filled_price"`
	TargetPrice float64   `json:"target_price"` // FilledPrice × (1 + target_pct/100)
	StopPrice   float64   `json:"stop_price"`   // FilledPrice × (1 - stop_pct/100)
	OrderID     int64     `json:"order_id"`
	Market      string    `json:"market"` // "KR" | "US"
	CreatedAt   time.Time `json:"created_at"`
}

// Balance is a point-in-time snapshot of the account balance.
type Balance struct {
	ID              int64     `json:"id"`
	TotalEval       float64   `json:"total_eval"`
	AvailableAmount float64   `json:"available_amount"`
	RecordedAt      time.Time `json:"recorded_at"`
}

// KISAPILog records every KIS API error response for audit and debugging.
type KISAPILog struct {
	ID          int64     `json:"id"`
	Endpoint    string    `json:"endpoint"`
	ErrorCode   string    `json:"error_code"`
	ErrorMsg    string    `json:"error_message"`
	RawResponse string    `json:"raw_response"`
	Timestamp   time.Time `json:"timestamp"`
}

// TraderSelectionLog records each LLM stock selection attempt for UI display.
type TraderSelectionLog struct {
	ID             int64  `json:"id"`
	Timestamp      string `json:"timestamp"`
	SentCount      int    `json:"sent_count"`
	Candidates     string `json:"candidates"`    // JSON string — full ranking list sent to LLM
	LLMResult      string `json:"llm_result"`    // JSON string — ordered StockCandidate list
	SelectedCode   string `json:"selected_code"` // empty if no fill occurred
	SelectedReason string `json:"selected_reason"`
	FailReason     string `json:"fail_reason"` // LLM 오류 또는 주문 실패 사유
}

// TraderRankingLog records each getRankings() attempt for UI display.
type TraderRankingLog struct {
	ID                int64     `json:"id"`
	Timestamp         time.Time `json:"timestamp"`
	RankingTypes      string    `json:"ranking_types"` // JSON string: ["volume","strength"]
	PriceMin          string    `json:"price_min"`
	PriceMax          string    `json:"price_max"`
	VolumeCount       int       `json:"volume_count"` // -1 = 타입 미사용
	StrengthCount     int       `json:"strength_count"`
	ExecCountCount    int       `json:"exec_count_count"`
	DisparityCount    int       `json:"disparity_count"`
	IntersectionCount int       `json:"intersection_count"` // AND 교집합 결과
	ErrorMessage      string    `json:"error_message"`
}

// Token stores the KIS OAuth access token and its validity window.
type Token struct {
	ID          int64     `json:"id"`
	AccessToken string    `json:"access_token"`
	IssuedAt    time.Time `json:"issued_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}
