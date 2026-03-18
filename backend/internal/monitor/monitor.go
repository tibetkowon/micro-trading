package monitor

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/micro-trading-for-agent/backend/internal/database"
	"github.com/micro-trading-for-agent/backend/internal/kis"
	"github.com/micro-trading-for-agent/backend/internal/logger"
	"github.com/micro-trading-for-agent/backend/internal/models"
)

// IndicatorSnapshot holds key technical indicators for sell condition evaluation.
type IndicatorSnapshot struct {
	RSI14      float64
	MACDLine   float64
	MACDSignal float64
}

// MonitoredEntry holds a buy position being actively monitored.
type MonitoredEntry struct {
	StockCode   string
	StockName   string
	FilledPrice float64
	TargetPrice float64
	StopPrice   float64
	OrderID     int64
	Market      string        // "KR" or "US" (empty defaults to "KR")
	ExchCode    string        // 거래소코드 for US: NASD/NYSE/AMEX (empty for KR)
	SoldCh      chan<- string // optional: engine receives sold signal (may be nil)
	// 트레일링 스탑
	TrailingTriggerPct float64 // 활성화 기준 수익률 (%). 0=비활성
	TrailingStopPct    float64 // 최고가 대비 하락 허용폭 (%)
	TrailingActivated  bool    // 트레일링 스탑 활성화 여부
	PeakPrice          float64 // 보유 중 도달한 최고가
}

// Monitor watches registered positions for target/stop price hits and
// handles end-of-day liquidation.
type Monitor struct {
	mu        sync.RWMutex
	positions map[string]*MonitoredEntry // stockCode → entry

	kisClient *kis.Client
	wsClient  *kis.WebSocketClient
	db        *database.DB

	// 횡보 감지
	stagnMu                sync.Mutex
	stagnantSince          map[string]*time.Time // stockCode → 횡보 시작 시각
	stagnationThresholdPct float64               // 횡보 판단 기준 변동폭 (%, 0=비활성)
	stagnationDurationMin  int                   // 횡보 지속 기준 시간 (분, 0=비활성)
}

// New creates a Monitor.
func New(db *database.DB, kisClient *kis.Client, wsClient *kis.WebSocketClient) *Monitor {
	return &Monitor{
		positions:     make(map[string]*MonitoredEntry),
		stagnantSince: make(map[string]*time.Time),
		kisClient:     kisClient,
		wsClient:      wsClient,
		db:            db,
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

// Register adds (or updates) a position to be monitored and persists it to DB.
// If wsClient is connected, it subscribes to real-time price updates.
func (m *Monitor) Register(ctx context.Context, pos MonitoredEntry) error {
	m.mu.Lock()
	m.positions[pos.StockCode] = &pos
	m.mu.Unlock()

	// Persist for server-restart recovery.
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO monitored_positions
		  (stock_code, stock_name, filled_price, target_price, stop_price, order_id, market)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(stock_code) DO UPDATE SET
		   stock_name=excluded.stock_name,
		   filled_price=excluded.filled_price,
		   target_price=excluded.target_price,
		   stop_price=excluded.stop_price,
		   order_id=excluded.order_id,
		   market=excluded.market,
		   created_at=CURRENT_TIMESTAMP`,
		pos.StockCode, pos.StockName, pos.FilledPrice,
		pos.TargetPrice, pos.StopPrice, pos.OrderID, pos.Market,
	)
	if err != nil {
		return fmt.Errorf("persist monitored_position: %w", err)
	}

	// Subscribe to real-time price stream.
	if m.wsClient != nil {
		if pos.Market == "US" && pos.ExchCode != "" {
			excd := exchCodeToEXCD(pos.ExchCode)
			if err := m.wsClient.SubscribeOverseasPrice(excd, pos.StockCode); err != nil {
				logger.Error("ws subscribe overseas price failed",
					map[string]any{"stock_code": pos.StockCode, "error": err.Error()})
			}
		} else {
			if err := m.wsClient.SubscribePrice(pos.StockCode); err != nil {
				logger.Error("ws subscribe price failed",
					map[string]any{"stock_code": pos.StockCode, "error": err.Error()})
			}
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
	pos := m.positions[stockCode]
	delete(m.positions, stockCode)
	m.mu.Unlock()

	m.stagnMu.Lock()
	delete(m.stagnantSince, stockCode)
	m.stagnMu.Unlock()

	m.db.ExecContext(ctx, `DELETE FROM monitored_positions WHERE stock_code = ?`, stockCode)

	if m.wsClient != nil {
		if pos != nil && pos.Market == "US" && pos.ExchCode != "" {
			m.wsClient.UnsubscribeOverseasPrice(exchCodeToEXCD(pos.ExchCode), stockCode) //nolint:errcheck
		} else {
			m.wsClient.UnsubscribePrice(stockCode) //nolint:errcheck
		}
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
						m.executeSell(stockCode, pos, "트레일링 스탑")
					}
					m.Remove(context.Background(), stockCode)
					return
				}
			}
		}
	}

	switch {
	case price >= pos.TargetPrice:
		logger.Info("monitor: TARGET hit",
			map[string]any{"stock_code": stockCode, "price": price, "target": pos.TargetPrice})
		if !isTest {
			m.executeSell(stockCode, pos, "목표가 도달")
		}
		m.Remove(context.Background(), stockCode)

	case price <= pos.StopPrice:
		logger.Info("monitor: STOP hit",
			map[string]any{"stock_code": stockCode, "price": price, "stop": pos.StopPrice})
		if !isTest {
			m.executeSell(stockCode, pos, "손절가 도달")
		}
		m.Remove(context.Background(), stockCode)

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

// executeSell places a market sell order for the given position and returns the qty sold.
// Returns 0 if holdings lookup fails, qty is 0, or sell order fails.
func (m *Monitor) executeSell(stockCode string, pos *MonitoredEntry, reason string) int {
	ctx := context.Background()

	// US positions: use overseas sell API
	if pos.Market == "US" && pos.ExchCode != "" {
		return m.executeOverseasSell(stockCode, pos, reason)
	}

	holdings, err := m.kisClient.GetHoldings(ctx)
	if err != nil {
		logger.Error("auto-sell: GetHoldings failed",
			map[string]any{"stock_code": stockCode, "error": err.Error()})
		m.db.InsertServiceLog(ctx, "MONITOR", "ERROR", "자동 매도 실패: GetHoldings 오류", fmt.Sprintf("stock_code=%s error=%s", stockCode, err.Error()))
		return 0
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
		return 0
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
		m.db.InsertServiceLog(ctx, "MONITOR", "ERROR", "자동 매도 실패: 주문 오류", fmt.Sprintf("stock_code=%s qty=%d error=%s", stockCode, qty, err.Error()))
		return 0
	}

	logger.Info("auto-sell: sell order placed",
		map[string]any{"stock_code": stockCode, "qty": qty, "filled_price": pos.FilledPrice, "reason": reason})

	kisOrderID := ""
	if resp != nil {
		kisOrderID = resp.KISOrderID
	}
	_, _ = m.db.ExecContext(ctx,
		`INSERT INTO orders (stock_code, stock_name, order_type, qty, price, status, kis_order_id, sell_reason, market, created_at)
		 VALUES (?, ?, 'SELL', ?, ?, 'PENDING', ?, ?, 'KR', ?)`,
		stockCode, pos.StockName, qty, pos.FilledPrice, kisOrderID, reason, time.Now().UTC())

	// Notify engine that this position was sold.
	if pos.SoldCh != nil {
		select {
		case pos.SoldCh <- stockCode:
		default:
		}
	}

	return qty
}

// executeOverseasSell places a sell order for a US position.
func (m *Monitor) executeOverseasSell(stockCode string, pos *MonitoredEntry, reason string) int {
	ctx := context.Background()

	holdings, err := m.kisClient.GetOverseasHoldings(ctx, pos.ExchCode, "USD")
	if err != nil {
		logger.Error("auto-sell US: GetOverseasHoldings failed",
			map[string]any{"stock_code": stockCode, "error": err.Error()})
		m.db.InsertServiceLog(ctx, "MONITOR", "ERROR", "미장 자동 매도 실패: GetOverseasHoldings 오류", fmt.Sprintf("stock_code=%s error=%s", stockCode, err.Error()))
		return 0
	}

	qty := 0
	for _, h := range holdings {
		if h.StockCode == stockCode {
			fmt.Sscanf(h.OrderablQty, "%d", &qty)
			break
		}
	}
	if qty <= 0 {
		logger.Info("auto-sell US: no holdings found", map[string]any{"stock_code": stockCode})
		return 0
	}

	resp, err := m.kisClient.PlaceOverseasSellOrder(ctx, pos.ExchCode, stockCode, qty)
	if err != nil {
		logger.Error("auto-sell US: PlaceOverseasSellOrder failed",
			map[string]any{"stock_code": stockCode, "qty": qty, "error": err.Error()})
		m.db.InsertServiceLog(ctx, "MONITOR", "ERROR", "미장 자동 매도 실패: 주문 오류", fmt.Sprintf("stock_code=%s qty=%d error=%s", stockCode, qty, err.Error()))
		return 0
	}

	kisOrderID := ""
	if resp != nil {
		kisOrderID = resp.KISOrderID
	}
	_, _ = m.db.ExecContext(ctx,
		`INSERT INTO orders (stock_code, stock_name, order_type, qty, price, status, kis_order_id, sell_reason, market, created_at)
		 VALUES (?, ?, 'SELL', ?, ?, 'PENDING', ?, ?, 'US', ?)`,
		stockCode, pos.StockName, qty, pos.FilledPrice, kisOrderID, reason, time.Now().UTC())

	if pos.SoldCh != nil {
		select {
		case pos.SoldCh <- stockCode:
		default:
		}
	}

	logger.Info("auto-sell US: sell order placed",
		map[string]any{"stock_code": stockCode, "qty": qty, "reason": reason})
	return qty
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

		// US positions use overseas API
		if pos.Market == "US" {
			usHoldings, usErr := m.kisClient.GetOverseasHoldings(ctx, pos.ExchCode, "USD")
			if usErr != nil {
				logger.Error("liquidate US: GetOverseasHoldings failed",
					map[string]any{"stock_code": code, "error": usErr.Error()})
				continue
			}
			usQty := 0
			for _, h := range usHoldings {
				if h.StockCode == code {
					fmt.Sscanf(h.OrderablQty, "%d", &usQty)
					break
				}
			}
			if usQty <= 0 {
				m.Remove(ctx, code)
				continue
			}
			usResp, usErr := m.kisClient.PlaceOverseasSellOrder(ctx, pos.ExchCode, code, usQty)
			if usErr != nil {
				logger.Error("liquidate US: PlaceOverseasSellOrder failed",
					map[string]any{"stock_code": code, "error": usErr.Error()})
			} else {
				kisOrderID := ""
				if usResp != nil {
					kisOrderID = usResp.KISOrderID
				}
				_, _ = m.db.ExecContext(ctx,
					`INSERT INTO orders (stock_code, stock_name, order_type, qty, price, status, kis_order_id, sell_reason, market, created_at)
					 VALUES (?, ?, 'SELL', ?, ?, 'PENDING', ?, ?, 'US', ?)`,
					code, pos.StockName, usQty, pos.FilledPrice, kisOrderID, "일일 자동 청산", time.Now().UTC())
				logger.Info("liquidate US: sell order placed", map[string]any{"stock_code": code, "qty": usQty})
			}
			m.Remove(ctx, code)
			continue
		}

		// KR positions
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
			logger.Error("liquidate: sell order failed",
				map[string]any{"stock_code": code, "error": err.Error()})
		} else {
			logger.Info("liquidate: sell order placed",
				map[string]any{"stock_code": code, "qty": qty})
			kisOrderID := ""
			if liqResp != nil {
				kisOrderID = liqResp.KISOrderID
			}
			_, _ = m.db.ExecContext(ctx,
				`INSERT INTO orders (stock_code, stock_name, order_type, qty, price, status, kis_order_id, sell_reason, market, created_at)
				 VALUES (?, ?, 'SELL', ?, ?, 'PENDING', ?, ?, 'KR', ?)`,
				code, pos.StockName, qty, pos.FilledPrice, kisOrderID, "일일 자동 청산", time.Now().UTC())
		}


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
			StockCode:   pos.StockCode,
			StockName:   pos.StockName,
			FilledPrice: pos.FilledPrice,
			TargetPrice: pos.TargetPrice,
			StopPrice:   pos.StopPrice,
			OrderID:     pos.OrderID,
			CreatedAt:   time.Now(),
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

		// DB에서 해당 종목의 오늘 마지막 매수 주문 ID 조회.
		var orderID int64
		m.db.QueryRowContext(ctx, //nolint:errcheck
			`SELECT id FROM orders
			 WHERE stock_code = ? AND order_type = 'BUY' AND status = 'FILLED'
			   AND source = 'AGENT'
			 ORDER BY id DESC LIMIT 1`,
			h.StockCode,
		).Scan(&orderID)

		entry := MonitoredEntry{
			StockCode:   h.StockCode,
			StockName:   h.StockName,
			FilledPrice: filledPrice,
			TargetPrice: filledPrice * (1 + settings.TakeProfitPct/100),
			StopPrice:   filledPrice * (1 - settings.StopLossPct/100),
			OrderID:     orderID,
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
		m.mu.RLock()
		pos := m.positions[code]
		m.mu.RUnlock()
		if pos != nil && pos.Market == "US" && pos.ExchCode != "" {
			if err := m.wsClient.SubscribeOverseasPrice(exchCodeToEXCD(pos.ExchCode), code); err != nil {
				logger.Error("ResubscribeAll: SubscribeOverseasPrice failed",
					map[string]any{"stock_code": code, "error": err.Error()})
			} else {
				logger.Info("ResubscribeAll: subscribed overseas", map[string]any{"stock_code": code})
			}
		} else {
			if err := m.wsClient.SubscribePrice(code); err != nil {
				logger.Error("ResubscribeAll: SubscribePrice failed",
					map[string]any{"stock_code": code, "error": err.Error()})
			} else {
				logger.Info("ResubscribeAll: subscribed", map[string]any{"stock_code": code})
			}
		}
	}
}

// exchCodeToEXCD converts order exchange code to WebSocket/quote EXCD.
// NASD→NAS, NYSE→NYS, AMEX→AMS, SEHK→HKS, SHAA→SHS, SZAA→SZS, TKSE→TSE
func exchCodeToEXCD(exchCode string) string {
	m := map[string]string{
		"NASD": "NAS", "NYSE": "NYS", "AMEX": "AMS",
		"SEHK": "HKS", "SHAA": "SHS", "SZAA": "SZS", "TKSE": "TSE",
	}
	if excd, ok := m[exchCode]; ok {
		return excd
	}
	return exchCode
}

// LoadFromDB restores monitored positions from the database after a server restart.
func (m *Monitor) LoadFromDB(ctx context.Context) error {
	rows, err := m.db.QueryContext(ctx,
		`SELECT stock_code, stock_name, filled_price, target_price, stop_price, order_id, COALESCE(market, 'KR')
		 FROM monitored_positions`)
	if err != nil {
		return fmt.Errorf("load monitored_positions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var pos MonitoredEntry
		var market string
		if err := rows.Scan(
			&pos.StockCode, &pos.StockName,
			&pos.FilledPrice, &pos.TargetPrice, &pos.StopPrice, &pos.OrderID, &market,
		); err != nil {
			continue
		}
		pos.Market = market
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
				m.stagnMu.Unlock()
				if hasSince && since != nil && durationMin > 0 && thresholdPct > 0 {
					elapsed := time.Since(*since)
					if elapsed >= time.Duration(durationMin)*time.Minute {
						triggered = true
						triggerReason = fmt.Sprintf("횡보 감지: %.1f%% 이내 변동 %.0f분 지속", thresholdPct, elapsed.Minutes())
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
		m.executeSell(code, pos, triggerReason)
		m.Remove(ctx, code)
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
