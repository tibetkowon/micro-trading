package ops

import (
	"context"
	"sync"
	"time"

	"github.com/micro-trading-for-agent/backend/internal/kis"
)

type stockInfoCache struct {
	mu      sync.Mutex
	entries map[string]*cacheEntry
}

type cacheEntry struct {
	info      *StockInfo
	fetchedAt time.Time
}

var siCache = &stockInfoCache{entries: make(map[string]*cacheEntry)}

const siCacheTTL = 30 * time.Second

// GetStockInfoCached returns a cached StockInfo if available and within TTL,
// otherwise fetches fresh data and stores it in the cache.
func GetStockInfoCached(ctx context.Context, client *kis.Client, stockCode string) (*StockInfo, error) {
	siCache.mu.Lock()
	if e, ok := siCache.entries[stockCode]; ok && time.Since(e.fetchedAt) < siCacheTTL {
		info := e.info
		siCache.mu.Unlock()
		return info, nil
	}
	siCache.mu.Unlock()

	info, err := GetStockInfo(ctx, client, stockCode)
	if err != nil {
		return nil, err
	}

	siCache.mu.Lock()
	siCache.entries[stockCode] = &cacheEntry{info: info, fetchedAt: time.Now()}
	siCache.mu.Unlock()

	return info, nil
}

// InvalidateStockInfoCache removes a specific stock from the cache (e.g. after order fill).
func InvalidateStockInfoCache(stockCode string) {
	siCache.mu.Lock()
	delete(siCache.entries, stockCode)
	siCache.mu.Unlock()
}
