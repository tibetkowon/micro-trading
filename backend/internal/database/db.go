package database

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/micro-trading-for-agent/backend/internal/models"
)

// Collection names in Firestore.
const (
	colSettings    = "settings"
	colOrders      = "orders"
	colPositions   = "monitored_positions"
	colBalances    = "balances"
	colServiceLogs = "service_logs"
	colKISAPILogs  = "kis_api_logs"
	colScanLogs    = "scan_logs"
	colTradeRpt    = "trade_reports"
	colDailyRpt    = "daily_reports"
	colSimResults  = "simulation_results"
	colTokens      = "tokens"
	colStockMaster = "stock_masters"

	docSettings = "config"
	docToken    = "current"
)

// TradingSettings holds all autonomous trading configuration values read from Firestore settings.
type TradingSettings struct {
	// 수익/손절 기준
	TakeProfitPct      float64
	StopLossPct        float64
	ETFTakeProfitPct   float64
	ETFStopLossPct     float64
	StockTakeProfitPct float64
	StockStopLossPct   float64
	StockTaxRate       float64
	// 포지션 / 주문
	MaxPositions   int
	OrderAmountPct float64
	// 순위 / 필터
	RankingTypes              []string
	RankingPriceMin           string
	RankingPriceMax           string
	RankingCondition          string
	RankingTopN               int
	RankingVolumeMinIncrRate  float64
	RankingStrengthMin        float64
	RankingFluctuationMinRate float64
	RankingFluctuationMaxRate float64
	RankingVIKindCode         string
	RankingExchanges          []string
	RankingVolumeBlngClsCodes []string
	RankingExcludeCls         string
	HardWatchSymbols          []string
	RankLeaseDurationMin      int
	// 매도 조건
	SellConditions            []string
	IndicatorCheckIntervalMin int
	IndicatorRSISellThreshold float64
	IndicatorMACDBearishSell  bool
	ScanIntervalMin           int
	// 거래 시간
	TradingStartTime string
	TradingEndTime   string
	// 횡보 감지
	StagnationThresholdPct        float64
	StagnationDurationMin         int
	StagnationPartialExitEnabled  bool
	StagnationBidAskSellThreshold float64
	// 트레일링 스탑
	TrailingTriggerPct float64
	TrailingStopPct    float64
	// 트레일링 모드 선택 (공존)
	TrailingMode string // "pct" (기본) | "tick"
	// 틱 트레일 설정 (TrailingMode == "tick" 일 때만 적용)
	TickTier0StopLossTicks int     // X: 진입가 대비 -X틱 손절
	TickTier1TriggerPct    float64 // A: +A% 수익 시 Tier1 활성화
	TickTier1TrailTicks    int     // Y: 매수1호가 고점 대비 -Y틱
	TickTier2TriggerPct    float64 // B: +B% 수익 시 Tier2 활성화
	TickTier2TrailTicks    int     // Z: 매수1호가 고점 대비 -Z틱
	// 부분 익절
	PartialTPEnabled   bool
	PartialTPPct       float64
	PartialTPRatio     float64
	PartialTPRaiseStop bool
	// 연속 손절 제한
	MaxConsecutiveLosses         int
	ConsecutiveLossResetOnProfit bool
	// 호가 스프레드 필터
	MaxBidAskSpreadPct float64
	// 상한가 매도
	SellOnUpperLimit bool
	// 일일 손익 한도
	DailyMaxLossPct      float64
	DailyTargetProfitPct float64
	// 하드 필터
	FilterRsiMax           float64
	FilterDisparityM5Max   float64
	FilterHighPriceDiffMin float64
	FilterOpenPriceDiffMax float64
	// 점수 시스템 (v2)
	MinScoreThreshold      float64
	ScoreWeightStrength    int
	ScoreWeightRSI         int
	ScoreWeightMACD        int
	ScoreWeightBidAsk      int
	ScoreWeightVWAP        int
	ScoreWeightVolume      int
	ScoreWeightProgramBuy  int
	ScoreWeightMicroBidAsk int
	ScoreWeightVIDisparity int
	// 기타
	MinTradingValue         float64
	StreamBypassEnabled     bool
	StreamBigTradeAmount    float64
	StreamVelocityThreshold float64
	BuyPauseStart           string
	BuyPauseEnd             string
	IndexCodes              []string
	IndexDropThresholdPct   float64
	TradingDays             []int
	MinMarketCap            float64
	MinExpectedProfitPct    float64
	VWAPDiffMin             float64
	VWAPDiffMax             float64
	RSIBuyMin               float64
	RSIBuyMax               float64
	BidAskRatioMin          float64
	// Hard Rule 상세 기준
	HardDisparityM5Min      float64
	HardDisparityM5Max      float64
	HardHighPriceDiffMax    float64
	HardHighPriceDiffMin    float64
	HardPrevVolRatioMax     float64
	HardStrengthMin         float64
	HardRSIMax              float64
	HardOpenPriceDiffMax    float64
	HardProgramBuyMin       float64
	HardMACDBearishEnabled  bool
	HardMA60SupportEnabled  bool
	HardMA120SupportEnabled bool
	HardHighFormedMinsMax   float64
	HardVolVs3AvgRatioMin   float64
	HardRelativeStrengthMin float64
	// 재진입 관련
	BlockReentryOnLoss      bool
	ReentryScorePenalty     float64
	ReentryCooldownMin      int
	LossCooldownMin         int     // 손절 후 시간 기반 쿨타임(분); 0=비활성. BlockReentryOnLoss=false일 때 사용
	LossReentryPriceGuard   bool    // true=현재가 < 직전매수가면 재진입 차단
	UniversalCooldownMin    int     // 매도 후 재진입 금지 시간(분). 0=비활성. 기본 20
	UniversalPriceGuard     bool    // true=쿨타임 후에도 현재가<직전매수가면 차단
	IndicatorSellMinLossPct float64 // MACD 등 지표 매도가 발동하려면 최소 이 손실%(-) 도달해야 함. 0=비활성
	// 지표 꺾임 감지
	HardPeakTurnEnabled bool    // true=RSI 고점 꺾임 또는 연속 하락봉 시 진입 차단
	HardPeakRSIMin      float64 // 꺾임 판단 기준 RSI 하한 (기본 65.0)
	// 매수 주문 방식: "limit"=현재가 지정가(기본), "ask1"=매도1호가 지정가, "ask2"=매도2호가 지정가, "market"=순수 시장가
	BuyOrderType string
}

// DB is the Firestore-backed data layer.
type DB struct {
	client    *firestore.Client
	projectID string
}

// New initialises the Firestore client.
// credJSON may be a file path ending in ".json" or inline JSON content.
// If empty, Application Default Credentials are used.
func New(ctx context.Context, projectID, credJSON string) (*DB, error) {
	var opts []option.ClientOption

	if credJSON != "" {
		var raw []byte
		if _, err := os.Stat(credJSON); err == nil {
			// file path
			raw, err = os.ReadFile(credJSON)
			if err != nil {
				return nil, fmt.Errorf("read credentials file: %w", err)
			}
		} else {
			if strings.HasPrefix(strings.TrimSpace(credJSON), "{") {
				raw = []byte(credJSON)
			} else {
				return nil, fmt.Errorf("credentials file '%s' not found or invalid JSON string: %v", credJSON, err)
			}
		}
		opts = append(opts, option.WithCredentialsJSON(raw))
	}

	client, err := firestore.NewClient(ctx, projectID, opts...)
	if err != nil {
		return nil, fmt.Errorf("firestore.NewClient: %w", err)
	}

	db := &DB{client: client, projectID: projectID}
	if err := db.initDefaults(ctx); err != nil {
		return nil, fmt.Errorf("initDefaults: %w", err)
	}
	return db, nil
}

// FirestoreClient exposes the underlying client for packages that need direct access (e.g. mst).
func (db *DB) FirestoreClient() *firestore.Client { return db.client }

// Close releases the Firestore client.
func (db *DB) Close() error { return db.client.Close() }

// newID returns a unique integer ID based on the current time in microseconds.
func newID() int64 { return time.Now().UnixMicro() }

// ─── Settings ────────────────────────────────────────────────────────────────

// GetSetting returns the value of a setting key, or "" if not found.
func (db *DB) GetSetting(ctx context.Context, key string) string {
	v, _ := db.GetSettingErr(ctx, key)
	return v
}

// GetSettingErr returns the value of a setting key with error.
func (db *DB) GetSettingErr(ctx context.Context, key string) (string, error) {
	snap, err := db.client.Collection(colSettings).Doc(docSettings).Get(ctx)
	if err != nil {
		return "", err
	}
	v, ok := snap.Data()[key]
	if !ok {
		return "", nil
	}
	return fmt.Sprintf("%v", v), nil
}

// GetAllSettings returns all settings as a map[string]string.
func (db *DB) GetAllSettings(ctx context.Context) (map[string]string, error) {
	snap, err := db.client.Collection(colSettings).Doc(docSettings).Get(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(snap.Data()))
	for k, v := range snap.Data() {
		result[k] = fmt.Sprintf("%v", v)
	}
	return result, nil
}

// SetSetting writes a single setting key-value pair.
func (db *DB) SetSetting(ctx context.Context, key, value string) error {
	_, err := db.client.Collection(colSettings).Doc(docSettings).Set(ctx,
		map[string]any{key: value}, firestore.MergeAll)
	return err
}

// GetTradingSettings reads all autonomous trading settings from Firestore in one call.
func (db *DB) GetTradingSettings(ctx context.Context) (TradingSettings, error) {
	m, err := db.GetAllSettings(ctx)
	if err != nil {
		return TradingSettings{}, fmt.Errorf("GetAllSettings: %w", err)
	}
	s := TradingSettings{}
	s.TakeProfitPct = pf(m, "take_profit_pct", 3.0)
	s.StopLossPct = pf(m, "stop_loss_pct", 2.0)
	s.ETFTakeProfitPct = pf(m, "etf_take_profit_pct", 0.5)
	s.ETFStopLossPct = pf(m, "etf_stop_loss_pct", 1.0)
	s.StockTakeProfitPct = pf(m, "stock_take_profit_pct", 1.5)
	s.StockStopLossPct = pf(m, "stock_stop_loss_pct", 1.0)
	s.StockTaxRate = pf(m, "stock_tax_rate", 0.0)
	s.MaxPositions = pi(m, "max_positions", 1)
	s.OrderAmountPct = pf(m, "order_amount_pct", 95.0)
	s.RankingTypes = pjsonStrSlice(m, "ranking_types", []string{"volume", "strength"})
	s.RankingPriceMin = ps(m, "ranking_price_min", "5000")
	s.RankingPriceMax = ps(m, "ranking_price_max", "200000")
	s.RankingCondition = ps(m, "ranking_condition", "OR")
	s.RankingTopN = pi(m, "ranking_top_n", 30)
	s.RankingVolumeMinIncrRate = pf(m, "ranking_volume_min_incr_rate", 0.0)
	s.RankingStrengthMin = pf(m, "ranking_strength_min", 0.0)
	s.RankingFluctuationMinRate = pf(m, "ranking_fluctuation_min_rate", 0.0)
	s.RankingFluctuationMaxRate = pf(m, "ranking_fluctuation_max_rate", 0.0)
	s.RankingVIKindCode = ps(m, "ranking_vi_kind_code", "")
	s.RankingExchanges = pjsonStrSlice(m, "ranking_exchanges", []string{"0001", "1001"})
	s.RankingVolumeBlngClsCodes = pjsonStrSlice(m, "ranking_volume_blng_cls_codes", []string{"0"})
	s.RankingExcludeCls = ps(m, "ranking_exclude_cls", "1111111111")
	s.HardWatchSymbols = pjsonStrSlice(m, "hard_watch_symbols", nil)
	s.RankLeaseDurationMin = pi(m, "rank_lease_duration_min", 0)
	s.SellConditions = pjsonStrSlice(m, "sell_conditions", []string{"take_profit", "stop_loss"})
	s.IndicatorCheckIntervalMin = pi(m, "indicator_check_interval_min", 5)
	s.IndicatorRSISellThreshold = pf(m, "indicator_rsi_sell_threshold", 70.0)
	s.IndicatorMACDBearishSell = pb(m, "indicator_macd_bearish_sell", false)
	s.ScanIntervalMin = pi(m, "scan_interval", 1)
	if s.ScanIntervalMin < 1 {
		s.ScanIntervalMin = 1
	}
	s.TradingStartTime = ps(m, "trading_start_time", "09:15")
	s.TradingEndTime = ps(m, "trading_end_time", "15:15")
	s.StagnationThresholdPct = pf(m, "stagnation_threshold_pct", 0.0)
	s.StagnationDurationMin = pi(m, "stagnation_duration_min", 0)
	s.StagnationPartialExitEnabled = pb(m, "stagnation_partial_exit_enabled", false)
	s.StagnationBidAskSellThreshold = pf(m, "stagnation_bidask_sell_threshold", 1.0)
	s.TrailingTriggerPct = pf(m, "trailing_trigger_pct", 0.0)
	s.TrailingStopPct = pf(m, "trailing_stop_pct", 0.0)
	s.TrailingMode = ps(m, "trailing_mode", "pct")
	s.TickTier0StopLossTicks = pi(m, "tick_tier0_stop_loss_ticks", 3)
	s.TickTier1TriggerPct = pf(m, "tick_tier1_trigger_pct", 0.0)
	s.TickTier1TrailTicks = pi(m, "tick_tier1_trail_ticks", 5)
	s.TickTier2TriggerPct = pf(m, "tick_tier2_trigger_pct", 0.0)
	s.TickTier2TrailTicks = pi(m, "tick_tier2_trail_ticks", 2)
	s.PartialTPEnabled = pb(m, "partial_tp_enabled", false)
	s.PartialTPPct = pf(m, "partial_tp_pct", 1.0)
	s.PartialTPRatio = pf(m, "partial_tp_ratio", 0.5)
	s.PartialTPRaiseStop = pb(m, "partial_tp_raise_stop", false)
	s.MaxConsecutiveLosses = pi(m, "max_consecutive_losses", 0)
	s.ConsecutiveLossResetOnProfit = pb(m, "consecutive_loss_reset_on_profit", true)
	s.MaxBidAskSpreadPct = pf(m, "max_bidask_spread_pct", 0.0)
	s.SellOnUpperLimit = pb(m, "sell_on_upper_limit", false)
	s.DailyMaxLossPct = pf(m, "daily_max_loss_pct", 0.0)
	s.DailyTargetProfitPct = pf(m, "daily_target_profit_pct", 0.0)
	s.FilterRsiMax = pf(m, "filter_rsi_max", 80.0)
	s.FilterDisparityM5Max = pf(m, "filter_disparity_m5_max", 3.0)
	s.FilterHighPriceDiffMin = pf(m, "filter_high_price_diff_min", -5.0)
	s.FilterOpenPriceDiffMax = pf(m, "filter_open_price_diff_max", 20.0)
	// v2 score system
	s.MinScoreThreshold = pf(m, "min_score_threshold", 0.0)
	s.ScoreWeightStrength = pi(m, "score_weight_strength", 30)
	s.ScoreWeightRSI = pi(m, "score_weight_rsi", 20)
	s.ScoreWeightMACD = pi(m, "score_weight_macd", 20)
	s.ScoreWeightBidAsk = pi(m, "score_weight_bidask", 15)
	s.ScoreWeightVWAP = pi(m, "score_weight_vwap", 10)
	s.ScoreWeightVolume = pi(m, "score_weight_volume", 5)
	s.ScoreWeightProgramBuy = pi(m, "score_weight_program_buy", 10)
	s.ScoreWeightMicroBidAsk = pi(m, "score_weight_micro_bidask", 10)
	s.ScoreWeightVIDisparity = pi(m, "score_weight_vi_disparity", 10)
	// misc
	s.MinTradingValue = pf(m, "min_trading_value", 0.0)
	s.StreamBypassEnabled = pb(m, "stream_bypass_enabled", true)
	s.StreamBigTradeAmount = pf(m, "stream_big_trade_amount", 30000000.0)
	s.StreamVelocityThreshold = pf(m, "stream_velocity_threshold", 5.0)
	s.BuyPauseStart = ps(m, "buy_pause_start", "")
	s.BuyPauseEnd = ps(m, "buy_pause_end", "")
	s.IndexCodes = pjsonStrSlice(m, "index_codes", nil)
	s.IndexDropThresholdPct = pf(m, "index_drop_threshold_pct", -1.0)
	s.TradingDays = pjsonIntSlice(m, "trading_days", nil)
	s.MinMarketCap = pf(m, "min_market_cap", 0.0)
	s.MinExpectedProfitPct = pf(m, "min_expected_profit_pct", 0.0)
	s.VWAPDiffMin = pf(m, "vwap_diff_min", 0.0)
	s.VWAPDiffMax = pf(m, "vwap_diff_max", 1.5)
	s.RSIBuyMin = pf(m, "rsi_buy_min", 40.0)
	s.RSIBuyMax = pf(m, "rsi_buy_max", 60.0)
	s.BidAskRatioMin = pf(m, "bid_ask_ratio_min", 1.2)
	s.HardDisparityM5Min = pf(m, "hard_disparity_m5_min", -1.5)
	s.HardDisparityM5Max = pf(m, "hard_disparity_m5_max", 3.0)
	s.HardHighPriceDiffMax = pf(m, "hard_high_price_diff_max", -0.5)
	s.HardHighPriceDiffMin = pf(m, "hard_high_price_diff_min", -5.0)
	s.HardPrevVolRatioMax = pf(m, "hard_prev_vol_ratio_max", 1.2)
	s.HardStrengthMin = pf(m, "hard_strength_min", 100.0)
	s.HardRSIMax = pf(m, "hard_rsi_max", 70.0)
	s.HardOpenPriceDiffMax = pf(m, "hard_open_price_diff_max", 15.0)
	s.HardProgramBuyMin = pf(m, "hard_program_buy_min", 0.0)
	s.HardMACDBearishEnabled = pb(m, "hard_macd_bearish_enabled", false)
	s.HardMA60SupportEnabled = pb(m, "hard_ma60_support_enabled", false)
	s.HardMA120SupportEnabled = pb(m, "hard_ma120_support_enabled", false)
	s.HardHighFormedMinsMax = pf(m, "hard_high_formed_mins_max", 0.0)
	s.HardVolVs3AvgRatioMin = pf(m, "hard_vol_vs3_avg_ratio_min", 0.0)
	s.HardRelativeStrengthMin = pf(m, "hard_relative_strength_min", 0.0)
	s.BlockReentryOnLoss = pb(m, "block_reentry_on_loss", true)
	s.ReentryScorePenalty = pf(m, "reentry_score_penalty", 10.0)
	s.ReentryCooldownMin = pi(m, "reentry_cooldown_min", 15)
	s.LossCooldownMin = pi(m, "loss_cooldown_min", 20)
	s.LossReentryPriceGuard = pb(m, "loss_reentry_price_guard", true)
	s.UniversalCooldownMin = pi(m, "universal_cooldown_min", 20)
	s.UniversalPriceGuard = pb(m, "universal_price_guard", true)
	s.IndicatorSellMinLossPct = pf(m, "indicator_sell_min_loss_pct", 1.0)
	s.HardPeakTurnEnabled = pb(m, "hard_peak_turn_enabled", false)
	s.HardPeakRSIMin = pf(m, "hard_peak_rsi_min", 65.0)
	s.BuyOrderType = ps(m, "buy_order_type", "limit")
	return s, nil
}

// ─── Token ────────────────────────────────────────────────────────────────────

// GetCurrentToken returns the stored KIS access token.
func (db *DB) GetCurrentToken(ctx context.Context) (*models.Token, error) {
	snap, err := db.client.Collection(colTokens).Doc(docToken).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("token not found: %w", err)
	}
	var tok models.Token
	if err := snap.DataTo(&tok); err != nil {
		return nil, fmt.Errorf("decode token: %w", err)
	}
	return &tok, nil
}

// SaveToken persists the KIS access token, overwriting any previous value.
func (db *DB) SaveToken(ctx context.Context, tok *models.Token) error {
	tok.ID = newID()
	tok.ExpireAt = time.Now().UTC().Add(48 * time.Hour)
	_, err := db.client.Collection(colTokens).Doc(docToken).Set(ctx, tok)
	return err
}

// DeleteAllTokens removes the stored token (e.g. on credential change).
func (db *DB) DeleteAllTokens(ctx context.Context) error {
	_, err := db.client.Collection(colTokens).Doc(docToken).Delete(ctx)
	return err
}

// ─── Orders ──────────────────────────────────────────────────────────────────

// CreateOrder persists a new order and returns its assigned ID.
func (db *DB) CreateOrder(ctx context.Context, o *models.Order) (int64, error) {
	id := newID()
	o.ID = id
	if o.Source == "" {
		o.Source = models.OrderSourceAgent
	}
	if o.CreatedAt.IsZero() {
		o.CreatedAt = time.Now().UTC()
	}
	o.ExpireAt = time.Now().UTC().Add(365 * 24 * time.Hour)
	_, err := db.client.Collection(colOrders).Doc(strconv.FormatInt(id, 10)).Set(ctx, o)
	return id, err
}

// GetOrderByID fetches a single order by its local ID.
func (db *DB) GetOrderByID(ctx context.Context, id int64) (*models.Order, error) {
	snap, err := db.client.Collection(colOrders).Doc(strconv.FormatInt(id, 10)).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("order %d not found: %w", id, err)
	}
	var o models.Order
	if err := snap.DataTo(&o); err != nil {
		return nil, err
	}
	return &o, nil
}

// GetOrderByKISID returns the first order matching the given KIS order ID.
func (db *DB) GetOrderByKISID(ctx context.Context, kisOrderID string) (*models.Order, error) {
	iter := db.client.Collection(colOrders).
		Where("kis_order_id", "==", kisOrderID).
		Limit(1).Documents(ctx)
	defer iter.Stop()
	snap, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("no order with kis_order_id=%s", kisOrderID)
	}
	if err != nil {
		return nil, err
	}
	var o models.Order
	if err := snap.DataTo(&o); err != nil {
		return nil, err
	}
	return &o, nil
}

// UpdateOrderStatus updates the status of an order by its local ID.
func (db *DB) UpdateOrderStatus(ctx context.Context, id int64, status models.OrderStatus) error {
	_, err := db.client.Collection(colOrders).Doc(strconv.FormatInt(id, 10)).
		Update(ctx, []firestore.Update{{Path: "status", Value: string(status)}})
	return err
}

// UpdateOrderFilled updates status and filled_price for orders matching the given KIS order ID.
func (db *DB) UpdateOrderFilled(ctx context.Context, kisOrderID string, status models.OrderStatus, filledPrice float64) error {
	o, err := db.GetOrderByKISID(ctx, kisOrderID)
	if err != nil {
		return err
	}
	_, err = db.client.Collection(colOrders).Doc(strconv.FormatInt(o.ID, 10)).
		Update(ctx, []firestore.Update{
			{Path: "status", Value: string(status)},
			{Path: "filled_price", Value: filledPrice},
		})
	return err
}

// UpdateOrderStockName sets the stock_name for orders with the given KIS order ID
// only when the current name is empty.
func (db *DB) UpdateOrderStockName(ctx context.Context, kisOrderID, stockName string) error {
	if stockName == "" {
		return nil
	}
	o, err := db.GetOrderByKISID(ctx, kisOrderID)
	if err != nil {
		return nil // not found — ignore
	}
	if o.StockName != "" {
		return nil // already set
	}
	_, err = db.client.Collection(colOrders).Doc(strconv.FormatInt(o.ID, 10)).
		Update(ctx, []firestore.Update{{Path: "stock_name", Value: stockName}})
	return err
}

// UpdateOrderSellReason sets the sell_reason field on an order.
func (db *DB) UpdateOrderSellReason(ctx context.Context, id int64, reason string) error {
	_, err := db.client.Collection(colOrders).Doc(strconv.FormatInt(id, 10)).
		Update(ctx, []firestore.Update{{Path: "sell_reason", Value: reason}})
	return err
}

// UpsertManualOrder inserts a MANUAL order if no order with the same KIS order ID exists.
func (db *DB) UpsertManualOrder(ctx context.Context, o *models.Order) error {
	_, err := db.GetOrderByKISID(ctx, o.KISOrderID)
	if err == nil {
		return nil // already exists
	}
	o.Source = models.OrderSourceManual
	_, err = db.CreateOrder(ctx, o)
	return err
}

// ListOrders returns orders sorted by created_at DESC, with pagination.
func (db *DB) ListOrders(ctx context.Context, limit, offset int) ([]models.Order, error) {
	q := db.client.Collection(colOrders).OrderBy("created_at", firestore.Desc)
	if limit > 0 {
		q = q.Limit(limit + offset)
	}
	iter := q.Documents(ctx)
	defer iter.Stop()

	var all []models.Order
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var o models.Order
		if err := snap.DataTo(&o); err != nil {
			continue
		}
		all = append(all, o)
	}
	if offset >= len(all) {
		return nil, nil
	}
	return all[offset:], nil
}

// InsertServiceLog is an alias for CreateServiceLog for backward compatibility.
func (db *DB) InsertServiceLog(ctx context.Context, source, level, message, detail string) error {
	return db.CreateServiceLog(ctx, source, level, message, detail)
}

// GetLastAgentBuyOrder returns the most recent AGENT BUY FILLED order for a stock code, or nil.
func (db *DB) GetLastAgentBuyOrder(ctx context.Context, stockCode string) (*models.Order, error) {
	iter := db.client.Collection(colOrders).
		Where("stock_code", "==", stockCode).
		Where("order_type", "==", string(models.OrderTypeBuy)).
		Where("status", "==", string(models.OrderStatusFilled)).
		Where("source", "==", string(models.OrderSourceAgent)).
		OrderBy("created_at", firestore.Desc).
		Limit(1).Documents(ctx)
	defer iter.Stop()
	snap, err := iter.Next()
	if err == iterator.Done {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var o models.Order
	if err := snap.DataTo(&o); err != nil {
		return nil, err
	}
	return &o, nil
}

// UpdateTradeReportOnSell fills in the sell-side of a trade report identified by the buy order ID.
func (db *DB) UpdateTradeReportOnSell(ctx context.Context, buyOrderID, sellOrderID int64, sellPrice float64, sellQty int, sellReason, sellIndicators string) error {
	iter := db.client.Collection(colTradeRpt).
		Where("buy_order_id", "==", buyOrderID).
		Limit(1).Documents(ctx)
	defer iter.Stop()
	snap, err := iter.Next()
	if err == iterator.Done {
		return fmt.Errorf("UpdateTradeReportOnSell: no trade report found for buy_order_id=%d", buyOrderID)
	}
	if err != nil {
		return err
	}
	var r models.TradeReport
	if err := snap.DataTo(&r); err != nil {
		return err
	}

	now := time.Now().UTC()
	sellAmount := sellPrice * float64(sellQty)
	profitAmount := (sellPrice - r.BuyPrice) * float64(sellQty)
	profitPct := 0.0
	if r.BuyPrice > 0 {
		profitPct = (sellPrice - r.BuyPrice) / r.BuyPrice * 100
	}

	updates := map[string]any{
		"sell_order_id":   sellOrderID,
		"sell_price":      sellPrice,
		"sell_qty":        sellQty,
		"sell_amount":     sellAmount,
		"sell_reason":     sellReason,
		"sell_indicators": sellIndicators,
		"profit_amount":   profitAmount,
		"profit_pct":      profitPct,
		"sold_at":         now,
	}
	return db.UpdateTradeReportSell(ctx, r.ID, updates)
}

// CountOrders returns the total number of orders. Uses a full scan — only for small datasets.
func (db *DB) CountOrders(ctx context.Context) (int, error) {
	iter := db.client.Collection(colOrders).Documents(ctx)
	defer iter.Stop()
	n := 0
	for {
		_, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		n++
	}
	return n, nil
}

// ─── MonitoredPositions ───────────────────────────────────────────────────────

// CreatePosition persists a new monitored position (doc ID = stock_code).
func (db *DB) CreatePosition(ctx context.Context, p *models.MonitoredPosition) error {
	if p.ID == 0 {
		p.ID = newID()
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now().UTC()
	}
	_, err := db.client.Collection(colPositions).Doc(p.StockCode).Set(ctx, p)
	return err
}

// DeletePosition removes a monitored position by stock code.
func (db *DB) DeletePosition(ctx context.Context, stockCode string) error {
	_, err := db.client.Collection(colPositions).Doc(stockCode).Delete(ctx)
	return err
}

// UpdatePositionQty updates the remaining_qty of a monitored position.
func (db *DB) UpdatePositionQty(ctx context.Context, stockCode string, remainingQty int) error {
	_, err := db.client.Collection(colPositions).Doc(stockCode).Update(ctx, []firestore.Update{
		{Path: "remaining_qty", Value: remainingQty},
	})
	return err
}

// ListPositions returns all currently monitored positions.
func (db *DB) ListPositions(ctx context.Context) ([]models.MonitoredPosition, error) {
	iter := db.client.Collection(colPositions).Documents(ctx)
	defer iter.Stop()
	var positions []models.MonitoredPosition
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var p models.MonitoredPosition
		if err := snap.DataTo(&p); err != nil {
			continue
		}
		positions = append(positions, p)
	}
	return positions, nil
}

// ─── Balances ─────────────────────────────────────────────────────────────────

// CreateBalance records a balance snapshot.
func (db *DB) CreateBalance(ctx context.Context, totalEval, availableAmt float64) error {
	id := newID()
	now := time.Now().UTC()
	bal := models.Balance{
		ID:              id,
		TotalEval:       totalEval,
		AvailableAmount: availableAmt,
		RecordedAt:      now,
		ExpireAt:        now.Add(30 * 24 * time.Hour),
	}
	_, err := db.client.Collection(colBalances).Doc(strconv.FormatInt(id, 10)).Set(ctx, bal)
	return err
}

// GetLatestBalance returns the most recent balance snapshot.
func (db *DB) GetLatestBalance(ctx context.Context) (*models.Balance, error) {
	iter := db.client.Collection(colBalances).
		OrderBy("recorded_at", firestore.Desc).
		Limit(1).Documents(ctx)
	defer iter.Stop()
	snap, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("no balance snapshot")
	}
	if err != nil {
		return nil, err
	}
	var b models.Balance
	if err := snap.DataTo(&b); err != nil {
		return nil, err
	}
	return &b, nil
}

// ─── ServiceLogs ─────────────────────────────────────────────────────────────

// CreateServiceLog persists a service-level log entry.
func (db *DB) CreateServiceLog(ctx context.Context, source, level, message, detail string) error {
	id := newID()
	now := time.Now().UTC()
	log := models.ServiceLog{
		ID:        id,
		Source:    source,
		Level:     level,
		Message:   message,
		Detail:    detail,
		Timestamp: now,
		ExpireAt:  now.Add(14 * 24 * time.Hour),
	}
	_, err := db.client.Collection(colServiceLogs).Doc(strconv.FormatInt(id, 10)).Set(ctx, log)
	return err
}

// ListServiceLogs returns the most recent service logs.
func (db *DB) ListServiceLogs(ctx context.Context, limit int) ([]models.ServiceLog, error) {
	q := db.client.Collection(colServiceLogs).OrderBy("timestamp", firestore.Desc)
	if limit > 0 {
		q = q.Limit(limit)
	}
	iter := q.Documents(ctx)
	defer iter.Stop()
	var logs []models.ServiceLog
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var l models.ServiceLog
		if err := snap.DataTo(&l); err != nil {
			continue
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// DeleteServiceLog removes a service log by ID.
func (db *DB) DeleteServiceLog(ctx context.Context, id int64) error {
	_, err := db.client.Collection(colServiceLogs).Doc(strconv.FormatInt(id, 10)).Delete(ctx)
	return err
}

// ─── KISAPILogs ───────────────────────────────────────────────────────────────

// CreateKISAPILog persists a KIS API error log entry.
func (db *DB) CreateKISAPILog(ctx context.Context, endpoint, errCode, errMsg, raw, requestContext string) error {
	id := newID()
	now := time.Now().UTC()
	log := models.KISAPILog{
		ID:             id,
		Endpoint:       endpoint,
		ErrorCode:      errCode,
		ErrorMsg:       errMsg,
		RawResponse:    raw,
		RequestContext: requestContext,
		Timestamp:      now,
		ExpireAt:       now.Add(3 * 24 * time.Hour),
	}
	_, err := db.client.Collection(colKISAPILogs).Doc(strconv.FormatInt(id, 10)).Set(ctx, log)
	return err
}

// ListKISAPILogs returns the most recent KIS API error logs.
func (db *DB) ListKISAPILogs(ctx context.Context, limit int) ([]models.KISAPILog, error) {
	q := db.client.Collection(colKISAPILogs).OrderBy("timestamp", firestore.Desc)
	if limit > 0 {
		q = q.Limit(limit)
	}
	iter := q.Documents(ctx)
	defer iter.Stop()
	var logs []models.KISAPILog
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var l models.KISAPILog
		if err := snap.DataTo(&l); err != nil {
			continue
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// DeleteKISAPILog removes a KIS API log by ID.
func (db *DB) DeleteKISAPILog(ctx context.Context, id int64) error {
	_, err := db.client.Collection(colKISAPILogs).Doc(strconv.FormatInt(id, 10)).Delete(ctx)
	return err
}

// ─── ScanLogs ─────────────────────────────────────────────────────────────────

// CreateScanLog persists a scanner cycle result.
func (db *DB) CreateScanLog(ctx context.Context, sl *models.ScanLog) (int64, error) {
	id := newID()
	sl.ID = id
	if sl.Timestamp == "" {
		sl.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	sl.ExpireAt = time.Now().UTC().Add(7 * 24 * time.Hour)
	_, err := db.client.Collection(colScanLogs).Doc(strconv.FormatInt(id, 10)).Set(ctx, sl)
	return id, err
}

// ListScanLogs returns the most recent scan logs.
func (db *DB) ListScanLogs(ctx context.Context, limit int) ([]models.ScanLog, error) {
	q := db.client.Collection(colScanLogs).OrderBy("timestamp", firestore.Desc)
	if limit > 0 {
		q = q.Limit(limit)
	}
	iter := q.Documents(ctx)
	defer iter.Stop()
	var logs []models.ScanLog
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var l models.ScanLog
		if err := snap.DataTo(&l); err != nil {
			continue
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// GetScanLog returns a single scan log by its ID (Firestore document point-read).
func (db *DB) GetScanLog(ctx context.Context, id int64) (*models.ScanLog, error) {
	snap, err := db.client.Collection(colScanLogs).Doc(strconv.FormatInt(id, 10)).Get(ctx)
	if err != nil {
		return nil, err
	}
	var l models.ScanLog
	if err := snap.DataTo(&l); err != nil {
		return nil, err
	}
	return &l, nil
}

// ─── TradeReports ─────────────────────────────────────────────────────────────

// CreateTradeReport persists a new trade report (buy side).
func (db *DB) CreateTradeReport(ctx context.Context, r *models.TradeReport) (int64, error) {
	id := newID()
	r.ID = id
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now().UTC()
	}
	r.ExpireAt = time.Now().UTC().Add(180 * 24 * time.Hour)
	_, err := db.client.Collection(colTradeRpt).Doc(strconv.FormatInt(id, 10)).Set(ctx, r)
	return id, err
}

// UpdateTradeReportSell fills in the sell side of a trade report.
func (db *DB) UpdateTradeReportSell(ctx context.Context, id int64, updates map[string]any) error {
	var fsUpdates []firestore.Update
	for k, v := range updates {
		fsUpdates = append(fsUpdates, firestore.Update{Path: k, Value: v})
	}
	_, err := db.client.Collection(colTradeRpt).Doc(strconv.FormatInt(id, 10)).Update(ctx, fsUpdates)
	return err
}

// GetTradeReport fetches a single trade report by ID.
func (db *DB) GetTradeReport(ctx context.Context, id int64) (*models.TradeReport, error) {
	snap, err := db.client.Collection(colTradeRpt).Doc(strconv.FormatInt(id, 10)).Get(ctx)
	if err != nil {
		return nil, err
	}
	var r models.TradeReport
	if err := snap.DataTo(&r); err != nil {
		return nil, err
	}
	return &r, nil
}

// ListOpenTradeReports returns all trade reports that have not been sold yet.
func (db *DB) ListOpenTradeReports(ctx context.Context) ([]models.TradeReport, error) {
	iter := db.client.Collection(colTradeRpt).
		Where("sell_order_id", "==", int64(0)).Documents(ctx)
	defer iter.Stop()
	var reports []models.TradeReport
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var r models.TradeReport
		if err := snap.DataTo(&r); err != nil {
			continue
		}
		reports = append(reports, r)
	}
	return reports, nil
}

// GetCompletedTradesBySoldDate returns all sold trade reports for a given date (KST YYYY-MM-DD).
func (db *DB) GetCompletedTradesBySoldDate(ctx context.Context, date string) ([]models.TradeReport, error) {
	kst, _ := time.LoadLocation("Asia/Seoul")
	start, err := time.ParseInLocation("2006-01-02", date, kst)
	if err != nil {
		return nil, fmt.Errorf("parse date: %w", err)
	}
	end := start.Add(24 * time.Hour)

	iter := db.client.Collection(colTradeRpt).
		Where("sold_at", ">=", start).
		Where("sold_at", "<", end).
		Documents(ctx)
	defer iter.Stop()
	var reports []models.TradeReport
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var r models.TradeReport
		if err := snap.DataTo(&r); err != nil {
			continue
		}
		reports = append(reports, r)
	}
	return reports, nil
}

// ListTradeReports returns the most recent trade reports.
func (db *DB) ListTradeReports(ctx context.Context, limit int) ([]models.TradeReport, error) {
	q := db.client.Collection(colTradeRpt).OrderBy("created_at", firestore.Desc)
	if limit > 0 {
		q = q.Limit(limit)
	}
	iter := q.Documents(ctx)
	defer iter.Stop()
	var reports []models.TradeReport
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var r models.TradeReport
		if err := snap.DataTo(&r); err != nil {
			continue
		}
		reports = append(reports, r)
	}
	return reports, nil
}

// GetLatestCompletedTradeByStock returns the most recent sold trade report for a stock.
// Returns nil, nil if none found.
func (db *DB) GetLatestCompletedTradeByStock(ctx context.Context, stockCode string) (*models.TradeReport, error) {
	iter := db.client.Collection(colTradeRpt).
		Where("stock_code", "==", stockCode).
		Where("sold_at", "!=", nil).
		OrderBy("sold_at", firestore.Desc).
		Limit(1).Documents(ctx)
	defer iter.Stop()
	snap, err := iter.Next()
	if err == iterator.Done {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var r models.TradeReport
	if err := snap.DataTo(&r); err != nil {
		return nil, err
	}
	return &r, nil
}

// ─── DailyReports ─────────────────────────────────────────────────────────────

// InsertOrUpdateDailyReport upserts a daily report (doc ID = date).
func (db *DB) InsertOrUpdateDailyReport(ctx context.Context, dr models.DailyReport) error {
	if dr.ID == 0 {
		dr.ID = newID()
	}
	if dr.CreatedAt.IsZero() {
		dr.CreatedAt = time.Now().UTC()
	}
	dr.ExpireAt = time.Now().UTC().Add(180 * 24 * time.Hour)
	_, err := db.client.Collection(colDailyRpt).Doc(dr.Date).Set(ctx, dr)
	return err
}

// GetDailyReport fetches a daily report by date string (YYYY-MM-DD).
func (db *DB) GetDailyReport(ctx context.Context, date string) (*models.DailyReport, error) {
	snap, err := db.client.Collection(colDailyRpt).Doc(date).Get(ctx)
	if err != nil {
		return nil, err
	}
	var dr models.DailyReport
	if err := snap.DataTo(&dr); err != nil {
		return nil, err
	}
	return &dr, nil
}

// ListDailyReports returns the most recent daily reports.
func (db *DB) ListDailyReports(ctx context.Context, limit int) ([]models.DailyReport, error) {
	q := db.client.Collection(colDailyRpt).OrderBy("date", firestore.Desc)
	if limit > 0 {
		q = q.Limit(limit)
	}
	iter := q.Documents(ctx)
	defer iter.Stop()
	var reports []models.DailyReport
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var dr models.DailyReport
		if err := snap.DataTo(&dr); err != nil {
			continue
		}
		reports = append(reports, dr)
	}
	return reports, nil
}

// ListDailyReportsByDateRange returns daily reports within [from, to] inclusive (YYYY-MM-DD).
func (db *DB) ListDailyReportsByDateRange(ctx context.Context, from, to string) ([]models.DailyReport, error) {
	iter := db.client.Collection(colDailyRpt).
		Where("date", ">=", from).
		Where("date", "<=", to).
		OrderBy("date", firestore.Asc).
		Documents(ctx)
	defer iter.Stop()
	var reports []models.DailyReport
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var dr models.DailyReport
		if err := snap.DataTo(&dr); err != nil {
			continue
		}
		reports = append(reports, dr)
	}
	return reports, nil
}

// UpsertSimulationResult saves or overwrites the simulation result for a date.
func (db *DB) UpsertSimulationResult(ctx context.Context, r models.SimulationResult) error {
	if r.ID == 0 {
		r.ID = newID()
	}
	r.CreatedAt = time.Now().UTC()
	r.ExpireAt = time.Now().UTC().Add(30 * 24 * time.Hour)
	_, err := db.client.Collection(colSimResults).Doc(r.Date).Set(ctx, r)
	return err
}

// GetSimulationResult fetches the simulation result for a date.
func (db *DB) GetSimulationResult(ctx context.Context, date string) (*models.SimulationResult, error) {
	snap, err := db.client.Collection(colSimResults).Doc(date).Get(ctx)
	if err != nil {
		return nil, err
	}
	var r models.SimulationResult
	if err := snap.DataTo(&r); err != nil {
		return nil, err
	}
	return &r, nil
}

// ─── initDefaults ─────────────────────────────────────────────────────────────

// defaultSettings contains the v2 baseline settings inserted on first run.
var defaultSettings = map[string]any{
	"trading_enabled":                  "true",
	"max_positions":                    "1",
	"order_amount_pct":                 "95",
	"daily_max_loss_pct":               "0",
	"daily_target_profit_pct":          "0",
	"stock_take_profit_pct":            "1.5",
	"stock_stop_loss_pct":              "1.0",
	"etf_take_profit_pct":              "0.5",
	"etf_stop_loss_pct":                "1.0",
	"take_profit_pct":                  "3.0",
	"stop_loss_pct":                    "2.0",
	"stock_tax_rate":                   "0",
	"trailing_trigger_pct":             "0",
	"trailing_stop_pct":                "0",
	"trailing_mode":                    "pct",
	"tick_tier0_stop_loss_ticks":       "3",
	"tick_tier1_trigger_pct":           "0",
	"tick_tier1_trail_ticks":           "5",
	"tick_tier2_trigger_pct":           "0",
	"tick_tier2_trail_ticks":           "2",
	"partial_tp_enabled":               "false",
	"partial_tp_pct":                   "1.0",
	"partial_tp_ratio":                 "0.5",
	"partial_tp_raise_stop":            "false",
	"ranking_types":                    `["volume","strength"]`,
	"ranking_price_min":                "5000",
	"ranking_price_max":                "200000",
	"ranking_condition":                "OR",
	"ranking_top_n":                    "30",
	"ranking_exchanges":                `["0001","1001"]`,
	"ranking_volume_blng_cls_codes":    `["0"]`,
	"ranking_volume_min_incr_rate":     "0",
	"ranking_strength_min":             "0",
	"ranking_fluctuation_min_rate":     "0",
	"ranking_fluctuation_max_rate":     "0",
	"ranking_vi_kind_code":             "",
	"ranking_exclude_cls":              "1111111111",
	"hard_watch_symbols":               `[]`,
	"rank_lease_duration_min":          "0",
	"sell_conditions":                  `["take_profit","stop_loss"]`,
	"indicator_check_interval_min":     "5",
	"scan_interval":                    "1",
	"indicator_rsi_sell_threshold":     "70",
	"indicator_macd_bearish_sell":      "false",
	"indicator_sell_min_loss_pct":      "1.0",
	"trading_start_time":               "09:15",
	"trading_end_time":                 "15:15",
	"trading_days":                     `[]`,
	"stagnation_threshold_pct":         "0",
	"stagnation_duration_min":          "0",
	"stagnation_partial_exit_enabled":  "false",
	"stagnation_bidask_sell_threshold": "1.0",
	"min_trading_value":                "0",
	"buy_pause_start":                  "",
	"buy_pause_end":                    "",
	"index_codes":                      `[]`,
	"index_drop_threshold_pct":         "-1.0",
	"filter_rsi_max":                   "80",
	"filter_disparity_m5_max":          "3.0",
	"filter_high_price_diff_min":       "-5.0",
	"filter_open_price_diff_max":       "20.0",
	"min_score_threshold":              "0",
	"score_weight_strength":            "30",
	"score_weight_rsi":                 "20",
	"score_weight_macd":                "20",
	"score_weight_bidask":              "15",
	"score_weight_vwap":                "10",
	"score_weight_volume":              "5",
	"min_market_cap":                   "0",
	"min_expected_profit_pct":          "0",
	"vwap_diff_min":                    "0",
	"vwap_diff_max":                    "1.5",
	"rsi_buy_min":                      "40",
	"rsi_buy_max":                      "60",
	"bid_ask_ratio_min":                "1.2",
	"hard_disparity_m5_min":            "-1.5",
	"hard_disparity_m5_max":            "3.0",
	"hard_high_price_diff_max":         "-0.5",
	"hard_high_price_diff_min":         "-5.0",
	"hard_prev_vol_ratio_max":          "1.2",
	"hard_strength_min":                "100",
	"hard_rsi_max":                     "70",
	"hard_open_price_diff_max":         "15",
	"hard_macd_bearish_enabled":        "false",
	"hard_ma60_support_enabled":        "false",
	"hard_ma120_support_enabled":       "false",
	"hard_high_formed_mins_max":        "0",
	"hard_vol_vs3_avg_ratio_min":       "0",
	"hard_relative_strength_min":       "0",
	"block_reentry_on_loss":            "true",
	"reentry_score_penalty":            "10.0",
	"reentry_cooldown_min":             "15",
	"universal_cooldown_min":           "20",
	"universal_price_guard":            "true",
}

// initDefaults writes default settings on first run (using Set with MergeAll so existing values are preserved).
// It creates the settings document with all defaults only if it does not yet exist.
func (db *DB) initDefaults(ctx context.Context) error {
	ref := db.client.Collection(colSettings).Doc(docSettings)
	snap, err := ref.Get(ctx)
	if err == nil && snap.Exists() {
		return nil // already initialised
	}
	_, err = ref.Set(ctx, defaultSettings)
	return err
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func ps(m map[string]string, key, def string) string {
	if v, ok := m[key]; ok && v != "" {
		return v
	}
	return def
}

func pf(m map[string]string, key string, def float64) float64 {
	if v, ok := m[key]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func pi(m map[string]string, key string, def int) int {
	if v, ok := m[key]; ok {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func pb(m map[string]string, key string, def bool) bool {
	if v, ok := m[key]; ok {
		return v == "true" || v == "1"
	}
	return def
}

func pjsonStrSlice(m map[string]string, key string, def []string) []string {
	v, ok := m[key]
	if !ok || v == "" {
		return def
	}
	var s []string
	if err := json.Unmarshal([]byte(v), &s); err != nil {
		return def
	}
	return s
}

func pjsonIntSlice(m map[string]string, key string, def []int) []int {
	v, ok := m[key]
	if !ok || v == "" {
		return def
	}
	var s []int
	if err := json.Unmarshal([]byte(v), &s); err != nil {
		return def
	}
	return s
}

// DeleteOrder removes an order by its local ID.
func (db *DB) DeleteOrder(ctx context.Context, id int64) error {
	_, err := db.client.Collection(colOrders).Doc(strconv.FormatInt(id, 10)).Delete(ctx)
	return err
}

// DailyPnLEntry holds the realized profit/loss for a single trading day.
type DailyPnLEntry struct {
	Date         string  `json:"date"`
	ProfitAmount float64 `json:"profit_amount"`
	TradeCount   int     `json:"trade_count"`
}

// GetDailyPnL returns realized PnL aggregated by day for the most recent N days.
// It queries completed trade_reports (sold_at != nil) and groups by KST date.
func (db *DB) GetDailyPnL(ctx context.Context, days int) ([]DailyPnLEntry, error) {
	kst, _ := time.LoadLocation("Asia/Seoul")
	cutoff := time.Now().In(kst).AddDate(0, 0, -days).Truncate(24 * time.Hour)

	iter := db.client.Collection(colTradeRpt).
		Where("sold_at", ">=", cutoff).
		Documents(ctx)
	defer iter.Stop()

	// aggregate by KST date
	type agg struct {
		profit float64
		count  int
	}
	byDate := map[string]*agg{}

	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var r models.TradeReport
		if err := snap.DataTo(&r); err != nil {
			continue
		}
		if r.SoldAt == nil {
			continue
		}
		dateKey := r.SoldAt.In(kst).Format("2006-01-02")
		if byDate[dateKey] == nil {
			byDate[dateKey] = &agg{}
		}
		byDate[dateKey].profit += r.ProfitAmount
		byDate[dateKey].count++
	}

	entries := make([]DailyPnLEntry, 0, len(byDate))
	for date, a := range byDate {
		entries = append(entries, DailyPnLEntry{
			Date:         date,
			ProfitAmount: a.profit,
			TradeCount:   a.count,
		})
	}
	// sort descending by date
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].Date < entries[j].Date {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
	return entries, nil
}
