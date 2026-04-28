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
	HardDisparityM5Min      float64 // 5분봉 이격도 하한 (이하 → 칼날 하락 스킵). 기본 -1.5
	HardDisparityM5Max      float64 // 5분봉 이격도 상한 (이상 → 과열 스킵). 기본 3.0
	HardHighPriceDiffMax    float64 // 고점 대비 최대% (이상 → 고점권 스킵). 기본 -0.5
	HardHighPriceDiffMin    float64 // 고점 대비 최소% (이하 + 거래량급증 → 추세이탈 스킵). 기본 -5.0
	HardPrevVolRatioMax     float64 // 전봉 대비 거래량 비율 상한. 기본 1.2
	HardStrengthMin         float64 // 체결강도 하한 (이하 → 매수세 소멸 스킵). 기본 100.0
	HardRSIMax              float64 // RSI 상한 (이상 → 과매수 스킵). 기본 70.0
	HardOpenPriceDiffMax    float64 // 시가 대비 상승률 상한 (이상 → 상한가권 스킵). 기본 15.0
	HardMACDBearishEnabled  bool    // true이면 macd_line < macd_signal 진입 차단. 기본 false
	HardHighFormedMinsMax   float64 // 고점 형성 후 경과 시간 상한(분). 초과 시 모멘텀 소진으로 스킵. 0=비활성
	HardVolVs3AvgRatioMin   float64 // 거래량 회복 비율 최솟값. 현재봉/직전3봉 평균 거래량. 0=비활성
	HardRelativeStrengthMin float64 // 시장 대비 상대강도 최솟값(%). 종목등락률-시장등락률. 0=비활성
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
	// Claude 후보 제한
	MaxClaudeCandidates int // Claude에 전달할 최대 후보 종목 수. 0=제한없음. 기본 15
	// 복합 모멘텀 스코어링
	MomentumScoreMin float64 // 최솟값(0~100). 0=비활성. BidAskRatio 조회 후 필터
	// 단계적 횡보 청산
	StagnationPartialExitEnabled  bool    // 횡보 감지 시 절반 청산 활성화 (false=기존 전량 청산)
	StagnationBidAskSellThreshold float64 // 이 값 미만이면 즉시 전량 청산 (기본 1.0)
	// 부분 익절 (Partial Take Profit)
	PartialTPEnabled   bool    // 중간 목표가 도달 시 부분 매도 활성화
	PartialTPPct       float64 // 중간 익절 트리거 수익률 % (기본 1.0)
	PartialTPRatio     float64 // 매도 비율 0~1 (기본 0.5 = 50%)
	PartialTPRaiseStop bool    // 부분 익절 후 손절가를 매입가(BEP)로 올리기
	// 복합 스코어링 가중치
	ScoringBidAskWeight   int // bid_ask 점수 가중치 (기본 30)
	ScoringStrengthWeight int // 체결강도 점수 가중치 (기본 25)
	ScoringMACDWeight     int // MACD 방향성 점수 가중치 (기본 20)
	ScoringRSIWeight      int // RSI 구간 점수 가중치 (기본 15)
	ScoringVWAPWeight     int // VWAP 이격도 점수 가중치 (기본 10)
	// Adaptive Threshold — Hard Rule 자동 완화
	AdaptiveThresholdEnabled bool    // N회 연속 실패 시 hard rule 완화 활성화
	AdaptiveThresholdTrigger int     // 완화 발동 연속 실패 횟수 (기본 10)
	AdaptiveRelaxPct         float64 // hard rule 완화 비율 % (기본 20.0)
	AdaptiveRelaxActive      bool    // 런타임 플래그 — 현재 완화 적용 중 (DB에 저장 안 함)
	// Market Phase Detection — 시장 국면 감지 자동 완화
	MarketPhaseRelaxEnabled     bool    // 약세장 감지 시 hard rule 완화 활성화 (기본 false)
	MarketPhaseIndexDropTrigger float64 // 완화 발동 전일 대비 하락률 기준 % (기본 -1.0)
	MarketPhaseRelaxPct         float64 // hard rule 완화 비율 % (기본 15.0)
	MarketPhaseRelaxActive      bool    // 런타임 플래그 — DB에 저장 안 함
	// Hard Rule Escalation — 단계적 자동 완화
	EscalationEnabled   bool    // N회 연속 실패마다 단계적으로 hard rule 완화 활성화
	EscalationTrigger   int     // 단계당 실패 횟수 (기본 20)
	EscalationStepPct   float64 // 단계당 완화 비율 % (기본 10.0)
	EscalationMaxStages int     // 최대 단계 수 (기본 5)
	EscalationStage     int     // 런타임 전용 — 현재 단계 (DB에 저장 안 함)
	// Hard Rule Feedback — 룰별 거부 빈도 기반 자동 완화
	HardRuleFeedbackEnabled      bool    // 특정 룰이 window 내 70%+ 사이클에서 발동 시 해당 룰만 선택 완화
	HardRuleFeedbackWindow       int     // 분석 대상 최근 사이클 수 (기본 10)
	HardRuleFeedbackThresholdPct float64 // 발동 임계 비율 % (기본 70.0)
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

		`CREATE TABLE IF NOT EXISTS trade_reports (
			id               INTEGER  PRIMARY KEY AUTOINCREMENT,
			date             TEXT     NOT NULL,
			stock_code       TEXT     NOT NULL,
			stock_name       TEXT     NOT NULL DEFAULT '',
			buy_order_id     INTEGER  NOT NULL DEFAULT 0,
			sell_order_id    INTEGER  NOT NULL DEFAULT 0,
			selection_log_id INTEGER  NOT NULL DEFAULT 0,
			buy_price        REAL     NOT NULL DEFAULT 0,
			buy_qty          INTEGER  NOT NULL DEFAULT 0,
			buy_amount       REAL     NOT NULL DEFAULT 0,
			buy_reason       TEXT     NOT NULL DEFAULT '',
			buy_indicators   TEXT     NOT NULL DEFAULT '',
			sell_price       REAL     NOT NULL DEFAULT 0,
			sell_qty         INTEGER  NOT NULL DEFAULT 0,
			sell_amount      REAL     NOT NULL DEFAULT 0,
			sell_reason      TEXT     NOT NULL DEFAULT '',
			sell_indicators  TEXT     NOT NULL DEFAULT '',
			profit_amount    REAL     NOT NULL DEFAULT 0,
			profit_pct       REAL     NOT NULL DEFAULT 0,
			created_at       DATETIME NOT NULL DEFAULT (datetime('now')),
			sold_at          DATETIME
		)`,

		`CREATE TABLE IF NOT EXISTS daily_reports (
			id                  INTEGER  PRIMARY KEY AUTOINCREMENT,
			date                TEXT     NOT NULL UNIQUE,
			total_trades        INTEGER  NOT NULL DEFAULT 0,
			winning_trades      INTEGER  NOT NULL DEFAULT 0,
			losing_trades       INTEGER  NOT NULL DEFAULT 0,
			total_profit_amount REAL     NOT NULL DEFAULT 0,
			avg_profit_pct      REAL     NOT NULL DEFAULT 0,
			best_trade          TEXT     NOT NULL DEFAULT '',
			worst_trade         TEXT     NOT NULL DEFAULT '',
			trade_summary       TEXT     NOT NULL DEFAULT '',
			created_at          DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,

		`CREATE TABLE IF NOT EXISTS optimization_reports (
			id                   INTEGER  PRIMARY KEY AUTOINCREMENT,
			date                 TEXT     NOT NULL UNIQUE,
			overall_assessment   TEXT     NOT NULL DEFAULT '',
			suggestions          TEXT     NOT NULL DEFAULT '[]',
			apply_mode_snapshot  TEXT     NOT NULL DEFAULT '',
			created_at           DATETIME NOT NULL DEFAULT (datetime('now'))
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
		`ALTER TABLE trader_ranking_logs ADD COLUMN type_counts TEXT NOT NULL DEFAULT '{}'`,
		`ALTER TABLE trader_selection_logs ADD COLUMN ranking_log_id INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE settings_presets ADD COLUMN market TEXT NOT NULL DEFAULT 'KR'`,
		`ALTER TABLE stock_masters ADD COLUMN listed_shares INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE trader_selection_logs ADD COLUMN hard_rule_stats TEXT NOT NULL DEFAULT '{}'`,
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
		{"ranking_volume_blng_cls_codes", `["1","3"]`},
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
		{"hard_macd_bearish_enabled", "false"},
		{"hard_high_formed_mins_max", "0"},
		{"hard_vol_vs_3avg_ratio_min", "0"},
		{"hard_relative_strength_min", "0"},
		{"vwap_diff_min", "0.0"},
		{"vwap_diff_max", "1.5"},
		{"rsi_buy_min", "40.0"},
		{"rsi_buy_max", "60.0"},
		{"bid_ask_ratio_min", "1.2"},
		{"min_market_cap", "0"},
		{"min_expected_profit_pct", "0"},
		{"active_preset_id", "0"},
		{"optimization_apply_mode", "all_manual"},
		{"max_claude_candidates", "15"},
		{"momentum_score_min", "0"},
		{"stagnation_partial_exit_enabled", "false"},
		{"stagnation_bid_ask_sell_threshold", "1.0"},
		// 부분 익절
		{"partial_tp_enabled", "false"},
		{"partial_tp_pct", "1.0"},
		{"partial_tp_ratio", "0.5"},
		{"partial_tp_raise_stop", "true"},
		// 복합 스코어링 가중치
		{"scoring_bidask_weight", "30"},
		{"scoring_strength_weight", "25"},
		{"scoring_macd_weight", "20"},
		{"scoring_rsi_weight", "15"},
		{"scoring_vwap_weight", "10"},
		// Adaptive Threshold
		{"adaptive_threshold_enabled", "false"},
		{"adaptive_threshold_trigger", "10"},
		{"adaptive_relax_pct", "20"},
		// Market Phase Detection
		{"market_phase_relax_enabled", "false"},
		{"market_phase_index_drop_trigger", "-1.0"},
		{"market_phase_relax_pct", "15"},
		// Hard Rule Escalation
		{"escalation_enabled", "false"},
		{"escalation_trigger", "20"},
		{"escalation_step_pct", "10.0"},
		{"escalation_max_stages", "5"},
		// Hard Rule Feedback
		{"hard_rule_feedback_enabled", "false"},
		{"hard_rule_feedback_window", "10"},
		{"hard_rule_feedback_threshold_pct", "70"},
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
			`'hard_macd_bearish_enabled','hard_high_formed_mins_max',`+
			`'hard_vol_vs_3avg_ratio_min','hard_relative_strength_min',`+
			`'vwap_diff_min','vwap_diff_max','rsi_buy_min','rsi_buy_max','bid_ask_ratio_min',`+
			`'min_market_cap','min_expected_profit_pct',`+
			`'max_claude_candidates',`+
			`'momentum_score_min',`+
			`'stagnation_partial_exit_enabled','stagnation_bid_ask_sell_threshold',`+
			`'partial_tp_enabled','partial_tp_pct','partial_tp_ratio','partial_tp_raise_stop',`+
			`'scoring_bidask_weight','scoring_strength_weight','scoring_macd_weight','scoring_rsi_weight','scoring_vwap_weight',`+
			`'adaptive_threshold_enabled','adaptive_threshold_trigger','adaptive_relax_pct',`+
			`'market_phase_relax_enabled','market_phase_index_drop_trigger','market_phase_relax_pct',`+
			`'escalation_enabled','escalation_trigger','escalation_step_pct','escalation_max_stages'`+
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
	// f64Default returns the stored float64 value, or def only when the key is absent/blank.
	// This prevents treating a user-saved 0 as "not set".
	f64Default := func(k string, def float64) float64 {
		v, ok := vals[k]
		if !ok || v == "" {
			return def
		}
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return def
		}
		return parsed
	}
	// i64Default is the integer equivalent of f64Default.
	i64Default := func(k string, def int) int {
		v, ok := vals[k]
		if !ok || v == "" {
			return def
		}
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return def
		}
		return parsed
	}
	strSlice := func(k string) []string {
		var s []string
		if v := vals[k]; v != "" {
			_ = json.Unmarshal([]byte(v), &s)
		}
		return s
	}

	takeProfitPct := f64Default("take_profit_pct", 3.0)
	stopLossPct := f64Default("stop_loss_pct", 2.0)
	maxPositions := i64Default("max_positions", 1)
	orderAmountPct := f64Default("order_amount_pct", 95)
	indicatorCheckInterval := i64Default("indicator_check_interval_min", 5)
	rsiThreshold := f64Default("indicator_rsi_sell_threshold", 70)
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
	stagnationThresholdPct := f64Default("stagnation_threshold_pct", 1.0)
	stagnationDurationMin := i64Default("stagnation_duration_min", 30)
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
		rankingVolumeBlngClsCodes = []string{"1", "3"}
	}

	filterRsiMax := f64Default("filter_rsi_max", 80)
	filterDisparityM5Max := f64Default("filter_disparity_m5_max", 3.0)
	filterHighPriceDiffMin := f64Default("filter_high_price_diff_min", -5.0)
	filterOpenPriceDiffMax := f64Default("filter_open_price_diff_max", 20.0)
	indexDropThresholdPct := f64Default("index_drop_threshold_pct", -1.0)

	var tradingDays []int
	if v, ok := vals["trading_days"]; ok && v != "" {
		if err := json.Unmarshal([]byte(v), &tradingDays); err != nil {
			tradingDays = nil
		}
	}

	hardDisparityM5Min := f64Default("hard_disparity_m5_min", -1.5)
	hardDisparityM5Max := f64Default("hard_disparity_m5_max", 3.0)
	hardHighPriceDiffMax := f64Default("hard_high_price_diff_max", -0.5)
	hardHighPriceDiffMin := f64Default("hard_high_price_diff_min", -5.0)
	hardPrevVolRatioMax := f64Default("hard_prev_vol_ratio_max", 1.2)
	hardStrengthMin := f64Default("hard_strength_min", 100.0)
	hardRSIMax := f64Default("hard_rsi_max", 70.0)
	hardOpenPriceDiffMax := f64Default("hard_open_price_diff_max", 15.0)

	vwapDiffMax := f64Default("vwap_diff_max", 1.5)
	rsiBuyMin := f64Default("rsi_buy_min", 40.0)
	rsiBuyMax := f64Default("rsi_buy_max", 60.0)
	bidAskRatioMin := f64Default("bid_ask_ratio_min", 1.2)

	return TradingSettings{
		TakeProfitPct:                 takeProfitPct,
		StopLossPct:                   stopLossPct,
		ETFTakeProfitPct:              f64("etf_take_profit_pct"),
		ETFStopLossPct:                f64("etf_stop_loss_pct"),
		StockTakeProfitPct:            f64("stock_take_profit_pct"),
		StockStopLossPct:              f64("stock_stop_loss_pct"),
		StockTaxRate:                  f64("stock_tax_rate"),
		HardWatchSymbols:              strSlice("hard_watch_symbols"),
		RankLeaseDurationMin:          i64("rank_lease_duration_min"),
		RankingTypes:                  rankingTypes,
		RankingPriceMin:               vals["ranking_price_min"],
		RankingPriceMax:               vals["ranking_price_max"],
		MaxPositions:                  maxPositions,
		OrderAmountPct:                orderAmountPct,
		SellConditions:                sellConditions,
		IndicatorCheckIntervalMin:     indicatorCheckInterval,
		IndicatorRSISellThreshold:     rsiThreshold,
		IndicatorMACDBearishSell:      vals["indicator_macd_bearish_sell"] == "true",
		ClaudeModel:                   claudeModel,
		RankingVolumeMinIncrRate:      f64("ranking_volume_min_incrrate"),
		RankingStrengthMin:            f64("ranking_strength_min"),
		RankingFluctuationMinRate:     f64("ranking_fluctuation_min_rate"),
		RankingFluctuationMaxRate:     f64("ranking_fluctuation_max_rate"),
		RankingVIKindCode:             vals["ranking_vi_kind_code"],
		RankingTopN:                   i64("ranking_top_n"),
		RankingExchanges:              rankingExchanges,
		RankingVolumeBlngClsCodes:     rankingVolumeBlngClsCodes,
		TradingStartTime:              tradingStartTime,
		TradingEndTime:                tradingEndTime,
		StagnationThresholdPct:        stagnationThresholdPct,
		StagnationDurationMin:         stagnationDurationMin,
		RankingCondition:              rankingCondition,
		MinTradingValue:               f64("min_trading_value"),
		BuyPauseStart:                 vals["buy_pause_start"],
		BuyPauseEnd:                   vals["buy_pause_end"],
		TrailingTriggerPct:            f64("trailing_trigger_pct"),
		TrailingStopPct:               f64("trailing_stop_pct"),
		DailyMaxLossPct:               f64("daily_max_loss_pct"),
		IndexCodes:                    strSlice("index_codes"),
		FilterRsiMax:                  filterRsiMax,
		FilterDisparityM5Max:          filterDisparityM5Max,
		FilterHighPriceDiffMin:        filterHighPriceDiffMin,
		FilterOpenPriceDiffMax:        filterOpenPriceDiffMax,
		IndexDropThresholdPct:         indexDropThresholdPct,
		TradingDays:                   tradingDays,
		HardDisparityM5Min:            hardDisparityM5Min,
		HardDisparityM5Max:            hardDisparityM5Max,
		HardHighPriceDiffMax:          hardHighPriceDiffMax,
		HardHighPriceDiffMin:          hardHighPriceDiffMin,
		HardPrevVolRatioMax:           hardPrevVolRatioMax,
		HardStrengthMin:               hardStrengthMin,
		HardRSIMax:                    hardRSIMax,
		HardOpenPriceDiffMax:          hardOpenPriceDiffMax,
		HardMACDBearishEnabled:        vals["hard_macd_bearish_enabled"] == "true",
		HardHighFormedMinsMax:         f64("hard_high_formed_mins_max"),
		HardVolVs3AvgRatioMin:         f64("hard_vol_vs_3avg_ratio_min"),
		HardRelativeStrengthMin:       f64("hard_relative_strength_min"),
		VWAPDiffMin:                   f64("vwap_diff_min"),
		VWAPDiffMax:                   vwapDiffMax,
		RSIBuyMin:                     rsiBuyMin,
		RSIBuyMax:                     rsiBuyMax,
		BidAskRatioMin:                bidAskRatioMin,
		MinMarketCap:                  f64("min_market_cap"),
		MinExpectedProfitPct:          f64("min_expected_profit_pct"),
		MaxClaudeCandidates:           i64Default("max_claude_candidates", 15),
		MomentumScoreMin:              f64("momentum_score_min"),
		StagnationPartialExitEnabled:  vals["stagnation_partial_exit_enabled"] == "true",
		StagnationBidAskSellThreshold: f64Default("stagnation_bid_ask_sell_threshold", 1.0),
		// 부분 익절
		PartialTPEnabled:   vals["partial_tp_enabled"] == "true",
		PartialTPPct:       f64Default("partial_tp_pct", 1.0),
		PartialTPRatio:     f64Default("partial_tp_ratio", 0.5),
		PartialTPRaiseStop: vals["partial_tp_raise_stop"] == "true",
		// 복합 스코어링 가중치
		ScoringBidAskWeight:   i64Default("scoring_bidask_weight", 30),
		ScoringStrengthWeight: i64Default("scoring_strength_weight", 25),
		ScoringMACDWeight:     i64Default("scoring_macd_weight", 20),
		ScoringRSIWeight:      i64Default("scoring_rsi_weight", 15),
		ScoringVWAPWeight:     i64Default("scoring_vwap_weight", 10),
		// Adaptive Threshold
		AdaptiveThresholdEnabled: vals["adaptive_threshold_enabled"] == "true",
		AdaptiveThresholdTrigger: i64Default("adaptive_threshold_trigger", 10),
		AdaptiveRelaxPct:         f64Default("adaptive_relax_pct", 20.0),
		// Market Phase Detection
		MarketPhaseRelaxEnabled:     vals["market_phase_relax_enabled"] == "true",
		MarketPhaseIndexDropTrigger: f64Default("market_phase_index_drop_trigger", -1.0),
		MarketPhaseRelaxPct:         f64Default("market_phase_relax_pct", 15.0),
		// Hard Rule Escalation
		EscalationEnabled:   vals["escalation_enabled"] == "true",
		EscalationTrigger:   i64Default("escalation_trigger", 20),
		EscalationStepPct:   f64Default("escalation_step_pct", 10.0),
		EscalationMaxStages: i64Default("escalation_max_stages", 5),
		// Hard Rule Feedback
		HardRuleFeedbackEnabled:      vals["hard_rule_feedback_enabled"] == "true",
		HardRuleFeedbackWindow:       i64Default("hard_rule_feedback_window", 10),
		HardRuleFeedbackThresholdPct: f64Default("hard_rule_feedback_threshold_pct", 70.0),
	}, nil
}

// GetSetting returns the value for the given key from the settings table.
// Returns an empty string if the key does not exist.
func (db *DB) GetSetting(ctx context.Context, key string) string {
	var value string
	db.QueryRowContext(ctx, "SELECT value FROM settings WHERE key = ?", key).Scan(&value) //nolint:errcheck
	return value
}

// GetAllSettings returns all key-value pairs from the settings table.
func (db *DB) GetAllSettings(ctx context.Context) map[string]string {
	rows, err := db.QueryContext(ctx, "SELECT key, value FROM settings")
	if err != nil {
		return nil
	}
	defer rows.Close()
	result := make(map[string]string)
	for rows.Next() {
		var k, v string
		if rows.Scan(&k, &v) == nil {
			result[k] = v
		}
	}
	return result
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

// GetTodaySelectionLogs returns today's (KST) selection log entries.
// candidates and llm_result columns are omitted to minimize memory usage.
func (db *DB) GetTodaySelectionLogs(ctx context.Context) ([]models.TraderSelectionLog, error) {
	kst, _ := time.LoadLocation("Asia/Seoul")
	today := time.Now().In(kst).Format("2006-01-02")
	rows, err := db.QueryContext(ctx,
		`SELECT id, timestamp, sent_count, selected_code, fail_reason, market, hard_rule_stats
		 FROM trader_selection_logs
		 WHERE date(timestamp) = date(?)
		 ORDER BY id ASC`, today)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []models.TraderSelectionLog
	for rows.Next() {
		var l models.TraderSelectionLog
		if err := rows.Scan(&l.ID, &l.Timestamp, &l.SentCount, &l.SelectedCode, &l.FailReason, &l.Market, &l.HardRuleStats); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	if logs == nil {
		logs = []models.TraderSelectionLog{}
	}
	return logs, nil
}

// GetTodayRankingLogs returns today's (KST) ranking log entries.
func (db *DB) GetTodayRankingLogs(ctx context.Context) ([]models.TraderRankingLog, error) {
	kst, _ := time.LoadLocation("Asia/Seoul")
	today := time.Now().In(kst).Format("2006-01-02")
	rows, err := db.QueryContext(ctx,
		`SELECT id, timestamp, ranking_condition, intersection_count, result_stocks, filtered_stocks, error_message, market
		 FROM trader_ranking_logs
		 WHERE date(timestamp) = date(?)
		 ORDER BY id ASC`, today)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []models.TraderRankingLog
	for rows.Next() {
		var l models.TraderRankingLog
		if err := rows.Scan(&l.ID, &l.Timestamp, &l.RankingCondition, &l.IntersectionCount, &l.ResultStocks, &l.FilteredStocks, &l.ErrorMessage, &l.Market); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	if logs == nil {
		logs = []models.TraderRankingLog{}
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
		  volume_count, strength_count, type_counts,
		  ranking_condition, intersection_count, result_stocks, error_message, market)
		 VALUES (datetime('now'), ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.RankingTypes, log.PriceMin, log.PriceMax,
		log.VolumeCount, log.StrengthCount, log.TypeCounts,
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
		        volume_count, strength_count, type_counts,
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
			&l.VolumeCount, &l.StrengthCount, &l.TypeCounts,
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

// ────────────────────────────────────────────────────────
// Trade Reports
// ────────────────────────────────────────────────────────

// InsertTradeReport inserts a new trade report record for a buy event.
func (db *DB) InsertTradeReport(ctx context.Context, r models.TradeReport) (int64, error) {
	res, err := db.ExecContext(ctx,
		`INSERT INTO trade_reports
		 (date, stock_code, stock_name, buy_order_id, selection_log_id,
		  buy_price, buy_qty, buy_amount, buy_reason, buy_indicators, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))`,
		r.Date, r.StockCode, r.StockName, r.BuyOrderID, r.SelectionLogID,
		r.BuyPrice, r.BuyQty, r.BuyAmount, r.BuyReason, r.BuyIndicators)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateTradeReportOnSell updates a trade_report when the position is sold.
// Matches by buy_order_id. Calculates profit_amount and profit_pct.
func (db *DB) UpdateTradeReportOnSell(ctx context.Context, buyOrderID, sellOrderID int64, sellPrice float64, sellQty int, sellReason, sellIndicators string) error {
	var buyPrice float64
	var buyQty int
	_ = db.QueryRowContext(ctx,
		`SELECT buy_price, buy_qty FROM trade_reports WHERE buy_order_id = ? AND sold_at IS NULL`,
		buyOrderID).Scan(&buyPrice, &buyQty)

	sellAmount := sellPrice * float64(sellQty)
	// profitAmount: (매도가 - 매수가) × 매도수량.
	// buyQty를 기준으로 계산하면 부분매도 또는 sellPrice=0(GetStockInfo 실패) 시 전액 손실로 기록되는 버그가 있음.
	profitAmount := (sellPrice - buyPrice) * float64(sellQty)
	profitPct := 0.0
	if buyPrice > 0 {
		profitPct = (sellPrice - buyPrice) / buyPrice * 100
	}

	_, err := db.ExecContext(ctx,
		`UPDATE trade_reports
		 SET sell_order_id=?, sell_price=?, sell_qty=?, sell_amount=?,
		     sell_reason=?, sell_indicators=?,
		     profit_amount=?, profit_pct=?, sold_at=datetime('now')
		 WHERE buy_order_id=? AND sold_at IS NULL`,
		sellOrderID, sellPrice, sellQty, sellAmount,
		sellReason, sellIndicators,
		profitAmount, profitPct, buyOrderID)
	return err
}

// GetTradeReports queries trade_reports with optional filters. Results sorted by id DESC.
func (db *DB) GetTradeReports(ctx context.Context, date, stockCode string, limit, offset int) ([]models.TradeReport, error) {
	where := "WHERE 1=1"
	args := []any{}
	if date != "" {
		where += " AND date = ?"
		args = append(args, date)
	}
	if stockCode != "" {
		where += " AND stock_code = ?"
		args = append(args, stockCode)
	}
	args = append(args, limit, offset)

	rows, err := db.QueryContext(ctx,
		`SELECT id, date, stock_code, stock_name,
		        buy_order_id, sell_order_id, selection_log_id,
		        buy_price, buy_qty, buy_amount, buy_reason, buy_indicators,
		        sell_price, sell_qty, sell_amount, sell_reason, sell_indicators,
		        profit_amount, profit_pct, created_at, sold_at
		 FROM trade_reports `+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []models.TradeReport
	for rows.Next() {
		var r models.TradeReport
		if err := rows.Scan(
			&r.ID, &r.Date, &r.StockCode, &r.StockName,
			&r.BuyOrderID, &r.SellOrderID, &r.SelectionLogID,
			&r.BuyPrice, &r.BuyQty, &r.BuyAmount, &r.BuyReason, &r.BuyIndicators,
			&r.SellPrice, &r.SellQty, &r.SellAmount, &r.SellReason, &r.SellIndicators,
			&r.ProfitAmount, &r.ProfitPct, &r.CreatedAt, &r.SoldAt,
		); err != nil {
			return nil, err
		}
		reports = append(reports, r)
	}
	if reports == nil {
		reports = []models.TradeReport{}
	}
	return reports, nil
}

// GetCompletedTradesBySoldDate returns trade_reports that were sold on the given date (KST "YYYY-MM-DD").
// Used by GenerateDailyReport to collect trades by sell date, not buy date.
func (db *DB) GetCompletedTradesBySoldDate(ctx context.Context, date string) ([]models.TradeReport, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, date, stock_code, stock_name,
		        buy_order_id, sell_order_id, selection_log_id,
		        buy_price, buy_qty, buy_amount, buy_reason, buy_indicators,
		        sell_price, sell_qty, sell_amount, sell_reason, sell_indicators,
		        profit_amount, profit_pct, created_at, sold_at
		 FROM trade_reports
		 WHERE sold_at IS NOT NULL
		   AND date(sold_at) = ?
		 ORDER BY id ASC`, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []models.TradeReport
	for rows.Next() {
		var r models.TradeReport
		if err := rows.Scan(
			&r.ID, &r.Date, &r.StockCode, &r.StockName,
			&r.BuyOrderID, &r.SellOrderID, &r.SelectionLogID,
			&r.BuyPrice, &r.BuyQty, &r.BuyAmount, &r.BuyReason, &r.BuyIndicators,
			&r.SellPrice, &r.SellQty, &r.SellAmount, &r.SellReason, &r.SellIndicators,
			&r.ProfitAmount, &r.ProfitPct, &r.CreatedAt, &r.SoldAt,
		); err != nil {
			return nil, err
		}
		reports = append(reports, r)
	}
	if reports == nil {
		reports = []models.TradeReport{}
	}
	return reports, nil
}

// ────────────────────────────────────────────────────────
// Daily Reports
// ────────────────────────────────────────────────────────

// InsertOrUpdateDailyReport upserts a daily_report record.
func (db *DB) InsertOrUpdateDailyReport(ctx context.Context, r models.DailyReport) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO daily_reports
		 (date, total_trades, winning_trades, losing_trades,
		  total_profit_amount, avg_profit_pct,
		  best_trade, worst_trade, trade_summary, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
		 ON CONFLICT(date) DO UPDATE SET
		   total_trades=excluded.total_trades,
		   winning_trades=excluded.winning_trades,
		   losing_trades=excluded.losing_trades,
		   total_profit_amount=excluded.total_profit_amount,
		   avg_profit_pct=excluded.avg_profit_pct,
		   best_trade=excluded.best_trade,
		   worst_trade=excluded.worst_trade,
		   trade_summary=excluded.trade_summary,
		   created_at=excluded.created_at`,
		r.Date, r.TotalTrades, r.WinningTrades, r.LosingTrades,
		r.TotalProfitAmount, r.AvgProfitPct,
		r.BestTrade, r.WorstTrade, r.TradeSummary)
	return err
}

// GetDailyReports returns daily_reports sorted by date DESC.
func (db *DB) GetDailyReports(ctx context.Context, from, to string, limit int) ([]models.DailyReport, error) {
	where := "WHERE 1=1"
	args := []any{}
	if from != "" {
		where += " AND date >= ?"
		args = append(args, from)
	}
	if to != "" {
		where += " AND date <= ?"
		args = append(args, to)
	}
	args = append(args, limit)

	rows, err := db.QueryContext(ctx,
		`SELECT id, date, total_trades, winning_trades, losing_trades,
		        total_profit_amount, avg_profit_pct,
		        best_trade, worst_trade, trade_summary, created_at
		 FROM daily_reports `+where+` ORDER BY date DESC LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []models.DailyReport
	for rows.Next() {
		var r models.DailyReport
		if err := rows.Scan(
			&r.ID, &r.Date, &r.TotalTrades, &r.WinningTrades, &r.LosingTrades,
			&r.TotalProfitAmount, &r.AvgProfitPct,
			&r.BestTrade, &r.WorstTrade, &r.TradeSummary, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		reports = append(reports, r)
	}
	if reports == nil {
		reports = []models.DailyReport{}
	}
	return reports, nil
}

// ────────────────────────────────────────────────────────
// Optimization Reports
// ────────────────────────────────────────────────────────

// UpsertOptimizationReport inserts or replaces an optimization_report for the given date.
func (db *DB) UpsertOptimizationReport(ctx context.Context, r models.OptimizationReport) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO optimization_reports
		 (date, overall_assessment, suggestions, apply_mode_snapshot, created_at)
		 VALUES (?, ?, ?, ?, datetime('now'))
		 ON CONFLICT(date) DO UPDATE SET
		   overall_assessment=excluded.overall_assessment,
		   suggestions=excluded.suggestions,
		   apply_mode_snapshot=excluded.apply_mode_snapshot,
		   created_at=excluded.created_at`,
		r.Date, r.OverallAssessment, r.Suggestions, r.ApplyModeSnapshot)
	return err
}

// GetOptimizationReports returns optimization_reports sorted by date DESC.
func (db *DB) GetOptimizationReports(ctx context.Context, from, to string, limit int) ([]models.OptimizationReport, error) {
	where := "WHERE 1=1"
	args := []any{}
	if from != "" {
		where += " AND date >= ?"
		args = append(args, from)
	}
	if to != "" {
		where += " AND date <= ?"
		args = append(args, to)
	}
	args = append(args, limit)

	rows, err := db.QueryContext(ctx,
		`SELECT id, date, overall_assessment, suggestions, apply_mode_snapshot, created_at
		 FROM optimization_reports `+where+` ORDER BY date DESC LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []models.OptimizationReport
	for rows.Next() {
		var r models.OptimizationReport
		if err := rows.Scan(&r.ID, &r.Date, &r.OverallAssessment, &r.Suggestions, &r.ApplyModeSnapshot, &r.CreatedAt); err != nil {
			return nil, err
		}
		reports = append(reports, r)
	}
	if reports == nil {
		reports = []models.OptimizationReport{}
	}
	return reports, nil
}

// GetOptimizationReportByDate returns the optimization_report for the given date, or nil if not found.
func (db *DB) GetOptimizationReportByDate(ctx context.Context, date string) (*models.OptimizationReport, error) {
	row := db.QueryRowContext(ctx,
		`SELECT id, date, overall_assessment, suggestions, apply_mode_snapshot, created_at
		 FROM optimization_reports WHERE date = ?`, date)
	var r models.OptimizationReport
	if err := row.Scan(&r.ID, &r.Date, &r.OverallAssessment, &r.Suggestions, &r.ApplyModeSnapshot, &r.CreatedAt); err != nil {
		return nil, err
	}
	return &r, nil
}
