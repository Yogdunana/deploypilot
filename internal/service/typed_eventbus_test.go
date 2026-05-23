package service

import (
	"context"
	"testing"
	"time"
)

func TestNewInMemoryTypedEventBus(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	if bus == nil {
		t.Fatal("NewInMemoryTypedEventBus returned nil")
	}
	if bus.topicSubs == nil {
		t.Error("topicSubs should be initialized")
	}
	if bus.typeSubs == nil {
		t.Error("typeSubs should be initialized")
	}
}

func TestInMemoryTypedEventBus_SubscribeType(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := bus.SubscribeType(ctx, EventAlert)
	if ch == nil {
		t.Fatal("SubscribeType returned nil channel")
	}

	bus.mu.RLock()
	count := len(bus.typeSubs[EventAlert])
	bus.mu.RUnlock()

	if count != 1 {
		t.Errorf("expected 1 subscriber, got %d", count)
	}
}

func TestInMemoryTypedEventBus_SubscribeType_MultipleSubscriptions(t *testing.T) {
	bus := NewInMemoryTypedEventBus()

	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	ch1 := bus.SubscribeType(ctx1, EventAlert)
	ch2 := bus.SubscribeType(ctx2, EventAlert)

	if ch1 == ch2 {
		t.Error("each subscription should get its own channel")
	}

	bus.mu.RLock()
	count := len(bus.typeSubs[EventAlert])
	bus.mu.RUnlock()

	if count != 2 {
		t.Errorf("expected 2 subscribers, got %d", count)
	}
}

func TestInMemoryTypedEventBus_SubscribeType_CancelUnsubscribes(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	ctx, cancel := context.WithCancel(context.Background())

	ch := bus.SubscribeType(ctx, EventAlert)

	cancel()
	time.Sleep(50 * time.Millisecond)

	bus.mu.RLock()
	_, exists := bus.typeSubs[EventAlert]
	bus.mu.RUnlock()

	if exists {
		t.Error("subscription should be removed after cancel")
	}

	_, ok := <-ch
	if ok {
		t.Error("channel should be closed after cancel")
	}
}

func TestInMemoryTypedEventBus_Publish_TopicSubscribers(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := bus.Subscribe(ctx, "app:test")

	event := BusEvent{
		ID:        "evt-1",
		Type:      EventDeploy,
		Topic:     "app:test",
		Payload:   map[string]interface{}{"status": "success"},
		Timestamp: time.Now(),
	}

	bus.Publish(event)

	select {
	case received := <-ch:
		if received.ID != event.ID {
			t.Errorf("ID = %q, want %q", received.ID, event.ID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for published event")
	}
}

func TestInMemoryTypedEventBus_Publish_TypeSubscribers(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := bus.SubscribeType(ctx, EventAlert)

	event := BusEvent{
		ID:        "evt-2",
		Type:      EventAlert,
		Topic:     "alert:critical",
		Payload:   map[string]interface{}{"severity": "critical"},
		Timestamp: time.Now(),
	}

	bus.Publish(event)

	select {
	case received := <-ch:
		if received.ID != event.ID {
			t.Errorf("ID = %q, want %q", received.ID, event.ID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for published event")
	}
}

func TestInMemoryTypedEventBus_Publish_DropOnFull(t *testing.T) {
	bus := NewInMemoryTypedEventBus()

	ch := make(chan BusEvent, 1)
	bus.mu.Lock()
	bus.typeSubs[EventAlert] = append(bus.typeSubs[EventAlert], ch)
	bus.mu.Unlock()

	event := BusEvent{
		ID:    "evt-drop",
		Type:  EventAlert,
		Topic: "alert:test",
	}

	bus.Publish(event)
	bus.Publish(event)

	select {
	case <-ch:
	case <-time.After(100 * time.Millisecond):
	}
}

func TestInMemoryTypedEventBus_Close(t *testing.T) {
	bus := NewInMemoryTypedEventBus()

	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	ch1 := bus.SubscribeType(ctx1, EventAlert)
	ch2 := bus.SubscribeType(ctx2, EventDeploy)

	if err := bus.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	bus.mu.RLock()
	topicCount := len(bus.topicSubs)
	typeCount := len(bus.typeSubs)
	bus.mu.RUnlock()

	if topicCount != 0 {
		t.Errorf("expected 0 topic subs after close, got %d", topicCount)
	}
	if typeCount != 0 {
		t.Errorf("expected 0 type subs after close, got %d", typeCount)
	}

	select {
	case _, ok := <-ch1:
		if ok {
			t.Error("channel 1 should be closed")
		}
	default:
		t.Error("channel 1 should be closed")
	}

	select {
	case _, ok := <-ch2:
		if ok {
			t.Error("channel 2 should be closed")
		}
	default:
		t.Error("channel 2 should be closed")
	}
}

func TestInMemoryTypedEventBus_UnsubscribeTopic(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	ctx, cancel := context.WithCancel(context.Background())

	_ = bus.Subscribe(ctx, "app:cleanup")

	cancel()
	time.Sleep(50 * time.Millisecond)

	bus.mu.RLock()
	_, exists := bus.topicSubs["app:cleanup"]
	bus.mu.RUnlock()

	if exists {
		t.Error("topic should be cleaned up after last subscriber cancels")
	}
}

func TestInMemoryTypedEventBus_UnsubscribeType(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	ctx, cancel := context.WithCancel(context.Background())

	ch := bus.SubscribeType(ctx, EventSecurity)

	cancel()
	time.Sleep(50 * time.Millisecond)

	bus.mu.RLock()
	_, exists := bus.typeSubs[EventSecurity]
	bus.mu.RUnlock()

	if exists {
		t.Error("event type should be cleaned up after last subscriber cancels")
	}

	_, ok := <-ch
	if ok {
		t.Error("channel should be closed")
	}
}

func TestInMemoryTypedEventBus_Publish_NoSubscribers(t *testing.T) {
	bus := NewInMemoryTypedEventBus()

	event := BusEvent{
		ID:        "evt-no-sub",
		Type:      EventSystem,
		Topic:     "system:startup",
		Payload:   nil,
		Timestamp: time.Now(),
	}

	bus.Publish(event)
}

func TestInMemoryTypedEventBus_Publish_MultipleSubscribers(t *testing.T) {
	bus := NewInMemoryTypedEventBus()

	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	ctx3, cancel3 := context.WithCancel(context.Background())
	defer cancel3()

	ch1 := bus.SubscribeType(ctx1, EventAlert)
	ch2 := bus.SubscribeType(ctx2, EventAlert)
	ch3 := bus.Subscribe(ctx3, "alert:cpu")

	event := BusEvent{
		ID:        "evt-multi",
		Type:      EventAlert,
		Topic:     "alert:cpu",
		Payload:   map[string]interface{}{"metric": "cpu"},
		Timestamp: time.Now(),
	}

	bus.Publish(event)

	received := 0
	timeout := time.After(2 * time.Second)

	for i := 0; i < 3; i++ {
		select {
		case <-ch1:
			received++
		case <-ch2:
			received++
		case <-ch3:
			received++
		case <-timeout:
			t.Fatalf("timeout waiting for events, received %d of 3", received)
		}
	}
}
