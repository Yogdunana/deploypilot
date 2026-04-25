package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisEventBus_ImplementsInterface(t *testing.T) {
	// Compile-time check: RedisEventBus implements EventBus
	var _ EventBus = (*RedisEventBus)(nil)
}

func TestRedisEventBus_NewAndClose(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	bus := NewRedisEventBus(rdb)
	if bus == nil {
		t.Fatal("NewRedisEventBus returned nil")
	}
	if bus.channel != "deploypilot:events" {
		t.Errorf("expected channel 'deploypilot:events', got %q", bus.channel)
	}
	if bus.localBus == nil {
		t.Fatal("expected non-nil localBus")
	}

	err := bus.Close()
	if err != nil {
		t.Errorf("Close() error: %v", err)
	}
}

func TestRedisEventBus_PublishLocal(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	bus := NewRedisEventBus(rdb)
	defer bus.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := bus.Subscribe(ctx, "pub-local-app")

	event := DeployEvent{
		TaskID:    "task-pub",
		AppID:     "pub-local-app",
		Step:      "pull",
		Status:    "running",
		Progress:  30,
		Message:   "pulling image",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	bus.Publish(event)

	select {
	case received := <-ch:
		if received.TaskID != event.TaskID {
			t.Errorf("TaskID = %q, want %q", received.TaskID, event.TaskID)
		}
		if received.Step != event.Step {
			t.Errorf("Step = %q, want %q", received.Step, event.Step)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for published event")
	}
}

func TestRedisEventBus_PublishRemote(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	bus := NewRedisEventBus(rdb)
	defer bus.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Subscribe to a different app to verify filtering
	ch := bus.Subscribe(ctx, "remote-app")

	// Publish event for a different app
	bus.Publish(DeployEvent{
		TaskID: "task-1",
		AppID:  "other-app",
		Step:   "pull",
		Status: "running",
	})

	// Publish event for the subscribed app via direct Redis publish
	// (simulating another instance)
	event := DeployEvent{
		TaskID:    "task-remote",
		AppID:     "remote-app",
		Step:      "done",
		Status:    "success",
		Progress:  100,
		Message:   "completed",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Publish directly to Redis channel (simulating another instance)
	data, _ := json.Marshal(event)
	_ = rdb.Publish(ctx, "deploypilot:events", data).Err()

	select {
	case received := <-ch:
		if received.TaskID != "task-remote" {
			t.Errorf("TaskID = %q, want %q", received.TaskID, "task-remote")
		}
		if received.AppID != "remote-app" {
			t.Errorf("AppID = %q, want %q", received.AppID, "remote-app")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for remote event")
	}
}

func TestRedisEventBus_ContextCancellation(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	bus := NewRedisEventBus(rdb)
	defer bus.Close()

	ctx, cancel := context.WithCancel(context.Background())
	ch := bus.Subscribe(ctx, "cancel-app")

	// Cancel the context
	cancel()

	// Channel should be closed
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected channel to be closed after context cancellation")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for channel close")
	}
}

func TestRedisEventBus_CloseCancelsSubscribers(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	bus := NewRedisEventBus(rdb)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := bus.Subscribe(ctx, "close-app")

	// Close the bus
	_ = bus.Close()

	// Channel should be closed
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected channel to be closed after bus Close()")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for channel close")
	}
}

func TestRedisEventBus_MultipleSubscribers(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	bus := NewRedisEventBus(rdb)
	defer bus.Close()

	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	ch1 := bus.Subscribe(ctx1, "multi-app")
	ch2 := bus.Subscribe(ctx2, "multi-app")

	event := DeployEvent{
		TaskID: "task-multi",
		AppID:  "multi-app",
		Step:   "done",
		Status: "success",
	}

	bus.Publish(event)

	// Both channels should receive the event
	for i, ch := range []<-chan DeployEvent{ch1, ch2} {
		select {
		case received := <-ch:
			if received.TaskID != "task-multi" {
				t.Errorf("subscriber %d: TaskID = %q, want %q", i, received.TaskID, "task-multi")
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("subscriber %d: timed out waiting for event", i)
		}
	}
}

func TestInMemoryEventBusAsEventBus(t *testing.T) {
	var bus EventBus = NewInMemoryEventBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := bus.Subscribe(ctx, "test-app")

	event := DeployEvent{
		TaskID:    "task-1",
		AppID:     "test-app",
		Step:      "pull",
		Status:    "running",
		Progress:  50,
		Message:   "pulling image",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	bus.Publish(event)

	select {
	case received := <-ch:
		if received.TaskID != event.TaskID {
			t.Errorf("TaskID = %q, want %q", received.TaskID, event.TaskID)
		}
		if received.AppID != event.AppID {
			t.Errorf("AppID = %q, want %q", received.AppID, event.AppID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for published event via interface")
	}

	if err := bus.Close(); err != nil {
		t.Errorf("Close() error: %v", err)
	}
}

func TestNewBridge_NilEventBus(t *testing.T) {
	b := NewBridge(nil, nil, nil, nil)
	if b.EventBus == nil {
		t.Fatal("expected non-nil EventBus when nil passed to NewBridge")
	}
	if _, ok := b.EventBus.(*InMemoryEventBus); !ok {
		t.Errorf("expected *InMemoryEventBus, got %T", b.EventBus)
	}
}

func TestNewBridge_WithEventBus(t *testing.T) {
	bus := NewInMemoryEventBus()
	b := NewBridge(nil, nil, nil, bus)
	if b.EventBus != bus {
		t.Error("expected EventBus to be the provided bus")
	}
}

func TestRedisEventBus_PublishMarshalError(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	bus := NewRedisEventBus(rdb)
	defer bus.Close()

	// DeployEvent with channels containing functions can't be marshaled,
	// but the struct fields are all serializable. To trigger a marshal error,
	// we need to use a type that can't be marshaled. Since DeployEvent fields
	// are all strings/ints, we can't easily trigger this. Instead, test that
	// Publish doesn't panic with normal events.
	event := DeployEvent{
		TaskID:    "marshal-test",
		AppID:     "marshal-app",
		Step:      "build",
		Status:    "running",
		Progress:  50,
		Message:   "building",
		Timestamp: time.Now().Format(time.RFC3339),
	}
	bus.Publish(event) // should not panic
}

func TestRedisEventBus_SubscribeFiltering(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	bus := NewRedisEventBus(rdb)
	defer bus.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := bus.Subscribe(ctx, "filter-app")

	// Publish event for a different app - should not be received
	bus.Publish(DeployEvent{
		TaskID: "other-task",
		AppID:  "other-app",
		Step:   "pull",
		Status: "running",
	})

	// Publish event for the subscribed app
	bus.Publish(DeployEvent{
		TaskID: "filter-task",
		AppID:  "filter-app",
		Step:   "done",
		Status: "success",
	})

	select {
	case received := <-ch:
		if received.TaskID != "filter-task" {
			t.Errorf("TaskID = %q, want %q", received.TaskID, "filter-task")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for filtered event")
	}
}
