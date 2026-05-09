package service

import (
	"context"
	"errors"
	"time"
)

// Cache defines the interface for a caching backend.
type Cache interface {
	// Get retrieves a string value by key. Returns ErrCacheMiss if not found.
	Get(ctx context.Context, key string) (string, error)

	// Set stores a string value with TTL.
	Set(ctx context.Context, key string, value string, ttl time.Duration) error

	// Delete removes a key.
	Delete(ctx context.Context, key string) error

	// GetJSON retrieves a JSON-serialized value into dest.
	GetJSON(ctx context.Context, key string, dest interface{}) error

	// SetJSON stores a value as JSON with TTL.
	SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Close closes the cache connection.
	Close() error
}

// ErrCacheMiss is returned when a cache key is not found.
var ErrCacheMiss = errors.New("cache miss")
