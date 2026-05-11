package trader

import (
	"sync"
	"time"

	"github.com/micro-trading-for-agent/backend/internal/kis"
)

// StreamMonitor tracks real-time ticks to detect big trades and velocity surges.
type StreamMonitor struct {
	mu                sync.Mutex
	ticks             map[string][]kis.PriceEvent
	bigTradeAmount    float64
	velocityThreshold float64 // ticks per second
	hijackCh          chan string
	hijacked          map[string]time.Time // to prevent multiple hijacks for the same stock quickly
}

// NewStreamMonitor creates a new StreamMonitor.
func NewStreamMonitor(bigTradeAmount, velocityThreshold float64, hijackCh chan string) *StreamMonitor {
	return &StreamMonitor{
		ticks:             make(map[string][]kis.PriceEvent),
		bigTradeAmount:    bigTradeAmount,
		velocityThreshold: velocityThreshold,
		hijackCh:          hijackCh,
		hijacked:          make(map[string]time.Time),
	}
}

// AddTick processes a new price event and evaluates bypass conditions.
func (sm *StreamMonitor) AddTick(ev kis.PriceEvent) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Clear out hijacked status if older than 1 minute to allow re-hijacking if needed
	if last, ok := sm.hijacked[ev.StockCode]; ok && time.Since(last) > time.Minute {
		delete(sm.hijacked, ev.StockCode)
	}
	if _, ok := sm.hijacked[ev.StockCode]; ok {
		return // Already recently hijacked, wait a bit
	}

	now := time.Now()
	cutoff := now.Add(-10 * time.Second)

	// Filter ticks older than 10 seconds
	var recent []kis.PriceEvent
	for _, tick := range sm.ticks[ev.StockCode] {
		if tick.Timestamp.After(cutoff) {
			recent = append(recent, tick)
		}
	}
	recent = append(recent, ev)
	sm.ticks[ev.StockCode] = recent

	// Evaluate conditions
	bigTradeCount := 0
	for _, tick := range recent {
		if tick.Price*float64(tick.Qty) >= sm.bigTradeAmount {
			bigTradeCount++
		}
	}

	velocity := float64(len(recent)) / 10.0

	if bigTradeCount >= 2 || velocity >= sm.velocityThreshold {
		// Trigger hijack
		sm.hijacked[ev.StockCode] = now
		select {
		case sm.hijackCh <- ev.StockCode:
		default:
			// channel full, drop it
		}
	}
}

// UpdateConfig updates the bypass settings dynamically.
func (sm *StreamMonitor) UpdateConfig(bigTradeAmount, velocityThreshold float64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.bigTradeAmount = bigTradeAmount
	sm.velocityThreshold = velocityThreshold
}
