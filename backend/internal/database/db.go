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
	TakeProfitPct float64 // 익절 기준 % (STOCK 기본값, ETF_DOMESTIC 미설정 시 폴백)
	StopLossPct   float64 // 손절 기준 % (STOCK 기본값, ETF_DOMESTIC 미설정 시 폴백)
	// ETF/주식 분리 수익·손절 기준 (0이면 TakeProfitPct/StopLossPct 기본값 사용)
	ETFTakeProfitPct   float64 // ETF 익절 기준 %
	ETFStopLossPct     float64 // ETF 손절 기준 %
	StockTakeProfitPct float64 // 주식 익절 기준 %
	StockStopLossPct   float64 // 주식 손절 기준 %
	StockTaxRate       float64 // 주식·과세ETF 세율 (0=미사용)
	// 하드 감시 종목 (항상 순위에 포함)
	HardWatchSymbols          []string // hard_watch 타입으로 강제 추가할 종목 코드 목록
	RankLeaseDurationMin      int      // 순위에서 사라진 종목 유지 시간(분). 0=비활성
	RankingTypes              []string // 순위 유형 우선순위 (volume, strength, fluctuation)
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
	RankingVolumeMinIncrRate  float64  // 거래량 증가율 최솟값 (0=필터없음)
	RankingStrengthMin        float64  // 체결강도 최솟값 (0=필터없음)
	RankingFluctuationMinRate float64  // 등락률 최솟값 % (0=필터없음)
	RankingFluctuationMaxRate float64  // 등락률 최댓값 % (0=제한없음)
	RankingVIKindCode         string   // VI 종류 필터: ""=전체, "1"=정적만, "2"=동적만
	RankingTopN               int      // 각 순위별 상위 N개만 교집합 대상 (0=전체)
	RankingExchanges          []string // 국장 순위 조회 거래소 코드 (0001=KOSPI, 1001=KOSDAQ, 2001=KOSPI200)
	RankingVolumeBlngClsCodes []string // 거래량순위 FID_BLNG_CLS_CODE 목록 (0=평균거래량, 1=거래량증가율, 2=거래회전율, 3=거래대금순, 4=평균거래대금)
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
	DailyMaxLossPct float64 // 일일 최대 손실 한도(%). 0=제한없음 (국장 KRW 기준)
	// 지수 필터
	IndexCodes []string // 지수 코드 목록 ("0001"=코스피, "1001"=코스닥). 빈 배열=비활성
	// 하드 필터 (매수 품질 필터)
	FilterRsiMax           float64 // RSI 과열 임계값 (0=필터없음). 기본 80
	FilterDisparityM5Max   float64 // 5분봉 이격도 최대값 (0=필터없음). 기본 3.0
	FilterHighPriceDiffMin float64 // 고가 대비 최소값% (0=필터없음). 기본 -5.0
	FilterOpenPriceDiffMax float64 // 시가 대비 최대값% (0=필터없음). 기본 20.0
	// 지수 하락 임계값
	IndexDropThresholdPct float64 // 지수 하락 시 매수 중단 기준 % (기본 -1.0)
	// 요일별 트레이딩 스케줄 (0=일 1=월 2=화 3=수 4=목 5=금 6=토). 빈 배열=매일
	TradingDays []int
	// AI 하드 거부 기준값
	HardDisparityM5Min   float64 // 5분봉 이격도 하한 (이하 → 칼날 하락 스킵). 기본 -1.5
	HardDisparityM5Max   float64 // 5분봉 이격도 상한 (이상 → 과열 스킵). 기본 3.0
	HardHighPriceDiffMax float64 // 고점 대비 최대% (이상 → 고점권 스킵). 기본 -0.5
	HardHighPriceDiffMin float64 // 고점 대비 최소% (이하 + 거래량급증 → 추세이탈 스킵). 기본 -5.0
	HardPrevVolRatioMax  float64 // 전봉 대비 거래량 비율 상한. 기본 1.2
	HardStrengthMin      float64 // 체결강도 하한 (이하 → 매수세 소멸 스킵). 기본 100.0
	HardRSIMax           float64 // RSI 상한 (이상 → 과매수 스킵). 기본 70.0
	HardOpenPriceDiffMax float64 // 시가 대비 상승률 상한 (이상 → 상한가권 스킵). 기본 15.0
	// AI 매수 구간 기준값
	VWAPDiffMin    float64 // VWAP 대비 이격도 하한 (%). 기본 0.0
	VWAPDiffMax    float64 // VWAP 대비 이격도 상한 (%). 기본 1.5
	RSIBuyMin      float64 // 이상적 매수 RSI 하한. 기본 40.0
	RSIBuyMax      float64 // 이상적 매수 RSI 상한. 기본 60.0
	BidAskRatioMin float64 // 매수호가 우세 최솟값. 기본 1.2 (0=미사용)
	// 시가총액 필터
	MinMarketCap float64 // 최소 시가총액 (억원). 0=필터없음. MST 상장주식수 × 현재가 기준
	// 세금보정
	MinExpectedProfitPct float64 // 주식 진입 시 세후 최소 기대수익 (%). 0=미사용
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

		`CREATE TABLE IF NOT EXISTS service_logs (
			id        INTEGER PRIMARY KEY AUTOINCREMENT,
			source    TEXT    NOT NULL DEFAULT 'SYSTEM',
			level     TEXT    NOT NULL DEFAULT 'ERROR',
			message   TEXT    NOT NULL DEFAULT '',
			detail    TEXT    NOT NULL DEFAULT '',
			timestamp DATETIME NOT NULL DEFAULT (datetime('now'))
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
			intersection_count INTEGER  NOT NULL DEFAULT 0,
			error_message      TEXT     NOT NULL DEFAULT ''
		)`,

		`CREATE TABLE IF NOT EXISTS settings_presets (
			id            INTEGER  PRIMARY KEY AUTOINCREMENT,
			name          TEXT     NOT NULL UNIQUE,
			description   TEXT     NOT NULL DEFAULT '',
			settings_json TEXT     NOT NULL DEFAULT '{}',
			created_at    DATETIME NOT NULL DEFAULT (datetime('now')),
			updated_at    DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,

		`CREATE TABLE IF NOT EXISTS stock_masters (
			stock_code             TEXT     PRIMARY KEY,
			stock_name             TEXT     NOT NULL DEFAULT '',
			isin                   TEXT     NOT NULL DEFAULT '',
			market_type            TEXT     NOT NULL DEFAULT '',
			group_code             TEXT     NOT NULL DEFAULT '',
			is_etf                 INTEGER  NOT NULL DEFAULT 0,
			is_domestic_equity_etf INTEGER  NOT NULL DEFAULT 0,
			listed_shares          INTEGER  NOT NULL DEFAULT 0,
			updated_at             DATETIME NOT NULL DEFAULT (datetime('now'))
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
		`ALTER TABLE orders ADD COLUMN market TEXT NOT NULL DEFAULT 'KR'`,
		`ALTER TABLE monitored_positions ADD COLUMN market TEXT NOT NULL DEFAULT 'KR'`,
		`ALTER TABLE trader_ranking_logs ADD COLUMN ranking_condition TEXT NOT NULL DEFAULT 'AND'`,
		`ALTER TABLE trader_ranking_logs ADD COLUMN result_stocks TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE trader_selection_logs ADD COLUMN market TEXT NOT NULL DEFAULT 'KR'`,
		`ALTER TABLE trader_ranking_logs ADD COLUMN market TEXT NOT NULL DEFAULT 'KR'`,
		`ALTER TABLE trader_ranking_logs ADD COLUMN filtered_stocks TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE trader_selection_logs ADD COLUMN ranking_log_id INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE settings_presets ADD COLUMN market TEXT NOT NULL DEFAULT 'KR'`,
		`ALTER TABLE stock_masters ADD COLUMN listed_shares INTEGER NOT NULL DEFAULT 0`,
	}
	for _, s := range alterStmts {
		// "duplicate column name" 에러는 정상 (이미 존재하는 경우) — 무시
		db.Exec(s) //nolint:errcheck
	}

	// Default trading settings (INSERT OR IGNORE — never overwrite user-set values)
	defaultSettings := []struct{ key, val string }{
		{"take_profit_pct", "3.0"},
		{"stop_loss_pct", "2.0"},
		{"etf_take_profit_pct", "0.5"},
		{"etf_stop_loss_pct", "0.4"},
		{"stock_take_profit_pct", "1.5"},
		{"stock_stop_loss_pct", "1.0"},
		{"stock_tax_rate", "0.002"},
		{"hard_watch_symbols", "[]"},
		{"rank_lease_duration_min", "5"},
		{"ranking_types", `["volume","strength"]`},
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
		{"ranking_fluctuation_min_rate", "0"},
		{"ranking_fluctuation_max_rate", "0"},
		{"ranking_vi_kind_code", ""},
		{"ranking_top_n", "20"},
		{"ranking_exchanges", `["0001","1001"]`},
		{"ranking_volume_blng_cls_codes", `["0","1","2","3","4"]`},
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
		{"filter_rsi_max", "80"},
		{"filter_disparity_m5_max", "3.0"},
		{"filter_high_price_diff_min", "-5.0"},
		{"filter_open_price_diff_max", "20.0"},
		{"index_drop_threshold_pct", "-1.0"},
		{"trading_days", "[1,2,3,4,5]"},
		{"hard_disparity_m5_min", "-1.5"},
		{"hard_disparity_m5_max", "3.0"},
		{"hard_high_price_diff_max", "-0.5"},
		{"hard_high_price_diff_min", "-5.0"},
		{"hard_prev_vol_ratio_max", "1.2"},
		{"hard_strength_min", "100.0"},
		{"hard_rsi_max", "70.0"},
		{"hard_open_price_diff_max", "15.0"},
		{"vwap_diff_min", "0.0"},
		{"vwap_diff_max", "1.5"},
		{"rsi_buy_min", "40.0"},
		{"rsi_buy_max", "60.0"},
		{"bid_ask_ratio_min", "1.2"},
		{"min_market_cap", "0"},
		{"min_expected_profit_pct", "0"},
		{"active_preset_id", "0"},
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
			`'take_profit_pct','stop_loss_pct',`+
			`'etf_take_profit_pct','etf_stop_loss_pct','stock_take_profit_pct','stock_stop_loss_pct','stock_tax_rate',`+
			`'hard_watch_symbols','rank_lease_duration_min',`+
			`'ranking_types',`+
			`'ranking_price_min','ranking_price_max','max_positions',`+
			`'order_amount_pct','sell_conditions','indicator_check_interval_min',`+
			`'indicator_rsi_sell_threshold','indicator_macd_bearish_sell','claude_model',`+
			`'ranking_volume_min_incrrate','ranking_strength_min',`+
			`'ranking_fluctuation_min_rate','ranking_fluctuation_max_rate','ranking_vi_kind_code',`+
			`'ranking_top_n','ranking_exchanges','ranking_volume_blng_cls_codes',`+
			`'trading_start_time','trading_end_time',`+
			`'stagnation_threshold_pct','stagnation_duration_min',`+
			`'ranking_condition',`+
			`'min_trading_value',`+
			`'buy_pause_start','buy_pause_end',`+
			`'trailing_trigger_pct','trailing_stop_pct',`+
			`'daily_max_loss_pct','index_codes',`+
			`'filter_rsi_max','filter_disparity_m5_max',`+
			`'filter_high_price_diff_min','filter_open_price_diff_max',`+
			`'index_drop_threshold_pct',`+
			`'trading_days',`+
			`'hard_disparity_m5_min','hard_disparity_m5_max',`+
			`'hard_high_price_diff_max','hard_high_price_diff_min',`+
			`'hard_prev_vol_ratio_max','hard_strength_min','hard_rsi_max','hard_open_price_diff_max',`+
			`'vwap_diff_min','vwap_diff_max','rsi_buy_min','rsi_buy_max','bid_ask_ratio_min',`+
			`'min_market_cap','min_expected_profit_pct'`+
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
		rankingTypes = []string{"volume", "strength"}
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
	rankingExchanges := strSlice("ranking_exchanges")
	if len(rankingExchanges) == 0 {
		rankingExchanges = []string{"0001", "1001"}
	}
	rankingVolumeBlngClsCodes := strSlice("ranking_volume_blng_cls_codes")
	if len(rankingVolumeBlngClsCodes) == 0 {
		rankingVolumeBlngClsCodes = []string{"0", "1", "2", "3", "4"}
	}

	filterRsiMax := f64("filter_rsi_max")
	if filterRsiMax == 0 {
		filterRsiMax = 80
	}
	filterDisparityM5Max := f64("filter_disparity_m5_max")
	if filterDisparityM5Max == 0 {
		filterDisparityM5Max = 3.0
	}
	filterHighPriceDiffMin := f64("filter_high_price_diff_min")
	if filterHighPriceDiffMin == 0 {
		filterHighPriceDiffMin = -5.0
	}
	filterOpenPriceDiffMax := f64("filter_open_price_diff_max")
	if filterOpenPriceDiffMax == 0 {
		filterOpenPriceDiffMax = 20.0
	}
	indexDropThresholdPct := f64("index_drop_threshold_pct")
	if indexDropThresholdPct == 0 {
		indexDropThresholdPct = -1.0
	}

	var tradingDays []int
	if v, ok := vals["trading_days"]; ok && v != "" {
		if err := json.Unmarshal([]byte(v), &tradingDays); err != nil {
			tradingDays = nil
		}
	}

	hardDisparityM5Min := f64("hard_disparity_m5_min")
	if hardDisparityM5Min == 0 {
		hardDisparityM5Min = -1.5
	}
	hardDisparityM5Max := f64("hard_disparity_m5_max")
	if hardDisparityM5Max == 0 {
		hardDisparityM5Max = 3.0
	}
	hardHighPriceDiffMax := f64("hard_high_price_diff_max")
	if hardHighPriceDiffMax == 0 {
		hardHighPriceDiffMax = -0.5
	}
	hardHighPriceDiffMin := f64("hard_high_price_diff_min")
	if hardHighPriceDiffMin == 0 {
		hardHighPriceDiffMin = -5.0
	}
	hardPrevVolRatioMax := f64("hard_prev_vol_ratio_max")
	if hardPrevVolRatioMax == 0 {
		hardPrevVolRatioMax = 1.2
	}
	hardStrengthMin := f64("hard_strength_min")
	if hardStrengthMin == 0 {
		hardStrengthMin = 100.0
	}
	hardRSIMax := f64("hard_rsi_max")
	if hardRSIMax == 0 {
		hardRSIMax = 70.0
	}
	hardOpenPriceDiffMax := f64("hard_open_price_diff_max")
	if hardOpenPriceDiffMax == 0 {
		hardOpenPriceDiffMax = 15.0
	}

	vwapDiffMax := f64("vwap_diff_max")
	if vwapDiffMax == 0 {
		vwapDiffMax = 1.5
	}
	rsiBuyMin := f64("rsi_buy_min")
	if rsiBuyMin == 0 {
		rsiBuyMin = 40.0
	}
	rsiBuyMax := f64("rsi_buy_max")
	if rsiBuyMax == 0 {
		rsiBuyMax = 60.0
	}
	bidAskRatioMin := f64("bid_ask_ratio_min")
	if bidAskRatioMin == 0 {
		bidAskRatioMin = 1.2
	}

	return TradingSettings{
		TakeProfitPct:             takeProfitPct,
		StopLossPct:               stopLossPct,
		ETFTakeProfitPct:          f64("etf_take_profit_pct"),
		ETFStopLossPct:            f64("etf_stop_loss_pct"),
		StockTakeProfitPct:        f64("stock_take_profit_pct"),
		StockStopLossPct:          f64("stock_stop_loss_pct"),
		StockTaxRate:              f64("stock_tax_rate"),
		HardWatchSymbols:          strSlice("hard_watch_symbols"),
		RankLeaseDurationMin:      i64("rank_lease_duration_min"),
		RankingTypes:              rankingTypes,
		RankingPriceMin:           vals["ranking_price_min"],
		RankingPriceMax:           vals["ranking_price_max"],
		MaxPositions:              maxPositions,
		OrderAmountPct:            orderAmountPct,
		SellConditions:            sellConditions,
		IndicatorCheckIntervalMin: indicatorCheckInterval,
		IndicatorRSISellThreshold: rsiThreshold,
		IndicatorMACDBearishSell:  vals["indicator_macd_bearish_sell"] == "true",
		ClaudeModel:               claudeModel,
		RankingVolumeMinIncrRate:  f64("ranking_volume_min_incrrate"),
		RankingStrengthMin:        f64("ranking_strength_min"),
		RankingFluctuationMinRate: f64("ranking_fluctuation_min_rate"),
		RankingFluctuationMaxRate: f64("ranking_fluctuation_max_rate"),
		RankingVIKindCode:         vals["ranking_vi_kind_code"],
		RankingTopN:               i64("ranking_top_n"),
		RankingExchanges:          rankingExchanges,
		RankingVolumeBlngClsCodes: rankingVolumeBlngClsCodes,
		TradingStartTime:          tradingStartTime,
		TradingEndTime:            tradingEndTime,
		StagnationThresholdPct:    stagnationThresholdPct,
		StagnationDurationMin:     stagnationDurationMin,
		RankingCondition:          rankingCondition,
		MinTradingValue:           f64("min_trading_value"),
		BuyPauseStart:             vals["buy_pause_start"],
		BuyPauseEnd:               vals["buy_pause_end"],
		TrailingTriggerPct:        f64("trailing_trigger_pct"),
		TrailingStopPct:           f64("trailing_stop_pct"),
		DailyMaxLossPct:           f64("daily_max_loss_pct"),
		IndexCodes:                strSlice("index_codes"),
		FilterRsiMax:              filterRsiMax,
		FilterDisparityM5Max:      filterDisparityM5Max,
		FilterHighPriceDiffMin:    filterHighPriceDiffMin,
		FilterOpenPriceDiffMax:    filterOpenPriceDiffMax,
		IndexDropThresholdPct:     indexDropThresholdPct,
		TradingDays:               tradingDays,
		HardDisparityM5Min:        hardDisparityM5Min,
		HardDisparityM5Max:        hardDisparityM5Max,
		HardHighPriceDiffMax:      hardHighPriceDiffMax,
		HardHighPriceDiffMin:      hardHighPriceDiffMin,
		HardPrevVolRatioMax:       hardPrevVolRatioMax,
		HardStrengthMin:           hardStrengthMin,
		HardRSIMax:                hardRSIMax,
		HardOpenPriceDiffMax:      hardOpenPriceDiffMax,
		VWAPDiffMin:               f64("vwap_diff_min"),
		VWAPDiffMax:               vwapDiffMax,
		RSIBuyMin:                 rsiBuyMin,
		RSIBuyMax:                 rsiBuyMax,
		BidAskRatioMin:            bidAskRatioMin,
		MinMarketCap:              f64("min_market_cap"),
		MinExpectedProfitPct:      f64("min_expected_profit_pct"),
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

// InsertServiceLog writes a service-level log entry. Non-fatal — errors are swallowed.
// source: "TRADER" | "MONITOR" | "SYSTEM"
// level:  "ERROR"  | "WARN"
func (db *DB) InsertServiceLog(ctx context.Context, source, level, message, detail string) {
	db.ExecContext(ctx, //nolint:errcheck
		`INSERT INTO service_logs (source, level, message, detail, timestamp)
		 VALUES (?, ?, ?, ?, datetime('now'))`,
		source, level, message, detail)
}

// GetServiceLogs returns recent service log entries (newest first).
// Auto-purges entries older than 7 days.
// source: "" or "ALL" = all sources; otherwise filter by exact source.
func (db *DB) GetServiceLogs(ctx context.Context, source string, limit int) ([]models.ServiceLog, error) {
	db.ExecContext(ctx, //nolint:errcheck
		`DELETE FROM service_logs WHERE timestamp < datetime('now', '-7 days')`)

	var (
		rows *sql.Rows
		err  error
	)
	if source == "" || source == "ALL" {
		rows, err = db.QueryContext(ctx,
			`SELECT id, source, level, message, detail, timestamp
			 FROM service_logs ORDER BY id DESC LIMIT ?`, limit)
	} else {
		rows, err = db.QueryContext(ctx,
			`SELECT id, source, level, message, detail, timestamp
			 FROM service_logs WHERE source = ? ORDER BY id DESC LIMIT ?`, source, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.ServiceLog
	for rows.Next() {
		var l models.ServiceLog
		if err := rows.Scan(&l.ID, &l.Source, &l.Level, &l.Message, &l.Detail, &l.Timestamp); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	if logs == nil {
		logs = []models.ServiceLog{}
	}
	return logs, nil
}

// GetTodayRealizedPnLByMarket returns today's realized P&L for AGENT SELL orders
// filtered by market ("KR" or "US"). Returns 0 on error or no qualifying orders.
func (db *DB) GetTodayRealizedPnLByMarket(ctx context.Context, market string) float64 {
	kst, _ := time.LoadLocation("Asia/Seoul")
	today := time.Now().In(kst).Format("2006-01-02")

	var pnl float64
	db.QueryRowContext(ctx, //nolint:errcheck
		`SELECT COALESCE(SUM((filled_price - price) * qty), 0)
		 FROM orders
		 WHERE date(created_at) = date(?)
		   AND order_type = 'SELL'
		   AND source = 'AGENT'
		   AND market = ?
		   AND filled_price > 0`, today, market,
	).Scan(&pnl)
	return pnl
}

// DailyPnL holds the realized P&L total for a single trading day.
type DailyPnL struct {
	Date string  `json:"date"` // "YYYY-MM-DD"
	PnL  float64 `json:"pnl"`
}

// GetDailyPnL returns daily realized P&L for AGENT SELL orders over the past `days` days.
// Results are ordered by date ascending. Days with no filled SELL orders are omitted.
func (db *DB) GetDailyPnL(ctx context.Context, days int) ([]DailyPnL, error) {
	query := fmt.Sprintf(`
		SELECT date(created_at) AS d,
		       COALESCE(SUM((filled_price - price) * qty), 0) AS pnl
		FROM orders
		WHERE order_type = 'SELL'
		  AND source = 'AGENT'
		  AND filled_price > 0
		  AND date(created_at) >= date('now', '-%d days')
		GROUP BY d
		ORDER BY d ASC
	`, days)

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []DailyPnL
	for rows.Next() {
		var dp DailyPnL
		if err := rows.Scan(&dp.Date, &dp.PnL); err != nil {
			return nil, err
		}
		result = append(result, dp)
	}
	if result == nil {
		result = []DailyPnL{}
	}
	return result, nil
}

// InsertRankingLog saves a single getRankings() attempt record and returns the new row ID.
func (db *DB) InsertRankingLog(ctx context.Context, log models.TraderRankingLog) (int64, error) {
	res, err := db.ExecContext(ctx,
		`INSERT INTO trader_ranking_logs
		 (timestamp, ranking_types, price_min, price_max,
		  volume_count, strength_count,
		  ranking_condition, intersection_count, result_stocks, error_message, market)
		 VALUES (datetime('now'), ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.RankingTypes, log.PriceMin, log.PriceMax,
		log.VolumeCount, log.StrengthCount,
		log.RankingCondition, log.IntersectionCount, log.ResultStocks, log.ErrorMessage, log.Market)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetRankingLogs returns the most recent ranking attempt logs (newest first).
// Logs older than 30 days are automatically purged on each call.
func (db *DB) GetRankingLogs(ctx context.Context, limit int) ([]models.TraderRankingLog, error) {
	db.ExecContext(ctx, //nolint:errcheck
		`DELETE FROM trader_ranking_logs WHERE timestamp < datetime('now', '-30 days')`)

	rows, err := db.QueryContext(ctx,
		`SELECT id, timestamp, ranking_types, price_min, price_max,
		        volume_count, strength_count,
		        ranking_condition, intersection_count, result_stocks, error_message, market,
		        filtered_stocks
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
			&l.VolumeCount, &l.StrengthCount,
			&l.RankingCondition, &l.IntersectionCount, &l.ResultStocks, &l.ErrorMessage, &l.Market,
			&l.FilteredStocks,
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

// ────────────────────────────────────────────────────────
// Settings Presets
// ────────────────────────────────────────────────────────

// SettingsPreset represents a named snapshot of trading settings.
type SettingsPreset struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Market       string `json:"market"` // "KR" or "US"
	SettingsJSON string `json:"settings_json"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// ListSettingsPresets returns all presets ordered by id.
func (db *DB) ListSettingsPresets(ctx context.Context) ([]SettingsPreset, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, name, description, market, settings_json, created_at, updated_at
		 FROM settings_presets ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var presets []SettingsPreset
	for rows.Next() {
		var p SettingsPreset
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Market, &p.SettingsJSON, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		presets = append(presets, p)
	}
	if presets == nil {
		presets = []SettingsPreset{}
	}
	return presets, nil
}

// CreateSettingsPreset inserts a new preset. Returns the new preset ID.
func (db *DB) CreateSettingsPreset(ctx context.Context, name, description, market, settingsJSON string) (int64, error) {
	res, err := db.ExecContext(ctx,
		`INSERT INTO settings_presets (name, description, market, settings_json, updated_at)
		 VALUES (?, ?, ?, ?, datetime('now'))`,
		name, description, market, settingsJSON)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetSettingsPreset returns a single preset by id.
func (db *DB) GetSettingsPreset(ctx context.Context, id int64) (*SettingsPreset, error) {
	var p SettingsPreset
	err := db.QueryRowContext(ctx,
		`SELECT id, name, description, market, settings_json, created_at, updated_at
		 FROM settings_presets WHERE id = ?`, id).
		Scan(&p.ID, &p.Name, &p.Description, &p.Market, &p.SettingsJSON, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// DeleteSettingsPreset removes a preset by id.
func (db *DB) DeleteSettingsPreset(ctx context.Context, id int64) error {
	_, err := db.ExecContext(ctx, `DELETE FROM settings_presets WHERE id = ?`, id)
	return err
}

// UpdateSettingsPreset overwrites an existing preset's settings_json snapshot.
func (db *DB) UpdateSettingsPreset(ctx context.Context, id int64, settingsJSON string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE settings_presets SET settings_json=?, updated_at=datetime('now') WHERE id=?`,
		settingsJSON, id)
	return err
}
