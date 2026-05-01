package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/redis/go-redis/v9"
)

// RedisTypedEventBus implements TypedEventBus using Redis Pub/Sub for multi-instance broadcasting.
// It uses a Source field to prevent processing self-published messages (ai-guide #28).
type RedisTypedEventBus struct {
	client   *redis.Client
	ctx      context.Context
	cancel   context.CancelFunc
	localBus *InMemoryTypedEventBus // for local subscribers
	instanceID string               // unique instance identifier for loop prevention
	channel  string                 // Redis Pub/Sub channel name
	mu       sync.RWMutex
}

// NewRedisTypedEventBus creates a new RedisTypedEventBus.
func NewRedisTypedEventBus(client *redis.Client, instanceID string) *RedisTypedEventBus {
	if instanceID == "" {
		instanceID = GenerateBusEventID()
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &RedisTypedEventBus{
		client:     client,
		ctx:        ctx,
		cancel:     cancel,
		localBus:   NewInMemoryTypedEventBus(),
		instanceID: instanceID,
		channel:    "deploypilot:typed-events",
	}
}

// Publish sends a typed event to Redis for other instances and locally for this instance's subscribers.
func (r *RedisTypedEventBus) Publish(event BusEvent) {
	// Set source to this instance for loop prevention
	event.Source = r.instanceID

	data, err := json.Marshal(event)
	if err != nil {
		slog.Error("failed to marshal redis typed eventbus event", "error", err)
		return
	}

	// Publish to Redis for other instances
	if err := r.client.Publish(r.ctx, r.channel, data).Err(); err != nil {
		slog.Error("failed to publish typed event to redis", "error", err)
	}

	// Also publish locally
	r.localBus.Publish(event)
}

// Subscribe returns a read-only channel receiving events for the given topic.
// It merges events from both the local bus and the Redis Pub/Sub channel.
func (r *RedisTypedEventBus) Subscribe(ctx context.Context, topic string) <-chan BusEvent {
	ch := make(chan BusEvent, 100)

	sub := r.client.Subscribe(ctx, r.channel)
	localCh := r.localBus.Subscribe(ctx, topic)

	go func() {
		defer func() { _ = sub.Close() }()
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
				return
			case <-r.ctx.Done():
				return
			case msg, ok := <-sub.Channel():
				if !ok {
					return
				}
				var event BusEvent
				if json.Unmarshal([]byte(msg.Payload), &event) == nil {
					// Skip self-published events (loop prevention)
					if event.Source == r.instanceID {
						continue
					}
					if event.Topic == topic {
						select {
						case ch <- event:
						default:
						}
					}
				}
			case event, ok := <-localCh:
				if !ok {
					return
				}
				select {
				case ch <- event:
				default:
				}
			}
		}
	}()

	return ch
}

// SubscribeType returns a read-only channel receiving events of the given type.
// It merges events from both the local bus and the Redis Pub/Sub channel.
func (r *RedisTypedEventBus) SubscribeType(ctx context.Context, eventType EventType) <-chan BusEvent {
	ch := make(chan BusEvent, 100)

	sub := r.client.Subscribe(ctx, r.channel)
	localCh := r.localBus.SubscribeType(ctx, eventType)

	go func() {
		defer func() { _ = sub.Close() }()
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
				return
			case <-r.ctx.Done():
				return
			case msg, ok := <-sub.Channel():
				if !ok {
					return
				}
				var event BusEvent
				if json.Unmarshal([]byte(msg.Payload), &event) == nil {
					// Skip self-published events (loop prevention)
					if event.Source == r.instanceID {
						continue
					}
					if event.Type == eventType {
						select {
						case ch <- event:
						default:
						}
					}
				}
			case event, ok := <-localCh:
				if !ok {
					return
				}
				select {
				case ch <- event:
				default:
				}
			}
		}
	}()

	return ch
}

// Close shuts down the Redis typed event bus.
func (r *RedisTypedEventBus) Close() error {
	r.cancel()
	return r.localBus.Close()
}
