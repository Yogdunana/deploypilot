package service

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// MemoryCache implements Cache using an in-memory map with TTL.
type MemoryCache struct {
	mu      sync.RWMutex
	items   map[string]*cacheItem
	prefix  string
	closeCh chan struct{}
}

type cacheItem struct {
	value    string
	expireAt time.Time
}

// NewMemoryCache creates a new in-memory cache.
func NewMemoryCache(prefix string) *MemoryCache {
	c := &MemoryCache{
		items:   make(map[string]*cacheItem),
		prefix:  prefix,
		closeCh: make(chan struct{}),
	}
	go c.cleanup()
	return c
}

func (c *MemoryCache) key(k string) string {
	return c.prefix + k
}

func (c *MemoryCache) Get(ctx context.Context, key string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, ok := c.items[c.key(key)]
	if !ok || time.Now().After(item.expireAt) {
		return "", ErrCacheMiss
	}
	return item.value, nil
}

func (c *MemoryCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[c.key(key)] = &cacheItem{value: value, expireAt: time.Now().Add(ttl)}
	return nil
}

func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, c.key(key))
	return nil
}

func (c *MemoryCache) GetJSON(ctx context.Context, key string, dest interface{}) error {
	val, err := c.Get(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), dest)
}

func (c *MemoryCache) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.Set(ctx, key, string(data), ttl)
}

func (c *MemoryCache) Close() error {
	close(c.closeCh)
	return nil
}

func (c *MemoryCache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for k, item := range c.items {
				if now.After(item.expireAt) {
					delete(c.items, k)
				}
			}
			c.mu.Unlock()
		case <-c.closeCh:
			return
		}
	}
}
