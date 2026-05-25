//go:build cgo

package stockmaster

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	_, err = db.Exec(`CREATE TABLE stock_masters (
		stock_code  TEXT PRIMARY KEY,
		data        TEXT NOT NULL,
		stock_name  TEXT,
		market_type TEXT,
		is_etf      INTEGER DEFAULT 0,
		updated_at  TEXT NOT NULL
	)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	return db
}

func insertStock(t *testing.T, db *sql.DB, m StockMaster) {
	t.Helper()
	data, _ := json.Marshal(m)
	_, err := db.Exec(
		`INSERT INTO stock_masters(stock_code,data,stock_name,market_type,is_etf,updated_at)
		 VALUES(?,?,?,?,?,?)`,
		m.StockCode, string(data), m.StockName, m.MarketType, boolInt(m.IsETF),
		time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("insert %s: %v", m.StockCode, err)
	}
}

func TestSearch_SQLFiltering(t *testing.T) {
	db := setupTestDB(t)
	store := NewStore(db)
	ctx := context.Background()

	insertStock(t, db, StockMaster{StockCode: "005930", StockName: "삼성전자", MarketType: "KOSPI", IsETF: false})
	insertStock(t, db, StockMaster{StockCode: "000660", StockName: "SK하이닉스", MarketType: "KOSPI", IsETF: false})
	insertStock(t, db, StockMaster{StockCode: "069500", StockName: "KODEX 200", MarketType: "KOSPI", IsETF: true})
	insertStock(t, db, StockMaster{StockCode: "293180", StockName: "KODEX 삼성그룹", MarketType: "KOSDAQ", IsETF: true})
	insertStock(t, db, StockMaster{StockCode: "035720", StockName: "카카오", MarketType: "KOSPI", IsETF: false})

	tests := []struct {
		name    string
		q       string
		etfOnly bool
		market  string
		limit   int
		want    int
	}{
		{"no filter", "", false, "", 10, 5},
		{"q=삼성 matches 삼성전자+KODEX삼성", "삼성", false, "", 10, 2},
		{"etf only", "", true, "", 10, 2},
		{"market=KOSDAQ", "", false, "KOSDAQ", 10, 1},
		{"q=kodex etf only", "kodex", true, "", 10, 2},
		{"limit=2", "", false, "", 2, 2},
		{"q=stock_code 005930", "005930", false, "", 10, 1},
		{"q=% returns no match (wildcard escape)", "%", false, "", 10, 0},
		{"q=_ returns no match (wildcard escape)", "_", false, "", 10, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			results, err := store.Search(ctx, tc.q, tc.etfOnly, tc.market, tc.limit)
			if err != nil {
				t.Fatalf("Search error: %v", err)
			}
			if len(results) != tc.want {
				t.Errorf("got %d results, want %d", len(results), tc.want)
			}
		})
	}
}
