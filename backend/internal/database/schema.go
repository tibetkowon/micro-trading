package database

import "database/sql"

const createTablesSQL = `
CREATE TABLE IF NOT EXISTS settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tokens (
    id           TEXT PRIMARY KEY DEFAULT 'current',
    access_token TEXT NOT NULL,
    issued_at    TEXT,
    expires_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS orders (
    id            INTEGER PRIMARY KEY,
    kis_order_id  TEXT,
    stock_code    TEXT NOT NULL,
    stock_name    TEXT,
    order_type    TEXT NOT NULL,
    qty           INTEGER NOT NULL,
    price         REAL NOT NULL,
    filled_price  REAL,
    status        TEXT NOT NULL,
    source        TEXT,
    target_pct    REAL,
    stop_pct      REAL,
    sell_reason   TEXT,
    created_at    TEXT NOT NULL,
    expire_at     TEXT
);
CREATE INDEX IF NOT EXISTS idx_orders_created ON orders(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_orders_code ON orders(stock_code);
CREATE INDEX IF NOT EXISTS idx_orders_kis_id ON orders(kis_order_id);

CREATE TABLE IF NOT EXISTS monitored_positions (
    code       TEXT PRIMARY KEY,
    data       TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS balances (
    id         INTEGER PRIMARY KEY,
    data       TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_balances_created ON balances(created_at DESC);

CREATE TABLE IF NOT EXISTS scan_logs (
    id                    INTEGER PRIMARY KEY,
    timestamp             TEXT NOT NULL,
    total_candidates      INTEGER,
    stocks_found          INTEGER,
    top_stocks            TEXT,
    stock_raw_data        TEXT,
    below_threshold_data  TEXT,
    ordered               INTEGER DEFAULT 0,
    ordered_code          TEXT,
    skip_reason           TEXT,
    score_stats           TEXT,
    created_at            TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_scan_logs_created ON scan_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_scan_logs_timestamp ON scan_logs(timestamp);

CREATE TABLE IF NOT EXISTS trade_reports (
    id              INTEGER PRIMARY KEY,
    data            TEXT NOT NULL,
    date            TEXT,
    stock_code      TEXT,
    buy_order_id    INTEGER,
    sell_order_id   INTEGER,
    created_at      TEXT NOT NULL,
    sold_at         TEXT
);
CREATE INDEX IF NOT EXISTS idx_trade_reports_date ON trade_reports(date DESC);
CREATE INDEX IF NOT EXISTS idx_trade_reports_created ON trade_reports(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_trade_reports_stock ON trade_reports(stock_code);
CREATE INDEX IF NOT EXISTS idx_trade_reports_buy_order ON trade_reports(buy_order_id);

CREATE TABLE IF NOT EXISTS daily_reports (
    date       TEXT PRIMARY KEY,
    data       TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS simulation_results (
    date       TEXT PRIMARY KEY,
    data       TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS stock_masters (
    stock_code TEXT PRIMARY KEY,
    data       TEXT NOT NULL,
    stock_name TEXT,
    market_type TEXT,
    is_etf INTEGER DEFAULT 0,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_stock_masters_name ON stock_masters(stock_name);
`

func migrate(db *sql.DB) error {
	_, err := db.Exec(createTablesSQL)
	return err
}
