package stockmaster

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

const colStockMaster = "stock_masters"

// Store handles persistence of StockMaster records in Firestore.
type Store struct {
	client *firestore.Client
}

// NewStore creates a Store backed by the given Firestore client.
func NewStore(client *firestore.Client) *Store {
	return &Store{client: client}
}

// Upsert inserts or updates a batch of StockMaster records (doc ID = stock_code).
func (s *Store) Upsert(ctx context.Context, records []*StockMaster) error {
	if len(records) == 0 {
		return nil
	}

	const batchSize = 500 // Firestore batch write limit
	now := time.Now().UTC()

	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}
		batch := s.client.Batch()
		for _, r := range records[i:end] {
			ref := s.client.Collection(colStockMaster).Doc(r.StockCode)
			batch.Set(ref, map[string]any{
				"stock_code":             r.StockCode,
				"stock_name":             r.StockName,
				"isin":                   r.ISIN,
				"market_type":            r.MarketType,
				"group_code":             r.GroupCode,
				"is_etf":                 r.IsETF,
				"is_etn":                 r.IsETN,
				"is_domestic_equity_etf": r.IsDomesticEquityETF,
				"listed_shares":          r.ListedShares,
				"updated_at":             now,
			})
		}
		if _, err := batch.Commit(ctx); err != nil {
			return fmt.Errorf("batch commit (offset %d): %w", i, err)
		}
	}
	return nil
}

// Count returns the number of documents in stock_masters.
func (s *Store) Count(ctx context.Context) (int, error) {
	iter := s.client.Collection(colStockMaster).Documents(ctx)
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

// GetByCode returns a StockMaster by stock code, or nil if not found.
func (s *Store) GetByCode(ctx context.Context, stockCode string) (*StockMaster, error) {
	snap, err := s.client.Collection(colStockMaster).Doc(stockCode).Get(ctx)
	if err != nil {
		return nil, nil // not found
	}
	data := snap.Data()
	m := &StockMaster{
		StockCode:  stockCode,
		StockName:  strField(data, "stock_name"),
		ISIN:       strField(data, "isin"),
		MarketType: strField(data, "market_type"),
		GroupCode:  strField(data, "group_code"),
	}
	if v, ok := data["is_etf"].(bool); ok {
		m.IsETF = v
	}
	if v, ok := data["is_etn"].(bool); ok {
		m.IsETN = v
	}
	if v, ok := data["is_domestic_equity_etf"].(bool); ok {
		m.IsDomesticEquityETF = v
	}
	if v, ok := data["listed_shares"].(int64); ok {
		m.ListedShares = v
	}
	return m, nil
}

// Search returns stock master records filtered by name/code substring, ETF flag, and market type.
// limit caps the number of results (0 = use default of 200).
func (s *Store) Search(ctx context.Context, q string, etfOnly bool, market string, limit int) ([]*StockMaster, error) {
	if limit <= 0 {
		limit = 200
	}

	iter := s.client.Collection(colStockMaster).Documents(ctx)
	defer iter.Stop()

	var results []*StockMaster
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		data := snap.Data()
		code := strField(data, "stock_code")
		name := strField(data, "stock_name")
		mkt := strField(data, "market_type")

		// Filter
		if q != "" {
			if !containsCI(code, q) && !containsCI(name, q) {
				continue
			}
		}
		if etfOnly {
			isETF, _ := data["is_etf"].(bool)
			if !isETF {
				continue
			}
		}
		if market != "" && mkt != market {
			continue
		}

		m := &StockMaster{
			StockCode:  code,
			StockName:  name,
			MarketType: mkt,
			ISIN:       strField(data, "isin"),
			GroupCode:  strField(data, "group_code"),
		}
		if v, ok := data["is_etf"].(bool); ok {
			m.IsETF = v
		}
		if v, ok := data["is_domestic_equity_etf"].(bool); ok {
			m.IsDomesticEquityETF = v
		}
		if v, ok := data["listed_shares"].(int64); ok {
			m.ListedShares = v
		}
		results = append(results, m)
		if len(results) >= limit {
			break
		}
	}
	return results, nil
}

// containsCI reports whether s contains substr (case-insensitive ASCII).
func containsCI(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	sl := toLowerASCII(s)
	subl := toLowerASCII(substr)
	for i := 0; i <= len(sl)-len(subl); i++ {
		if sl[i:i+len(subl)] == subl {
			return true
		}
	}
	return false
}

func toLowerASCII(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

func strField(data map[string]any, key string) string {
	v, _ := data[key].(string)
	return v
}
