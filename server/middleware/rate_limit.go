package middleware

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter provides rate limiting functionality.
type RateLimiter struct {
	mu     sync.RWMutex
	limits map[string]*rate.Limiter
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		limits: make(map[string]*rate.Limiter),
	}
}

// getLimiter gets or creates a limiter for the given key.
func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if limiter, ok := rl.limits[key]; ok {
		return limiter
	}

	// 10 requests per second, with burst of 20
	limiter := rate.NewLimiter(rate.Every(time.Second/10), 20)
	rl.limits[key] = limiter
	return limiter
}

// Allow checks if a request is allowed for the given key.
func (rl *RateLimiter) Allow(key string) bool {
	return rl.getLimiter(key).Allow()
}

// Wait waits for a request to be allowed.
// Returns error if the context is cancelled or rate limit exceeded.
func (rl *RateLimiter) Wait(ctx context.Context, key string) error {
	return rl.getLimiter(key).Wait(ctx)
}
