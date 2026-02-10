package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type client struct {
	requests int
	lastSeen time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*client
	maxReqs  int
	window   time.Duration
	stopChan chan struct{}
}

var defaultLimiter *RateLimiter
var limiterOnce sync.Once

// GetDefaultLimiter returns the singleton rate limiter instance
func GetDefaultLimiter() *RateLimiter {
	limiterOnce.Do(func() {
		defaultLimiter = NewRateLimiter(10, time.Minute)
	})
	return defaultLimiter
}

// NewRateLimiter creates a new rate limiter with cleanup goroutine
func NewRateLimiter(maxReqs int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		clients:  make(map[string]*client),
		maxReqs:  maxReqs,
		window:   window,
		stopChan: make(chan struct{}),
	}
	go rl.cleanup()
	return rl
}

// cleanup periodically removes stale client entries to prevent memory leaks
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, c := range rl.clients {
				if now.Sub(c.lastSeen) > rl.window*2 {
					delete(rl.clients, ip)
				}
			}
			rl.mu.Unlock()
		case <-rl.stopChan:
			return
		}
	}
}

// Stop gracefully stops the cleanup goroutine
func (rl *RateLimiter) Stop() {
	close(rl.stopChan)
}

// Allow checks if a request from the given IP is allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	c, exists := rl.clients[ip]
	if !exists {
		rl.clients[ip] = &client{
			requests: 1,
			lastSeen: time.Now(),
		}
		return true
	}

	// Reset window if expired
	if time.Since(c.lastSeen) > rl.window {
		c.requests = 1
		c.lastSeen = time.Now()
		return true
	}

	if c.requests >= rl.maxReqs {
		return false
	}

	c.requests++
	return true
}

// RateLimitMiddleware creates an HTTP middleware for rate limiting
func RateLimitMiddleware(next http.Handler) http.Handler {
	limiter := GetDefaultLimiter()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, "Invalid IP", http.StatusInternalServerError)
			return
		}

		if !limiter.Allow(ip) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": "rate_limit_exceeded", "message": "Too many requests. Please try again later."}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// NewRateLimitMiddlewareWithConfig creates a rate limit middleware with custom settings
func NewRateLimitMiddlewareWithConfig(maxReqs int, window time.Duration) func(http.Handler) http.Handler {
	limiter := NewRateLimiter(maxReqs, window)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				http.Error(w, "Invalid IP", http.StatusInternalServerError)
				return
			}

			if !limiter.Allow(ip) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error": "rate_limit_exceeded", "message": "Too many requests. Please try again later."}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
