package service

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

// RedisEventBus implements EventBus using Redis Pub/Sub for multi-instance broadcasting.
type RedisEventBus struct {
	client   *redis.Client
	ctx      context.Context
	cancel   context.CancelFunc
	localBus *InMemoryEventBus // for local subscribers
	channel  string            // Redis Pub/Sub channel name
}

// NewRedisEventBus creates a new RedisEventBus.
func NewRedisEventBus(client *redis.Client) *RedisEventBus {
	ctx, cancel := context.WithCancel(context.Background())
	return &RedisEventBus{
		client:   client,
		ctx:      ctx,
		cancel:   cancel,
		localBus: NewInMemoryEventBus(),
		channel:  "deploypilot:events",
	}
}

// Publish sends an event to Redis for other instances and locally for this instance's subscribers.
func (r *RedisEventBus) Publish(event DeployEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		slog.Error("failed to marshal redis eventbus event", "error", err)
		return
	}
	// Publish to Redis for other instances (fire-and-forget)
	if err := r.client.Publish(r.ctx, r.channel, data).Err(); err != nil {
		slog.Warn("failed to publish redis event", "channel", r.channel, "error", err)
	}
	// Also publish locally for this instance's subscribers
	r.localBus.Publish(event)
}

// Subscribe returns a read-only channel that receives events for the given appID.
// It merges events from both the local bus and the Redis Pub/Sub channel.
func (r *RedisEventBus) Subscribe(ctx context.Context, appID string) <-chan DeployEvent {
	ch := make(chan DeployEvent, 100)

	// Subscribe to Redis channel for events from other instances
	sub := r.client.Subscribe(ctx, r.channel)

	// Also subscribe locally
	localCh := r.localBus.Subscribe(ctx, appID)

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
				var event DeployEvent
				if json.Unmarshal([]byte(msg.Payload), &event) == nil {
					if event.AppID == appID {
						select {
						case ch <- event:
						default:
							// channel full, drop event
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
					// channel full, drop event
				}
			}
		}
	}()

	return ch
}

// Close shuts down the Redis event bus.
func (r *RedisEventBus) Close() error {
	r.cancel()
	return r.localBus.Close()
}
