package handlers

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements a simple token-bucket rate limiter.
type RateLimiter struct {
	mu       sync.Mutex
	tokens   int
	max      int
	interval time.Duration
	last     time.Time
}

// NewRateLimiter creates a rate limiter allowing max requests per interval.
func NewRateLimiter(max int, interval time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:   max,
		max:      max,
		interval: interval,
		last:     time.Now(),
	}
}

// Allow returns true if the request is within rate limits.
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.last)

	// Replenish tokens based on elapsed time
	replenish := int(elapsed / rl.interval)
	if replenish > 0 {
		rl.tokens += replenish
		if rl.tokens > rl.max {
			rl.tokens = rl.max
		}
		rl.last = now
	}

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}
	return false
}

// RateLimitMiddleware wraps a handler with rate limiting.
func RateLimitMiddleware(limiter *RateLimiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			writeJSON(w, http.StatusTooManyRequests, map[string]string{
				"error": "too many requests, please try again later",
			})
			return
		}
		next(w, r)
	}
}
