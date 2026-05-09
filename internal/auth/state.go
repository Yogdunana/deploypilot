package auth

import (
	"context"
	"sync"
	"time"
)

// StateStore manages OAuth state parameters for CSRF protection.
type StateStore interface {
	Generate(state string, ttl time.Duration) error
	Validate(state string) bool
}

// MemoryStateStore implements StateStore using in-memory storage.
type MemoryStateStore struct {
	mu    sync.RWMutex
	items map[string]time.Time
}

// NewMemoryStateStore creates a new in-memory state store.
func NewMemoryStateStore() *MemoryStateStore {
	return &MemoryStateStore{
		items: make(map[string]time.Time),
	}
}

func (s *MemoryStateStore) Generate(state string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[state] = time.Now().Add(ttl)
	return nil
}

func (s *MemoryStateStore) Validate(state string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	expiry, exists := s.items[state]
	if !exists {
		return false
	}
	if time.Now().After(expiry) {
		delete(s.items, state)
		return false
	}
	delete(s.items, state)
	return true
}

// StartCleanup starts a background goroutine that removes expired state entries.
func (s *MemoryStateStore) StartCleanup(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.mu.Lock()
				now := time.Now()
				for state, expiry := range s.items {
					if now.After(expiry) {
						delete(s.items, state)
					}
				}
				s.mu.Unlock()
			}
		}
	}()
}
