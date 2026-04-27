package auth

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// TokenBlacklist provides token revocation capabilities.
type TokenBlacklist interface {
	Revoke(jti string, ttl time.Duration) error
	IsRevoked(jti string) (bool, error)
}

// RedisTokenBlacklist implements TokenBlacklist using Redis.
type RedisTokenBlacklist struct {
	client *redis.Client
}

// NewRedisTokenBlacklist creates a new Redis-backed token blacklist.
func NewRedisTokenBlacklist(client *redis.Client) *RedisTokenBlacklist {
	return &RedisTokenBlacklist{client: client}
}

func (b *RedisTokenBlacklist) Revoke(jti string, ttl time.Duration) error {
	key := fmt.Sprintf("jti:%s", jti)
	if err := b.client.Set(context.Background(), key, "1", ttl).Err(); err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}
	return nil
}

func (b *RedisTokenBlacklist) IsRevoked(jti string) (bool, error) {
	key := fmt.Sprintf("jti:%s", jti)
	val, err := b.client.Exists(context.Background(), key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check token revocation: %w", err)
	}
	return val > 0, nil
}

// MemoryTokenBlacklist implements TokenBlacklist using in-memory storage.
// Suitable for single-instance deployments or as a fallback when Redis is unavailable.
type MemoryTokenBlacklist struct {
	mu    sync.RWMutex
	items map[string]time.Time
}

// NewMemoryTokenBlacklist creates a new in-memory token blacklist.
func NewMemoryTokenBlacklist() *MemoryTokenBlacklist {
	return &MemoryTokenBlacklist{
		items: make(map[string]time.Time),
	}
}

func (b *MemoryTokenBlacklist) Revoke(jti string, ttl time.Duration) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.items[jti] = time.Now().Add(ttl)
	return nil
}

func (b *MemoryTokenBlacklist) IsRevoked(jti string) (bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	expiry, exists := b.items[jti]
	if !exists {
		return false, nil
	}
	if time.Now().After(expiry) {
		return false, nil
	}
	return true, nil
}

// StartCleanup starts a background goroutine that periodically removes expired entries.
func (b *MemoryTokenBlacklist) StartCleanup(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				b.cleanup()
			}
		}
	}()
}

func (b *MemoryTokenBlacklist) cleanup() {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	count := 0
	for jti, expiry := range b.items {
		if now.After(expiry) {
			delete(b.items, jti)
			count++
		}
	}
	if count > 0 {
		slog.Debug("cleaned up expired token blacklist entries", "count", count)
	}
}
