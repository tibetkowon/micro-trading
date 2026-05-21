package api

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// SetupRouter registers all API routes and returns a configured gin.Engine.
// frontendDist is the path to the React build output (dist/) directory.
func SetupRouter(h *Handler, frontendDist string) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(jsonLogger())
	r.Use(corsMiddleware(h.cfg.FrontendOrigin))

	api := r.Group("/api")
	{
		api.GET("/balance", h.GetBalance)
		api.GET("/positions", h.GetPositions)
		api.GET("/stock/:code", h.GetStock)
		api.GET("/stock/:code/chart", h.GetStockChart)
		api.GET("/orders", h.GetOrders)
		api.POST("/orders", h.PlaceOrder)
		api.POST("/orders/:id/cancel", h.CancelOrder)
		api.DELETE("/orders/:id", h.DeleteOrder)
		api.GET("/orders/feasibility", h.GetFeasibility)
		api.GET("/logs/scan", h.GetScanLogs)
		api.GET("/logs/scan/:id/raw", h.GetScanLogRaw)
		api.GET("/settings", h.GetSettings)
		api.PATCH("/settings", h.UpdateSettings)

		api.GET("/server/status", h.GetServerStatus)

		monitorGroup := api.Group("/monitor")
		{
			monitorGroup.GET("/positions", h.GetMonitorPositions)
			monitorGroup.DELETE("/positions/:code", h.RemoveMonitorPosition)
			monitorGroup.POST("/positions/:code/sell", h.ForceSellMonitorPosition)
			monitorGroup.POST("/liquidate-all", h.LiquidateAll)
		}

		market := api.Group("/market")
		{
			market.GET("/status", h.GetMarketStatus)
		}

		api.GET("/stocks", h.GetStocks)

		ranking := api.Group("/ranking")
		{
			ranking.GET("/volume", h.GetVolumeRank)
			ranking.GET("/strength", h.GetStrengthRank)
			ranking.GET("/fluctuation", h.GetFluctuationRank)
			ranking.GET("/vi-status", h.GetVIStatus)
		}

		ws := api.Group("/ws")
		{
			ws.POST("/connect", h.ConnectWebSocket)
			ws.POST("/disconnect", h.DisconnectWebSocket)
		}

		traderGroup := api.Group("/trader")
		{
			traderGroup.POST("/force-run", h.ForceRunTrader)
		}

		stats := api.Group("/stats")
		{
			stats.GET("/daily-pnl", h.GetDailyPnL)
		}

		reportsGroup := api.Group("/reports")
		{
			reportsGroup.GET("/trades", h.GetTradeReports)
			reportsGroup.GET("/daily", h.GetDailyReports)
			reportsGroup.POST("/daily/generate", h.GenerateDailyReport)
			reportsGroup.GET("/export", h.HandleExportReport)
		}

		sim := api.Group("/simulation")
		{
			sim.GET("/:date", h.HandleGetSimulationResult)
			sim.POST("/run", h.HandleRunSimulation)
		}
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// SPA static file serving.
	// Requests not matched by /api/* or /health are served from frontendDist.
	// If the exact file exists, serve it; otherwise fall back to index.html
	// so that React Router can handle client-side navigation.
	r.NoRoute(func(c *gin.Context) {
		filePath := filepath.Join(frontendDist, filepath.Clean(c.Request.URL.Path))
		if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
			c.File(filePath)
			return
		}
		index := filepath.Join(frontendDist, "index.html")
		if _, err := os.Stat(index); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "frontend not found"})
			return
		}
		c.File(index)
	})

	return r
}

func corsMiddleware(allowedOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if allowedOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowedOrigin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// jsonLogger is a minimal structured request logger middleware.
func jsonLogger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Output is handled by the logger package; keep gin's default quiet.
		return ""
	})
}
