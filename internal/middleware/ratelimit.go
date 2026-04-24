package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter implements token bucket rate limiting.
type RateLimiter struct {
	buckets     map[string]*bucket
	mu          sync.Mutex
	rates       map[string]int // role -> rate per minute
	defaultRate int
}

type bucket struct {
	tokens     int
	lastRefill time.Time
	rate       int // tokens per minute
}

// NewRateLimiter creates a new RateLimiter with per-role rate limits.
func NewRateLimiter(defaultRate, ownerRate, adminRate, devRate, viewerRate int) *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*bucket),
		rates: map[string]int{
			"owner":  ownerRate,
			"admin":  adminRate,
			"dev":    devRate,
			"viewer": viewerRate,
		},
		defaultRate: defaultRate,
	}
}

// Middleware returns a Gin middleware that enforces rate limits.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get identifier: user ID if authenticated, otherwise IP
		key := c.GetString("userID")
		if key == "" {
			key = "ip:" + c.ClientIP()
		}

		// Get role for role-based limits
		role := c.GetString("role")

		if !rl.allow(key, role) {
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("Retry-After", "60")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate limit exceeded",
				"code":    "E017",
				"message": "too many requests, please try again later",
			})
			c.Abort()
			return
		}

		remaining := rl.remaining(key, role)
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Next()
	}
}

// allow checks if a request is allowed under the rate limit.
func (rl *RateLimiter) allow(key, role string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b := rl.getOrCreateBucket(key, role)
	rl.refill(b)

	if b.tokens > 0 {
		b.tokens--
		return true
	}
	return false
}

// remaining returns the number of remaining tokens for the key.
func (rl *RateLimiter) remaining(key, role string) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b := rl.getOrCreateBucket(key, role)
	rl.refill(b)
	return b.tokens
}

func (rl *RateLimiter) getOrCreateBucket(key, role string) *bucket {
	if b, ok := rl.buckets[key]; ok {
		// Update rate if role changed
		if rate, ok := rl.rates[role]; ok {
			b.rate = rate
		}
		return b
	}
	rate := rl.defaultRate
	if r, ok := rl.rates[role]; ok {
		rate = r
	}
	b := &bucket{
		tokens:     rate,
		lastRefill: time.Now(),
		rate:       rate,
	}
	rl.buckets[key] = b
	return b
}

// refill adds tokens based on elapsed time since last refill.
func (rl *RateLimiter) refill(b *bucket) {
	now := time.Now()
	elapsed := now.Sub(b.lastRefill)
	if elapsed < time.Minute {
		return
	}
	// Calculate how many tokens to add based on elapsed minutes
	minutes := int(elapsed.Minutes())
	added := minutes * b.rate
	b.tokens += added
	// Cap at rate (one minute's worth of tokens)
	if b.tokens > b.rate {
		b.tokens = b.rate
	}
	b.lastRefill = b.lastRefill.Add(time.Duration(minutes) * time.Minute)
}
