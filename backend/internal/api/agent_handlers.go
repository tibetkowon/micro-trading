package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/micro-trading-for-agent/backend/internal/ops"
)

func (h *Handler) AgentGetSettings(c *gin.Context) {
	settings, err := h.db.GetAllSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"settings": settings})
}

func (h *Handler) AgentUpdateSetting(c *gin.Context) {
	var req struct {
		Key   string `json:"key" binding:"required"`
		Value string `json:"value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.db.SetSetting(c.Request.Context(), req.Key, req.Value); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"updated": req.Key, "value": req.Value})
}

func (h *Handler) AgentGetPositions(c *gin.Context) {
	if h.monitor == nil {
		c.JSON(http.StatusOK, gin.H{"positions": []gin.H{}, "count": 0})
		return
	}
	positions := h.monitor.GetPositions()
	result := make([]gin.H, 0, len(positions))
	for _, p := range positions {
		profitPct := 0.0
		profitAmount := 0.0
		if p.FilledPrice > 0 {
			profitPct = (p.CurrentPrice - p.FilledPrice) / p.FilledPrice * 100
			profitAmount = (p.CurrentPrice - p.FilledPrice) * float64(p.Qty)
		}
		result = append(result, gin.H{
			"code":          p.StockCode,
			"name":          p.StockName,
			"qty":           p.Qty,
			"buy_price":     p.FilledPrice,
			"current_price": p.CurrentPrice,
			"target_price":  p.TargetPrice,
			"stop_price":    p.StopPrice,
			"profit_pct":    profitPct,
			"profit_amount": profitAmount,
		})
	}
	c.JSON(http.StatusOK, gin.H{"positions": result, "count": len(result)})
}

func (h *Handler) AgentSellPosition(c *gin.Context) {
	if h.monitor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "monitor unavailable"})
		return
	}
	code := c.Param("code")
	qty, err := h.monitor.ForceSell(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"sold": code, "qty": qty})
}

func (h *Handler) AgentLiquidateAll(c *gin.Context) {
	if h.monitor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "monitor unavailable"})
		return
	}
	h.monitor.LiquidateAll(c.Request.Context(), "")
	c.JSON(http.StatusOK, gin.H{"liquidated": true})
}

func (h *Handler) AgentGetStatus(c *gin.Context) {
	h.GetServerStatus(c)
}

func (h *Handler) AgentGetStats(c *gin.Context) {
	today := time.Now().In(ops.KSTLocation()).Format("2006-01-02")
	report, err := h.db.GetDailyReport(c.Request.Context(), today)
	if err != nil || report == nil {
		c.JSON(http.StatusOK, gin.H{"date": today, "no_data": true})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"date":                report.Date,
		"total_trades":        report.TotalTrades,
		"winning_trades":      report.WinningTrades,
		"losing_trades":       report.LosingTrades,
		"total_profit_amount": report.TotalProfitAmount,
		"avg_profit_pct":      report.AvgProfitPct,
	})
}
