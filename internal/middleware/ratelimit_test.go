package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimiter_AllowUnderLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := NewRateLimiter(5, 10, 8, 5, 3)

	r := gin.New()
	r.Use(rl.Middleware())
	r.GET("/test", func(c *gin.Context) { c.String(200, "ok") })

	// Should allow 5 requests (default rate)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("request %d: expected 200, got %d", i+1, w.Code)
		}
	}
}

func TestRateLimiter_BlockOverLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := NewRateLimiter(2, 10, 8, 5, 3)

	r := gin.New()
	r.Use(rl.Middleware())
	r.GET("/test", func(c *gin.Context) { c.String(200, "ok") })

	// First 2 requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("request %d: expected 200, got %d", i+1, w.Code)
		}
	}

	// Third request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}
	if w.Header().Get("Retry-After") != "60" {
		t.Errorf("Retry-After = %q, want %q", w.Header().Get("Retry-After"), "60")
	}
}

func TestRateLimiter_RoleBasedLimits(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := NewRateLimiter(2, 10, 5, 3, 1)

	r := gin.New()
	r.Use(rl.Middleware())
	r.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	// Test default role (limit 2) - first 2 should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("default request %d: expected 200, got %d", i+1, w.Code)
		}
	}

	// Third request should be blocked
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 after exhausting default rate, got %d", w.Code)
	}
}

func TestRateLimiter_IPBasedKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := NewRateLimiter(1, 10, 5, 3, 1)

	r := gin.New()
	r.Use(rl.Middleware())
	r.GET("/test", func(c *gin.Context) { c.String(200, "ok") })

	// First request from IP should succeed
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("first request: expected 200, got %d", w.Code)
	}

	// Second request from same IP should be blocked
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:54321"
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("second request from same IP: expected 429, got %d", w.Code)
	}
}

func TestRateLimiter_Refill(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := NewRateLimiter(2, 10, 5, 3, 1)

	// Manually create a bucket and set lastRefill to the past
	rl.mu.Lock()
	rl.buckets["test-key"] = &bucket{
		tokens:     0,
		lastRefill: time.Now().Add(-2 * time.Minute),
		rate:       2,
	}
	rl.mu.Unlock()

	// After refill, should have tokens again
	if !rl.allow("test-key", "") {
		t.Error("expected request to be allowed after refill")
	}
}

func TestRateLimiter_RemainingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := NewRateLimiter(5, 10, 8, 5, 3)

	r := gin.New()
	r.Use(rl.Middleware())
	r.GET("/test", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	remaining := w.Header().Get("X-RateLimit-Remaining")
	if remaining == "" {
		t.Error("expected X-RateLimit-Remaining header")
	}
}

func TestRateLimiter_RateLimitResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := NewRateLimiter(1, 10, 5, 3, 1)

	r := gin.New()
	r.Use(rl.Middleware())
	r.GET("/test", func(c *gin.Context) { c.String(200, "ok") })

	// Exhaust tokens
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Next request should get rate limited
	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}

	// Check response body
	body := w.Body.String()
	if body == "" {
		t.Error("expected non-empty response body")
	}
}

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(100, 200, 150, 100, 50)

	if rl.defaultRate != 100 {
		t.Errorf("defaultRate = %d, want 100", rl.defaultRate)
	}
	if rl.rates["owner"] != 200 {
		t.Errorf("owner rate = %d, want 200", rl.rates["owner"])
	}
	if rl.rates["admin"] != 150 {
		t.Errorf("admin rate = %d, want 150", rl.rates["admin"])
	}
	if rl.rates["dev"] != 100 {
		t.Errorf("dev rate = %d, want 100", rl.rates["dev"])
	}
	if rl.rates["viewer"] != 50 {
		t.Errorf("viewer rate = %d, want 50", rl.rates["viewer"])
	}
}
