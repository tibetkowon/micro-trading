package ops

import (
	"testing"
	"time"
)

func TestStockInfoCache_Hit(t *testing.T) {
	code := "TEST01"
	info := &StockInfo{StockCode: code, CurrentPrice: "10000"}
	siCache.mu.Lock()
	siCache.entries[code] = &cacheEntry{info: info, fetchedAt: time.Now()}
	siCache.mu.Unlock()

	siCache.mu.Lock()
	e, ok := siCache.entries[code]
	siCache.mu.Unlock()
	if !ok {
		t.Fatal("cache entry not found")
	}
	if time.Since(e.fetchedAt) >= siCacheTTL {
		t.Fatal("cache entry already expired")
	}
	if e.info != info {
		t.Error("expected same pointer")
	}
}

func TestStockInfoCache_Invalidate(t *testing.T) {
	code := "TEST02"
	siCache.mu.Lock()
	siCache.entries[code] = &cacheEntry{info: &StockInfo{StockCode: code}, fetchedAt: time.Now()}
	siCache.mu.Unlock()

	InvalidateStockInfoCache(code)

	siCache.mu.Lock()
	_, ok := siCache.entries[code]
	siCache.mu.Unlock()
	if ok {
		t.Error("expected cache entry to be removed after invalidation")
	}
}

func TestStockInfoCache_Expiry(t *testing.T) {
	code := "TEST03"
	siCache.mu.Lock()
	siCache.entries[code] = &cacheEntry{
		info:      &StockInfo{StockCode: code},
		fetchedAt: time.Now().Add(-siCacheTTL - time.Second), // already expired
	}
	siCache.mu.Unlock()

	siCache.mu.Lock()
	e, ok := siCache.entries[code]
	siCache.mu.Unlock()
	if !ok {
		t.Fatal("entry should still be in map (eviction is lazy)")
	}
	if time.Since(e.fetchedAt) < siCacheTTL {
		t.Error("expected entry to be expired")
	}
}
