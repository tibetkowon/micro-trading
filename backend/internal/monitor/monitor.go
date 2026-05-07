package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/micro-trading-for-agent/backend/internal/database"
	"github.com/micro-trading-for-agent/backend/internal/kis"
	"github.com/micro-trading-for-agent/backend/internal/logger"
	"github.com/micro-trading-for-agent/backend/internal/models"
	"github.com/micro-trading-for-agent/backend/internal/ops"
	"github.com/micro-trading-for-agent/backend/internal/stockmaster"
)

// IndicatorSnapshot holds key technical indicators for sell condition evaluation.
type IndicatorSnapshot struct {
	RSI14      float64
	MACDLine   float64
	MACDSignal float64
}

// MonitoredEntry holds a buy position being actively monitored.
type MonitoredEntry struct {
	StockCode    string
	StockName    string
	FilledPrice  float64
	CurrentPrice float64
	TargetPrice  float64
	StopPrice   float64
	OrderID     int64
	Qty         int    // 현재 보유 수량 (부분 매도 시 감소)
	Market      string        // "KR" (empty defaults to "KR")
	AssetType   string        // "STOCK" | "ETF" | "ETF_DOMESTIC" (MST 기반)
	SoldCh      chan<- string // optional: engine receives sold signal (may be nil)
	// 트레일링 스탑
	TrailingTriggerPct float64 // 활성화 기준 수익률 (%). 0=비활성
	TrailingStopPct    float64 // 최고가 대비 하락 허용폭 (%)
	TrailingActivated  bool    // 트레일링 스탑 활성화 여부
	PeakPrice          float64 // 보유 중 도달한 최고가
	// 단계적 횡보 청산
	PartialExitDone bool // 절반 청산 완료 여부 (true이면 다음 횡보 감지 시 전량 청산)
	// 부분 익절 (TP partial) — stagnation과 독립적인 별도 플래그
	PartialTPDone bool // 중간 목표가 도달 후 부분 매도 완료 여부
	// 상한가 매도
	UpperLimitPrice  float64 // 당일 상한가. 0=비활성
	SellOnUpperLimit bool    // 상한가 도달 시 자동 매도
}

// Monitor watches registered positions for target/stop price hits and
// handles end-of-day liquidation.
type Monitor struct {
	mu        sync.RWMutex
	positions map[string]*MonitoredEntry // stockCode → entry

	kisClient *kis.Client
	wsClient  *kis.WebSocketClient
	db        *database.DB
	mstStore  *stockmaster.Store

	// 횡보 감지
	stagnMu                       sync.Mutex
	stagnantSince                 map[string]*time.Time // stockCode → 횡보 시작 시각
	stagnationThresholdPct        float64               // 횡보 판단 기준 변동폭 (%, 0=비활성)
	stagnationDurationMin         int                   // 횡보 지속 기준 시간 (분, 0=비활성)
	stagnationPartialExitEnabled  bool                  // 단계적 횡보 청산 활성화
	stagnationBidAskSellThreshold float64               // 이 값 미만이면 즉시 전량 청산 (기본 1.0)
	// 부분 익절 설정
	partialTPEnabled   bool    // 중간 목표가 도달 시 부분 매도 활성화
	partialTPPct       float64 // 중간 익절 트리거 수익률 %
	partialTPRatio     float64 // 매도 비율 (0~1)
	partialTPRaiseStop bool    // 부분 익절 후 손절가를 매입가(BEP)로 올리기
}

// New creates a Monitor.
func New(db *database.DB, kisClient *kis.Client, wsClient *kis.WebSocketClient, mstStore *stockmaster.Store) *Monitor {
	return &Monitor{
		positions:     make(map[string]*MonitoredEntry),
		stagnantSince: make(map[string]*time.Time),
		kisClient:     kisClient,
		wsClient:      wsClient,
		db:            db,
		mstStore:      mstStore,
	}
}

// SetStagnationConfig updates the stagnation detection parameters.
// Call this before starting the indicator checker.
func (m *Monitor) SetStagnationConfig(thresholdPct float64, durationMin int) {
	m.stagnMu.Lock()
	m.stagnationThresholdPct = thresholdPct
	m.stagnationDurationMin = durationMin
	m.stagnMu.Unlock()
}

// SetStagnationExitConfig sets the staged exit behaviour for stagnation.
// partialExitEnabled=true: first stagnation → sell 50%, second → sell all.
// bidAskSellThreshold: if bid_ask_ratio is below this during stagnation, sell all immediately (default 1.0).
// Call this before starting the indicator checker.
func (m *Monitor) SetStagnationExitConfig(partialExitEnabled bool, bidAskSellThreshold float64) {
	if bidAskSellThreshold <= 0 {
		bidAskSellThreshold = 1.0
	}
	m.stagnMu.Lock()
	m.stagnationPartialExitEnabled = partialExitEnabled
	m.stagnationBidAskSellThreshold = bidAskSellThreshold
	m.stagnMu.Unlock()
}

// SetPartialTPConfig configures the partial take-profit behaviour.
// enabled: feature on/off; pct: intermediate trigger %; ratio: fraction to sell (e.g. 0.5=50%);
// raiseStop: raise StopPrice to FilledPrice (breakeven) after partial TP fires.
func (m *Monitor) SetPartialTPConfig(enabled bool, pct, ratio float64, raiseStop bool) {
	if pct <= 0 {
		pct = 1.0
	}
	if ratio <= 0 || ratio >= 1 {
		ratio = 0.5
	}
	m.mu.Lock()
	m.partialTPEnabled = enabled
	m.partialTPPct = pct
	m.partialTPRatio = ratio
	m.partialTPRaiseStop = raiseStop
	m.mu.Unlock()
}

// Register adds (or updates) a position to be monitored and persists it to DB.
// If wsClient is connected, it subscribes to real-time price updates.
func (m *Monitor) Register(ctx context.Context, pos MonitoredEntry) error {
	m.mu.Lock()
	m.positions[pos.StockCode] = &pos
	m.mu.Unlock()

	// Persist for server-restart recovery.
	dbPos := &models.MonitoredPosition{
		StockCode:    pos.StockCode,
		StockName:    pos.StockName,
		FilledPrice:  pos.FilledPrice,
		TargetPrice:  pos.TargetPrice,
		StopPrice:    pos.StopPrice,
		OrderID:      pos.OrderID,
		RemainingQty: pos.Qty,
	}
	if err := m.db.CreatePosition(ctx, dbPos); err != nil {
		return fmt.Errorf("persist monitored_position: %w", err)
	}

	// Subscribe to real-time price stream.
	if m.wsClient != nil {
		if err := m.wsClient.SubscribePrice(pos.StockCode); err != nil {
			logger.Error("ws subscribe price failed",
				map[string]any{"stock_code": pos.StockCode, "error": err.Error()})
		}
	}

	logger.Info("monitor: position registered",
		map[string]any{
			"stock_code":   pos.StockCode,
			"target_price": pos.TargetPrice,
			"stop_price":   pos.StopPrice,
		})
	return nil
}

// Remove removes a position from monitoring and deletes it from DB.
func (m *Monitor) Remove(ctx context.Context, stockCode string) {
	m.mu.Lock()
	delete(m.positions, stockCode)
	m.mu.Unlock()

	m.stagnMu.Lock()
	delete(m.stagnantSince, stockCode)
	m.stagnMu.Unlock()

	_ = m.db.DeletePosition(ctx, stockCode)

	if m.wsClient != nil {
		m.wsClient.UnsubscribePrice(stockCode) //nolint:errcheck
	}

	logger.Info("monitor: position removed", map[string]any{"stock_code": stockCode})
}

// HandlePrice evaluates a price update against registered positions.
// Called by the WebSocket price event consumer goroutine.
// isTest=true: KIS 매도 주문을 건너뛰고 MQTT만 발행 (장 외 테스트용).
func (m *Monitor) HandlePrice(stockCode string, price float64, isTest bool) {
	m.mu.RLock()
	pos, ok := m.positions[stockCode]
	m.mu.RUnlock()
	if !ok {
		return
	}

	// 현재가 갱신
	m.mu.Lock()
	if p, ok2 := m.positions[stockCode]; ok2 {
		p.CurrentPrice = price
	}
	m.mu.Unlock()

	// 트레일링 스탑: 활성화 여부 갱신 및 트리거 체크
	if pos.TrailingTriggerPct > 0 && pos.FilledPrice > 0 {
		triggerThreshold := pos.FilledPrice * (1 + pos.TrailingTriggerPct/100)
		if !pos.TrailingActivated && price >= triggerThreshold {
			// 트레일링 스탑 활성화
			m.mu.Lock()
			if p, ok2 := m.positions[stockCode]; ok2 {
				p.TrailingActivated = true
				p.PeakPrice = price
			}
			m.mu.Unlock()
			logger.Info("monitor: trailing stop activated",
				map[string]any{"stock_code": stockCode, "price": price, "trigger": triggerThreshold})
		} else if pos.TrailingActivated {
			// 최고가 갱신
			if price > pos.PeakPrice {
				m.mu.Lock()
				if p, ok2 := m.positions[stockCode]; ok2 {
					p.PeakPrice = price
				}
				m.mu.Unlock()
				// 최신 pos 재조회
				m.mu.RLock()
				pos, ok = m.positions[stockCode]
				m.mu.RUnlock()
				if !ok {
					return
				}
			}
			// 트레일링 스탑 트리거 체크
			if pos.TrailingStopPct > 0 && pos.PeakPrice > 0 {
				stopThreshold := pos.PeakPrice * (1 - pos.TrailingStopPct/100)
				if price < stopThreshold {
					logger.Info("monitor: TRAILING STOP hit",
						map[string]any{"stock_code": stockCode, "price": price, "peak": pos.PeakPrice, "stop_threshold": stopThreshold})
					if !isTest {
						if soldQty := m.executeSell(stockCode, pos, "트레일링 스탑"); soldQty < 0 {
							return // 주문 실패 — 포지션 유지, 다음 틱 재시도
						}
					}
					m.Remove(context.Background(), stockCode)
					return
				}
			}
		}
	}

	// 부분 익절: full TP 체크 전에 중간 목표가 도달 여부 확인
	m.mu.RLock()
	partialTPEnabled := m.partialTPEnabled
	partialTPPct := m.partialTPPct
	partialTPRatio := m.partialTPRatio
	partialTPRaiseStop := m.partialTPRaiseStop
	m.mu.RUnlock()
	if partialTPEnabled && !pos.PartialTPDone && pos.FilledPrice > 0 {
		triggerPrice := pos.FilledPrice * (1 + partialTPPct/100)
		if price >= triggerPrice && price < pos.TargetPrice {
			logger.Info("monitor: PARTIAL TP hit",
				map[string]any{"stock_code": stockCode, "price": price, "trigger": triggerPrice})
			if !isTest {
				m.executePartialSellWithRatio(stockCode, pos, "부분 익절", partialTPRatio)
			}
			m.mu.Lock()
			if p, ok2 := m.positions[stockCode]; ok2 {
				p.PartialTPDone = true
				if partialTPRaiseStop {
					p.StopPrice = p.FilledPrice // BEP 손절가
				}
			}
			m.mu.Unlock()
			return // 다음 틱에서 full TP 체크
		}
	}

	// 상한가 도달 시 매도
	if pos.SellOnUpperLimit && pos.UpperLimitPrice > 0 && price >= pos.UpperLimitPrice {
		logger.Info("monitor: UPPER LIMIT hit",
			map[string]any{"stock_code": stockCode, "price": price, "upper_limit": pos.UpperLimitPrice})
		if !isTest {
			if soldQty := m.executeSell(stockCode, pos, "상한가 도달"); soldQty >= 0 {
				m.Remove(context.Background(), stockCode)
			}
		} else {
			m.Remove(context.Background(), stockCode)
		}
		return
	}

	switch {
	case price >= pos.TargetPrice:
		logger.Info("monitor: TARGET hit",
			map[string]any{"stock_code": stockCode, "price": price, "target": pos.TargetPrice})
		if !isTest {
			if soldQty := m.executeSell(stockCode, pos, "목표가 도달"); soldQty >= 0 {
				m.Remove(context.Background(), stockCode)
			}
			// soldQty == -1: 주문 실패, 잔고 보존 → 다음 틱에서 재시도
		} else {
			m.Remove(context.Background(), stockCode)
		}

	case price <= pos.StopPrice:
		logger.Info("monitor: STOP hit",
			map[string]any{"stock_code": stockCode, "price": price, "stop": pos.StopPrice})
		if !isTest {
			if soldQty := m.executeSell(stockCode, pos, "손절가 도달"); soldQty >= 0 {
				m.Remove(context.Background(), stockCode)
			}
			// soldQty == -1: 주문 실패, 잔고 보존 → 다음 틱에서 재시도
		} else {
			m.Remove(context.Background(), stockCode)
		}

	default:
		// 목표/손절 미도달 — 횡보 여부 추적
		m.stagnMu.Lock()
		threshold := m.stagnationThresholdPct
		if threshold > 0 && pos.FilledPrice > 0 {
			changePct := math.Abs(price-pos.FilledPrice) / pos.FilledPrice * 100
			if changePct < threshold {
				if _, exists := m.stagnantSince[stockCode]; !exists {
					now := time.Now()
					m.stagnantSince[stockCode] = &now
				}
			} else {
				delete(m.stagnantSince, stockCode)
			}
		}
		m.stagnMu.Unlock()
	}
}

// executeSell places a market sell order for the given position.
// Returns qty > 0 on success, 0 when no holdings (safe to Remove), -1 when sell order failed (do NOT Remove).
func (m *Monitor) executeSell(stockCode string, pos *MonitoredEntry, reason string) int {
	ctx := context.Background()

	holdings, err := m.kisClient.GetHoldings(ctx)
	if err != nil {
		logger.Error("auto-sell: GetHoldings failed",
			map[string]any{"stock_code": stockCode, "error": err.Error()})
		m.db.InsertServiceLog(ctx, "MONITOR", "ERROR", "자동 매도 실패: GetHoldings 오류", fmt.Sprintf("stock_code=%s error=%s", stockCode, err.Error()))
		return -1 // 잔고 확인 불가 — 안전을 위해 Remove 차단
	}

	qty := 0
	for _, h := range holdings {
		if h.StockCode == stockCode {
			fmt.Sscanf(h.HoldingQty, "%d", &qty)
			break
		}
	}
	if qty <= 0 {
		logger.Info("auto-sell: no holdings found", map[string]any{"stock_code": stockCode})
		return 0 // 잔고 없음 — Remove 허용
	}

	resp, err := m.kisClient.PlaceSellOrder(ctx, kis.OrderRequest{
		StockCode: stockCode,
		OrderDivn: "01", // 시장가
		Qty:       fmt.Sprintf("%d", qty),
		Price:     "0",
	})
	if err != nil {
		logger.Error("auto-sell: PlaceSellOrder failed",
			map[string]any{"stock_code": stockCode, "qty": qty, "error": err.Error()})
		m.db.InsertServiceLog(ctx, "MONITOR", "ERROR", "자동 매도 실패: 주문 오류 — 포지션 모니터링 유지", fmt.Sprintf("stock_code=%s qty=%d error=%s", stockCode, qty, err.Error()))
		return -1 // 주문 실패 — 실제 잔고 있음, Remove 차단
	}

	logger.Info("auto-sell: sell order placed",
		map[string]any{"stock_code": stockCode, "qty": qty, "filled_price": pos.FilledPrice, "reason": reason})

	kisOrderID := ""
	if resp != nil {
		kisOrderID = resp.KISOrderID
	}
	sellOrder := &models.Order{
		StockCode:  stockCode,
		StockName:  pos.StockName,
		OrderType:  models.OrderTypeSell,
		Qty:        qty,
		Price:      pos.FilledPrice,
		Status:     models.OrderStatusPending,
		KISOrderID: kisOrderID,
		SellReason: reason,
		Source:     models.OrderSourceAgent,
		CreatedAt:  time.Now().UTC(),
	}
	sellOrderID, _ := m.db.CreateOrder(ctx, sellOrder)

	// Update trade report with sell data (non-blocking).
	go func(buyOrderID, soID int64, sellQty int, sellReason string) {
		rCtx := context.Background()
		var sellPrice float64
		var sellIndJSON string
		info, infoErr := ops.GetStockInfo(rCtx, m.kisClient, stockCode)
		if infoErr == nil {
			if p, e := strconv.ParseFloat(info.CurrentPrice, 64); e == nil {
				sellPrice = p
			}
			b, _ := json.Marshal(info)
			sellIndJSON = string(b)
		}
		if err := m.db.UpdateTradeReportOnSell(rCtx, buyOrderID, soID, sellPrice, sellQty, sellReason, sellIndJSON); err != nil {
			logger.Error("monitor: UpdateTradeReportOnSell failed",
				map[string]any{"stock_code": stockCode, "error": err.Error()})
		}
	}(pos.OrderID, sellOrderID, qty, reason)

	// Notify engine that this position was sold.
	if pos.SoldCh != nil {
		select {
		case pos.SoldCh <- stockCode:
		default:
		}
	}

	return qty
}

// executePartialSell sells approximately half of the current holdings for the given position.
// It does NOT remove the position from monitoring. Returns the qty sold (0 on failure).
func (m *Monitor) executePartialSell(stockCode string, pos *MonitoredEntry, reason string) int {
	ctx := context.Background()

	holdings, err := m.kisClient.GetHoldings(ctx)
	if err != nil {
		logger.Error("partial-sell: GetHoldings failed",
			map[string]any{"stock_code": stockCode, "error": err.Error()})
		return 0
	}

	totalQty := 0
	for _, h := range holdings {
		if h.StockCode == stockCode {
			fmt.Sscanf(h.HoldingQty, "%d", &totalQty)
			break
		}
	}
	if totalQty <= 0 {
		logger.Info("partial-sell: no holdings found", map[string]any{"stock_code": stockCode})
		return 0
	}

	sellQty := totalQty / 2
	if sellQty <= 0 {
		sellQty = 1 // 최소 1주
	}

	resp, err := m.kisClient.PlaceSellOrder(ctx, kis.OrderRequest{
		StockCode: stockCode,
		OrderDivn: "01", // 시장가
		Qty:       fmt.Sprintf("%d", sellQty),
		Price:     "0",
	})
	if err != nil {
		logger.Error("partial-sell: PlaceSellOrder failed",
			map[string]any{"stock_code": stockCode, "qty": sellQty, "error": err.Error()})
		return 0
	}

	logger.Info("partial-sell: sell order placed",
		map[string]any{"stock_code": stockCode, "sell_qty": sellQty, "total_qty": totalQty, "reason": reason})

	kisOrderID := ""
	if resp != nil {
		kisOrderID = resp.KISOrderID
	}
	partialOrder := &models.Order{
		StockCode:  stockCode,
		StockName:  pos.StockName,
		OrderType:  models.OrderTypeSell,
		Qty:        sellQty,
		Price:      pos.FilledPrice,
		Status:     models.OrderStatusPending,
		KISOrderID: kisOrderID,
		SellReason: reason,
		Source:     models.OrderSourceAgent,
		CreatedAt:  time.Now().UTC(),
	}
	partialSellOrderID, _ := m.db.CreateOrder(ctx, partialOrder)

	go func(buyOrderID, soID int64, qty int, sellReason string) {
		rCtx := context.Background()
		var sellPrice float64
		var sellIndJSON string
		info, infoErr := ops.GetStockInfo(rCtx, m.kisClient, stockCode)
		if infoErr == nil {
			if p, e := strconv.ParseFloat(info.CurrentPrice, 64); e == nil {
				sellPrice = p
			}
			b, _ := json.Marshal(info)
			sellIndJSON = string(b)
		}
		if err := m.db.UpdateTradeReportOnSell(rCtx, buyOrderID, soID, sellPrice, qty, sellReason, sellIndJSON); err != nil {
			logger.Error("monitor: UpdateTradeReportOnSell (partial) failed",
				map[string]any{"stock_code": stockCode, "error": err.Error()})
		}
	}(pos.OrderID, partialSellOrderID, sellQty, reason)

	// 부분 매도 성공 후 잔여 수량 업데이트
	m.mu.Lock()
	if p, ok := m.positions[stockCode]; ok {
		p.Qty -= sellQty
		if p.Qty < 0 {
			p.Qty = 0
		}
		remaining := p.Qty
		m.mu.Unlock()
		if err := m.db.UpdatePositionQty(ctx, stockCode, remaining); err != nil {
			logger.Error("monitor: UpdatePositionQty failed", map[string]any{"stock_code": stockCode, "error": err.Error()})
		}
	} else {
		m.mu.Unlock()
	}

	return sellQty
}

// executePartialSellWithRatio은 ratio 비율만큼 부분 매도한다.
// stagnation 경로(executePartialSell, 고정 50%)와 독립적으로 사용된다.
func (m *Monitor) executePartialSellWithRatio(stockCode string, pos *MonitoredEntry, reason string, ratio float64) int {
	if ratio <= 0 || ratio >= 1 {
		ratio = 0.5
	}
	ctx := context.Background()

	holdings, err := m.kisClient.GetHoldings(ctx)
	if err != nil {
		logger.Error("partial-tp-sell: GetHoldings failed",
			map[string]any{"stock_code": stockCode, "error": err.Error()})
		return 0
	}

	totalQty := 0
	for _, h := range holdings {
		if h.StockCode == stockCode {
			fmt.Sscanf(h.HoldingQty, "%d", &totalQty)
			break
		}
	}
	if totalQty <= 0 {
		logger.Info("partial-tp-sell: no holdings found", map[string]any{"stock_code": stockCode})
		return 0
	}

	sellQty := int(math.Round(float64(totalQty) * ratio))
	if sellQty <= 0 {
		sellQty = 1
	}
	if sellQty >= totalQty {
		sellQty = totalQty - 1 // 최소 1주 잔여 보장
	}

	resp, err := m.kisClient.PlaceSellOrder(ctx, kis.OrderRequest{
		StockCode: stockCode,
		OrderDivn: "01", // 시장가
		Qty:       fmt.Sprintf("%d", sellQty),
		Price:     "0",
	})
	if err != nil {
		logger.Error("partial-tp-sell: PlaceSellOrder failed",
			map[string]any{"stock_code": stockCode, "qty": sellQty, "error": err.Error()})
		return 0
	}

	logger.Info("partial-tp-sell: sell order placed",
		map[string]any{"stock_code": stockCode, "sell_qty": sellQty, "total_qty": totalQty, "ratio": ratio, "reason": reason})

	kisOrderID := ""
	if resp != nil {
		kisOrderID = resp.KISOrderID
	}
	tpOrder := &models.Order{
		StockCode:  stockCode,
		StockName:  pos.StockName,
		OrderType:  models.OrderTypeSell,
		Qty:        sellQty,
		Price:      pos.FilledPrice,
		Status:     models.OrderStatusPending,
		KISOrderID: kisOrderID,
		SellReason: reason,
		Source:     models.OrderSourceAgent,
		CreatedAt:  time.Now().UTC(),
	}
	tpSellOrderID, _ := m.db.CreateOrder(ctx, tpOrder)

	go func(buyOrderID, soID int64, qty int, sellReason string) {
		rCtx := context.Background()
		var sellPrice float64
		var sellIndJSON string
		info, infoErr := ops.GetStockInfo(rCtx, m.kisClient, stockCode)
		if infoErr == nil {
			if p, e := strconv.ParseFloat(info.CurrentPrice, 64); e == nil {
				sellPrice = p
			}
			b, _ := json.Marshal(info)
			sellIndJSON = string(b)
		}
		if err := m.db.UpdateTradeReportOnSell(rCtx, buyOrderID, soID, sellPrice, qty, sellReason, sellIndJSON); err != nil {
			logger.Error("monitor: UpdateTradeReportOnSell (partial-tp) failed",
				map[string]any{"stock_code": stockCode, "error": err.Error()})
		}
	}(pos.OrderID, tpSellOrderID, sellQty, reason)

	// 부분 매도 성공 후 잔여 수량 업데이트
	m.mu.Lock()
	if p, ok := m.positions[stockCode]; ok {
		p.Qty -= sellQty
		if p.Qty < 0 {
			p.Qty = 0
		}
		remaining := p.Qty
		m.mu.Unlock()
		if err := m.db.UpdatePositionQty(ctx, stockCode, remaining); err != nil {
			logger.Error("monitor: UpdatePositionQty failed", map[string]any{"stock_code": stockCode, "error": err.Error()})
		}
	} else {
		m.mu.Unlock()
	}

	return sellQty
}

// ForceSell places a market sell order for a specific monitored position and removes it from monitoring.
// Called by the user via the manual "강제매도" button. Returns the sold qty, or error on failure.
func (m *Monitor) ForceSell(ctx context.Context, stockCode string) (int, error) {
	m.mu.RLock()
	pos, ok := m.positions[stockCode]
	m.mu.RUnlock()
	if !ok {
		return 0, fmt.Errorf("모니터링 중인 포지션을 찾을 수 없습니다: %s", stockCode)
	}

	qty := m.executeSell(stockCode, pos, "강제매도")
	m.Remove(ctx, stockCode)
	if qty <= 0 {
		return 0, fmt.Errorf("매도 주문 실패 (보유수량 없음 또는 API 오류)")
	}
	return qty, nil
}

// LiquidateAll places market sell orders for all monitored positions (장마감).
// market: optional filter — "KR" or "US". Empty means all positions.
func (m *Monitor) LiquidateAll(ctx context.Context, market ...string) {
	filterMarket := ""
	if len(market) > 0 {
		filterMarket = market[0]
	}

	m.mu.RLock()
	codes := make([]string, 0, len(m.positions))
	for code, pos := range m.positions {
		if filterMarket == "" || pos.Market == filterMarket || (filterMarket == "KR" && pos.Market == "") {
			codes = append(codes, code)
		}
	}
	m.mu.RUnlock()

	if len(codes) == 0 {
		return
	}

	logger.Info("monitor: liquidating all positions", map[string]any{"count": len(codes), "filter": filterMarket})

	for _, code := range codes {
		m.mu.RLock()
		pos, ok := m.positions[code]
		m.mu.RUnlock()
		if !ok {
			continue
		}

		holdings, err := m.kisClient.GetHoldings(ctx)
		if err != nil {
			logger.Error("liquidate: GetHoldings failed",
				map[string]any{"stock_code": code, "error": err.Error()})
			continue
		}

		qty := 0
		var currentPrice float64
		for _, h := range holdings {
			if h.StockCode == code {
				fmt.Sscanf(h.HoldingQty, "%d", &qty)
				fmt.Sscanf(h.CurrentPrice, "%f", &currentPrice)
				break
			}
		}
		if qty <= 0 {
			m.Remove(ctx, code)
			continue
		}

		liqResp, err := m.kisClient.PlaceSellOrder(ctx, kis.OrderRequest{
			StockCode: code,
			OrderDivn: "01", // 시장가
			Qty:       fmt.Sprintf("%d", qty),
			Price:     "0",
		})
		if err != nil {
			// 매도 주문 실패 시 Remove 하지 않음 — 실제 잔고가 남아있으므로 모니터링 유지
			logger.Error("liquidate: sell order failed — position retained in monitor",
				map[string]any{"stock_code": code, "qty": qty, "error": err.Error()})
			m.db.InsertServiceLog(ctx, "MONITOR", "ERROR", "장마감 청산 실패: 잔고 남아있음", fmt.Sprintf("stock_code=%s qty=%d error=%s", code, qty, err.Error()))
			continue
		}

		logger.Info("liquidate: sell order placed",
			map[string]any{"stock_code": code, "qty": qty})
		kisOrderID := ""
		if liqResp != nil {
			kisOrderID = liqResp.KISOrderID
		}
		liqOrder := &models.Order{
			StockCode:  code,
			StockName:  pos.StockName,
			OrderType:  models.OrderTypeSell,
			Qty:        qty,
			Price:      pos.FilledPrice,
			Status:     models.OrderStatusPending,
			KISOrderID: kisOrderID,
			SellReason: "일일 자동 청산",
			Source:     models.OrderSourceAgent,
			CreatedAt:  time.Now().UTC(),
		}
		liqSellOrderID, _ := m.db.CreateOrder(ctx, liqOrder)

		// Update trade report (non-blocking).
		go func(codeSnap string, buyOrderID, soID int64, sellQty int, sellPriceEst float64) {
			rCtx := context.Background()
			var sellIndJSON string
			info, infoErr := ops.GetStockInfo(rCtx, m.kisClient, codeSnap)
			sellPrice := sellPriceEst
			if infoErr == nil {
				if p, e := strconv.ParseFloat(info.CurrentPrice, 64); e == nil && p > 0 {
					sellPrice = p
				}
				b, _ := json.Marshal(info)
				sellIndJSON = string(b)
			}
			if err := m.db.UpdateTradeReportOnSell(rCtx, buyOrderID, soID, sellPrice, sellQty, "일일 자동 청산", sellIndJSON); err != nil {
				logger.Error("monitor: UpdateTradeReportOnSell (liquidate) failed",
					map[string]any{"stock_code": codeSnap, "error": err.Error()})
			}
		}(code, pos.OrderID, liqSellOrderID, qty, currentPrice)

		m.Remove(ctx, code)
	}
}

// List returns a snapshot of all currently monitored positions.
func (m *Monitor) List() []models.MonitoredPosition {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]models.MonitoredPosition, 0, len(m.positions))
	for _, pos := range m.positions {
		result = append(result, models.MonitoredPosition{
			StockCode:    pos.StockCode,
			StockName:    pos.StockName,
			FilledPrice:  pos.FilledPrice,
			CurrentPrice: pos.CurrentPrice,
			TargetPrice:  pos.TargetPrice,
			StopPrice:    pos.StopPrice,
			OrderID:      pos.OrderID,
			RemainingQty: pos.Qty,
			CreatedAt:    time.Now(),
		})
	}
	return result
}

// Count returns the number of monitored positions.
func (m *Monitor) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.positions)
}

// Has reports whether the stock code is currently being monitored.
func (m *Monitor) Has(stockCode string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.positions[stockCode]
	return ok
}

// RecoverFromHoldings compares KIS actual holdings with the current monitored positions map
// and registers any holding that is not yet being monitored.
// 서버 재시작 시 호출 — DB에 등록 안 된 포지션(버그·장애로 누락된 경우)도 자동 복구.
//
// 체결가: KIS 잔고의 매입평균가(pchs_avg_pric) 사용.
// 목표가/손절가: DB trading settings의 take_profit_pct / stop_loss_pct 적용.
// OrderID: DB orders 테이블에서 해당 종목의 오늘 마지막 매수 체결 주문 조회 (없으면 0).
func (m *Monitor) RecoverFromHoldings(ctx context.Context, soldCh chan<- string) {
	holdings, err := m.kisClient.GetHoldings(ctx)
	if err != nil {
		logger.Error("RecoverFromHoldings: GetHoldings failed", map[string]any{"error": err.Error()})
		return
	}
	if len(holdings) == 0 {
		return
	}

	settings, err := m.db.GetTradingSettings(ctx)
	if err != nil {
		logger.Error("RecoverFromHoldings: GetTradingSettings failed", map[string]any{"error": err.Error()})
		return
	}

	registered := 0
	for _, h := range holdings {
		var qty int
		fmt.Sscanf(h.HoldingQty, "%d", &qty)
		if qty <= 0 {
			continue
		}

		// 이미 모니터링 중이면 건너뜀.
		m.mu.RLock()
		_, alreadyMonitored := m.positions[h.StockCode]
		m.mu.RUnlock()
		if alreadyMonitored {
			continue
		}

		var filledPrice float64
		fmt.Sscanf(h.AvgPrice, "%f", &filledPrice)
		if filledPrice <= 0 {
			logger.Warn("RecoverFromHoldings: avg price is 0, skipping",
				map[string]any{"stock_code": h.StockCode})
			continue
		}

		// DB에서 해당 종목의 마지막 AGENT BUY 주문 ID 조회.
		var orderID int64
		if lastOrder, err := m.db.GetLastAgentBuyOrder(ctx, h.StockCode); err == nil && lastOrder != nil {
			orderID = lastOrder.ID
		}

		// MST 기반 AssetType 결정 — ETF/주식별 익절/손절 분리 적용.
		assetType := "STOCK"
		if m.mstStore != nil {
			if sm, err := m.mstStore.GetByCode(ctx, h.StockCode); err == nil && sm != nil {
				if sm.IsDomesticEquityETF {
					assetType = "ETF_DOMESTIC"
				} else if sm.IsETF {
					assetType = "ETF"
				}
			}
		}

		takePct := settings.TakeProfitPct
		stopPct := settings.StopLossPct
		if assetType == "ETF_DOMESTIC" || assetType == "ETF" {
			if settings.ETFTakeProfitPct > 0 {
				takePct = settings.ETFTakeProfitPct
			}
			if settings.ETFStopLossPct > 0 {
				stopPct = settings.ETFStopLossPct
			}
		} else {
			if settings.StockTakeProfitPct > 0 {
				takePct = settings.StockTakeProfitPct
			}
			if settings.StockStopLossPct > 0 {
				stopPct = settings.StockStopLossPct
			}
		}

		entry := MonitoredEntry{
			StockCode:   h.StockCode,
			StockName:   h.StockName,
			FilledPrice: filledPrice,
			TargetPrice: filledPrice * (1 + takePct/100),
			StopPrice:   filledPrice * (1 - stopPct/100),
			OrderID:     orderID,
			Qty:         qty,
			AssetType:   assetType,
			SoldCh:      soldCh,
		}

		if err := m.Register(ctx, entry); err != nil {
			logger.Error("RecoverFromHoldings: Register failed",
				map[string]any{"stock_code": h.StockCode, "error": err.Error()})
			continue
		}
		registered++
		logger.Info("RecoverFromHoldings: position recovered from KIS holdings",
			map[string]any{
				"stock_code":   h.StockCode,
				"stock_name":   h.StockName,
				"filled_price": filledPrice,
				"target_price": entry.TargetPrice,
				"stop_price":   entry.StopPrice,
				"asset_type":   assetType,
			})
	}

	if registered > 0 {
		logger.Info("RecoverFromHoldings: recovery complete",
			map[string]any{"recovered": registered})
	}
}

// PurgeStalePositions compares monitored positions against actual KIS holdings
// and removes any position that is no longer held (qty == 0).
// WebSocket 재연결 후 또는 주기적으로 호출하여 매도 완료 종목을 정리합니다.
func (m *Monitor) PurgeStalePositions(ctx context.Context) {
	holdings, err := m.kisClient.GetHoldings(ctx)
	if err != nil {
		logger.Error("PurgeStalePositions: GetHoldings failed", map[string]any{"error": err.Error()})
		return
	}

	held := make(map[string]bool, len(holdings))
	for _, h := range holdings {
		var qty int
		fmt.Sscanf(h.HoldingQty, "%d", &qty)
		if qty > 0 {
			held[h.StockCode] = true
		}
	}

	m.mu.RLock()
	monitored := make([]string, 0, len(m.positions))
	for code := range m.positions {
		monitored = append(monitored, code)
	}
	m.mu.RUnlock()

	for _, code := range monitored {
		if !held[code] {
			logger.Info("PurgeStalePositions: removing position no longer held",
				map[string]any{"stock_code": code})
			m.Remove(ctx, code)
		}
	}
}

// ResubscribeAll sends WebSocket price subscriptions for every currently monitored position.
// WS 연결 직후 호출 — LoadFromDB/RecoverFromHoldings 시점엔 WS 미연결이라
// SubscribePrice 가 실패하므로, 연결 완료 후 이 함수로 일괄 재구독한다.
func (m *Monitor) ResubscribeAll() {
	if m.wsClient == nil {
		return
	}
	m.mu.RLock()
	codes := make([]string, 0, len(m.positions))
	for code := range m.positions {
		codes = append(codes, code)
	}
	m.mu.RUnlock()

	for _, code := range codes {
		if err := m.wsClient.SubscribePrice(code); err != nil {
			logger.Error("ResubscribeAll: SubscribePrice failed",
				map[string]any{"stock_code": code, "error": err.Error()})
		} else {
			logger.Info("ResubscribeAll: subscribed", map[string]any{"stock_code": code})
		}
	}
}

// LoadFromDB restores monitored positions from the database after a server restart.
func (m *Monitor) LoadFromDB(ctx context.Context, soldCh chan<- string) error {
	dbPositions, err := m.db.ListPositions(ctx)
	if err != nil {
		return fmt.Errorf("load monitored_positions: %w", err)
	}

	for _, p := range dbPositions {
		pos := MonitoredEntry{
			StockCode:   p.StockCode,
			StockName:   p.StockName,
			FilledPrice: p.FilledPrice,
			TargetPrice: p.TargetPrice,
			StopPrice:   p.StopPrice,
			OrderID:     p.OrderID,
			Qty:         p.RemainingQty,
			Market:      "KR",
			SoldCh:      soldCh,
		}
		m.mu.Lock()
		m.positions[pos.StockCode] = &pos
		m.mu.Unlock()

		if m.wsClient != nil {
			m.wsClient.SubscribePrice(pos.StockCode) //nolint:errcheck
		}
	}
	count := m.Count()
	if count > 0 {
		logger.Info("monitor: restored positions from DB", map[string]any{"count": count})
	}
	return nil
}

// StartIndicatorChecker periodically checks technical indicators for each monitored position
// and executes a sell if a configured condition is met.
// getInfoFn is a callback (injected to avoid circular imports) that returns the current indicators.
// conditions controls which checks are active and their priority order.
// Supported values: "rsi_overbought", "macd_bearish" (target_pct/stop_pct are handled by HandlePrice).
func (m *Monitor) StartIndicatorChecker(
	ctx context.Context,
	intervalMin int,
	conditions []string,
	rsiThreshold float64,
	macdBearish bool,
	getInfoFn func(ctx context.Context, code string) (*IndicatorSnapshot, error),
) {
	if intervalMin <= 0 {
		intervalMin = 5
	}
	ticker := time.NewTicker(time.Duration(intervalMin) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.checkIndicators(ctx, conditions, rsiThreshold, macdBearish, getInfoFn)
		}
	}
}

func (m *Monitor) checkIndicators(
	ctx context.Context,
	conditions []string,
	rsiThreshold float64,
	macdBearish bool,
	getInfoFn func(ctx context.Context, code string) (*IndicatorSnapshot, error),
) {
	m.mu.RLock()
	codes := make([]string, 0, len(m.positions))
	for code := range m.positions {
		codes = append(codes, code)
	}
	m.mu.RUnlock()

	for _, code := range codes {
		snap, err := getInfoFn(ctx, code)
		if err != nil {
			logger.Error("indicator check: getInfoFn failed",
				map[string]any{"stock_code": code, "error": err.Error()})
			continue
		}

		m.mu.RLock()
		pos, ok := m.positions[code]
		m.mu.RUnlock()
		if !ok {
			continue
		}

		triggered := false
		triggerReason := ""

		for _, cond := range conditions {
			switch cond {
			case "rsi_overbought":
				if snap.RSI14 > 0 && rsiThreshold > 0 && snap.RSI14 >= rsiThreshold {
					triggered = true
					triggerReason = fmt.Sprintf("RSI %.2f >= threshold %.2f", snap.RSI14, rsiThreshold)
				}
			case "macd_bearish":
				if macdBearish && snap.MACDLine != 0 && snap.MACDLine < snap.MACDSignal {
					triggered = true
					triggerReason = fmt.Sprintf("MACD bearish crossover: line=%.4f signal=%.4f", snap.MACDLine, snap.MACDSignal)
				}
			case "stagnation":
				m.stagnMu.Lock()
				since, hasSince := m.stagnantSince[code]
				durationMin := m.stagnationDurationMin
				thresholdPct := m.stagnationThresholdPct
				partialExitEnabled := m.stagnationPartialExitEnabled
				bidAskThreshold := m.stagnationBidAskSellThreshold
				m.stagnMu.Unlock()
				if hasSince && since != nil && durationMin > 0 && thresholdPct > 0 {
					elapsed := time.Since(*since)
					if elapsed >= time.Duration(durationMin)*time.Minute {
						if !partialExitEnabled {
							// 기존 동작: 전량 청산
							triggered = true
							triggerReason = fmt.Sprintf("횡보 감지: %.1f%% 이내 변동 %.0f분 지속", thresholdPct, elapsed.Minutes())
						} else {
							// 단계적 청산: bid_ask_ratio 조회
							bidAsk, baErr := m.kisClient.GetBidAskRatio(ctx, code)
							if baErr != nil {
								bidAsk = 0
							}
							if bidAsk > 0 && bidAsk < bidAskThreshold {
								// 매도 우세 전환 → 즉시 전량 청산
								triggered = true
								triggerReason = fmt.Sprintf("횡보 중 매도우세 전환: bid_ask=%.2f < %.2f (%.0f분 횡보)", bidAsk, bidAskThreshold, elapsed.Minutes())
							} else if !pos.PartialExitDone {
								// 첫 번째 횡보 + 매수 우세 → 절반 청산
								reason := fmt.Sprintf("횡보 절반청산(1차): bid_ask=%.2f, %.0f분 횡보", bidAsk, elapsed.Minutes())
								soldQty := m.executePartialSell(code, pos, reason)
								if soldQty > 0 {
									// 포지션 플래그 설정 + 횡보 타이머 리셋
									m.mu.Lock()
									if p, ok2 := m.positions[code]; ok2 {
										p.PartialExitDone = true
									}
									m.mu.Unlock()
									m.stagnMu.Lock()
									delete(m.stagnantSince, code)
									m.stagnMu.Unlock()
									logger.Info("monitor: stagnation partial exit done, timer reset",
										map[string]any{"stock_code": code, "sold_qty": soldQty})
								}
								// triggered=false → 전량 청산하지 않고 계속 모니터링
							} else {
								// 두 번째 횡보 + 절반 청산 이미 완료 → 전량 청산
								triggered = true
								triggerReason = fmt.Sprintf("횡보 전량청산(2차): bid_ask=%.2f, %.0f분 횡보 (1차 절반청산 후 재횡보)", bidAsk, elapsed.Minutes())
							}
						}
					}
				}
			}
			if triggered {
				break
			}
		}

		if !triggered {
			continue
		}

		logger.Info("indicator check: sell condition triggered",
			map[string]any{"stock_code": code, "reason": triggerReason})
		if soldQty := m.executeSell(code, pos, triggerReason); soldQty >= 0 {
			m.Remove(ctx, code)
		}
		// soldQty == -1: 주문 실패 — 다음 인터벌에서 재시도
	}
}

// StartPriceConsumer reads from wsClient.PriceCh and calls HandlePrice.
// Runs until ctx is cancelled.
func (m *Monitor) StartPriceConsumer(ctx context.Context) {
	if m.wsClient == nil {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-m.wsClient.PriceCh:
			if !ok {
				return
			}
			m.HandlePrice(ev.StockCode, ev.Price, false)
		}
	}
}
