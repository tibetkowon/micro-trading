package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // NCP Micro 이미지에 tzdata 없을 경우 Asia/Seoul 로드 실패 방지

	"github.com/micro-trading-for-agent/backend/internal/agent"
	"github.com/micro-trading-for-agent/backend/internal/api"
	"github.com/micro-trading-for-agent/backend/internal/config"
	"github.com/micro-trading-for-agent/backend/internal/database"
	"github.com/micro-trading-for-agent/backend/internal/kis"
	"github.com/micro-trading-for-agent/backend/internal/logger"
	"github.com/micro-trading-for-agent/backend/internal/monitor"
	mqttpkg "github.com/micro-trading-for-agent/backend/internal/mqtt"
	"github.com/micro-trading-for-agent/backend/internal/trader"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	db, err := database.New(cfg.DatabasePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "database error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("database initialized", map[string]any{"path": cfg.DatabasePath})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize KIS token manager.
	tokenManager := kis.NewTokenManager(cfg.KISBaseURL, cfg.KISAppKey, cfg.KISAppSecret, db)
	if cfg.KISAppKey != "" && cfg.KISAppSecret != "" {
		if err := tokenManager.InvalidateIfCredentialsChanged(ctx); err != nil {
			logger.Warn("credential check failed — token cache may be stale",
				map[string]any{"error": err.Error()})
		}
		if _, err := tokenManager.EnsureToken(ctx); err != nil {
			logger.Warn("initial KIS token issue failed — API calls will fail until resolved",
				map[string]any{"error": err.Error()})
		}
		// 토큰 갱신은 매일 08:50 market scheduler에서 WebSocket 연결과 함께 수행.
		// 별도 20시간 타이머 없음.
	} else {
		logger.Warn("KIS_APP_KEY or KIS_APP_SECRET not set — running without KIS integration", nil)
	}

	kisClient := kis.NewClient(
		cfg.KISBaseURL,
		cfg.KISAppKey,
		cfg.KISAppSecret,
		cfg.KISAccountNo,
		cfg.KISAccountType,
		tokenManager,
		db,
	)

	// --- MQTT Publisher (optional) ---
	var mqttPub *mqttpkg.Publisher
	if cfg.MQTTBrokerURL != "" {
		pub, mqttErr := mqttpkg.NewPublisher(cfg.MQTTBrokerURL, cfg.MQTTClientID)
		if mqttErr != nil {
			logger.Warn("MQTT broker unavailable — alerts will be logged only",
				map[string]any{"broker": cfg.MQTTBrokerURL, "error": mqttErr.Error()})
		} else {
			mqttPub = pub
			defer mqttPub.Close()
			logger.Info("MQTT publisher ready", map[string]any{"broker": cfg.MQTTBrokerURL})
		}
	}

	// --- KIS WebSocket client (optional — requires credentials) ---
	var wsClient *kis.WebSocketClient
	if cfg.KISAppKey != "" && cfg.KISAppSecret != "" {
		wsClient = kis.NewWebSocketClient("", cfg.KISHTSID)
		// approval_key is fetched just before market open; start with empty key.
	}

	// --- Position monitor ---
	mon := monitor.New(db, kisClient, wsClient, mqttPub)
	if err := mon.LoadFromDB(ctx); err != nil {
		logger.Warn("failed to restore monitored positions from DB",
			map[string]any{"error": err.Error()})
	}

	// --- Order sync scheduler (폴링 폴백) ---
	if cfg.KISAppKey != "" && cfg.KISAppSecret != "" {
		agent.StartOrderSyncScheduler(ctx, kisClient, db, 5*time.Minute)
		logger.Info("order sync scheduler started", map[string]any{"interval": "5m"})
	}

	// --- Trading engine (Claude-based autonomous trader) ---
	var claudeClient *trader.ClaudeClient
	if cfg.AnthropicAPIKey != "" {
		settings, _ := db.GetTradingSettings(ctx)
		claudeClient = trader.NewClaudeClient(cfg.AnthropicAPIKey, settings.ClaudeModel)
		logger.Info("Claude client initialized", map[string]any{"model": settings.ClaudeModel})
	} else {
		logger.Warn("ANTHROPIC_API_KEY not set — autonomous trading disabled", nil)
	}

	tradingEngine := trader.NewEngine(db, kisClient, wsClient, mon, mqttPub, claudeClient, "KR")
	usEngine := trader.NewEngine(db, kisClient, wsClient, mon, mqttPub, claudeClient, "US")

	// KIS 실제 잔고와 대조하여 누락된 포지션 자동 복구.
	// DB에 등록되지 않은 보유 종목(버그·장애·수동 주문 등)을 모니터링에 추가.
	if cfg.KISAppKey != "" && cfg.KISAppSecret != "" {
		mon.RecoverFromHoldings(ctx, tradingEngine.SoldCh())
	}

	// --- Market hours scheduler ---
	if cfg.KISAppKey != "" && cfg.KISAppSecret != "" && wsClient != nil {
		go runMarketScheduler(ctx, db, kisClient, wsClient, mon, tokenManager, tradingEngine, usEngine)
	}

	// --- Price consumer ---
	if wsClient != nil {
		go mon.StartPriceConsumer(ctx)
	}

	handler := api.NewHandler(db, kisClient, tokenManager, cfg, mon, wsClient)
	handler.SetEngine(tradingEngine)
	router := api.SetupRouter(handler, cfg.FrontendDist)

	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("server starting", map[string]any{"port": cfg.ServerPort})
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			os.Exit(1)
		}
	}()

	<-quit
	logger.Info("shutting down server...", nil)

	if wsClient != nil {
		wsClient.Disconnect()
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("forced shutdown", map[string]any{"error": err.Error()})
	}
	logger.Info("server exited", nil)
}

// parseHHMM parses a "HH:MM" string into an integer (e.g. "09:15" → 915).
// Returns def if the string is empty or malformed.
func parseHHMM(s string, def int) int {
	if s == "" {
		return def
	}
	t, err := time.Parse("15:04", s)
	if err != nil {
		return def
	}
	return t.Hour()*100 + t.Minute()
}

// runMarketScheduler manages WebSocket lifecycle and trading engine based on KST market hours:
//
//	08:50 → issue fresh token → fetch approval_key → connect → subscribe
//	09:00 → check trading_enabled + market open → set tradingReady
//	09:15 → start trading engine + indicator checker (if tradingReady)
//	15:15 → stop engine → liquidate all positions
//	15:20 → generate daily report → save to DB
//	16:00 → disconnect
// isActiveUSTrading returns true if the current hhmm is within the US trading window.
// Handles midnight crossover (e.g., 22:30~05:00).
func isActiveUSTrading(hhmm, startHHMM, endHHMM int) bool {
	if startHHMM > endHHMM {
		// Midnight crossover: active if hhmm >= start OR hhmm < end
		return hhmm >= startHHMM || hhmm < endHHMM
	}
	return hhmm >= startHHMM && hhmm < endHHMM
}

func runMarketScheduler(ctx context.Context,
	db *database.DB, kisClient *kis.Client, wsClient *kis.WebSocketClient, mon *monitor.Monitor,
	tokenManager *kis.TokenManager, eng *trader.Engine, usEng *trader.Engine) {

	kst, _ := time.LoadLocation("Asia/Seoul")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	var wsRunning bool
	var engineRunning bool
	var tradingReady bool
	var stopEngine func()
	var stopIndicator context.CancelFunc

	var usEngineRunning bool
	var stopUSEngine func()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now().In(kst)
			hhmm := now.Hour()*100 + now.Minute()

			// US market scheduling (runs any day of week — US market hours)
			usStartHHMM := parseHHMM(db.GetSetting(ctx, "us_trading_start_time"), 2230)
			usEndHHMM := parseHHMM(db.GetSetting(ctx, "us_trading_end_time"), 500)
			usEnabled := db.GetSetting(ctx, "us_trading_enabled") == "true"

			switch {
			case usEng != nil && usEnabled && !usEngineRunning && isActiveUSTrading(hhmm, usStartHHMM, usEndHHMM):
				stopUSEngine = usEng.Start(ctx)
				usEngineRunning = true
				logger.Info("market scheduler: US trading engine started", map[string]any{"hhmm": hhmm})

			case usEngineRunning && !isActiveUSTrading(hhmm, usStartHHMM, usEndHHMM):
				if stopUSEngine != nil {
					stopUSEngine()
					stopUSEngine = nil
					usEngineRunning = false
				}
				mon.LiquidateAll(ctx, "US")
				logger.Info("market scheduler: US trading engine stopped", map[string]any{"hhmm": hhmm})
			}

			// KR market scheduling (weekdays only)
			wd := now.Weekday()
			if wd == time.Saturday || wd == time.Sunday {
				continue
			}
			startHHMM := parseHHMM(db.GetSetting(ctx, "trading_start_time"), 915)
			endHHMM := parseHHMM(db.GetSetting(ctx, "trading_end_time"), 1515)

			switch {
			case !wsRunning && hhmm >= 850 && hhmm < 1600:
				// 08:50 또는 장중 서버 재시작 시 WebSocket 연결.
				// hhmm == 850 에만 의존하면 서버가 08:50 이후 시작될 때 영구 미연결됨.
				if _, err := tokenManager.IssueToken(ctx); err != nil {
					logger.Error("market scheduler: token refresh failed", map[string]any{"error": err.Error()})
				} else {
					logger.Info("market scheduler: KIS token refreshed", map[string]any{"hhmm": hhmm})
				}
				approvalKey, err := kisClient.GetApprovalKey(ctx)
				if err != nil {
					logger.Error("GetApprovalKey failed", map[string]any{"error": err.Error()})
					continue
				}

				wsCtx, wsCancel := context.WithCancel(ctx)
				wsClient.SetApprovalKey(approvalKey)
				wsClient.SetReconnectCancel(wsCancel)
				go wsClient.StartWithReconnect(wsCtx)
				wsRunning = true
				tradingReady = false

				// Wait for connection, then subscribe.
				time.Sleep(2 * time.Second)
				if err := wsClient.SubscribeExecNotice(); err != nil {
					logger.Warn("exec notice subscribe failed", map[string]any{"error": err.Error()})
				}
				// 이미 매도된 종목을 monitored_positions에서 제거 후 재구독.
				mon.PurgeStalePositions(ctx)
				mon.ResubscribeAll()
				logger.Info("market scheduler: WebSocket connected", map[string]any{"hhmm": hhmm})

			case wsRunning && !tradingReady && hhmm >= 900 && hhmm < endHHMM:
				// 09:00 이후 — 트레이딩 활성화 여부 및 장 개장 확인.
				// 서버가 09:00 이후 시작된 경우도 처리.
				tradingEnabled := db.GetSetting(ctx, "trading_enabled") != "false"
				if !tradingEnabled {
					logger.Info("market scheduler: trading disabled — engine will not start", nil)
					break
				}
				isOpen, err := agent.IsMarketOpen(ctx, kisClient)
				if err != nil || !isOpen {
					logger.Info("market scheduler: market not open — engine will not start",
						map[string]any{"is_open": isOpen, "hhmm": hhmm})
					break
				}
				tradingReady = true
				logger.Info("market scheduler: trading ready confirmed", map[string]any{"hhmm": hhmm})

			case tradingReady && !engineRunning && hhmm >= startHHMM && hhmm < endHHMM:
				// 거래 시작 시간 — start autonomous trading engine
				settings, err := db.GetTradingSettings(ctx)
				if err != nil {
					logger.Error("market scheduler: GetTradingSettings failed", map[string]any{"error": err.Error()})
					break
				}

				stopEngine = eng.Start(ctx)
				engineRunning = true
				logger.Info("market scheduler: trading engine started", map[string]any{"hhmm": hhmm, "start": startHHMM, "end": endHHMM})

				// Start indicator checker (횡보 감지 설정 포함)
				mon.SetStagnationConfig(settings.StagnationThresholdPct, settings.StagnationDurationMin)
				indCtx, indCancel := context.WithCancel(ctx)
				stopIndicator = indCancel
				go mon.StartIndicatorChecker(
					indCtx,
					settings.IndicatorCheckIntervalMin,
					settings.SellConditions,
					settings.IndicatorRSISellThreshold,
					settings.IndicatorMACDBearishSell,
					func(iCtx context.Context, code string) (*monitor.IndicatorSnapshot, error) {
						info, err := agent.GetStockInfo(iCtx, kisClient, code)
						if err != nil {
							return nil, err
						}
						return &monitor.IndicatorSnapshot{
							RSI14:      info.RSI14,
							MACDLine:   info.MACDLine,
							MACDSignal: info.MACDSignal,
						}, nil
					},
				)
				logger.Info("market scheduler: indicator checker started", nil)

			case engineRunning && hhmm >= endHHMM && hhmm < 1600:
				// 거래 종료 시간 — stop engine, stop indicator checker, liquidate all
				if stopEngine != nil {
					stopEngine()
					engineRunning = false
				}
				if stopIndicator != nil {
					stopIndicator()
					stopIndicator = nil
				}
				logger.Info("market scheduler: end-time liquidation triggered", map[string]any{"hhmm": hhmm, "end": endHHMM})
				mon.LiquidateAll(ctx, "KR")

			case hhmm == 1600 && wsRunning:
				// 16:00 — disconnect
				wsClient.Disconnect()
				wsRunning = false
				tradingReady = false
				engineRunning = false
				logger.Info("market scheduler: WebSocket disconnected at 16:00", nil)
			}
		}
	}
}
