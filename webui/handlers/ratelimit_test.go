package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterAllow(t *testing.T) {
	rl := NewRateLimiter(3, time.Second)

	// First 3 should pass
	for i := 0; i < 3; i++ {
		if !rl.Allow() {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 4th should be blocked
	if rl.Allow() {
		t.Error("4th request should be rate limited")
	}
}

func TestRateLimiterReplenish(t *testing.T) {
	rl := NewRateLimiter(1, 50*time.Millisecond)

	if !rl.Allow() {
		t.Error("first request should be allowed")
	}
	if rl.Allow() {
		t.Error("second request should be blocked")
	}

	// Wait for replenish
	time.Sleep(60 * time.Millisecond)

	if !rl.Allow() {
		t.Error("request after replenish should be allowed")
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)
	handler := RateLimitMiddleware(rl, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// First request: OK
	req := httptest.NewRequest("POST", "/api/login", nil)
	w := httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Second request: rate limited
	w = httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}
}
