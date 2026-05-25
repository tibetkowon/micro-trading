package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/micro-trading-for-agent/backend/internal/models"
)

type DB struct {
	sql *sql.DB
}

// New opens the SQLite database and applies the schema migration.
func New(path string) (*DB, error) {
	if path == "" {
		path = "/data/trading.db"
	}
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create sqlite dir: %w", err)
		}
	}
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(3)
	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	d := &DB{sql: db}
	if err := d.initDefaults(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("initDefaults: %w", err)
	}
	return d, nil
}

func (db *DB) Close() error { return db.sql.Close() }

func (db *DB) SQLDB() *sql.DB { return db.sql }

func newID() int64 { return time.Now().UnixMicro() }

func timeString(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func nullTimeString(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return timeString(t)
}

func nullPtrTimeString(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return timeString(*t)
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func scanTime(ns sql.NullString) time.Time {
	if !ns.Valid {
		return time.Time{}
	}
	return parseTime(ns.String)
}

// ─── Settings ────────────────────────────────────────────────────────────────

func (db *DB) GetSetting(ctx context.Context, key string) string {
	v, _ := db.GetSettingErr(ctx, key)
	return v
}

func (db *DB) GetSettingErr(ctx context.Context, key string) (string, error) {
	var value string
	err := db.sql.QueryRowContext(ctx, "SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (db *DB) GetAllSettings(ctx context.Context) (map[string]string, error) {
	rows, err := db.sql.QueryContext(ctx, "SELECT key, value FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		result[k] = v
	}
	return result, rows.Err()
}

func (db *DB) SetSetting(ctx context.Context, key, value string) error {
	_, err := db.sql.ExecContext(ctx, `INSERT INTO settings(key,value,updated_at) VALUES(?,?,?)
		ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=excluded.updated_at`,
		key, value, timeString(time.Now()))
	return err
}

// TradingSettings holds all autonomous trading configuration values read from settings.
type TradingSettings struct {
	TakeProfitPct                 float64
	StopLossPct                   float64
	ETFTakeProfitPct              float64
	ETFStopLossPct                float64
	StockTakeProfitPct            float64
	StockStopLossPct              float64
	StockTaxRate                  float64
	MaxPositions                  int
	OrderAmountPct                float64
	RankingTypes                  []string
	RankingPriceMin               string
	RankingPriceMax               string
	RankingCondition              string
	RankingTopN                   int
	RankingVolumeMinIncrRate      float64
	RankingStrengthMin            float64
	RankingFluctuationMinRate     float64
	RankingFluctuationMaxRate     float64
	RankingVIKindCode             string
	RankingExchanges              []string
	RankingVolumeBlngClsCodes     []string
	RankingExcludeCls             string
	HardWatchSymbols              []string
	RankLeaseDurationMin          int
	SellConditions                []string
	IndicatorCheckIntervalMin     int
	IndicatorRSISellThreshold     float64
	IndicatorMACDBearishSell      bool
	ScanIntervalMin               int
	TradingStartTime              string
	TradingEndTime                string
	StagnationThresholdPct        float64
	StagnationDurationMin         int
	StagnationPartialExitEnabled  bool
	StagnationBidAskSellThreshold float64
	TrailingTriggerPct            float64
	TrailingStopPct               float64
	TrailingMode                  string
	TickTier0StopLossTicks        int
	TickTier1TriggerPct           float64
	TickTier1TrailTicks           int
	TickTier2TriggerPct           float64
	TickTier2TrailTicks           int
	PartialTPEnabled              bool
	PartialTPPct                  float64
	PartialTPRatio                float64
	PartialTPRaiseStop            bool
	MaxConsecutiveLosses          int
	ConsecutiveLossResetOnProfit  bool
	MaxBidAskSpreadPct            float64
	SellOnUpperLimit              bool
	DailyMaxLossPct               float64
	DailyTargetProfitPct          float64
	FilterRsiMax                  float64
	FilterDisparityM5Max          float64
	FilterHighPriceDiffMin        float64
	FilterOpenPriceDiffMax        float64
	MinScoreThreshold             float64
	ScoreWeightStrength           int
	ScoreWeightRSI                int
	ScoreWeightMACD               int
	ScoreWeightBidAsk             int
	ScoreWeightVWAP               int
	ScoreWeightVolume             int
	ScoreWeightProgramBuy         int
	ScoreWeightMicroBidAsk        int
	ScoreWeightVIDisparity        int
	MinTradingValue               float64
	StreamBypassEnabled           bool
	StreamBigTradeAmount          float64
	StreamVelocityThreshold       float64
	BuyPauseStart                 string
	BuyPauseEnd                   string
	IndexCodes                    []string
	IndexDropThresholdPct         float64
	TradingDays                   []int
	MinMarketCap                  float64
	MinExpectedProfitPct          float64
	VWAPDiffMin                   float64
	VWAPDiffMax                   float64
	RSIBuyMin                     float64
	RSIBuyMax                     float64
	BidAskRatioMin                float64
	HardDisparityM5Min            float64
	HardDisparityM5Max            float64
	HardHighPriceDiffMax          float64
	HardHighPriceDiffMin          float64
	HardPrevVolRatioMax           float64
	HardStrengthMin               float64
	HardRSIMax                    float64
	HardOpenPriceDiffMax          float64
	HardProgramBuyMin             float64
	HardMACDBearishEnabled        bool
	HardMA60SupportEnabled        bool
	HardMA120SupportEnabled       bool
	HardHighFormedMinsMax         float64
	HardVolVs3AvgRatioMin         float64
	HardRelativeStrengthMin       float64
	BlockReentryOnLoss            bool
	ReentryScorePenalty           float64
	ReentryCooldownMin            int
	LossCooldownMin               int
	LossReentryPriceGuard         bool
	UniversalCooldownMin          int
	UniversalPriceGuard           bool
	IndicatorSellMinLossPct       float64
	HardPeakTurnEnabled           bool
	HardPeakRSIMin                float64
	BuyOrderType                  string
}

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

// ─── Tokens ──────────────────────────────────────────────────────────────────

func (db *DB) GetCurrentToken(ctx context.Context) (*models.Token, error) {
	var tok models.Token
	var issuedAt, expiresAt sql.NullString
	err := db.sql.QueryRowContext(ctx, "SELECT access_token, issued_at, expires_at FROM tokens WHERE id='current'").
		Scan(&tok.AccessToken, &issuedAt, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	tok.IssuedAt = scanTime(issuedAt)
	tok.ExpiresAt = scanTime(expiresAt)
	return &tok, nil
}

func (db *DB) SaveToken(ctx context.Context, tok *models.Token) error {
	now := time.Now().UTC()
	if tok.IssuedAt.IsZero() {
		tok.IssuedAt = now
	}
	if tok.ExpiresAt.IsZero() {
		tok.ExpiresAt = now.Add(48 * time.Hour)
	}
	_, err := db.sql.ExecContext(ctx, `INSERT INTO tokens(id,access_token,issued_at,expires_at,updated_at)
		VALUES('current',?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET access_token=excluded.access_token, issued_at=excluded.issued_at,
		expires_at=excluded.expires_at, updated_at=excluded.updated_at`,
		tok.AccessToken, timeString(tok.IssuedAt), timeString(tok.ExpiresAt), timeString(now))
	return err
}

func (db *DB) DeleteAllTokens(ctx context.Context) error {
	_, err := db.sql.ExecContext(ctx, "DELETE FROM tokens")
	return err
}

// ─── Orders ──────────────────────────────────────────────────────────────────

func (db *DB) CreateOrder(ctx context.Context, o *models.Order) (int64, error) {
	if o.ID == 0 {
		o.ID = newID()
	}
	if o.Source == "" {
		o.Source = models.OrderSourceAgent
	}
	if o.CreatedAt.IsZero() {
		o.CreatedAt = time.Now().UTC()
	}
	_, err := db.sql.ExecContext(ctx, `INSERT OR REPLACE INTO orders(id,kis_order_id,stock_code,stock_name,order_type,
		qty,price,filled_price,status,source,target_pct,stop_pct,sell_reason,created_at,expire_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		o.ID, o.KISOrderID, o.StockCode, o.StockName, string(o.OrderType), o.Qty, o.Price, o.FilledPrice,
		string(o.Status), string(o.Source), o.TargetPct, o.StopPct, o.SellReason, timeString(o.CreatedAt), nullTimeString(o.ExpireAt))
	return o.ID, err
}

func (db *DB) scanOrder(row interface {
	Scan(dest ...any) error
}) (*models.Order, error) {
	var o models.Order
	var orderType, status, source string
	var createdAt, expireAt sql.NullString
	if err := row.Scan(&o.ID, &o.KISOrderID, &o.StockCode, &o.StockName, &orderType, &o.Qty, &o.Price,
		&o.FilledPrice, &status, &source, &o.TargetPct, &o.StopPct, &o.SellReason, &createdAt, &expireAt); err != nil {
		return nil, err
	}
	o.OrderType = models.OrderType(orderType)
	o.Status = models.OrderStatus(status)
	o.Source = models.OrderSource(source)
	o.CreatedAt = scanTime(createdAt)
	o.ExpireAt = scanTime(expireAt)
	return &o, nil
}

func (db *DB) GetOrderByID(ctx context.Context, id int64) (*models.Order, error) {
	o, err := db.scanOrder(db.sql.QueryRowContext(ctx, `SELECT id,kis_order_id,stock_code,stock_name,order_type,qty,price,
		filled_price,status,source,target_pct,stop_pct,sell_reason,created_at,expire_at FROM orders WHERE id=?`, id))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("order %d not found", id)
	}
	return o, err
}

func (db *DB) GetOrderByKISID(ctx context.Context, kisOrderID string) (*models.Order, error) {
	o, err := db.scanOrder(db.sql.QueryRowContext(ctx, `SELECT id,kis_order_id,stock_code,stock_name,order_type,qty,price,
		filled_price,status,source,target_pct,stop_pct,sell_reason,created_at,expire_at FROM orders WHERE kis_order_id=? LIMIT 1`, kisOrderID))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no order with kis_order_id=%s", kisOrderID)
	}
	return o, err
}

func (db *DB) UpdateOrderStatus(ctx context.Context, id int64, status models.OrderStatus) error {
	_, err := db.sql.ExecContext(ctx, "UPDATE orders SET status=? WHERE id=?", string(status), id)
	return err
}

func (db *DB) UpdateOrderFilled(ctx context.Context, kisOrderID string, status models.OrderStatus, filledPrice float64) error {
	_, err := db.sql.ExecContext(ctx, "UPDATE orders SET status=?, filled_price=? WHERE kis_order_id=?",
		string(status), filledPrice, kisOrderID)
	return err
}

func (db *DB) UpdateOrderStockName(ctx context.Context, kisOrderID, stockName string) error {
	if stockName == "" {
		return nil
	}
	_, err := db.sql.ExecContext(ctx, "UPDATE orders SET stock_name=? WHERE kis_order_id=? AND (stock_name IS NULL OR stock_name='')",
		stockName, kisOrderID)
	return err
}

func (db *DB) UpdateOrderSellReason(ctx context.Context, id int64, reason string) error {
	_, err := db.sql.ExecContext(ctx, "UPDATE orders SET sell_reason=? WHERE id=?", reason, id)
	return err
}

func (db *DB) UpsertManualOrder(ctx context.Context, o *models.Order) error {
	if existing, err := db.GetOrderByKISID(ctx, o.KISOrderID); err == nil && existing != nil {
		return nil
	}
	o.Source = models.OrderSourceManual
	_, err := db.CreateOrder(ctx, o)
	return err
}

func (db *DB) ListOrders(ctx context.Context, limit, offset int) ([]models.Order, error) {
	rows, err := db.sql.QueryContext(ctx, `SELECT id,kis_order_id,stock_code,stock_name,order_type,qty,price,
		filled_price,status,source,target_pct,stop_pct,sell_reason,created_at,expire_at
		FROM orders ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var orders []models.Order
	for rows.Next() {
		o, err := db.scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, *o)
	}
	return orders, rows.Err()
}

func (db *DB) ListOrdersSince(ctx context.Context, since time.Time, limit, offset int) ([]models.Order, error) {
	rows, err := db.sql.QueryContext(ctx, `SELECT id,kis_order_id,stock_code,stock_name,order_type,qty,price,
		filled_price,status,source,target_pct,stop_pct,sell_reason,created_at,expire_at
		FROM orders WHERE datetime(created_at) >= datetime(?) ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		since.UTC().Format("2006-01-02 15:04:05"), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var orders []models.Order
	for rows.Next() {
		o, err := db.scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, *o)
	}
	return orders, rows.Err()
}

func (db *DB) GetLastAgentBuyOrder(ctx context.Context, stockCode string) (*models.Order, error) {
	o, err := db.scanOrder(db.sql.QueryRowContext(ctx, `SELECT id,kis_order_id,stock_code,stock_name,order_type,qty,price,
		filled_price,status,source,target_pct,stop_pct,sell_reason,created_at,expire_at
		FROM orders WHERE stock_code=? AND order_type=? AND status=? AND source=?
		ORDER BY created_at DESC LIMIT 1`, stockCode, string(models.OrderTypeBuy), string(models.OrderStatusFilled), string(models.OrderSourceAgent)))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return o, err
}

func (db *DB) CountOrders(ctx context.Context) (int, error) {
	var n int
	err := db.sql.QueryRowContext(ctx, "SELECT COUNT(*) FROM orders").Scan(&n)
	return n, err
}

func (db *DB) CountOrdersSince(ctx context.Context, since time.Time) (int, error) {
	var n int
	err := db.sql.QueryRowContext(ctx, "SELECT COUNT(*) FROM orders WHERE datetime(created_at) >= datetime(?)",
		since.UTC().Format("2006-01-02 15:04:05")).Scan(&n)
	return n, err
}

func (db *DB) DeleteOrder(ctx context.Context, id int64) error {
	_, err := db.sql.ExecContext(ctx, "DELETE FROM orders WHERE id=?", id)
	return err
}

// ─── MonitoredPositions ───────────────────────────────────────────────────────

func (db *DB) CreatePosition(ctx context.Context, p *models.MonitoredPosition) error {
	if p.ID == 0 {
		p.ID = newID()
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now().UTC()
	}
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	now := timeString(time.Now())
	_, err = db.sql.ExecContext(ctx, `INSERT OR REPLACE INTO monitored_positions(code,data,created_at,updated_at)
		VALUES(?,?,COALESCE((SELECT created_at FROM monitored_positions WHERE code=?),?),?)`,
		p.StockCode, string(data), p.StockCode, timeString(p.CreatedAt), now)
	return err
}

func (db *DB) DeletePosition(ctx context.Context, stockCode string) error {
	_, err := db.sql.ExecContext(ctx, "DELETE FROM monitored_positions WHERE code=?", stockCode)
	return err
}

func (db *DB) UpdatePositionQty(ctx context.Context, stockCode string, remainingQty int) error {
	var blob string
	if err := db.sql.QueryRowContext(ctx, "SELECT data FROM monitored_positions WHERE code=?", stockCode).Scan(&blob); err != nil {
		return err
	}
	var p models.MonitoredPosition
	if err := json.Unmarshal([]byte(blob), &p); err != nil {
		return err
	}
	p.RemainingQty = remainingQty
	updated, err := json.Marshal(&p)
	if err != nil {
		return err
	}
	_, err = db.sql.ExecContext(ctx, "UPDATE monitored_positions SET data=?, updated_at=? WHERE code=?",
		string(updated), timeString(time.Now()), stockCode)
	return err
}

func (db *DB) ListPositions(ctx context.Context) ([]models.MonitoredPosition, error) {
	rows, err := db.sql.QueryContext(ctx, "SELECT data FROM monitored_positions ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var positions []models.MonitoredPosition
	for rows.Next() {
		var blob string
		if err := rows.Scan(&blob); err != nil {
			return nil, err
		}
		var p models.MonitoredPosition
		if err := json.Unmarshal([]byte(blob), &p); err != nil {
			slog.Warn("failed to unmarshal monitored position", "error", err)
			continue
		}
		positions = append(positions, p)
	}
	return positions, rows.Err()
}

// ─── Balances ─────────────────────────────────────────────────────────────────

func (db *DB) CreateBalance(ctx context.Context, totalEval, availableAmt float64) error {
	now := time.Now().UTC()
	bal := models.Balance{
		ID:              newID(),
		TotalEval:       totalEval,
		AvailableAmount: availableAmt,
		RecordedAt:      now,
		ExpireAt:        now.Add(30 * 24 * time.Hour),
	}
	data, err := json.Marshal(&bal)
	if err != nil {
		return err
	}
	_, err = db.sql.ExecContext(ctx, "INSERT INTO balances(id,data,created_at) VALUES(?,?,?)",
		bal.ID, string(data), timeString(bal.RecordedAt))
	return err
}

func (db *DB) GetLatestBalance(ctx context.Context) (*models.Balance, error) {
	var blob string
	err := db.sql.QueryRowContext(ctx, "SELECT data FROM balances ORDER BY created_at DESC LIMIT 1").Scan(&blob)
	if err != nil {
		return nil, fmt.Errorf("no balance snapshot")
	}
	var b models.Balance
	return &b, json.Unmarshal([]byte(blob), &b)
}

// ─── ScanLogs ─────────────────────────────────────────────────────────────────

func (db *DB) CreateScanLog(ctx context.Context, sl *models.ScanLog) (int64, error) {
	if sl.ID == 0 {
		sl.ID = newID()
	}
	if sl.Timestamp == "" {
		sl.Timestamp = timeString(time.Now())
	}
	_, err := db.sql.ExecContext(ctx, `INSERT OR REPLACE INTO scan_logs(id,timestamp,total_candidates,stocks_found,top_stocks,
		stock_raw_data,below_threshold_data,ordered,ordered_code,skip_reason,score_stats,created_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, sl.ID, sl.Timestamp, sl.TotalCandidates, sl.StocksFound,
		sl.TopStocks, sl.StockRawData, sl.BelowThresholdData, boolInt(sl.Ordered), sl.OrderedCode,
		sl.SkipReason, sl.ScoreStats, timeString(time.Now()))
	return sl.ID, err
}

func (db *DB) scanScanLog(row interface {
	Scan(dest ...any) error
}) (models.ScanLog, error) {
	var l models.ScanLog
	var ordered int
	err := row.Scan(&l.ID, &l.Timestamp, &l.TotalCandidates, &l.StocksFound, &l.TopStocks,
		&l.StockRawData, &l.BelowThresholdData, &ordered, &l.OrderedCode, &l.SkipReason, &l.ScoreStats)
	l.Ordered = ordered != 0
	return l, err
}

func (db *DB) ListScanLogs(ctx context.Context, limit int) ([]models.ScanLog, error) {
	rows, err := db.sql.QueryContext(ctx, `SELECT id,timestamp,total_candidates,stocks_found,top_stocks,
		stock_raw_data,below_threshold_data,ordered,ordered_code,skip_reason,score_stats
		FROM scan_logs ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []models.ScanLog
	for rows.Next() {
		l, err := db.scanScanLog(rows)
		if err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

func (db *DB) GetScanLogsByDate(ctx context.Context, date string) ([]models.ScanLog, error) {
	kst, _ := time.LoadLocation("Asia/Seoul")
	day, err := time.ParseInLocation("2006-01-02", date, kst)
	if err != nil {
		return nil, err
	}
	start := timeString(day.UTC())
	end := timeString(day.Add(24*time.Hour - time.Second).UTC())
	rows, err := db.sql.QueryContext(ctx, `SELECT id,timestamp,total_candidates,stocks_found,top_stocks,
		stock_raw_data,below_threshold_data,ordered,ordered_code,skip_reason,score_stats
		FROM scan_logs WHERE timestamp >= ? AND timestamp <= ? ORDER BY timestamp ASC`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []models.ScanLog
	for rows.Next() {
		l, err := db.scanScanLog(rows)
		if err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

func (db *DB) GetScanLog(ctx context.Context, id int64) (*models.ScanLog, error) {
	l, err := db.scanScanLog(db.sql.QueryRowContext(ctx, `SELECT id,timestamp,total_candidates,stocks_found,top_stocks,
		stock_raw_data,below_threshold_data,ordered,ordered_code,skip_reason,score_stats FROM scan_logs WHERE id=?`, id))
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// ─── TradeReports ─────────────────────────────────────────────────────────────

func (db *DB) saveTradeReport(ctx context.Context, r *models.TradeReport) error {
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	_, err = db.sql.ExecContext(ctx, `INSERT OR REPLACE INTO trade_reports(id,data,date,stock_code,buy_order_id,sell_order_id,created_at,sold_at)
		VALUES(?,?,?,?,?,?,?,?)`, r.ID, string(data), r.Date, r.StockCode, r.BuyOrderID, r.SellOrderID,
		timeString(r.CreatedAt), nullPtrTimeString(r.SoldAt))
	return err
}

func (db *DB) CreateTradeReport(ctx context.Context, r *models.TradeReport) (int64, error) {
	if r.ID == 0 {
		r.ID = newID()
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now().UTC()
	}
	r.ExpireAt = time.Now().UTC().Add(180 * 24 * time.Hour)
	return r.ID, db.saveTradeReport(ctx, r)
}

func (db *DB) getTradeReportByQuery(ctx context.Context, query string, args ...any) (*models.TradeReport, error) {
	var blob string
	if err := db.sql.QueryRowContext(ctx, query, args...).Scan(&blob); err != nil {
		return nil, err
	}
	var r models.TradeReport
	return &r, json.Unmarshal([]byte(blob), &r)
}

func (db *DB) UpdateTradeReportOnSell(ctx context.Context, buyOrderID, sellOrderID int64, sellPrice float64, sellQty int, sellReason, sellIndicators string) error {
	r, err := db.getTradeReportByQuery(ctx, "SELECT data FROM trade_reports WHERE buy_order_id=? LIMIT 1", buyOrderID)
	if err != nil {
		return fmt.Errorf("UpdateTradeReportOnSell: no trade report found for buy_order_id=%d", buyOrderID)
	}
	now := time.Now().UTC()
	r.SellOrderID = sellOrderID
	r.SellPrice = sellPrice
	r.SellQty = sellQty
	r.SellAmount = sellPrice * float64(sellQty)
	r.SellReason = sellReason
	r.SellIndicators = sellIndicators
	r.ProfitAmount = (sellPrice - r.BuyPrice) * float64(sellQty)
	if r.BuyPrice > 0 {
		r.ProfitPct = (sellPrice - r.BuyPrice) / r.BuyPrice * 100
	}
	r.SoldAt = &now
	return db.saveTradeReport(ctx, r)
}

func (db *DB) UpdateTradeReportSell(ctx context.Context, id int64, updates map[string]any) error {
	r, err := db.GetTradeReport(ctx, id)
	if err != nil {
		return err
	}
	for k, v := range updates {
		switch k {
		case "sell_order_id":
			r.SellOrderID = asInt64(v)
		case "sell_price":
			r.SellPrice = asFloat(v)
		case "sell_qty":
			r.SellQty = int(asInt64(v))
		case "sell_amount":
			r.SellAmount = asFloat(v)
		case "sell_reason":
			r.SellReason, _ = v.(string)
		case "sell_indicators":
			r.SellIndicators, _ = v.(string)
		case "profit_amount":
			r.ProfitAmount = asFloat(v)
		case "profit_pct":
			r.ProfitPct = asFloat(v)
		case "sold_at":
			if t, ok := v.(time.Time); ok {
				r.SoldAt = &t
			}
		}
	}
	return db.saveTradeReport(ctx, r)
}

func (db *DB) GetTradeReport(ctx context.Context, id int64) (*models.TradeReport, error) {
	return db.getTradeReportByQuery(ctx, "SELECT data FROM trade_reports WHERE id=?", id)
}

func (db *DB) ListOpenTradeReports(ctx context.Context) ([]models.TradeReport, error) {
	rows, err := db.sql.QueryContext(ctx, "SELECT data FROM trade_reports WHERE sell_order_id=0 ORDER BY created_at DESC")
	return db.readTradeReports(rows, err)
}

func (db *DB) GetCompletedTradesBySoldDate(ctx context.Context, date string) ([]models.TradeReport, error) {
	kst, _ := time.LoadLocation("Asia/Seoul")
	start, err := time.ParseInLocation("2006-01-02", date, kst)
	if err != nil {
		return nil, fmt.Errorf("parse date: %w", err)
	}
	end := start.Add(24 * time.Hour)
	rows, err := db.sql.QueryContext(ctx, "SELECT data FROM trade_reports WHERE sold_at >= ? AND sold_at < ? ORDER BY sold_at ASC",
		timeString(start.UTC()), timeString(end.UTC()))
	return db.readTradeReports(rows, err)
}

func (db *DB) ListTradeReports(ctx context.Context, limit int) ([]models.TradeReport, error) {
	rows, err := db.sql.QueryContext(ctx, "SELECT data FROM trade_reports ORDER BY created_at DESC LIMIT ?", limit)
	return db.readTradeReports(rows, err)
}

func (db *DB) GetLatestCompletedTradeByStock(ctx context.Context, stockCode string) (*models.TradeReport, error) {
	r, err := db.getTradeReportByQuery(ctx, `SELECT data FROM trade_reports
		WHERE stock_code=? AND sold_at IS NOT NULL ORDER BY sold_at DESC LIMIT 1`, stockCode)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return r, err
}

func (db *DB) readTradeReports(rows *sql.Rows, err error) ([]models.TradeReport, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reports []models.TradeReport
	for rows.Next() {
		var blob string
		if err := rows.Scan(&blob); err != nil {
			return nil, err
		}
		var r models.TradeReport
		if err := json.Unmarshal([]byte(blob), &r); err != nil {
			return nil, err
		}
		reports = append(reports, r)
	}
	return reports, rows.Err()
}

// ─── DailyReports / Simulation ────────────────────────────────────────────────

func (db *DB) InsertOrUpdateDailyReport(ctx context.Context, dr models.DailyReport) error {
	if dr.ID == 0 {
		dr.ID = newID()
	}
	if dr.CreatedAt.IsZero() {
		dr.CreatedAt = time.Now().UTC()
	}
	dr.ExpireAt = time.Now().UTC().Add(180 * 24 * time.Hour)
	data, err := json.Marshal(&dr)
	if err != nil {
		return err
	}
	now := timeString(time.Now())
	_, err = db.sql.ExecContext(ctx, `INSERT INTO daily_reports(date,data,created_at,updated_at) VALUES(?,?,?,?)
		ON CONFLICT(date) DO UPDATE SET data=excluded.data, updated_at=excluded.updated_at`,
		dr.Date, string(data), timeString(dr.CreatedAt), now)
	return err
}

func (db *DB) GetDailyReport(ctx context.Context, date string) (*models.DailyReport, error) {
	var blob string
	if err := db.sql.QueryRowContext(ctx, "SELECT data FROM daily_reports WHERE date=?", date).Scan(&blob); err != nil {
		return nil, err
	}
	var dr models.DailyReport
	return &dr, json.Unmarshal([]byte(blob), &dr)
}

func (db *DB) ListDailyReports(ctx context.Context, limit int) ([]models.DailyReport, error) {
	rows, err := db.sql.QueryContext(ctx, "SELECT data FROM daily_reports ORDER BY date DESC LIMIT ?", limit)
	return db.readDailyReports(rows, err)
}

func (db *DB) ListDailyReportsByDateRange(ctx context.Context, from, to string) ([]models.DailyReport, error) {
	rows, err := db.sql.QueryContext(ctx, "SELECT data FROM daily_reports WHERE date >= ? AND date <= ? ORDER BY date ASC", from, to)
	return db.readDailyReports(rows, err)
}

func (db *DB) readDailyReports(rows *sql.Rows, err error) ([]models.DailyReport, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reports []models.DailyReport
	for rows.Next() {
		var blob string
		if err := rows.Scan(&blob); err != nil {
			return nil, err
		}
		var dr models.DailyReport
		if err := json.Unmarshal([]byte(blob), &dr); err != nil {
			return nil, err
		}
		reports = append(reports, dr)
	}
	return reports, rows.Err()
}

func (db *DB) UpsertSimulationResult(ctx context.Context, r models.SimulationResult) error {
	if r.ID == 0 {
		r.ID = newID()
	}
	r.CreatedAt = time.Now().UTC()
	r.ExpireAt = time.Now().UTC().Add(30 * 24 * time.Hour)
	data, err := json.Marshal(&r)
	if err != nil {
		return err
	}
	_, err = db.sql.ExecContext(ctx, `INSERT INTO simulation_results(date,data,created_at,updated_at) VALUES(?,?,?,?)
		ON CONFLICT(date) DO UPDATE SET data=excluded.data, updated_at=excluded.updated_at`,
		r.Date, string(data), timeString(r.CreatedAt), timeString(time.Now()))
	return err
}

func (db *DB) GetSimulationResult(ctx context.Context, date string) (*models.SimulationResult, error) {
	var blob string
	if err := db.sql.QueryRowContext(ctx, "SELECT data FROM simulation_results WHERE date=?", date).Scan(&blob); err != nil {
		return nil, err
	}
	var r models.SimulationResult
	return &r, json.Unmarshal([]byte(blob), &r)
}

// ─── Admin / Maintenance ─────────────────────────────────────────────────────

func (db *DB) ListTable(ctx context.Context, table string, page, limit int) ([]map[string]any, int, error) {
	allowed := map[string]bool{
		"orders": true, "monitored_positions": true, "scan_logs": true,
		"trade_reports": true, "daily_reports": true, "simulation_results": true,
		"balances": true, "settings": true, "tokens": true,
	}
	if !allowed[table] {
		return nil, 0, fmt.Errorf("table not allowed: %s", table)
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}
	var total int
	if err := db.sql.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&total); err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * limit
	rows, err := db.sql.QueryContext(ctx, fmt.Sprintf("SELECT * FROM %s LIMIT ? OFFSET ?", table), limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, 0, err
	}
	var results []map[string]any
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, 0, err
		}
		row := map[string]any{}
		for i, col := range cols {
			if b, ok := vals[i].([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = vals[i]
			}
		}
		results = append(results, row)
	}
	return results, total, rows.Err()
}

func (db *DB) PurgeOldRecords(ctx context.Context) error {
	stmts := []struct{ table, col, cutoff string }{
		{"scan_logs", "created_at", timeString(time.Now().AddDate(0, 0, -30))},
		{"balances", "created_at", timeString(time.Now().AddDate(0, 0, -90))},
		{"trade_reports", "created_at", timeString(time.Now().AddDate(0, -6, 0))},
	}
	for _, s := range stmts {
		res, err := db.sql.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE %s < ?", s.table, s.col), s.cutoff)
		if err != nil {
			return fmt.Errorf("purge %s: %w", s.table, err)
		}
		if n, _ := res.RowsAffected(); n > 0 {
			slog.Info("purged old records", "table", s.table, "count", n)
		}
	}
	return nil
}

type DailyPnLEntry struct {
	Date         string  `json:"date"`
	ProfitAmount float64 `json:"profit_amount"`
	TradeCount   int     `json:"trade_count"`
}

func (db *DB) GetDailyPnL(ctx context.Context, days int) ([]DailyPnLEntry, error) {
	kst, _ := time.LoadLocation("Asia/Seoul")
	cutoff := time.Now().In(kst).AddDate(0, 0, -days).Truncate(24 * time.Hour)
	rows, err := db.sql.QueryContext(ctx, "SELECT data FROM trade_reports WHERE sold_at >= ?", timeString(cutoff.UTC()))
	if err != nil {
		return nil, err
	}
	reports, err := db.readTradeReports(rows, nil)
	if err != nil {
		return nil, err
	}
	byDate := map[string]*struct {
		profit float64
		count  int
	}{}
	for _, r := range reports {
		if r.SoldAt == nil {
			continue
		}
		dateKey := r.SoldAt.In(kst).Format("2006-01-02")
		if byDate[dateKey] == nil {
			byDate[dateKey] = &struct {
				profit float64
				count  int
			}{}
		}
		byDate[dateKey].profit += r.ProfitAmount
		byDate[dateKey].count++
	}
	entries := make([]DailyPnLEntry, 0, len(byDate))
	for date, a := range byDate {
		entries = append(entries, DailyPnLEntry{Date: date, ProfitAmount: a.profit, TradeCount: a.count})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Date > entries[j].Date })
	return entries, nil
}

// ─── Defaults / parse helpers ─────────────────────────────────────────────────

func (db *DB) initDefaults(ctx context.Context) error {
	var count int
	if err := db.sql.QueryRowContext(ctx, "SELECT COUNT(*) FROM settings").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	for k, v := range defaultSettings {
		if err := db.SetSetting(ctx, k, fmt.Sprintf("%v", v)); err != nil {
			return err
		}
	}
	return nil
}

var defaultSettings = map[string]any{
	"trading_enabled": "true", "max_positions": "1", "order_amount_pct": "95",
	"daily_max_loss_pct": "0", "daily_target_profit_pct": "0",
	"stock_take_profit_pct": "1.5", "stock_stop_loss_pct": "1.0",
	"etf_take_profit_pct": "0.5", "etf_stop_loss_pct": "1.0",
	"take_profit_pct": "3.0", "stop_loss_pct": "2.0", "stock_tax_rate": "0",
	"trailing_trigger_pct": "0", "trailing_stop_pct": "0", "trailing_mode": "pct",
	"tick_tier0_stop_loss_ticks": "3", "tick_tier1_trigger_pct": "0",
	"tick_tier1_trail_ticks": "5", "tick_tier2_trigger_pct": "0", "tick_tier2_trail_ticks": "2",
	"partial_tp_enabled": "false", "partial_tp_pct": "1.0", "partial_tp_ratio": "0.5",
	"partial_tp_raise_stop": "false", "sell_on_upper_limit": "false",
	"max_consecutive_losses":           "0",
	"consecutive_loss_reset_on_profit": "true", "max_bidask_spread_pct": "0",
	"indicator_check_interval_min": "5", "indicator_rsi_sell_threshold": "70",
	"indicator_macd_bearish_sell": "false", "indicator_sell_min_loss_pct": "1.0",
	"trading_start_time": "09:15", "trading_end_time": "15:15", "trading_days": `[]`,
	"stagnation_threshold_pct": "0", "stagnation_duration_min": "0",
	"stagnation_partial_exit_enabled": "false", "stagnation_bidask_sell_threshold": "1.0",
	"min_trading_value": "0", "buy_pause_start": "", "buy_pause_end": "",
	"index_codes": `[]`, "index_drop_threshold_pct": "-1.0",
	"ranking_types": `["volume","strength"]`, "ranking_condition": "OR", "ranking_top_n": "30",
	"ranking_price_min": "5000", "ranking_price_max": "200000",
	"ranking_exchanges": `["0001","1001"]`, "ranking_exclude_cls": "1111111111",
	"ranking_volume_blng_cls_codes": `["0"]`, "ranking_vi_kind_code": "",
	"rank_lease_duration_min": "0", "sell_conditions": `["take_profit","stop_loss"]`,
	"filter_rsi_max": "80", "filter_disparity_m5_max": "3.0",
	"filter_high_price_diff_min": "-5.0", "filter_open_price_diff_max": "20.0",
	"min_score_threshold": "0", "score_weight_strength": "30", "score_weight_rsi": "20",
	"score_weight_macd": "20", "score_weight_bidask": "15", "score_weight_vwap": "10",
	"score_weight_volume": "5", "score_weight_program_buy": "10",
	"score_weight_micro_bidask": "10", "score_weight_vi_disparity": "10",
	"stream_bypass_enabled": "true", "stream_big_trade_amount": "30000000",
	"stream_velocity_threshold": "5.0", "min_market_cap": "0", "min_expected_profit_pct": "0",
	"vwap_diff_min": "0", "vwap_diff_max": "1.5", "rsi_buy_min": "40", "rsi_buy_max": "60",
	"bid_ask_ratio_min": "1.2", "hard_disparity_m5_min": "-1.5",
	"hard_disparity_m5_max": "3.0", "hard_high_price_diff_max": "-0.5",
	"hard_high_price_diff_min": "-5.0", "hard_prev_vol_ratio_max": "1.2",
	"hard_strength_min": "100", "hard_rsi_max": "70", "hard_open_price_diff_max": "15",
	"hard_program_buy_min": "0", "hard_macd_bearish_enabled": "false",
	"hard_ma60_support_enabled": "false", "hard_ma120_support_enabled": "false",
	"hard_high_formed_mins_max": "0", "hard_vol_vs3_avg_ratio_min": "0",
	"hard_relative_strength_min": "0", "block_reentry_on_loss": "true",
	"reentry_score_penalty": "10.0", "reentry_cooldown_min": "15",
	"loss_cooldown_min": "20", "loss_reentry_price_guard": "true",
	"universal_cooldown_min": "20", "universal_price_guard": "true",
	"hard_peak_turn_enabled": "false", "hard_peak_rsi_min": "65.0", "buy_order_type": "limit",
}

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

func asFloat(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case string:
		f, _ := strconv.ParseFloat(x, 64)
		return f
	default:
		return 0
	}
}

func asInt64(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case float64:
		return int64(x)
	case string:
		i, _ := strconv.ParseInt(x, 10, 64)
		return i
	default:
		return 0
	}
}
