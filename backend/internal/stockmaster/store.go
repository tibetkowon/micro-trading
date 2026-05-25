package stockmaster

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Store handles persistence of StockMaster records in SQLite.
type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Upsert(ctx context.Context, records []*StockMaster) error {
	if len(records) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO stock_masters(stock_code,data,stock_name,market_type,is_etf,updated_at)
		VALUES(?,?,?,?,?,?)
		ON CONFLICT(stock_code) DO UPDATE SET data=excluded.data, stock_name=excluded.stock_name,
		market_type=excluded.market_type, is_etf=excluded.is_etf, updated_at=excluded.updated_at`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, r := range records {
		data, err := json.Marshal(r)
		if err != nil {
			tx.Rollback()
			return err
		}
		if _, err := stmt.ExecContext(ctx, r.StockCode, string(data), r.StockName, r.MarketType, boolInt(r.IsETF), now); err != nil {
			tx.Rollback()
			return fmt.Errorf("upsert stock master %s: %w", r.StockCode, err)
		}
	}
	return tx.Commit()
}

func (s *Store) Count(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM stock_masters").Scan(&n)
	return n, err
}

func (s *Store) GetByCode(ctx context.Context, stockCode string) (*StockMaster, error) {
	var blob string
	if err := s.db.QueryRowContext(ctx, "SELECT data FROM stock_masters WHERE stock_code=?", stockCode).Scan(&blob); err != nil {
		return nil, nil
	}
	var m StockMaster
	if err := json.Unmarshal([]byte(blob), &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *Store) Search(ctx context.Context, q string, etfOnly bool, market string, limit int) ([]*StockMaster, error) {
	if limit <= 0 {
		limit = 200
	}
	q = strings.ToLower(q)

	query := "SELECT data FROM stock_masters WHERE 1=1"
	args := make([]any, 0, 4)

	if q != "" {
		escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q)
		pattern := "%" + escaped + "%"
		query += " AND (LOWER(stock_code) LIKE ? ESCAPE '\\' OR LOWER(stock_name) LIKE ? ESCAPE '\\')"
		args = append(args, pattern, pattern)
	}
	if etfOnly {
		query += " AND is_etf = 1"
	}
	if market != "" {
		query += " AND market_type = ?"
		args = append(args, market)
	}
	query += " ORDER BY stock_code LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*StockMaster
	for rows.Next() {
		var blob string
		if err := rows.Scan(&blob); err != nil {
			return nil, err
		}
		var m StockMaster
		if err := json.Unmarshal([]byte(blob), &m); err != nil {
			continue
		}
		results = append(results, &m)
	}
	return results, rows.Err()
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
