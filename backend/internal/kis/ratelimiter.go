package kis

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter wraps golang.org/x/time/rate to enforce KIS API TPS limits.
// KIS allows up to 20 requests/second for most endpoints, but in practice
// sustained bursts above ~5-7 RPS trigger EGW00201 (TPS exceeded) errors.
type RateLimiter struct {
	mu           sync.Mutex
	limiter      *rate.Limiter
	normalLimit  rate.Limit
	restoreTimer *time.Timer
}

// NewRateLimiter creates a limiter allowing `rps` requests per second
// with a burst of `burst` (set burst == 1 for strict per-request spacing).
func NewRateLimiter(rps, burst int) *RateLimiter {
	lim := rate.Limit(rps)
	return &RateLimiter{
		limiter:     rate.NewLimiter(lim, burst),
		normalLimit: lim,
	}
}

// Wait blocks until a request token is available or ctx is cancelled.
func (rl *RateLimiter) Wait(ctx context.Context) error {
	return rl.limiter.Wait(ctx)
}

// Throttle temporarily reduces the rate to `slowRPS` for `duration`,
// then restores the normal rate. Safe to call concurrently; multiple
// TPS errors in flight only extend the throttle window.
func (rl *RateLimiter) Throttle(slowRPS int, duration time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.limiter.SetLimit(rate.Limit(slowRPS))

	if rl.restoreTimer != nil {
		rl.restoreTimer.Reset(duration)
		return
	}
	rl.restoreTimer = time.AfterFunc(duration, func() {
		rl.mu.Lock()
		defer rl.mu.Unlock()
		rl.limiter.SetLimit(rl.normalLimit)
		rl.restoreTimer = nil
	})
}
