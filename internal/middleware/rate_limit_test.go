package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	limiter := NewRateLimiter(5, time.Minute)
	defer limiter.Stop()

	ip := "192.168.1.1"

	// Should allow first 5 requests
	for i := 0; i < 5; i++ {
		if !limiter.Allow(ip) {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 6th request should be denied
	if limiter.Allow(ip) {
		t.Error("6th request should be rate limited")
	}
}

func TestRateLimiter_WindowReset(t *testing.T) {
	limiter := NewRateLimiter(2, 50*time.Millisecond)
	defer limiter.Stop()

	ip := "192.168.1.1"

	// Use up the limit
	limiter.Allow(ip)
	limiter.Allow(ip)

	if limiter.Allow(ip) {
		t.Error("3rd request should be denied")
	}

	// Wait for window to reset
	time.Sleep(60 * time.Millisecond)

	if !limiter.Allow(ip) {
		t.Error("request after window reset should be allowed")
	}
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)
	defer limiter.Stop()

	ip1 := "192.168.1.1"
	ip2 := "192.168.1.2"

	// Use up limit for ip1
	limiter.Allow(ip1)
	limiter.Allow(ip1)

	// ip2 should still be allowed
	if !limiter.Allow(ip2) {
		t.Error("ip2 should be allowed (separate limit)")
	}

	// ip1 should be denied
	if limiter.Allow(ip1) {
		t.Error("ip1 should be denied")
	}
}

func TestRateLimitMiddleware_Integration(t *testing.T) {
	// Create a custom limiter with low limit for testing
	limiter := NewRateLimiter(2, time.Minute)
	defer limiter.Stop()

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with rate limit middleware
	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow("127.0.0.1") {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	wrapped := middleware(handler)

	// First 2 requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		rr := httptest.NewRecorder()
		wrapped.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, rr.Code)
		}
	}

	// 3rd request should fail
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("3rd request: expected 429, got %d", rr.Code)
	}
}
