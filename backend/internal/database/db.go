package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/micro-trading-for-agent/backend/internal/models"
)

// TradingSettings holds all autonomous trading configuration values.
type TradingSettings struct {
	TakeProfitPct             float64  // 익절 기준 %
	StopLossPct               float64  // 손절 기준 %
	RankingTypes              []string // 순위 유형 우선순위 (volume, strength, exec_count, disparity)
	RankingPriceMin           string   // 순위 조회 최소 주가
	RankingPriceMax           string   // 순위 조회 최대 주가
	MaxPositions              int      // 동시 보유 최대 종목 수
	OrderAmountPct            float64  // 가용자금 대비 주문 비율(%)
	SellConditions            []string // 매도 조건 우선순위 배열
	IndicatorCheckIntervalMin int      // 지표 확인 주기(분)
	IndicatorRSISellThreshold float64  // RSI 매도 기준값
	IndicatorMACDBearishSell  bool     // MACD 데드크로스 매도 여부
	ClaudeModel               string   // 사용할 Claude 모델
	// 순위별 필터
	RankingVolumeMinIncrRate   float64 // 거래량 증가율 최솟값 (0=필터없음)
	RankingStrengthMin         float64 // 체결강도 최솟값 (0=필터없음)
	RankingExecCountNetBuyOnly bool    // 대량체결: 순매수 우세 종목만
	RankingDisparityD20Min     float64 // 20일 이격도 최솟값 (0=필터없음)
	RankingDisparityD20Max     float64 // 20일 이격도 최댓값 (0=필터없음)
	RankingTopN                int     // 각 순위별 상위 N개만 교집합 대상 (0=전체)
	// 거래 시간
	TradingStartTime string // 자율 거래 시작 시간 (HH:MM)
	TradingEndTime   string // 자율 거래 종료 시간 (HH:MM)
	// 횡보 감지
	StagnationThresholdPct float64 // 횡보 판단 기준 변동폭 (%)
	StagnationDurationMin  int     // 횡보 지속 시간 (분)
	// 순위 조건
	RankingCondition string // "AND" | "OR"
	// 거래대금 하한선
	MinTradingValue float64 // 최소 거래대금(원). 0=필터없음
	// 매수 중단 시간대
	BuyPauseStart string // 매수 중단 시작 (HH:MM). 비어있으면 비활성
	BuyPauseEnd   string // 매수 중단 종료 (HH:MM)
	// 트레일링 스탑
	TrailingTriggerPct float64 // 활성화 기준 수익률(%). 0=비활성
	TrailingStopPct    float64 // 최고가 대비 하락 허용폭(%)
	// 일일 최대 손실
	DailyMaxLossPct float64 // 일일 최대 손실 한도(%). 0=제한없음
	// 지수 필터
	IndexCodes []string // 지수 코드 목록 ("0001"=코스피, "1001"=코스닥). 빈 배열=비활성
}

// DB wraps the sql.DB connection.
type DB struct {
	*sql.DB
}

// New opens (or creates) the SQLite database at the given path,
// runs schema migrations, and returns a ready-to-use DB instance.
func New(dsn string) (*DB, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(dsn)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	sqlDB, err := sql.Open("sqlite3", dsn+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// Limit connections to avoid memory pressure on NCP Micro (1 GB RAM).
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	db := &DB{sqlDB}
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

// migrate creates all required tables if they do not already exist.
func (db *DB) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS settings (
			key        TEXT PRIMARY KEY,
			value      TEXT NOT NULL DEFAULT '',
			updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,

		`CREATE TABLE IF NOT EXISTS tokens (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			access_token TEXT    NOT NULL,
			issued_at    DATETIME NOT NULL DEFAULT (datetime('now')),
			expires_at   DATETIME NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS orders (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			stock_code   TEXT    NOT NULL,
			stock_name   TEXT    NOT NULL DEFAULT '',
			order_type   TEXT    NOT NULL CHECK(order_type IN ('BUY','SELL')),
			qty          INTEGER NOT NULL CHECK(qty > 0),
			price        REAL    NOT NULL CHECK(price >= 0),
			filled_price REAL    NOT NULL DEFAULT 0,
			status       TEXT    NOT NULL DEFAULT 'PENDING'
			                CHECK(status IN ('PENDING','FILLED','PARTIALLY_FILLED','CANCELLED','FAILED')),
			kis_order_id TEXT    NOT NULL DEFAULT '',
			created_at   DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,

		`CREATE TABLE IF NOT EXISTS balances (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			total_eval       REAL    NOT NULL DEFAULT 0,
			available_amount REAL    NOT NULL DEFAULT 0,
			recorded_at      DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,

		`CREATE TABLE IF NOT EXISTS kis_api_logs (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			endpoint     TEXT    NOT NULL,
			error_code   TEXT    NOT NULL DEFAULT '',
			error_message TEXT   NOT NULL DEFAULT '',
			raw_response TEXT    NOT NULL DEFAULT '',
			timestamp    DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,

		`CREATE TABLE IF NOT EXISTS monitored_positions (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			stock_code   TEXT    NOT NULL UNIQUE,
			stock_name   TEXT    NOT NULL DEFAULT '',
			filled_price REAL    NOT NULL DEFAULT 0,
			target_price REAL    NOT NULL DEFAULT 0,
			stop_price   REAL    NOT NULL DEFAULT 0,
			order_id     INTEGER NOT NULL DEFAULT 0,
			created_at   DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,

		`CREATE TABLE IF NOT EXISTS reports (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			report_date TEXT    NOT NULL UNIQUE,
			content     TEXT    NOT NULL DEFAULT '',
			created_at  DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,

		`CREATE TABLE IF NOT EXISTS trader_selection_logs (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp       DATETIME NOT NULL DEFAULT (datetime('now')),
			sent_count      INTEGER  NOT NULL DEFAULT 0,
			candidates      TEXT     NOT NULL DEFAULT '',
			llm_result      TEXT     NOT NULL DEFAULT '',
			selected_code   TEXT     NOT NULL DEFAULT '',
			selected_reason TEXT     NOT NULL DEFAULT ''
		)`,

		`CREATE TABLE IF NOT EXISTS trader_ranking_logs (
			id                 INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp          DATETIME NOT NULL DEFAULT (datetime('now')),
			ranking_types      TEXT     NOT NULL DEFAULT '',
			price_min          TEXT     NOT NULL DEFAULT '',
			price_max          TEXT     NOT NULL DEFAULT '',
			volume_count       INTEGER  NOT NULL DEFAULT -1,
			strength_count     INTEGER  NOT NULL DEFAULT -1,
			exec_count_count   INTEGER  NOT NULL DEFAULT -1,
			disparity_count    INTEGER  NOT NULL DEFAULT -1,
			intersection_count INTEGER  NOT NULL DEFAULT 0,
			error_message      TEXT     NOT NULL DEFAULT ''
		)`,
	}

	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("exec migration: %w\nSQL: %s", err, s)
		}
	}

	// 기존 DB 인스턴스를 위한 컬럼 추가 마이그레이션 (이미 존재하면 무시)
	alterStmts := []string{
		`ALTER TABLE orders ADD COLUMN stock_name   TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE orders ADD COLUMN filled_price REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE orders ADD COLUMN source       TEXT NOT NULL DEFAULT 'AGENT'`,
		`ALTER TABLE orders ADD COLUMN target_pct   REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE orders ADD COLUMN stop_pct     REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE trader_selection_logs ADD COLUMN fail_reason TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE orders ADD COLUMN sell_reason TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE trader_ranking_logs ADD COLUMN ranking_condition TEXT NOT NULL DEFAULT 'AND'`,
	}
	for _, s := range alterStmts {
		// "duplicate column name" 에러는 정상 (이미 존재하는 경우) — 무시
		db.Exec(s) //nolint:errcheck
	}

	// Default trading settings (INSERT OR IGNORE — never overwrite user-set values)
	defaultSettings := []struct{ key, val string }{
		{"take_profit_pct", "3.0"},
		{"stop_loss_pct", "2.0"},
		{"ranking_types", `["volume","strength","exec_count","disparity"]`},
		{"ranking_price_min", "5000"},
		{"ranking_price_max", "100000"},
		{"max_positions", "1"},
		{"order_amount_pct", "95"},
		{"sell_conditions", `["target_pct","stop_pct"]`},
		{"indicator_check_interval_min", "5"},
		{"indicator_rsi_sell_threshold", "70"},
		{"indicator_macd_bearish_sell", "false"},
		{"claude_model", "claude-sonnet-4-6"},
		{"ranking_volume_min_incrrate", "0"},
		{"ranking_strength_min", "100"},
		{"ranking_execcount_net_buy_only", "true"},
		{"ranking_disparity_d20_min", "0"},
		{"ranking_disparity_d20_max", "0"},
		{"ranking_top_n", "20"},
		{"trading_start_time", "09:15"},
		{"trading_end_time", "15:15"},
		{"stagnation_threshold_pct", "1.0"},
		{"stagnation_duration_min", "30"},
		{"ranking_condition", "AND"},
		{"min_trading_value", "0"},
		{"buy_pause_start", "11:00"},
		{"buy_pause_end", "14:00"},
		{"trailing_trigger_pct", "0"},
		{"trailing_stop_pct", "1.0"},
		{"daily_max_loss_pct", "0"},
		{"index_codes", "[]"},
	}
	for _, s := range defaultSettings {
		db.Exec( //nolint:errcheck
			`INSERT OR IGNORE INTO settings (key, value, updated_at) VALUES (?, ?, datetime('now'))`,
			s.key, s.val)
	}

	return nil
}

// GetTradingSettings reads all autonomous trading settings from the DB in one call.
func (db *DB) GetTradingSettings(ctx context.Context) (TradingSettings, error) {
	keys := []string{
		"take_profit_pct", "stop_loss_pct", "ranking_types",
		"ranking_price_min", "ranking_price_max", "max_positions",
		"order_amount_pct", "sell_conditions", "indicator_check_interval_min",
		"indicator_rsi_sell_threshold", "indicator_macd_bearish_sell", "claude_model",
	}
	vals := make(map[string]string, len(keys))
	rows, err := db.QueryContext(ctx,
		`SELECT key, value FROM settings WHERE key IN (`+
			`'take_profit_pct','stop_loss_pct','ranking_types',`+
			`'ranking_price_min','ranking_price_max','max_positions',`+
			`'order_amount_pct','sell_conditions','indicator_check_interval_min',`+
			`'indicator_rsi_sell_threshold','indicator_macd_bearish_sell','claude_model',`+
			`'ranking_volume_min_incrrate','ranking_strength_min',`+
			`'ranking_execcount_net_buy_only','ranking_disparity_d20_min','ranking_disparity_d20_max',`+
			`'ranking_top_n',`+
			`'trading_start_time','trading_end_time',`+
			`'stagnation_threshold_pct','stagnation_duration_min',`+
			`'ranking_condition',`+
			`'min_trading_value',`+
			`'buy_pause_start','buy_pause_end',`+
			`'trailing_trigger_pct','trailing_stop_pct',`+
			`'daily_max_loss_pct','index_codes'`+
			`)`)
	if err != nil {
		return TradingSettings{}, fmt.Errorf("GetTradingSettings query: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err == nil {
			vals[k] = v
		}
	}
	_ = keys

	// Parse helpers
	f64 := func(k string) float64 {
		v, _ := strconv.ParseFloat(vals[k], 64)
		return v
	}
	i64 := func(k string) int {
		v, _ := strconv.Atoi(vals[k])
		return v
	}
	strSlice := func(k string) []string {
		var s []string
		if v := vals[k]; v != "" {
			_ = json.Unmarshal([]byte(v), &s)
		}
		return s
	}

	takeProfitPct := f64("take_profit_pct")
	if takeProfitPct == 0 {
		takeProfitPct = 3.0
	}
	stopLossPct := f64("stop_loss_pct")
	if stopLossPct == 0 {
		stopLossPct = 2.0
	}
	maxPositions := i64("max_positions")
	if maxPositions == 0 {
		maxPositions = 1
	}
	orderAmountPct := f64("order_amount_pct")
	if orderAmountPct == 0 {
		orderAmountPct = 95
	}
	indicatorCheckInterval := i64("indicator_check_interval_min")
	if indicatorCheckInterval == 0 {
		indicatorCheckInterval = 5
	}
	rsiThreshold := f64("indicator_rsi_sell_threshold")
	if rsiThreshold == 0 {
		rsiThreshold = 70
	}
	claudeModel := vals["claude_model"]
	if claudeModel == "" {
		claudeModel = "claude-sonnet-4-6"
	}

	rankingTypes := strSlice("ranking_types")
	if len(rankingTypes) == 0 {
		rankingTypes = []string{"volume", "strength", "exec_count", "disparity"}
	}
	sellConditions := strSlice("sell_conditions")
	if len(sellConditions) == 0 {
		sellConditions = []string{"target_pct", "stop_pct"}
	}

	tradingStartTime := vals["trading_start_time"]
	if tradingStartTime == "" {
		tradingStartTime = "09:15"
	}
	tradingEndTime := vals["trading_end_time"]
	if tradingEndTime == "" {
		tradingEndTime = "15:15"
	}
	stagnationThresholdPct := f64("stagnation_threshold_pct")
	if stagnationThresholdPct == 0 {
		stagnationThresholdPct = 1.0
	}
	stagnationDurationMin := i64("stagnation_duration_min")
	if stagnationDurationMin == 0 {
		stagnationDurationMin = 30
	}
	rankingCondition := vals["ranking_condition"]
	if rankingCondition != "AND" && rankingCondition != "OR" {
		rankingCondition = "AND"
	}

	return TradingSettings{
		TakeProfitPct:              takeProfitPct,
		StopLossPct:                stopLossPct,
		RankingTypes:               rankingTypes,
		RankingPriceMin:            vals["ranking_price_min"],
		RankingPriceMax:            vals["ranking_price_max"],
		MaxPositions:               maxPositions,
		OrderAmountPct:             orderAmountPct,
		SellConditions:             sellConditions,
		IndicatorCheckIntervalMin:  indicatorCheckInterval,
		IndicatorRSISellThreshold:  rsiThreshold,
		IndicatorMACDBearishSell:   vals["indicator_macd_bearish_sell"] == "true",
		ClaudeModel:                claudeModel,
		RankingVolumeMinIncrRate:   f64("ranking_volume_min_incrrate"),
		RankingStrengthMin:         f64("ranking_strength_min"),
		RankingExecCountNetBuyOnly: vals["ranking_execcount_net_buy_only"] != "false",
		RankingDisparityD20Min:     f64("ranking_disparity_d20_min"),
		RankingDisparityD20Max:     f64("ranking_disparity_d20_max"),
		RankingTopN:                i64("ranking_top_n"),
		TradingStartTime:           tradingStartTime,
		TradingEndTime:             tradingEndTime,
		StagnationThresholdPct:     stagnationThresholdPct,
		StagnationDurationMin:      stagnationDurationMin,
		RankingCondition:           rankingCondition,
		MinTradingValue:            f64("min_trading_value"),
		BuyPauseStart:              vals["buy_pause_start"],
		BuyPauseEnd:                vals["buy_pause_end"],
		TrailingTriggerPct:         f64("trailing_trigger_pct"),
		TrailingStopPct:            f64("trailing_stop_pct"),
		DailyMaxLossPct:            f64("daily_max_loss_pct"),
		IndexCodes:                 strSlice("index_codes"),
	}, nil
}

// GetSetting returns the value for the given key from the settings table.
// Returns an empty string if the key does not exist.
func (db *DB) GetSetting(ctx context.Context, key string) string {
	var value string
	db.QueryRowContext(ctx, "SELECT value FROM settings WHERE key = ?", key).Scan(&value) //nolint:errcheck
	return value
}

// SetSetting upserts a key-value pair in the settings table.
func (db *DB) SetSetting(ctx context.Context, key, value string) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO settings (key, value, updated_at) VALUES (?, ?, datetime('now'))
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, value)
	return err
}

// GetTodayRealizedPnL returns today's realized P&L (KRW) for AGENT SELL orders.
// P&L = sum((filled_price - price) * qty) for SELL orders with filled_price > 0.
// 'price' in SELL orders is set to the buy filled_price at order creation time.
// Returns 0 if there are no qualifying orders or on query error.
func (db *DB) GetTodayRealizedPnL(ctx context.Context) float64 {
	kst, _ := time.LoadLocation("Asia/Seoul")
	today := time.Now().In(kst).Format("2006-01-02")

	var pnl float64
	db.QueryRowContext(ctx, //nolint:errcheck
		`SELECT COALESCE(SUM((filled_price - price) * qty), 0)
		 FROM orders
		 WHERE date(created_at) = date(?)
		   AND order_type = 'SELL'
		   AND source = 'AGENT'
		   AND filled_price > 0`, today,
	).Scan(&pnl)
	return pnl
}

// InsertRankingLog saves a single getRankings() attempt record.
func (db *DB) InsertRankingLog(ctx context.Context, log models.TraderRankingLog) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO trader_ranking_logs
		 (timestamp, ranking_types, price_min, price_max,
		  volume_count, strength_count, exec_count_count, disparity_count,
		  ranking_condition, intersection_count, error_message)
		 VALUES (datetime('now'), ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.RankingTypes, log.PriceMin, log.PriceMax,
		log.VolumeCount, log.StrengthCount, log.ExecCountCount, log.DisparityCount,
		log.RankingCondition, log.IntersectionCount, log.ErrorMessage)
	return err
}

// GetRankingLogs returns the most recent ranking attempt logs (newest first).
// Logs older than 30 days are automatically purged on each call.
func (db *DB) GetRankingLogs(ctx context.Context, limit int) ([]models.TraderRankingLog, error) {
	db.ExecContext(ctx, //nolint:errcheck
		`DELETE FROM trader_ranking_logs WHERE timestamp < datetime('now', '-30 days')`)

	rows, err := db.QueryContext(ctx,
		`SELECT id, timestamp, ranking_types, price_min, price_max,
		        volume_count, strength_count, exec_count_count, disparity_count,
		        ranking_condition, intersection_count, error_message
		 FROM trader_ranking_logs ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.TraderRankingLog
	for rows.Next() {
		var l models.TraderRankingLog
		if err := rows.Scan(
			&l.ID, &l.Timestamp, &l.RankingTypes, &l.PriceMin, &l.PriceMax,
			&l.VolumeCount, &l.StrengthCount, &l.ExecCountCount, &l.DisparityCount,
			&l.RankingCondition, &l.IntersectionCount, &l.ErrorMessage,
		); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	if logs == nil {
		logs = []models.TraderRankingLog{}
	}
	return logs, nil
}
