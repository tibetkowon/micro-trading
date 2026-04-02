package mst

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Store handles persistence of StockMaster records in the SQLite stock_masters table.
type Store struct {
	db *sql.DB
}

// NewStore creates a Store backed by the given *sql.DB.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Upsert inserts or updates a batch of StockMaster records.
func (s *Store) Upsert(ctx context.Context, records []*StockMaster) error {
	if len(records) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO stock_masters
			(stock_code, stock_name, isin, market_type, group_code, is_etf, is_domestic_equity_etf, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(stock_code) DO UPDATE SET
			stock_name           = excluded.stock_name,
			isin                 = excluded.isin,
			market_type          = excluded.market_type,
			group_code           = excluded.group_code,
			is_etf               = excluded.is_etf,
			is_domestic_equity_etf = excluded.is_domestic_equity_etf,
			updated_at           = excluded.updated_at`)
	if err != nil {
		return fmt.Errorf("prepare stmt: %w", err)
	}
	defer stmt.Close()

	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	for _, r := range records {
		if _, err := stmt.ExecContext(ctx,
			r.StockCode, r.StockName, r.ISIN, r.MarketType, r.GroupCode,
			boolToInt(r.IsETF), boolToInt(r.IsDomesticEquityETF), now,
		); err != nil {
			return fmt.Errorf("upsert %s: %w", r.StockCode, err)
		}
	}
	return tx.Commit()
}

// Count returns the number of rows in stock_masters.
func (s *Store) Count(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM stock_masters`).Scan(&n)
	return n, err
}

// GetByCode returns a StockMaster by stock code, or nil if not found.
func (s *Store) GetByCode(ctx context.Context, stockCode string) (*StockMaster, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT stock_code, stock_name, isin, market_type, group_code, is_etf, is_domestic_equity_etf
		 FROM stock_masters WHERE stock_code = ?`, stockCode)
	m := &StockMaster{}
	var isETF, isDomestic int
	err := row.Scan(&m.StockCode, &m.StockName, &m.ISIN, &m.MarketType, &m.GroupCode, &isETF, &isDomestic)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	m.IsETF = isETF == 1
	m.IsDomesticEquityETF = isDomestic == 1
	return m, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
