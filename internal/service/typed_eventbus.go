package service

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// InMemoryTypedEventBus is an in-memory implementation of TypedEventBus for single-instance deployments.
type InMemoryTypedEventBus struct {
	topicSubs map[string][]chan BusEvent  // topic -> channels
	typeSubs  map[EventType][]chan BusEvent // event type -> channels
	mu        sync.RWMutex
}

// NewInMemoryTypedEventBus creates a new InMemoryTypedEventBus.
func NewInMemoryTypedEventBus() *InMemoryTypedEventBus {
	return &InMemoryTypedEventBus{
		topicSubs: make(map[string][]chan BusEvent),
		typeSubs:  make(map[EventType][]chan BusEvent),
	}
}

// Publish sends a typed event to all topic and type subscribers.
func (b *InMemoryTypedEventBus) Publish(event BusEvent) {
	b.mu.RLock()

	// Collect topic subscribers
	var topicChannels []chan BusEvent
	if chs, ok := b.topicSubs[event.Topic]; ok {
		topicChannels = make([]chan BusEvent, len(chs))
		copy(topicChannels, chs)
	}

	// Collect type subscribers
	var typeChannels []chan BusEvent
	if chs, ok := b.typeSubs[event.Type]; ok {
		typeChannels = make([]chan BusEvent, len(chs))
		copy(typeChannels, chs)
	}

	b.mu.RUnlock()

	// Send to topic subscribers (non-blocking, drop on full)
	for _, ch := range topicChannels {
		select {
		case ch <- event:
		default:
		}
	}

	// Send to type subscribers (non-blocking, drop on full)
	for _, ch := range typeChannels {
		select {
		case ch <- event:
		default:
		}
	}
}

// Subscribe returns a buffered channel for the given topic.
// The channel is closed when ctx is cancelled.
func (b *InMemoryTypedEventBus) Subscribe(ctx context.Context, topic string) <-chan BusEvent {
	ch := make(chan BusEvent, 100)
	b.mu.Lock()
	b.topicSubs[topic] = append(b.topicSubs[topic], ch)
	b.mu.Unlock()

	go func() {
		<-ctx.Done()
		b.unsubscribeTopic(topic, ch)
	}()

	return ch
}

// SubscribeType returns a buffered channel for the given event type.
// The channel is closed when ctx is cancelled.
func (b *InMemoryTypedEventBus) SubscribeType(ctx context.Context, eventType EventType) <-chan BusEvent {
	ch := make(chan BusEvent, 100)
	b.mu.Lock()
	b.typeSubs[eventType] = append(b.typeSubs[eventType], ch)
	b.mu.Unlock()

	go func() {
		<-ctx.Done()
		b.unsubscribeType(eventType, ch)
	}()

	return ch
}

// Close closes all subscriber channels and clears the subscriber maps.
func (b *InMemoryTypedEventBus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	for topic, channels := range b.topicSubs {
		for _, ch := range channels {
			close(ch)
		}
		delete(b.topicSubs, topic)
	}
	for eventType, channels := range b.typeSubs {
		for _, ch := range channels {
			close(ch)
		}
		delete(b.typeSubs, eventType)
	}
	return nil
}

// unsubscribeTopic removes a channel for the given topic.
func (b *InMemoryTypedEventBus) unsubscribeTopic(topic string, ch chan BusEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	channels := b.topicSubs[topic]
	for i, c := range channels {
		if c == ch {
			b.topicSubs[topic] = append(channels[:i], channels[i+1:]...)
			close(ch)
			break
		}
	}
	if len(b.topicSubs[topic]) == 0 {
		delete(b.topicSubs, topic)
	}
}

// unsubscribeType removes a channel for the given event type.
func (b *InMemoryTypedEventBus) unsubscribeType(eventType EventType, ch chan BusEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	channels := b.typeSubs[eventType]
	for i, c := range channels {
		if c == ch {
			b.typeSubs[eventType] = append(channels[:i], channels[i+1:]...)
			close(ch)
			break
		}
	}
	if len(b.typeSubs[eventType]) == 0 {
		delete(b.typeSubs, eventType)
	}
}

// GenerateBusEventID creates a unique event ID.
func GenerateBusEventID() string {
	return fmt.Sprintf("evt-%d", time.Now().UnixNano())
}
