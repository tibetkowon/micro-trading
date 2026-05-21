package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/micro-trading-for-agent/backend/internal/config"
	"github.com/micro-trading-for-agent/backend/internal/database"
	"github.com/micro-trading-for-agent/backend/internal/kis"
	"github.com/micro-trading-for-agent/backend/internal/models"
	"github.com/micro-trading-for-agent/backend/internal/monitor"
	"github.com/micro-trading-for-agent/backend/internal/ops"
	"github.com/micro-trading-for-agent/backend/internal/report"
	"github.com/micro-trading-for-agent/backend/internal/simulation"
	"github.com/micro-trading-for-agent/backend/internal/stockmaster"
	"github.com/micro-trading-for-agent/backend/internal/trader"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler holds shared dependencies for all HTTP handlers.
type Handler struct {
	db           *database.DB
	client       *kis.Client
	tokenManager *kis.TokenManager
	cfg          *config.Config
	monitor      *monitor.Monitor
	wsClient     *kis.WebSocketClient
	engine       *trader.Engine
	mstStore     *stockmaster.Store
	// available_cash 캐시 (GetServerStatus 호출 시 KIS API 중복 호출 방지)
	cashCacheMu  sync.Mutex
	cashCacheVal float64
	cashCacheExp time.Time
}

// NewHandler creates a new Handler with the given dependencies.
func NewHandler(db *database.DB, client *kis.Client, tokenManager *kis.TokenManager,
	cfg *config.Config, mon *monitor.Monitor, wsClient *kis.WebSocketClient) *Handler {
	return &Handler{
		db:           db,
		client:       client,
		tokenManager: tokenManager,
		cfg:          cfg,
		monitor:      mon,
		wsClient:     wsClient,
	}
}

// SetMSTStore injects the MST store for stock master lookups.
func (h *Handler) SetMSTStore(s *stockmaster.Store) {
	h.mstStore = s
}

// SetEngine injects the trading engine (called after engine is created in main).
func (h *Handler) SetEngine(e *trader.Engine) {
	h.engine = e
}

func (h *Handler) GetAdminTables(c *gin.Context) {
	tables := []string{
		"orders", "monitored_positions", "scan_logs",
		"trade_reports", "daily_reports", "simulation_results",
		"balances", "settings", "tokens",
	}
	c.JSON(http.StatusOK, gin.H{"tables": tables})
}

func (h *Handler) GetAdminTableData(c *gin.Context) {
	table := c.Param("table")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit > 200 {
		limit = 200
	}
	rows, total, err := h.db.ListTable(c.Request.Context(), table, page, limit)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"table": table, "page": page, "limit": limit,
		"total": total, "rows": rows,
	})
}

// GET /api/balance
func (h *Handler) GetBalance(c *gin.Context) {
	bal, err := ops.GetAccountBalance(c.Request.Context(), h.client, h.db)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	assetChangePct := 0.0
	if v, err2 := strconv.ParseFloat(bal.AssetChangeRate, 64); err2 == nil {
		assetChangePct = v
	}
	// 당일 체결 완료 거래에서 실시간 손익 계산
	kst := ops.KSTLocation()
	todayStr := time.Now().In(kst).Format("2006-01-02")
	todayPnl, todayPnlPct, winRate := 0.0, 0.0, 0.0
	if trades, err2 := h.db.GetCompletedTradesBySoldDate(c.Request.Context(), todayStr); err2 == nil && len(trades) > 0 {
		var sumProfitPct float64
		winning := 0
		for _, t := range trades {
			todayPnl += t.ProfitAmount
			sumProfitPct += t.ProfitPct
			if t.ProfitAmount >= 0 {
				winning++
			}
		}
		todayPnlPct = sumProfitPct / float64(len(trades))
		winRate = float64(winning) / float64(len(trades)) * 100
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"total_assets":            bal.TotalEval,
			"available_cash":          bal.OrderableAmt,
			"today_pnl":               todayPnl,
			"today_pnl_pct":           todayPnlPct,
			"win_rate":                winRate,
			"total_assets_change_pct": assetChangePct,
		},
	})
}

// GET /api/orders?sync=true
// sync=true 이면 KIS 체결 내역을 먼저 동기화 (PENDING → FILLED/PARTIALLY_FILLED 갱신)
func (h *Handler) GetOrders(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	days, _ := strconv.Atoi(c.DefaultQuery("days", "1"))
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	if days < 1 || days > 90 {
		days = 1
	}
	since := time.Now().AddDate(0, 0, -(days - 1))
	since = time.Date(since.Year(), since.Month(), since.Day(), 0, 0, 0, 0, since.Location())

	var syncError string
	if c.Query("sync") == "true" {
		syncCtx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()
		endDate := time.Now().Format("20060102")
		startDate := since.Format("20060102")
		if _, err := ops.GetOrderHistory(syncCtx, h.client, h.db, startDate, endDate); err != nil {
			syncError = err.Error()
		}
	}

	orders, err := ops.GetLocalOrderHistory(c.Request.Context(), h.db, since, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if orders == nil {
		orders = []models.Order{}
	}
	total, _ := h.db.CountOrdersSince(c.Request.Context(), since)
	resp := gin.H{"orders": orders, "data": orders, "total": total, "limit": limit, "offset": offset}
	if syncError != "" {
		resp["sync_error"] = syncError
	}
	c.JSON(http.StatusOK, resp)
}

// POST /api/orders/:id/cancel — KIS 미체결 주문 취소 (TTTC0084R 확인 후 TTTC0013U 취소 요청)
// 이미 체결된 주문(FILLED)이나 존재하지 않는 KIS 주문번호는 오류 반환.
func (h *Handler) CancelOrder(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	result, err := ops.CancelOrder(c.Request.Context(), h.client, h.db, id)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"order_id":     result.OrderID,
		"kis_order_id": result.KISOrderID,
		"status":       "CANCELLED",
	})
}

// DELETE /api/orders/:id
func (h *Handler) DeleteOrder(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	// Firestore: 해당 주문 존재 확인 후 삭제
	order, err := h.db.GetOrderByID(c.Request.Context(), id)
	if err != nil || order == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	if err := h.db.DeleteOrder(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

// POST /api/orders — place a buy/sell order
// Optional: target_pct and stop_pct register a position for real-time monitoring.
func (h *Handler) PlaceOrder(c *gin.Context) {
	var req struct {
		StockCode string  `json:"stock_code" binding:"required"`
		OrderType string  `json:"order_type" binding:"required"`
		Qty       int     `json:"qty" binding:"required,min=1"`
		Price     float64 `json:"price"`
		TargetPct float64 `json:"target_pct"` // 목표 수익률 (%)
		StopPct   float64 `json:"stop_pct"`   // 손절 비율 (%)
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.db.GetSetting(c.Request.Context(), "trading_enabled") == "false" {
		c.JSON(http.StatusForbidden, gin.H{"error": "거래가 비활성화되어 있습니다. 설정에서 Trading을 ON으로 변경하세요."})
		return
	}

	var orderType models.OrderType
	switch req.OrderType {
	case "BUY":
		orderType = models.OrderTypeBuy
	case "SELL":
		orderType = models.OrderTypeSell
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "order_type must be BUY or SELL"})
		return
	}

	divn := "00" // 지정가
	if req.Price == 0 {
		divn = "01" // 시장가
	}

	result, err := ops.PlaceOrder(c.Request.Context(), h.client, h.db, ops.PlaceOrderRequest{
		StockCode: req.StockCode,
		OrderType: orderType,
		Qty:       req.Qty,
		Price:     req.Price,
		OrderDivn: divn,
		TargetPct: req.TargetPct,
		StopPct:   req.StopPct,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// BUY 주문이고 target/stop 설정 시 → 모니터링 등록 (체결가는 주문가로 근사)
	// 실제 체결가는 WebSocket 체결통보로 업데이트됨.
	if orderType == models.OrderTypeBuy && req.TargetPct > 0 && req.StopPct > 0 && h.monitor != nil {
		filledPrice := req.Price
		if filledPrice <= 0 {
			// 시장가 주문: 현재가로 목표/손절 계산 (이후 체결통보로 재보정 가능)
			if info, priceErr := h.client.GetStockPrice(c.Request.Context(), req.StockCode); priceErr == nil {
				if p := parseFloat(info.CurrentPrice); p > 0 {
					filledPrice = p
				}
			}
		}
		if filledPrice > 0 {
			entry := monitor.MonitoredEntry{
				StockCode:   req.StockCode,
				StockName:   req.StockCode, // 종목명은 DB 동기화 전까지 코드로 대체
				FilledPrice: filledPrice,
				TargetPrice: filledPrice * (1 + req.TargetPct/100),
				StopPrice:   filledPrice * (1 - req.StopPct/100),
				OrderID:     result.OrderID,
			}
			if regErr := h.monitor.Register(c.Request.Context(), entry); regErr != nil {
				// 등록 실패는 치명적이지 않음 — 로그만 남김
				_ = regErr
			}
		}
	}

	c.JSON(http.StatusCreated, result)
}

// GET /api/server/status — 통합 서버 상태 (시장개장/WebSocket연결/모니터링 수)
func (h *Handler) GetServerStatus(c *gin.Context) {
	now := time.Now().In(ops.KSTLocation())

	// KR market open check (weekdays 09:00~15:30 KST)
	marketOpen := false
	if wd := now.Weekday(); wd != time.Saturday && wd != time.Sunday {
		min := now.Hour()*60 + now.Minute()
		if min >= 9*60 && min < 15*60+30 {
			marketOpen = true
		}
	}

	wsConnected := false
	if h.wsClient != nil {
		wsConnected = h.wsClient.IsConnected()
	}

	monitoredCount := 0
	if h.monitor != nil {
		monitoredCount = h.monitor.Count()
	}

	// Available cash from balance (30초 캐시 — KIS API 중복 호출 방지)
	availableCash := float64(0)
	h.cashCacheMu.Lock()
	if time.Now().Before(h.cashCacheExp) {
		availableCash = h.cashCacheVal
		h.cashCacheMu.Unlock()
	} else {
		h.cashCacheMu.Unlock()
		if bal, err := h.client.GetInquireBalance(c.Request.Context()); err == nil {
			availableCash = parseFloat(bal.OrderableAmt)
			h.cashCacheMu.Lock()
			h.cashCacheVal = availableCash
			h.cashCacheExp = time.Now().Add(30 * time.Second)
			h.cashCacheMu.Unlock()
		}
	}

	tradingEnabled := h.db.GetSetting(c.Request.Context(), "trading_enabled") != "false"

	traderState := string(trader.StateIdle)
	haltReason := ""
	if h.engine != nil {
		traderState = string(h.engine.GetState())
		haltReason = h.engine.GetHaltReason()
	}

	c.JSON(http.StatusOK, gin.H{
		"market_open":     marketOpen,
		"trading_enabled": tradingEnabled,
		"available_cash":  availableCash,
		"ws_connected":    wsConnected,
		"monitored_count": monitoredCount,
		"trader_state":    traderState,
		"halt_reason":     haltReason,
	})
}

// POST /api/monitor/liquidate-all — 전체 보유 종목 즉시 시장가 매도 (패닉셀)
func (h *Handler) LiquidateAll(c *gin.Context) {
	if h.monitor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "monitor not available"})
		return
	}
	go h.monitor.LiquidateAll(c.Request.Context(), "KR")
	c.JSON(http.StatusOK, gin.H{"message": "전체 매도 실행 중"})
}

// GET /api/monitor/positions — 현재 모니터링 중인 포지션 목록
func (h *Handler) GetMonitorPositions(c *gin.Context) {
	if h.monitor == nil {
		c.JSON(http.StatusOK, gin.H{"data": []any{}})
		return
	}
	positions := h.monitor.List()
	if positions == nil {
		positions = []models.MonitoredPosition{}
	}
	type posView struct {
		ID           int64   `json:"id"`
		StockCode    string  `json:"stock_code"`
		StockName    string  `json:"stock_name"`
		AvgPrice     float64 `json:"avg_price"`
		CurrentPrice float64 `json:"current_price"`
		TargetPrice  float64 `json:"target_price"`
		StopPrice    float64 `json:"stop_price"`
		Quantity     int     `json:"quantity"`
		PnlPct       float64 `json:"pnl_pct"`
		PnlAmount    float64 `json:"pnl_amount"`
		HeldDays     int     `json:"held_days"`
		EntryTime    string  `json:"entry_time"`
		Status       string  `json:"status"`
	}
	kst := ops.KSTLocation()
	now := time.Now().In(kst)
	views := make([]posView, len(positions))
	for i, p := range positions {
		heldDays := int(now.Sub(p.CreatedAt.In(kst)).Hours() / 24)
		// RemainingQty 우선 사용, 0이면 원래 주문 수량 fallback
		qty := p.RemainingQty
		if qty == 0 && p.OrderID > 0 {
			if order, err := h.db.GetOrderByID(c.Request.Context(), p.OrderID); err == nil && order != nil {
				qty = order.Qty
			}
		}
		cp := p.CurrentPrice
		if cp == 0 {
			cp = p.FilledPrice
		}
		pnlPct := 0.0
		pnlAmount := 0.0
		if p.FilledPrice > 0 {
			pnlPct = (cp - p.FilledPrice) / p.FilledPrice * 100
			pnlAmount = (cp - p.FilledPrice) * float64(qty)
		}
		views[i] = posView{
			ID:           p.ID,
			StockCode:    p.StockCode,
			StockName:    p.StockName,
			AvgPrice:     p.FilledPrice,
			CurrentPrice: cp,
			TargetPrice:  p.TargetPrice,
			StopPrice:    p.StopPrice,
			Quantity:     qty,
			PnlPct:       pnlPct,
			PnlAmount:    pnlAmount,
			HeldDays:     heldDays,
			EntryTime:    p.CreatedAt.In(kst).Format("2006-01-02 15:04"),
			Status:       "MONITORING",
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": views})
}

// POST /api/monitor/positions/:code/sell — 강제 시장가 매도 + 모니터링 해제
func (h *Handler) ForceSellMonitorPosition(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "stock code required"})
		return
	}
	if h.monitor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "모니터링 서비스가 비활성화되어 있습니다"})
		return
	}
	qty, err := h.monitor.ForceSell(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"sold": code, "qty": qty})
}

// DELETE /api/monitor/positions/:code — 모니터링 포지션 제거
func (h *Handler) RemoveMonitorPosition(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "stock code required"})
		return
	}
	if h.monitor != nil {
		h.monitor.Remove(c.Request.Context(), code)
	}
	c.JSON(http.StatusOK, gin.H{"removed": code})
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// GET /api/positions — KIS 실시간 보유 종목 조회 (inquire-balance output1)
func (h *Handler) GetPositions(c *gin.Context) {
	holdings, err := h.client.GetHoldings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"positions": holdings})
}

// GET /api/logs/scan?limit=20 — 스캔 로그 조회 (최신 순)
func (h *Handler) GetScanLogs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 || limit > 500 {
		limit = 20
	}

	logs, err := h.db.ListScanLogs(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if logs == nil {
		logs = []models.ScanLog{}
	}
	type scanLogView struct {
		ID                int64   `json:"id"`
		ScannedAt         string  `json:"scanned_at"`
		CreatedAt         string  `json:"created_at"`
		EvaluatedCount    int     `json:"evaluated_count"`
		PassedHardFilter  int     `json:"passed_hard_filter"`
		ScoredCount       int     `json:"scored_count"`
		SelectedStockCode string  `json:"selected_stock_code"`
		SelectedStockName string  `json:"selected_stock_name"`
		RejectionSummary  string  `json:"rejection_summary"`
		HasRawData        bool    `json:"has_raw_data"`
		Stocks            []gin.H `json:"stocks"`
	}
	type topStockEntry struct {
		Code          string  `json:"code"`
		Name          string  `json:"name"`
		Strength      float64 `json:"strength"`
		RSI           float64 `json:"rsi"`
		MACDBullish   bool    `json:"macd_bullish"`
		BidAsk        float64 `json:"bid_ask"`
		VWAPDiff      float64 `json:"vwap_diff"`
		VolRatio      float64 `json:"vol_ratio"`
		ProgramNetBuy float64 `json:"program_net_buy"`
		MicroBidAsk   float64 `json:"micro_bid_ask"`
		VIDisparity   float64 `json:"vi_disparity"`
		Volume        string  `json:"volume"`
		Total         float64 `json:"total"`
		HasChart      bool    `json:"has_chart"` // 차트 API 성공 여부
	}

	kst := ops.KSTLocation()
	views := make([]scanLogView, len(logs))
	for i, l := range logs {
		stocks := []gin.H{}
		if l.TopStocks != "" {
			// JSON 형식 (신규) vs code(score) 형식 (구버전) 하위 호환
			var entries []topStockEntry
			if err := json.Unmarshal([]byte(l.TopStocks), &entries); err == nil {
				for _, e := range entries {
					stockName := e.Name
					if h.mstStore != nil && e.Code != "" {
						if sm, err2 := h.mstStore.GetByCode(c.Request.Context(), e.Code); err2 == nil && sm != nil {
							stockName = sm.StockName
						}
					}
					stocks = append(stocks, gin.H{
						"stock_code":      e.Code,
						"stock_name":      stockName,
						"strength":        e.Strength,
						"rsi":             e.RSI,
						"macd_bullish":    e.MACDBullish,
						"bid_ask_ratio":   e.BidAsk,
						"vwap_disparity":  e.VWAPDiff,
						"volume_ratio":    e.VolRatio,
						"program_net_buy": e.ProgramNetBuy,
						"micro_bid_ask":   e.MicroBidAsk,
						"vi_disparity":    e.VIDisparity,
						"volume":          e.Volume,
						"total_score":     e.Total,
						"has_chart":       e.HasChart,
					})
				}
			} else {
				// 구버전: "code(score), code(score)" 형식
				for _, entry := range splitTrimmed(l.TopStocks, ",") {
					code := entry
					var score *float64
					if idx := strings.Index(entry, "("); idx > 0 && strings.HasSuffix(entry, ")") {
						code = strings.TrimSpace(entry[:idx])
						if v, err2 := strconv.ParseFloat(entry[idx+1:len(entry)-1], 64); err2 == nil {
							score = &v
						}
					}
					if code == "" {
						continue
					}
					stockName := code
					if h.mstStore != nil {
						if sm, err2 := h.mstStore.GetByCode(c.Request.Context(), code); err2 == nil && sm != nil {
							stockName = sm.StockName
						}
					}
					stocks = append(stocks, gin.H{"stock_code": code, "stock_name": stockName, "total_score": score})
				}
			}
		}

		selectedName := l.OrderedCode
		if h.mstStore != nil && l.OrderedCode != "" {
			if sm, err := h.mstStore.GetByCode(c.Request.Context(), l.OrderedCode); err == nil && sm != nil {
				selectedName = sm.StockName
			}
		}

		kstTime := l.Timestamp
		if t, err2 := time.Parse(time.RFC3339, l.Timestamp); err2 == nil {
			kstTime = t.In(kst).Format("2006-01-02 15:04:05")
		}

		views[i] = scanLogView{
			ID:                l.ID,
			ScannedAt:         kstTime,
			CreatedAt:         kstTime,
			EvaluatedCount:    l.TotalCandidates,
			PassedHardFilter:  l.StocksFound,
			ScoredCount:       l.StocksFound,
			SelectedStockCode: l.OrderedCode,
			SelectedStockName: selectedName,
			RejectionSummary:  l.SkipReason,
			HasRawData:        l.StockRawData != "",
			Stocks:            stocks,
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": views})
}

// GetScanLogRaw returns the full raw StockInfo + ScoreDetail for each top stock in a scan log.
func (h *Handler) GetScanLogRaw(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	l, err := h.db.GetScanLog(c.Request.Context(), id)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	if l.StockRawData == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "no_raw_data"})
		return
	}

	var raw []json.RawMessage
	if err := json.Unmarshal([]byte(l.StockRawData), &raw); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "parse_error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": raw})
}

// splitTrimmed splits s by sep and trims whitespace from each element.
func splitTrimmed(s, sep string) []string {
	parts := make([]string, 0)
	for _, p := range strings.Split(s, sep) {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

// POST /api/trader/force-run — 강제 실행 (즉시 매수 사이클 트리거)
func (h *Handler) ForceRunTrader(c *gin.Context) {
	if h.engine == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "엔진이 설정되지 않았습니다"})
		return
	}
	h.engine.ForceRun(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// GET /api/stock/:code — 현재가 + MA5 + MA20
func (h *Handler) GetStock(c *gin.Context) {
	code := c.Param("code")
	info, err := ops.GetStockInfo(c.Request.Context(), h.client, code)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}

// GET /api/stock/:code/chart?interval=1m|5m|1h
func (h *Handler) GetStockChart(c *gin.Context) {
	code := c.Param("code")
	interval := c.DefaultQuery("interval", "1m")
	candles, err := ops.GetChart(c.Request.Context(), h.client, code, interval)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"stock_code": code, "interval": interval, "candles": candles})
}

// settingFloat reads a DB setting as float64 with a default fallback.
func settingFloat(val string, def float64) float64 {
	if v, err := strconv.ParseFloat(val, 64); err == nil {
		return v
	}
	return def
}

// settingInt reads a DB setting as int with a default fallback.
func settingInt(val string, def int) int {
	if v, err := strconv.Atoi(val); err == nil {
		return v
	}
	return def
}

// GET /api/settings — 서버 상태 및 런타임 설정 조회
func (h *Handler) GetSettings(c *gin.Context) {
	accountNo := h.cfg.KISAccountNo
	maskedAccount := ""
	if len(accountNo) >= 4 {
		maskedAccount = "****" + accountNo[len(accountNo)-4:]
	}

	wsConnected := false
	if h.wsClient != nil {
		wsConnected = h.wsClient.IsConnected()
	}

	tradingEnabled := h.db.GetSetting(c.Request.Context(), "trading_enabled") != "false"
	rankingExclCls := h.db.GetSetting(c.Request.Context(), "ranking_exclude_cls")
	if rankingExclCls == "" {
		rankingExclCls = "1111111111"
	}

	ts, _ := h.db.GetTradingSettings(c.Request.Context())

	ctx := c.Request.Context()
	db := h.db

	// Weights (stored as weight_<key> in DB)
	weightKeys := []string{"strength", "rsi", "macd", "bidask", "vwap", "volume"}
	weights := map[string]float64{}
	for _, k := range weightKeys {
		weights[k] = settingFloat(db.GetSetting(ctx, "weight_"+k), 0)
	}

	// Filters (stored as filter_<key>_enabled / filter_<key>_value in DB)
	filterKeys := []string{"rsi_upper_limit", "strength_lower", "vwap_min", "vwap_max",
		"high_disparity", "open_rise_limit", "high_elapsed_min", "volume_ratio_lower"}
	filters := map[string]gin.H{}
	for _, k := range filterKeys {
		filters[k] = gin.H{
			"enabled": db.GetSetting(ctx, "filter_"+k+"_enabled") == "true",
			"value":   settingFloat(db.GetSetting(ctx, "filter_"+k+"_value"), 0),
		}
	}

	// Schedule
	tradeStart := ts.TradingStartTime
	if tradeStart == "" {
		tradeStart = "09:15"
	}
	tradeEnd := ts.TradingEndTime
	if tradeEnd == "" {
		tradeEnd = "15:15"
	}
	schedule := gin.H{
		"trade_start":              tradeStart,
		"trade_end":                tradeEnd,
		"scan_interval":            settingInt(db.GetSetting(ctx, "scan_interval"), 60),
		"indicator_check_interval": ts.IndicatorCheckIntervalMin,
	}

	indicatorRSISellEnabled := db.GetSetting(ctx, "indicator_rsi_sell_enabled") == "true"
	minScore := settingFloat(db.GetSetting(ctx, "min_score_threshold"), 0)

	data := gin.H{
		// 거래 조건
		"max_positions":               ts.MaxPositions,
		"order_amount_pct":            ts.OrderAmountPct,
		"take_profit_pct":             ts.TakeProfitPct,
		"stop_loss_pct":               ts.StopLossPct,
		"etf_take_profit_pct":         ts.ETFTakeProfitPct,
		"etf_stop_loss_pct":           ts.ETFStopLossPct,
		"daily_loss_limit_pct":        ts.DailyMaxLossPct,
		"ranking_condition":           ts.RankingCondition,
		"ranking_top_n":               ts.RankingTopN,
		"ranking_exclude_cls":         rankingExclCls,
		"indicator_rsi_sell_enabled":  indicatorRSISellEnabled,
		"indicator_macd_bearish_sell": ts.IndicatorMACDBearishSell,
		"buy_pause_start":             ts.BuyPauseStart,
		"buy_pause_end":               ts.BuyPauseEnd,
		// 점수 시스템
		"min_score": minScore,
		"weights":   weights,
		// 하드 필터
		"filters": filters,
		// 스케줄
		"schedule": schedule,
		// 트레일링
		"trailing_trigger_pct":       ts.TrailingTriggerPct,
		"trailing_stop_pct":          ts.TrailingStopPct,
		"trailing_mode":              ts.TrailingMode,
		"tick_tier0_stop_loss_ticks": ts.TickTier0StopLossTicks,
		"tick_tier1_trigger_pct":     ts.TickTier1TriggerPct,
		"tick_tier1_trail_ticks":     ts.TickTier1TrailTicks,
		"tick_tier2_trigger_pct":     ts.TickTier2TriggerPct,
		"tick_tier2_trail_ticks":     ts.TickTier2TrailTicks,
		// 스태그네이션
		"stagnation_threshold_pct":         ts.StagnationThresholdPct,
		"stagnation_duration_min":          ts.StagnationDurationMin,
		"stagnation_partial_exit_enabled":  ts.StagnationPartialExitEnabled,
		"stagnation_bidask_sell_threshold": ts.StagnationBidAskSellThreshold,
		// 랭킹 (누락)
		"ranking_types":     ts.RankingTypes,
		"ranking_price_min": ts.RankingPriceMin,
		"ranking_price_max": ts.RankingPriceMax,
		"ranking_exchanges": ts.RankingExchanges,
		// 재진입 / 손절
		"sell_on_upper_limit":              ts.SellOnUpperLimit,
		"max_consecutive_losses":           ts.MaxConsecutiveLosses,
		"consecutive_loss_reset_on_profit": ts.ConsecutiveLossResetOnProfit,
		"max_bidask_spread_pct":            ts.MaxBidAskSpreadPct,
		"block_reentry_on_loss":            ts.BlockReentryOnLoss,
		"reentry_score_penalty":            ts.ReentryScorePenalty,
		"reentry_cooldown_min":             ts.ReentryCooldownMin,
		"loss_cooldown_min":                ts.LossCooldownMin,
		"loss_reentry_price_guard":         ts.LossReentryPriceGuard,
		// 하드 필터 (누락)
		"hard_ma60_support_enabled":  ts.HardMA60SupportEnabled,
		"hard_ma120_support_enabled": ts.HardMA120SupportEnabled,
		"hard_program_buy_min":       ts.HardProgramBuyMin,
		"hard_peak_turn_enabled":     ts.HardPeakTurnEnabled,
		"hard_peak_rsi_min":          ts.HardPeakRSIMin,
		// 스트림
		"stream_bypass_enabled":     ts.StreamBypassEnabled,
		"stream_big_trade_amount":   ts.StreamBigTradeAmount,
		"stream_velocity_threshold": ts.StreamVelocityThreshold,
		// 매도 조건
		"sell_conditions": ts.SellConditions,
		"buy_order_type":  ts.BuyOrderType,
		// 스코어 가중치 (flat 키로 노출 — transformSettings fallback 호환)
		"score_weight_strength":     ts.ScoreWeightStrength,
		"score_weight_rsi":          ts.ScoreWeightRSI,
		"score_weight_macd":         ts.ScoreWeightMACD,
		"score_weight_bidask":       ts.ScoreWeightBidAsk,
		"score_weight_vwap":         ts.ScoreWeightVWAP,
		"score_weight_volume":       ts.ScoreWeightVolume,
		"score_weight_program_buy":  ts.ScoreWeightProgramBuy,
		"score_weight_micro_bidask": ts.ScoreWeightMicroBidAsk,
		"score_weight_vi_disparity": ts.ScoreWeightVIDisparity,
		// 시스템 / KIS 설정 (읽기 전용)
		"account_no":      maskedAccount,
		"account_type":    h.cfg.KISAccountType,
		"kis_configured":  h.cfg.KISAppKey != "" && h.cfg.KISAppSecret != "",
		"ws_connected":    wsConnected,
		"trading_enabled": tradingEnabled,
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

// PATCH /api/settings — 런타임 설정 업데이트
func (h *Handler) UpdateSettings(c *gin.Context) {
	var req struct {
		TradingEnabled *bool  `json:"trading_enabled"`
		RankingExclCls string `json:"ranking_exclude_cls"`
		// Autonomous trading settings (all optional)
		TakeProfitPct             *float64 `json:"take_profit_pct"`
		StopLossPct               *float64 `json:"stop_loss_pct"`
		ETFTakeProfitPct          *float64 `json:"etf_take_profit_pct"`
		ETFStopLossPct            *float64 `json:"etf_stop_loss_pct"`
		StockTakeProfitPct        *float64 `json:"stock_take_profit_pct"`
		StockStopLossPct          *float64 `json:"stock_stop_loss_pct"`
		StockTaxRate              *float64 `json:"stock_tax_rate"`
		RankingTypes              []string `json:"ranking_types"`
		RankingPriceMin           string   `json:"ranking_price_min"`
		RankingPriceMax           string   `json:"ranking_price_max"`
		MaxPositions              *int     `json:"max_positions"`
		OrderAmountPct            *float64 `json:"order_amount_pct"`
		SellConditions            []string `json:"sell_conditions"`
		IndicatorCheckIntervalMin *int     `json:"indicator_check_interval_min"`
		IndicatorRSISellThreshold *float64 `json:"indicator_rsi_sell_threshold"`
		IndicatorMACDBearishSell  *bool    `json:"indicator_macd_bearish_sell"`
		RankingVolumeMinIncrRate  *float64 `json:"ranking_volume_min_incrrate"`
		RankingStrengthMin        *float64 `json:"ranking_strength_min"`
		RankingFluctuationMinRate *float64 `json:"ranking_fluctuation_min_rate"`
		RankingFluctuationMaxRate *float64 `json:"ranking_fluctuation_max_rate"`
		RankingVIKindCode         *string  `json:"ranking_vi_kind_code"`
		RankingTopN               *int     `json:"ranking_top_n"`
		TradingStartTime          string   `json:"trading_start_time"`
		TradingEndTime            string   `json:"trading_end_time"`
		StagnationThresholdPct    *float64 `json:"stagnation_threshold_pct"`
		StagnationDurationMin     *int     `json:"stagnation_duration_min"`
		RankingCondition          string   `json:"ranking_condition"`
		RankingExchanges          []string `json:"ranking_exchanges"`
		RankingVolumeBlngClsCodes []string `json:"ranking_volume_blng_cls_codes"`
		// 거래대금 하한선
		MinTradingValue *float64 `json:"min_trading_value"`
		// 매수 중단 시간대
		BuyPauseStart string `json:"buy_pause_start"`
		BuyPauseEnd   string `json:"buy_pause_end"`
		// 트레일링 스탑
		TrailingTriggerPct *float64 `json:"trailing_trigger_pct"`
		TrailingStopPct    *float64 `json:"trailing_stop_pct"`
		// 트레일링 모드 + 틱 트레일
		TrailingMode           *string  `json:"trailing_mode"`
		TickTier0StopLossTicks *int     `json:"tick_tier0_stop_loss_ticks"`
		TickTier1TriggerPct    *float64 `json:"tick_tier1_trigger_pct"`
		TickTier1TrailTicks    *int     `json:"tick_tier1_trail_ticks"`
		TickTier2TriggerPct    *float64 `json:"tick_tier2_trigger_pct"`
		TickTier2TrailTicks    *int     `json:"tick_tier2_trail_ticks"`
		// 일일 최대 손실
		DailyMaxLossPct *float64 `json:"daily_max_loss_pct"`
		// 지수 필터 (nil = 변경 안 함)
		IndexCodes []string `json:"index_codes"`
		// 하드 필터
		FilterRsiMax           *float64 `json:"filter_rsi_max"`
		FilterDisparityM5Max   *float64 `json:"filter_disparity_m5_max"`
		FilterHighPriceDiffMin *float64 `json:"filter_high_price_diff_min"`
		FilterOpenPriceDiffMax *float64 `json:"filter_open_price_diff_max"`
		// 지수 하락 임계값
		IndexDropThresholdPct *float64 `json:"index_drop_threshold_pct"`
		// 요일 스케줄
		TradingDays []int `json:"trading_days"`
		// 하드 감시 종목 / 순위 유지 시간
		HardWatchSymbols     []string `json:"hard_watch_symbols"`
		RankLeaseDurationMin *int     `json:"rank_lease_duration_min"`
		// AI 매매 기준값 — 하드 리젝션 룰
		HardDisparityM5Min      *float64 `json:"hard_disparity_m5_min"`
		HardDisparityM5Max      *float64 `json:"hard_disparity_m5_max"`
		HardHighPriceDiffMax    *float64 `json:"hard_high_price_diff_max"`
		HardHighPriceDiffMin    *float64 `json:"hard_high_price_diff_min"`
		HardPrevVolRatioMax     *float64 `json:"hard_prev_vol_ratio_max"`
		HardStrengthMin         *float64 `json:"hard_strength_min"`
		HardRSIMax              *float64 `json:"hard_rsi_max"`
		HardOpenPriceDiffMax    *float64 `json:"hard_open_price_diff_max"`
		HardMACDBearishEnabled  *bool    `json:"hard_macd_bearish_enabled"`
		HardMA60SupportEnabled  *bool    `json:"hard_ma60_support_enabled"`
		HardMA120SupportEnabled *bool    `json:"hard_ma120_support_enabled"`
		HardHighFormedMinsMax   *float64 `json:"hard_high_formed_mins_max"`
		HardVolVs3AvgRatioMin   *float64 `json:"hard_vol_vs_3avg_ratio_min"`
		HardRelativeStrengthMin *float64 `json:"hard_relative_strength_min"`
		// AI 매매 기준값 — 랭킹 기준
		VWAPDiffMin                   *float64 `json:"vwap_diff_min"`
		VWAPDiffMax                   *float64 `json:"vwap_diff_max"`
		RSIBuyMin                     *float64 `json:"rsi_buy_min"`
		RSIBuyMax                     *float64 `json:"rsi_buy_max"`
		BidAskRatioMin                *float64 `json:"bid_ask_ratio_min"`
		MinMarketCap                  *float64 `json:"min_market_cap"`
		MinExpectedProfitPct          *float64 `json:"min_expected_profit_pct"`
		StagnationPartialExitEnabled  *bool    `json:"stagnation_partial_exit_enabled"`
		StagnationBidAskSellThreshold *float64 `json:"stagnation_bid_ask_sell_threshold"`
		// 부분 익절
		PartialTPEnabled   *bool    `json:"partial_tp_enabled"`
		PartialTPPct       *float64 `json:"partial_tp_pct"`
		PartialTPRatio     *float64 `json:"partial_tp_ratio"`
		PartialTPRaiseStop *bool    `json:"partial_tp_raise_stop"`
		// UI 거래조건 추가 필드
		DailyLossLimitPct *float64 `json:"daily_loss_limit_pct"`
		// 재진입 / 손절 제어
		SellOnUpperLimit             *bool    `json:"sell_on_upper_limit"`
		MaxConsecutiveLosses         *int     `json:"max_consecutive_losses"`
		ConsecutiveLossResetOnProfit *bool    `json:"consecutive_loss_reset_on_profit"`
		MaxBidAskSpreadPct           *float64 `json:"max_bidask_spread_pct"`
		BlockReentryOnLoss           *bool    `json:"block_reentry_on_loss"`
		ReentryScorePenalty          *float64 `json:"reentry_score_penalty"`
		ReentryCooldownMin           *int     `json:"reentry_cooldown_min"`
		LossCooldownMin              *int     `json:"loss_cooldown_min"`
		LossReentryPriceGuard        *bool    `json:"loss_reentry_price_guard"`
		// 하드 피크 감지
		HardPeakTurnEnabled *bool    `json:"hard_peak_turn_enabled"`
		HardPeakRSIMin      *float64 `json:"hard_peak_rsi_min"`
		// 주문 유형
		BuyOrderType *string `json:"buy_order_type"`
		// 스트림
		StreamBypassEnabled     *bool    `json:"stream_bypass_enabled"`
		StreamBigTradeAmount    *float64 `json:"stream_big_trade_amount"`
		StreamVelocityThreshold *float64 `json:"stream_velocity_threshold"`
		IndicatorRSISellEnabled *bool    `json:"indicator_rsi_sell_enabled"`
		MinScore                *float64 `json:"min_score"`
		MinScoreThreshold       *float64 `json:"min_score_threshold"`
		UniversalCooldownMin    *int     `json:"universal_cooldown_min"`
		ScoreWeightStrength     *int     `json:"score_weight_strength"`
		ScoreWeightRSI          *int     `json:"score_weight_rsi"`
		ScoreWeightMACD         *int     `json:"score_weight_macd"`
		ScoreWeightBidAsk       *int     `json:"score_weight_bid_ask"`
		ScoreWeightVWAP         *int     `json:"score_weight_vwap"`
		ScoreWeightVolume       *int     `json:"score_weight_volume"`
		ScoreWeightProgramBuy   *int     `json:"score_weight_program_buy"`
		ScoreWeightMicroBidAsk  *int     `json:"score_weight_micro_bid_ask"`
		ScoreWeightVIDisparity  *int     `json:"score_weight_vi_disparity"`
		// UI 점수시스템 — weights (nested)
		Weights *struct {
			Strength float64 `json:"strength"`
			RSI      float64 `json:"rsi"`
			MACD     float64 `json:"macd"`
			BidAsk   float64 `json:"bidask"`
			VWAP     float64 `json:"vwap"`
			Volume   float64 `json:"volume"`
		} `json:"weights"`
		// UI 하드필터 — filters (nested map)
		Filters map[string]struct {
			Enabled bool    `json:"enabled"`
			Value   float64 `json:"value"`
		} `json:"filters"`
		// UI 스케줄 — schedule (nested)
		Schedule *struct {
			TradeStart             string `json:"trade_start"`
			TradeEnd               string `json:"trade_end"`
			ScanInterval           int    `json:"scan_interval"`
			IndicatorCheckInterval int    `json:"indicator_check_interval"`
		} `json:"schedule"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	changed := false

	save := func(key, val string) bool {
		if err := h.db.SetSetting(ctx, key, val); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "저장 실패: " + err.Error()})
			return false
		}
		changed = true
		return true
	}

	if req.TradingEnabled != nil {
		val := "true"
		if !*req.TradingEnabled {
			val = "false"
		}
		if !save("trading_enabled", val) {
			return
		}
	}

	if req.RankingExclCls != "" {
		if len(req.RankingExclCls) != 10 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ranking_exclude_cls는 10자리 문자열이어야 합니다"})
			return
		}
		if !save("ranking_exclude_cls", req.RankingExclCls) {
			return
		}
	}

	if req.TakeProfitPct != nil {
		if !save("take_profit_pct", strconv.FormatFloat(*req.TakeProfitPct, 'f', -1, 64)) {
			return
		}
	}
	if req.StopLossPct != nil {
		if !save("stop_loss_pct", strconv.FormatFloat(*req.StopLossPct, 'f', -1, 64)) {
			return
		}
	}
	if req.ETFTakeProfitPct != nil {
		if !save("etf_take_profit_pct", strconv.FormatFloat(*req.ETFTakeProfitPct, 'f', -1, 64)) {
			return
		}
	}
	if req.ETFStopLossPct != nil {
		if !save("etf_stop_loss_pct", strconv.FormatFloat(*req.ETFStopLossPct, 'f', -1, 64)) {
			return
		}
	}
	if req.StockTakeProfitPct != nil {
		if !save("stock_take_profit_pct", strconv.FormatFloat(*req.StockTakeProfitPct, 'f', -1, 64)) {
			return
		}
	}
	if req.StockStopLossPct != nil {
		if !save("stock_stop_loss_pct", strconv.FormatFloat(*req.StockStopLossPct, 'f', -1, 64)) {
			return
		}
	}
	if req.StockTaxRate != nil {
		if !save("stock_tax_rate", strconv.FormatFloat(*req.StockTaxRate, 'f', -1, 64)) {
			return
		}
	}
	if len(req.RankingTypes) > 0 {
		b, _ := json.Marshal(req.RankingTypes)
		if !save("ranking_types", string(b)) {
			return
		}
	}
	if req.RankingPriceMin != "" {
		if !save("ranking_price_min", req.RankingPriceMin) {
			return
		}
	}
	if req.RankingPriceMax != "" {
		if !save("ranking_price_max", req.RankingPriceMax) {
			return
		}
	}
	if req.MaxPositions != nil {
		if !save("max_positions", strconv.Itoa(*req.MaxPositions)) {
			return
		}
	}
	if req.OrderAmountPct != nil {
		if !save("order_amount_pct", strconv.FormatFloat(*req.OrderAmountPct, 'f', -1, 64)) {
			return
		}
	}
	if len(req.SellConditions) > 0 {
		b, _ := json.Marshal(req.SellConditions)
		if !save("sell_conditions", string(b)) {
			return
		}
	}
	if req.IndicatorCheckIntervalMin != nil {
		if !save("indicator_check_interval_min", strconv.Itoa(*req.IndicatorCheckIntervalMin)) {
			return
		}
	}
	if req.IndicatorRSISellThreshold != nil {
		if !save("indicator_rsi_sell_threshold", strconv.FormatFloat(*req.IndicatorRSISellThreshold, 'f', -1, 64)) {
			return
		}
	}
	if req.IndicatorMACDBearishSell != nil {
		val := "false"
		if *req.IndicatorMACDBearishSell {
			val = "true"
		}
		if !save("indicator_macd_bearish_sell", val) {
			return
		}
	}
	if req.RankingVolumeMinIncrRate != nil {
		if !save("ranking_volume_min_incr_rate", strconv.FormatFloat(*req.RankingVolumeMinIncrRate, 'f', -1, 64)) {
			return
		}
	}
	if req.RankingStrengthMin != nil {
		if !save("ranking_strength_min", strconv.FormatFloat(*req.RankingStrengthMin, 'f', -1, 64)) {
			return
		}
	}
	if req.RankingFluctuationMinRate != nil {
		if !save("ranking_fluctuation_min_rate", strconv.FormatFloat(*req.RankingFluctuationMinRate, 'f', -1, 64)) {
			return
		}
	}
	if req.RankingFluctuationMaxRate != nil {
		if !save("ranking_fluctuation_max_rate", strconv.FormatFloat(*req.RankingFluctuationMaxRate, 'f', -1, 64)) {
			return
		}
	}
	if req.RankingVIKindCode != nil {
		if !save("ranking_vi_kind_code", *req.RankingVIKindCode) {
			return
		}
	}
	if req.RankingTopN != nil {
		if !save("ranking_top_n", strconv.Itoa(*req.RankingTopN)) {
			return
		}
	}

	// Validate trading time formats before saving
	if req.TradingStartTime != "" {
		if _, err := time.Parse("15:04", req.TradingStartTime); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "trading_start_time 형식이 잘못되었습니다 (HH:MM)"})
			return
		}
	}
	if req.TradingEndTime != "" {
		if _, err := time.Parse("15:04", req.TradingEndTime); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "trading_end_time 형식이 잘못되었습니다 (HH:MM)"})
			return
		}
	}
	if req.TradingStartTime != "" && req.TradingEndTime != "" {
		startT, _ := time.Parse("15:04", req.TradingStartTime)
		endT, _ := time.Parse("15:04", req.TradingEndTime)
		if !startT.Before(endT) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "거래 시작 시간이 종료 시간보다 앞서야 합니다"})
			return
		}
	}
	if req.TradingStartTime != "" {
		if !save("trading_start_time", req.TradingStartTime) {
			return
		}
	}
	if req.TradingEndTime != "" {
		if !save("trading_end_time", req.TradingEndTime) {
			return
		}
	}
	if req.StagnationThresholdPct != nil {
		if *req.StagnationThresholdPct <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "stagnation_threshold_pct는 0보다 커야 합니다"})
			return
		}
		if !save("stagnation_threshold_pct", strconv.FormatFloat(*req.StagnationThresholdPct, 'f', -1, 64)) {
			return
		}
	}
	if req.StagnationDurationMin != nil {
		if *req.StagnationDurationMin < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "stagnation_duration_min은 1 이상이어야 합니다"})
			return
		}
		if !save("stagnation_duration_min", strconv.Itoa(*req.StagnationDurationMin)) {
			return
		}
	}

	if req.RankingCondition == "AND" || req.RankingCondition == "OR" {
		if !save("ranking_condition", req.RankingCondition) {
			return
		}
	}
	if len(req.RankingExchanges) > 0 {
		b, _ := json.Marshal(req.RankingExchanges)
		if !save("ranking_exchanges", string(b)) {
			return
		}
	}
	if len(req.RankingVolumeBlngClsCodes) > 0 {
		b, _ := json.Marshal(req.RankingVolumeBlngClsCodes)
		if !save("ranking_volume_blng_cls_codes", string(b)) {
			return
		}
	}

	if req.MinTradingValue != nil {
		if *req.MinTradingValue < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "min_trading_value는 0 이상이어야 합니다"})
			return
		}
		if !save("min_trading_value", strconv.FormatFloat(*req.MinTradingValue, 'f', -1, 64)) {
			return
		}
	}

	if req.BuyPauseStart != "" {
		if _, err := time.Parse("15:04", req.BuyPauseStart); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "buy_pause_start 형식이 잘못되었습니다 (HH:MM)"})
			return
		}
		if !save("buy_pause_start", req.BuyPauseStart) {
			return
		}
	}
	if req.BuyPauseEnd != "" {
		if _, err := time.Parse("15:04", req.BuyPauseEnd); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "buy_pause_end 형식이 잘못되었습니다 (HH:MM)"})
			return
		}
		if !save("buy_pause_end", req.BuyPauseEnd) {
			return
		}
	}

	if req.TrailingTriggerPct != nil {
		if *req.TrailingTriggerPct < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "trailing_trigger_pct는 0 이상이어야 합니다"})
			return
		}
		if !save("trailing_trigger_pct", strconv.FormatFloat(*req.TrailingTriggerPct, 'f', -1, 64)) {
			return
		}
	}
	if req.TrailingStopPct != nil {
		if *req.TrailingStopPct < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "trailing_stop_pct는 0 이상이어야 합니다"})
			return
		}
		if !save("trailing_stop_pct", strconv.FormatFloat(*req.TrailingStopPct, 'f', -1, 64)) {
			return
		}
	}

	// 0은 비활성(disabled) 의미이므로 둘 다 활성(> 0)일 때만 순서 검증
	// Tier1 < Tier2 순서 교차 검증 (같은 요청에서 둘 다 제공된 경우)
	if req.TickTier1TriggerPct != nil && req.TickTier2TriggerPct != nil &&
		*req.TickTier1TriggerPct > 0 && *req.TickTier2TriggerPct > 0 &&
		*req.TickTier2TriggerPct <= *req.TickTier1TriggerPct {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tick_tier2_trigger_pct는 tick_tier1_trigger_pct보다 커야 합니다"})
		return
	}

	if req.TrailingMode != nil {
		if *req.TrailingMode != "pct" && *req.TrailingMode != "tick" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "trailing_mode는 'pct' 또는 'tick' 이어야 합니다"})
			return
		}
		if !save("trailing_mode", *req.TrailingMode) {
			return
		}
	}
	if req.TickTier0StopLossTicks != nil {
		if *req.TickTier0StopLossTicks < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tick_tier0_stop_loss_ticks는 0 이상이어야 합니다"})
			return
		}
		if !save("tick_tier0_stop_loss_ticks", strconv.Itoa(*req.TickTier0StopLossTicks)) {
			return
		}
	}
	if req.TickTier1TriggerPct != nil {
		if *req.TickTier1TriggerPct < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tick_tier1_trigger_pct는 0 이상이어야 합니다"})
			return
		}
		if !save("tick_tier1_trigger_pct", strconv.FormatFloat(*req.TickTier1TriggerPct, 'f', -1, 64)) {
			return
		}
	}
	if req.TickTier1TrailTicks != nil {
		if *req.TickTier1TrailTicks < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tick_tier1_trail_ticks는 0 이상이어야 합니다"})
			return
		}
		if !save("tick_tier1_trail_ticks", strconv.Itoa(*req.TickTier1TrailTicks)) {
			return
		}
	}
	if req.TickTier2TriggerPct != nil {
		if *req.TickTier2TriggerPct < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tick_tier2_trigger_pct는 0 이상이어야 합니다"})
			return
		}
		if !save("tick_tier2_trigger_pct", strconv.FormatFloat(*req.TickTier2TriggerPct, 'f', -1, 64)) {
			return
		}
	}
	if req.TickTier2TrailTicks != nil {
		if *req.TickTier2TrailTicks < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tick_tier2_trail_ticks는 0 이상이어야 합니다"})
			return
		}
		if !save("tick_tier2_trail_ticks", strconv.Itoa(*req.TickTier2TrailTicks)) {
			return
		}
	}
	if req.DailyMaxLossPct != nil {
		if *req.DailyMaxLossPct < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "daily_max_loss_pct는 0 이상이어야 합니다"})
			return
		}
		if !save("daily_max_loss_pct", strconv.FormatFloat(*req.DailyMaxLossPct, 'f', -1, 64)) {
			return
		}
	}

	if req.IndexCodes != nil {
		b, _ := json.Marshal(req.IndexCodes)
		if !save("index_codes", string(b)) {
			return
		}
	}

	if req.FilterRsiMax != nil {
		if !save("filter_rsi_max", strconv.FormatFloat(*req.FilterRsiMax, 'f', -1, 64)) {
			return
		}
	}
	if req.FilterDisparityM5Max != nil {
		if !save("filter_disparity_m5_max", strconv.FormatFloat(*req.FilterDisparityM5Max, 'f', -1, 64)) {
			return
		}
	}
	if req.FilterHighPriceDiffMin != nil {
		if !save("filter_high_price_diff_min", strconv.FormatFloat(*req.FilterHighPriceDiffMin, 'f', -1, 64)) {
			return
		}
	}
	if req.FilterOpenPriceDiffMax != nil {
		if !save("filter_open_price_diff_max", strconv.FormatFloat(*req.FilterOpenPriceDiffMax, 'f', -1, 64)) {
			return
		}
	}
	if req.IndexDropThresholdPct != nil {
		if !save("index_drop_threshold_pct", strconv.FormatFloat(*req.IndexDropThresholdPct, 'f', -1, 64)) {
			return
		}
	}

	// 요일 스케줄 (trading_days)
	if req.TradingDays != nil {
		b, _ := json.Marshal(req.TradingDays)
		if !save("trading_days", string(b)) {
			return
		}
	}

	// AI 매매 기준값 — 하드 리젝션 룰
	if req.HardDisparityM5Min != nil {
		if !save("hard_disparity_m5_min", strconv.FormatFloat(*req.HardDisparityM5Min, 'f', -1, 64)) {
			return
		}
	}
	if req.HardDisparityM5Max != nil {
		if !save("hard_disparity_m5_max", strconv.FormatFloat(*req.HardDisparityM5Max, 'f', -1, 64)) {
			return
		}
	}
	if req.HardHighPriceDiffMax != nil {
		if !save("hard_high_price_diff_max", strconv.FormatFloat(*req.HardHighPriceDiffMax, 'f', -1, 64)) {
			return
		}
	}
	if req.HardHighPriceDiffMin != nil {
		if !save("hard_high_price_diff_min", strconv.FormatFloat(*req.HardHighPriceDiffMin, 'f', -1, 64)) {
			return
		}
	}
	if req.HardPrevVolRatioMax != nil {
		if !save("hard_prev_vol_ratio_max", strconv.FormatFloat(*req.HardPrevVolRatioMax, 'f', -1, 64)) {
			return
		}
	}
	if req.HardStrengthMin != nil {
		if !save("hard_strength_min", strconv.FormatFloat(*req.HardStrengthMin, 'f', -1, 64)) {
			return
		}
	}
	if req.HardRSIMax != nil {
		if !save("hard_rsi_max", strconv.FormatFloat(*req.HardRSIMax, 'f', -1, 64)) {
			return
		}
	}
	if req.HardOpenPriceDiffMax != nil {
		if !save("hard_open_price_diff_max", strconv.FormatFloat(*req.HardOpenPriceDiffMax, 'f', -1, 64)) {
			return
		}
	}
	if req.HardMACDBearishEnabled != nil {
		v := "false"
		if *req.HardMACDBearishEnabled {
			v = "true"
		}
		if !save("hard_macd_bearish_enabled", v) {
			return
		}
	}
	if req.HardMA60SupportEnabled != nil {
		v := "false"
		if *req.HardMA60SupportEnabled {
			v = "true"
		}
		if !save("hard_ma60_support_enabled", v) {
			return
		}
	}
	if req.HardMA120SupportEnabled != nil {
		v := "false"
		if *req.HardMA120SupportEnabled {
			v = "true"
		}
		if !save("hard_ma120_support_enabled", v) {
			return
		}
	}
	if req.HardHighFormedMinsMax != nil {
		if !save("hard_high_formed_mins_max", strconv.FormatFloat(*req.HardHighFormedMinsMax, 'f', -1, 64)) {
			return
		}
	}
	if req.HardVolVs3AvgRatioMin != nil {
		if !save("hard_vol_vs3_avg_ratio_min", strconv.FormatFloat(*req.HardVolVs3AvgRatioMin, 'f', -1, 64)) {
			return
		}
	}
	if req.HardRelativeStrengthMin != nil {
		if !save("hard_relative_strength_min", strconv.FormatFloat(*req.HardRelativeStrengthMin, 'f', -1, 64)) {
			return
		}
	}
	// AI 매매 기준값 — 랭킹 기준
	if req.VWAPDiffMin != nil {
		if !save("vwap_diff_min", strconv.FormatFloat(*req.VWAPDiffMin, 'f', -1, 64)) {
			return
		}
	}
	if req.VWAPDiffMax != nil {
		if !save("vwap_diff_max", strconv.FormatFloat(*req.VWAPDiffMax, 'f', -1, 64)) {
			return
		}
	}
	if req.RSIBuyMin != nil {
		if !save("rsi_buy_min", strconv.FormatFloat(*req.RSIBuyMin, 'f', -1, 64)) {
			return
		}
	}
	if req.RSIBuyMax != nil {
		if !save("rsi_buy_max", strconv.FormatFloat(*req.RSIBuyMax, 'f', -1, 64)) {
			return
		}
	}
	if req.BidAskRatioMin != nil {
		if !save("bid_ask_ratio_min", strconv.FormatFloat(*req.BidAskRatioMin, 'f', -1, 64)) {
			return
		}
	}
	if req.MinMarketCap != nil {
		if !save("min_market_cap", strconv.FormatFloat(*req.MinMarketCap, 'f', -1, 64)) {
			return
		}
	}
	if req.MinExpectedProfitPct != nil {
		if !save("min_expected_profit_pct", strconv.FormatFloat(*req.MinExpectedProfitPct, 'f', -1, 64)) {
			return
		}
	}
	if req.StagnationPartialExitEnabled != nil {
		val := "false"
		if *req.StagnationPartialExitEnabled {
			val = "true"
		}
		if !save("stagnation_partial_exit_enabled", val) {
			return
		}
	}
	if req.StagnationBidAskSellThreshold != nil {
		if !save("stagnation_bidask_sell_threshold", strconv.FormatFloat(*req.StagnationBidAskSellThreshold, 'f', -1, 64)) {
			return
		}
	}
	if req.PartialTPEnabled != nil {
		val := "false"
		if *req.PartialTPEnabled {
			val = "true"
		}
		if !save("partial_tp_enabled", val) {
			return
		}
	}
	if req.PartialTPPct != nil {
		if !save("partial_tp_pct", strconv.FormatFloat(*req.PartialTPPct, 'f', -1, 64)) {
			return
		}
	}
	if req.PartialTPRatio != nil {
		if *req.PartialTPRatio <= 0 || *req.PartialTPRatio >= 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "partial_tp_ratio는 0보다 크고 1보다 작아야 합니다"})
			return
		}
		if !save("partial_tp_ratio", strconv.FormatFloat(*req.PartialTPRatio, 'f', -1, 64)) {
			return
		}
	}
	if req.PartialTPRaiseStop != nil {
		val := "false"
		if *req.PartialTPRaiseStop {
			val = "true"
		}
		if !save("partial_tp_raise_stop", val) {
			return
		}
	}
	if req.HardWatchSymbols != nil {
		b, _ := json.Marshal(req.HardWatchSymbols)
		if !save("hard_watch_symbols", string(b)) {
			return
		}
	}
	if req.RankLeaseDurationMin != nil {
		if *req.RankLeaseDurationMin < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "rank_lease_duration_min은 0 이상이어야 합니다"})
			return
		}
		if !save("rank_lease_duration_min", strconv.Itoa(*req.RankLeaseDurationMin)) {
			return
		}
	}

	// UI 거래조건 추가 필드
	if req.DailyLossLimitPct != nil {
		if !save("daily_max_loss_pct", strconv.FormatFloat(*req.DailyLossLimitPct, 'f', -1, 64)) {
			return
		}
	}
	if req.IndicatorRSISellEnabled != nil {
		v := "false"
		if *req.IndicatorRSISellEnabled {
			v = "true"
		}
		if !save("indicator_rsi_sell_enabled", v) {
			return
		}
	}
	if req.MinScore != nil {
		if !save("min_score_threshold", strconv.FormatFloat(*req.MinScore, 'f', -1, 64)) {
			return
		}
	}
	if req.MinScoreThreshold != nil {
		if *req.MinScoreThreshold < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "min_score_threshold는 0 이상이어야 합니다"})
			return
		}
		if !save("min_score_threshold", strconv.FormatFloat(*req.MinScoreThreshold, 'f', -1, 64)) {
			return
		}
	}
	if req.UniversalCooldownMin != nil {
		if *req.UniversalCooldownMin < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "universal_cooldown_min은 0 이상이어야 합니다"})
			return
		}
		if !save("universal_cooldown_min", strconv.Itoa(*req.UniversalCooldownMin)) {
			return
		}
	}
	// 재진입 / 손절 제어
	if req.SellOnUpperLimit != nil {
		v := "false"
		if *req.SellOnUpperLimit {
			v = "true"
		}
		if !save("sell_on_upper_limit", v) {
			return
		}
	}
	if req.MaxConsecutiveLosses != nil {
		if !save("max_consecutive_losses", strconv.Itoa(*req.MaxConsecutiveLosses)) {
			return
		}
	}
	if req.ConsecutiveLossResetOnProfit != nil {
		v := "false"
		if *req.ConsecutiveLossResetOnProfit {
			v = "true"
		}
		if !save("consecutive_loss_reset_on_profit", v) {
			return
		}
	}
	if req.MaxBidAskSpreadPct != nil {
		if !save("max_bidask_spread_pct", strconv.FormatFloat(*req.MaxBidAskSpreadPct, 'f', -1, 64)) {
			return
		}
	}
	if req.BlockReentryOnLoss != nil {
		v := "false"
		if *req.BlockReentryOnLoss {
			v = "true"
		}
		if !save("block_reentry_on_loss", v) {
			return
		}
	}
	if req.ReentryScorePenalty != nil {
		if !save("reentry_score_penalty", strconv.FormatFloat(*req.ReentryScorePenalty, 'f', -1, 64)) {
			return
		}
	}
	if req.ReentryCooldownMin != nil {
		if !save("reentry_cooldown_min", strconv.Itoa(*req.ReentryCooldownMin)) {
			return
		}
	}
	if req.LossCooldownMin != nil {
		if !save("loss_cooldown_min", strconv.Itoa(*req.LossCooldownMin)) {
			return
		}
	}
	if req.LossReentryPriceGuard != nil {
		v := "false"
		if *req.LossReentryPriceGuard {
			v = "true"
		}
		if !save("loss_reentry_price_guard", v) {
			return
		}
	}
	if req.HardPeakTurnEnabled != nil {
		v := "false"
		if *req.HardPeakTurnEnabled {
			v = "true"
		}
		if !save("hard_peak_turn_enabled", v) {
			return
		}
	}
	if req.HardPeakRSIMin != nil {
		if !save("hard_peak_rsi_min", strconv.FormatFloat(*req.HardPeakRSIMin, 'f', -1, 64)) {
			return
		}
	}
	if req.BuyOrderType != nil {
		if !save("buy_order_type", *req.BuyOrderType) {
			return
		}
	}
	if req.StreamBypassEnabled != nil {
		v := "false"
		if *req.StreamBypassEnabled {
			v = "true"
		}
		if !save("stream_bypass_enabled", v) {
			return
		}
	}
	if req.StreamBigTradeAmount != nil {
		if !save("stream_big_trade_amount", strconv.FormatFloat(*req.StreamBigTradeAmount, 'f', -1, 64)) {
			return
		}
	}
	if req.StreamVelocityThreshold != nil {
		if !save("stream_velocity_threshold", strconv.FormatFloat(*req.StreamVelocityThreshold, 'f', -1, 64)) {
			return
		}
	}

	scoreWeights := []struct {
		ptr *int
		key string
	}{
		{req.ScoreWeightStrength, "score_weight_strength"},
		{req.ScoreWeightRSI, "score_weight_rsi"},
		{req.ScoreWeightMACD, "score_weight_macd"},
		{req.ScoreWeightBidAsk, "score_weight_bidask"},
		{req.ScoreWeightVWAP, "score_weight_vwap"},
		{req.ScoreWeightVolume, "score_weight_volume"},
		{req.ScoreWeightProgramBuy, "score_weight_program_buy"},
		{req.ScoreWeightMicroBidAsk, "score_weight_micro_bidask"},
		{req.ScoreWeightVIDisparity, "score_weight_vi_disparity"},
	}
	for _, weight := range scoreWeights {
		if weight.ptr != nil {
			if *weight.ptr < 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": weight.key + "는 0 이상이어야 합니다"})
				return
			}
			if !save(weight.key, strconv.Itoa(*weight.ptr)) {
				return
			}
		}
	}

	// Weights
	if req.Weights != nil {
		w := req.Weights
		weightMap := map[string]float64{
			"strength": w.Strength,
			"rsi":      w.RSI,
			"macd":     w.MACD,
			"bidask":   w.BidAsk,
			"vwap":     w.VWAP,
			"volume":   w.Volume,
		}
		for k, v := range weightMap {
			if !save("weight_"+k, strconv.FormatFloat(v, 'f', -1, 64)) {
				return
			}
		}
	}

	// Filters
	for fKey, fVal := range req.Filters {
		v := "false"
		if fVal.Enabled {
			v = "true"
		}
		if !save("filter_"+fKey+"_enabled", v) {
			return
		}
		if !save("filter_"+fKey+"_value", strconv.FormatFloat(fVal.Value, 'f', -1, 64)) {
			return
		}
	}

	// Schedule
	if req.Schedule != nil {
		s := req.Schedule
		if s.TradeStart != "" {
			if _, err := time.Parse("15:04", s.TradeStart); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "schedule.trade_start 형식이 잘못되었습니다 (HH:MM)"})
				return
			}
			if !save("trading_start_time", s.TradeStart) {
				return
			}
		}
		if s.TradeEnd != "" {
			if _, err := time.Parse("15:04", s.TradeEnd); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "schedule.trade_end 형식이 잘못되었습니다 (HH:MM)"})
				return
			}
			if !save("trading_end_time", s.TradeEnd) {
				return
			}
		}
		if s.ScanInterval > 0 {
			if !save("scan_interval", strconv.Itoa(s.ScanInterval)) {
				return
			}
		}
		if s.IndicatorCheckInterval > 0 {
			if !save("indicator_check_interval_min", strconv.Itoa(s.IndicatorCheckInterval)) {
				return
			}
		}
	}

	if !changed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "변경할 항목이 없습니다"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "설정이 저장되었습니다."})
}

// GET /api/orders/feasibility?code=:code — 주문가능수량 및 주문가능금액 조회 (TTTC8908R)
// qty > 0 이면 주문 가능. qty == 0 이면 available_cash 기준으로 종목 재선정.
func (h *Handler) GetFeasibility(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code query param is required"})
		return
	}
	result, err := ops.CheckOrderFeasibility(c.Request.Context(), h.client, code)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"orderable_qty":  result.OrderableQty,
		"available_cash": result.AvailableCash,
	})
}

// resolvePriceFilter returns priceMin/priceMax for ranking API calls.
// use_balance_filter=true: 잔액 API(TTTC8434R)로 예수금 조회 후 priceMax로 자동 설정.
// price_min/price_max: 직접 입력값 (use_balance_filter가 true이면 무시됨).
// 잔액 조회 실패 또는 예수금=0이면 필터 미적용(빈값 반환).
func (h *Handler) resolvePriceFilter(c *gin.Context) (priceMin, priceMax string) {
	if c.Query("use_balance_filter") == "true" {
		summary, err := h.client.GetInquireBalance(c.Request.Context())
		if err == nil && summary.DepositAmt != "" && summary.DepositAmt != "0" {
			return "", summary.DepositAmt
		}
		return "", ""
	}
	return c.Query("price_min"), c.Query("price_max")
}

// GET /api/ranking/volume?market=J&input_iscd=0000&sort=0 — 거래량 순위 (FHPST01710000, max 30)
// input_iscd: 0000=전체(default), 0001=KOSPI, 1001=KOSDAQ, 2001=KOSPI200
// sort (FID_BLNG_CLS_CODE): 0=평균거래량(default), 1=거래량증가율, 2=거래회전율, 3=거래대금순, 4=평균거래대금
// price_min/price_max: 가격 범위 직접 입력 (원). use_balance_filter=true: 예수금 기준 자동 설정.
// ETF/ETN/우선주 등 비정상 종목은 항상 제외됨.
func (h *Handler) GetVolumeRank(c *gin.Context) {
	market := c.DefaultQuery("market", "J")
	inputIscd := c.DefaultQuery("input_iscd", "0000")
	sort := c.DefaultQuery("sort", "0")
	priceMin, priceMax := h.resolvePriceFilter(c)
	excludeCls := h.db.GetSetting(c.Request.Context(), "ranking_exclude_cls")
	items, err := ops.GetVolumeRank(c.Request.Context(), h.client, market, inputIscd, sort, priceMin, priceMax, excludeCls)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ranking": items})
}

// GET /api/ranking/strength?market=0000 — 체결강도 상위 (FHPST01680000, max 30)
// market: 0000=전체(default), 0001=거래소, 1001=코스닥, 2001=코스피200
// price_min/price_max: 가격 범위 직접 입력 (원). use_balance_filter=true: 예수금 기준 자동 설정.
// ETF/ETN/우선주 등 비정상 종목은 항상 제외 시도.
func (h *Handler) GetStrengthRank(c *gin.Context) {
	market := c.DefaultQuery("market", "0000")
	priceMin, priceMax := h.resolvePriceFilter(c)
	excludeCls := h.db.GetSetting(c.Request.Context(), "ranking_exclude_cls")
	items, err := ops.GetStrengthRank(c.Request.Context(), h.client, market, priceMin, priceMax, excludeCls)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ranking": items})
}

// GET /api/ranking/fluctuation?market=0000 — 등락률 상위 (FHPST01700000, max 30)
// market: 0000=전체(default), 0001=거래소(KOSPI), 1001=코스닥
// price_min/price_max: 가격 범위 직접 입력 (원). use_balance_filter=true: 예수금 기준 자동 설정.
// ETF/ETN/우선주 등 비정상 종목은 항상 제외 시도.
func (h *Handler) GetFluctuationRank(c *gin.Context) {
	market := c.DefaultQuery("market", "0000")
	priceMin, priceMax := h.resolvePriceFilter(c)
	excludeCls := h.db.GetSetting(c.Request.Context(), "ranking_exclude_cls")
	items, err := ops.GetFluctuationRank(c.Request.Context(), h.client, market, priceMin, priceMax, excludeCls)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ranking": items})
}

// GET /api/ranking/vi-status — VI 발동현황 (FHPST01390000)
// date: YYYYMMDD (default: 오늘). vi_cncl_hour가 비어있는 미해제 건은 제외하여 반환.
func (h *Handler) GetVIStatus(c *gin.Context) {
	kst, _ := time.LoadLocation("Asia/Seoul")
	date := c.DefaultQuery("date", time.Now().In(kst).Format("20060102"))
	items, err := ops.GetVIStatus(c.Request.Context(), h.client, date)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	// 미해제(vi_cncl_hour=="") 건 제외
	released := items[:0]
	for _, item := range items {
		if item.ViCnclHour != "" {
			released = append(released, item)
		}
	}
	c.JSON(http.StatusOK, gin.H{"ranking": released})
}

// GET /api/market/status — 현재 장운영 여부 조회
// Response: { "is_open": bool, "checked_at": RFC3339, "reason": "open"|"weekend"|"outside_hours"|"holiday"|"check_failed" }
func (h *Handler) GetMarketStatus(c *gin.Context) {
	now := time.Now().In(ops.KSTLocation())
	checkedAt := now.Format(time.RFC3339)

	if wd := now.Weekday(); wd == time.Saturday || wd == time.Sunday {
		c.JSON(http.StatusOK, gin.H{"is_open": false, "checked_at": checkedAt, "reason": "weekend"})
		return
	}

	openMinute := now.Hour()*60 + now.Minute()
	if openMinute < 9*60 || openMinute >= 15*60+30 {
		c.JSON(http.StatusOK, gin.H{"is_open": false, "checked_at": checkedAt, "reason": "outside_hours"})
		return
	}

	isOpen, err := ops.IsMarketOpen(c.Request.Context(), h.client)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"is_open": false, "checked_at": checkedAt, "reason": "check_failed"})
		return
	}
	if !isOpen {
		c.JSON(http.StatusOK, gin.H{"is_open": false, "checked_at": checkedAt, "reason": "holiday"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"is_open": true, "checked_at": checkedAt, "reason": "open"})
}

// POST /api/ws/connect — 수동 WebSocket 연결 (테스트·장애 복구용)
// 1) approval_key 발급 → 2) StartWithReconnect → 3) ResubscribeAll
func (h *Handler) ConnectWebSocket(c *gin.Context) {
	if h.wsClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "WebSocket client not initialized (KIS credentials missing)"})
		return
	}
	if h.wsClient.IsConnected() {
		c.JSON(http.StatusOK, gin.H{"message": "already connected"})
		return
	}

	ctx := c.Request.Context()

	if _, err := h.tokenManager.EnsureToken(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "토큰 발급 실패 (KIS AppKey/Secret 확인 필요): " + err.Error()})
		return
	}

	approvalKey, err := h.client.GetApprovalKey(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "GetApprovalKey failed: " + err.Error()})
		return
	}

	// 기존 reconnect 고루틴이 있으면 먼저 중단 후 재시작
	ctx2, cancel := context.WithCancel(context.Background())
	h.wsClient.SetReconnectCancel(cancel)
	h.wsClient.SetApprovalKey(approvalKey)
	go h.wsClient.StartWithReconnect(ctx2)

	// 최대 4초 대기하며 실제 연결 여부 확인 (500ms 간격)
	connected := false
	for i := 0; i < 8; i++ {
		time.Sleep(500 * time.Millisecond)
		if h.wsClient.IsConnected() {
			connected = true
			break
		}
	}

	if !connected {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "WebSocket 연결 실패 — KIS 서버가 연결을 거부했습니다. 로그를 확인하세요."})
		return
	}

	if err := h.wsClient.SubscribeExecNotice(); err != nil {
		// non-fatal: log only
		_ = err
	}
	// 이미 매도된 종목을 monitored_positions에서 제거 후 재구독.
	h.monitor.PurgeStalePositions(c.Request.Context())
	h.monitor.ResubscribeAll()

	c.JSON(http.StatusOK, gin.H{"message": "WebSocket connected", "subscribed": h.monitor.Count()})
}

// POST /api/ws/disconnect — 수동 WebSocket 해제
func (h *Handler) DisconnectWebSocket(c *gin.Context) {
	if h.wsClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "WebSocket client not initialized"})
		return
	}
	h.wsClient.Disconnect()
	c.JSON(http.StatusOK, gin.H{"message": "WebSocket disconnected"})
}

// GET /api/stats/daily-pnl?days=7|30
// 최근 N일간 일별 실현 손익을 반환합니다 (1 ≤ days ≤ 365, default 30).
func (h *Handler) GetDailyPnL(c *gin.Context) {
	days, err := strconv.Atoi(c.DefaultQuery("days", "30"))
	if err != nil || days < 1 || days > 365 {
		days = 30
	}
	data, dbErr := h.db.GetDailyPnL(c.Request.Context(), days)
	if dbErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": dbErr.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"days": days, "data": data})
}

// GET /api/stocks?q=&etf_only=&market= — 종목 마스터 검색
// q: 종목명/코드 검색어 (부분 일치)
// etf_only: "true" 이면 ETF만 반환
// market: "KOSPI" 또는 "KOSDAQ" (빈값이면 전체)
func (h *Handler) GetStocks(c *gin.Context) {
	ctx := c.Request.Context()
	if h.mstStore == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "종목 마스터가 초기화되지 않았습니다."})
		return
	}

	q := c.Query("q")
	etfOnly := c.Query("etf_only") == "true"
	market := c.Query("market")

	// Hard Watch Symbols 목록 조회 (hard_watch 여부 표시용)
	settings, _ := h.db.GetTradingSettings(ctx)
	hardSet := make(map[string]bool, len(settings.HardWatchSymbols))
	for _, code := range settings.HardWatchSymbols {
		hardSet[code] = true
	}

	type StockItem struct {
		StockCode           string `json:"stock_code"`
		StockName           string `json:"stock_name"`
		MarketType          string `json:"market_type"`
		IsETF               bool   `json:"is_etf"`
		IsDomesticEquityETF bool   `json:"is_domestic_equity_etf"`
		IsHardWatch         bool   `json:"is_hard_watch"`
	}

	items, err := h.mstStore.Search(ctx, q, etfOnly, market, 200)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := make([]StockItem, 0, len(items))
	for _, s := range items {
		result = append(result, StockItem{
			StockCode:           s.StockCode,
			StockName:           s.StockName,
			MarketType:          s.MarketType,
			IsETF:               s.IsETF,
			IsDomesticEquityETF: s.IsDomesticEquityETF,
			IsHardWatch:         hardSet[s.StockCode],
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": result, "total": len(result)})
}

// GET /api/reports/trades?date=YYYY-MM-DD&stock_code=XXXXXX&page=1&limit=20
func (h *Handler) GetTradeReports(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	reports, err := h.db.ListTradeReports(c.Request.Context(), limit*5) // fetch more for grouping
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if reports == nil {
		reports = []models.TradeReport{}
	}

	// 날짜/종목코드 필터 (클라이언트 사이드)
	date := c.Query("date")
	stockCode := c.Query("stock_code")
	if date != "" || stockCode != "" {
		filtered := reports[:0]
		for _, r := range reports {
			if date != "" && r.Date != date {
				continue
			}
			if stockCode != "" && r.StockCode != stockCode {
				continue
			}
			filtered = append(filtered, r)
		}
		reports = filtered
	}

	kst := ops.KSTLocation()
	// Group by date
	type tradeView struct {
		ID         int64          `json:"id"`
		StockCode  string         `json:"stock_code"`
		StockName  string         `json:"stock_name"`
		BuyPrice   float64        `json:"buy_price"`
		SellPrice  float64        `json:"sell_price"`
		PnlAmount  float64        `json:"pnl_amount"`
		PnlPct     float64        `json:"pnl_pct"`
		SellReason string         `json:"sell_reason"`
		BuyTime    string         `json:"buy_time"`
		SellTime   string         `json:"sell_time"`
		HoldPeriod string         `json:"hold_period"`
		Indicators map[string]any `json:"indicators"`
	}
	type groupView struct {
		Date   string      `json:"date"`
		DayPnl float64     `json:"day_pnl"`
		Trades []tradeView `json:"trades"`
	}

	dateOrder := []string{}
	dateMap := map[string]*groupView{}
	for _, r := range reports {
		g, ok := dateMap[r.Date]
		if !ok {
			g = &groupView{Date: r.Date}
			dateMap[r.Date] = g
			dateOrder = append(dateOrder, r.Date)
		}
		g.DayPnl += r.ProfitAmount

		buyTime := ""
		if !r.CreatedAt.IsZero() {
			buyTime = r.CreatedAt.In(kst).Format("15:04")
		}
		sellTime := ""
		if r.SoldAt != nil {
			sellTime = r.SoldAt.In(kst).Format("15:04")
		}
		holdPeriod := ""
		if r.SoldAt != nil && !r.CreatedAt.IsZero() {
			dur := r.SoldAt.Sub(r.CreatedAt)
			h := int(dur.Hours())
			m := int(dur.Minutes()) % 60
			if h > 0 {
				holdPeriod = strconv.Itoa(h) + "시간 " + strconv.Itoa(m) + "분"
			} else {
				holdPeriod = strconv.Itoa(m) + "분"
			}
		}

		// Parse buy_indicators JSON
		var indicators map[string]any
		if r.BuyIndicators != "" {
			_ = json.Unmarshal([]byte(r.BuyIndicators), &indicators)
		}

		g.Trades = append(g.Trades, tradeView{
			ID:         r.ID,
			StockCode:  r.StockCode,
			StockName:  r.StockName,
			BuyPrice:   r.BuyPrice,
			SellPrice:  r.SellPrice,
			PnlAmount:  r.ProfitAmount,
			PnlPct:     r.ProfitPct,
			SellReason: r.SellReason,
			BuyTime:    buyTime,
			SellTime:   sellTime,
			HoldPeriod: holdPeriod,
			Indicators: indicators,
		})
	}

	groups := make([]groupView, 0, len(dateOrder))
	for _, d := range dateOrder {
		groups = append(groups, *dateMap[d])
	}

	c.JSON(http.StatusOK, gin.H{"data": groups})
}

// GET /api/reports/daily?from=YYYY-MM-DD&to=YYYY-MM-DD&limit=30
func (h *Handler) GetDailyReports(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit < 1 || limit > 365 {
		limit = 50
	}

	reports, err := h.db.ListDailyReports(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if reports == nil {
		reports = []models.DailyReport{}
	}

	// from/to 범위 필터 (클라이언트 사이드)
	from := c.Query("from")
	to := c.Query("to")
	if from != "" || to != "" {
		filtered := reports[:0]
		for _, r := range reports {
			if from != "" && r.Date < from {
				continue
			}
			if to != "" && r.Date > to {
				continue
			}
			filtered = append(filtered, r)
		}
		reports = filtered
	}

	type dailyView struct {
		Date          string  `json:"date"`
		TradeCount    int     `json:"trade_count"`
		Wins          int     `json:"wins"`
		Losses        int     `json:"losses"`
		Pnl           float64 `json:"pnl"`
		PnlPct        float64 `json:"pnl_pct"`
		WinRate       float64 `json:"win_rate"`
		ReportSummary string  `json:"report_summary"`
	}
	views := make([]dailyView, len(reports))
	for i, r := range reports {
		winRate := 0.0
		if r.TotalTrades > 0 {
			winRate = float64(r.WinningTrades) / float64(r.TotalTrades) * 100
		}
		views[i] = dailyView{
			Date:          r.Date,
			TradeCount:    r.TotalTrades,
			Wins:          r.WinningTrades,
			Losses:        r.LosingTrades,
			Pnl:           r.TotalProfitAmount,
			PnlPct:        r.AvgProfitPct,
			WinRate:       winRate,
			ReportSummary: r.TradeSummary,
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": views})
}

// POST /api/reports/daily/generate?date=YYYY-MM-DD
func (h *Handler) GenerateDailyReport(c *gin.Context) {
	date := c.Query("date")
	if err := report.GenerateDailyReport(c.Request.Context(), h.db, date); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// HandleExportReport handles GET /api/reports/export?from=YYYY-MM-DD&to=YYYY-MM-DD.
func (h *Handler) HandleExportReport(c *gin.Context) {
	from := c.Query("from")
	to := c.Query("to")
	if from == "" || to == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from and to query params required (YYYY-MM-DD)"})
		return
	}

	fromT, err := time.Parse("2006-01-02", from)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from date"})
		return
	}
	toT, err := time.Parse("2006-01-02", to)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to date"})
		return
	}
	if toT.Sub(fromT) > 90*24*time.Hour {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date range must be 90 days or less"})
		return
	}

	result, err := report.GenerateExportReport(c.Request.Context(), h.db, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// HandleGetSimulationResult handles GET /api/simulation/:date.
func (h *Handler) HandleGetSimulationResult(c *gin.Context) {
	date := c.Param("date")
	if date == "" {
		kst, _ := time.LoadLocation("Asia/Seoul")
		date = time.Now().In(kst).Format("2006-01-02")
	}
	result, err := h.db.GetSimulationResult(c.Request.Context(), date)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "simulation result not found for " + date})
		return
	}

	var scenarios []simulation.ScenarioSummary
	var recommended simulation.RecommendedSettings
	_ = json.Unmarshal([]byte(result.ScenariosJSON), &scenarios)
	_ = json.Unmarshal([]byte(result.RecommendedJSON), &recommended)

	c.JSON(http.StatusOK, gin.H{
		"date":        result.Date,
		"scenarios":   scenarios,
		"recommended": recommended,
		"created_at":  result.CreatedAt,
	})
}

// HandleRunSimulation handles POST /api/simulation/run?date=YYYY-MM-DD.
func (h *Handler) HandleRunSimulation(c *gin.Context) {
	date := c.Query("date")
	if date == "" {
		kst, _ := time.LoadLocation("Asia/Seoul")
		date = time.Now().In(kst).Format("2006-01-02")
	}
	go func() {
		if err := simulation.RunDailySimulation(context.Background(), h.db, h.client, date); err != nil {
			log.Printf("[api] manual simulation failed: %v", err)
		}
	}()
	c.JSON(http.StatusAccepted, gin.H{"message": "simulation started for " + date})
}
