package service

import (
	"context"
	"sync"
)

// EventBus defines the interface for deploy event broadcasting.
// Implementations can be in-memory (single instance) or Redis-backed (multi-instance).
type EventBus interface {
	// Publish sends an event to all subscribers of the given appID.
	Publish(event DeployEvent)

	// Subscribe returns a read-only channel that receives events for the given appID.
	// The channel is closed when the provided context is cancelled.
	Subscribe(ctx context.Context, appID string) <-chan DeployEvent

	// Close shuts down the event bus and releases all resources.
	Close() error
}

// InMemoryEventBus is an in-memory implementation of EventBus for single-instance deployments.
type InMemoryEventBus struct {
	subscribers map[string][]chan DeployEvent // appID -> channels
	mu          sync.RWMutex
}

// NewInMemoryEventBus creates a new InMemoryEventBus.
func NewInMemoryEventBus() *InMemoryEventBus {
	return &InMemoryEventBus{
		subscribers: make(map[string][]chan DeployEvent),
	}
}

// Subscribe returns a buffered channel for the given appID.
// The channel is closed when ctx is cancelled.
func (b *InMemoryEventBus) Subscribe(ctx context.Context, appID string) <-chan DeployEvent {
	ch := make(chan DeployEvent, 100)
	b.mu.Lock()
	b.subscribers[appID] = append(b.subscribers[appID], ch)
	b.mu.Unlock()

	go func() {
		<-ctx.Done()
		b.unsubscribe(appID, ch)
	}()

	return ch
}

// unsubscribe removes a channel for the given appID.
func (b *InMemoryEventBus) unsubscribe(appID string, ch chan DeployEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	channels := b.subscribers[appID]
	for i, c := range channels {
		if c == ch {
			b.subscribers[appID] = append(channels[:i], channels[i+1:]...)
			close(ch)
			break
		}
	}
	if len(b.subscribers[appID]) == 0 {
		delete(b.subscribers, appID)
	}
}

// Publish sends an event to all subscribers of the given appID.
func (b *InMemoryEventBus) Publish(event DeployEvent) {
	b.mu.RLock()
	channels := b.subscribers[event.AppID]
	// Copy to avoid holding lock during sends
	copied := make([]chan DeployEvent, len(channels))
	copy(copied, channels)
	b.mu.RUnlock()
	for _, ch := range copied {
		select {
		case ch <- event:
		default:
			// channel full, drop event to avoid blocking publisher
		}
	}
}

// Close is a no-op for the in-memory implementation.
func (b *InMemoryEventBus) Close() error {
	return nil
}
