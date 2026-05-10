package service

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewInMemoryTypedEventBus(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	if bus == nil {
		t.Fatal("NewInMemoryTypedEventBus() returned nil")
	}
	if bus.topicSubs == nil {
		t.Fatal("expected non-nil topicSubs map")
	}
	if bus.typeSubs == nil {
		t.Fatal("expected non-nil typeSubs map")
	}
}

func TestInMemoryTypedEventBus_ImplementsInterface(t *testing.T) {
	var _ TypedEventBus = NewInMemoryTypedEventBus()
}

func TestInMemoryTypedEventBus_Subscribe(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := bus.Subscribe(ctx, "alert:all")
	if ch == nil {
		t.Fatal("Subscribe returned nil channel")
	}

	bus.mu.RLock()
	count := len(bus.topicSubs["alert:all"])
	bus.mu.RUnlock()
	if count != 1 {
		t.Errorf("expected 1 topic subscriber, got %d", count)
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
		t.Errorf("expected 1 type subscriber, got %d", count)
	}
}

func TestInMemoryTypedEventBus_PublishTopic(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := bus.Subscribe(ctx, "deploy:app-1")

	event := BusEvent{
		ID:        "evt-1",
		Type:      EventDeploy,
		Topic:     "deploy:app-1",
		Payload:   map[string]interface{}{"status": "success"},
		Timestamp: time.Now(),
	}

	bus.Publish(event)

	select {
	case received := <-ch:
		if received.ID != event.ID {
			t.Errorf("ID = %q, want %q", received.ID, event.ID)
		}
		if received.Topic != event.Topic {
			t.Errorf("Topic = %q, want %q", received.Topic, event.Topic)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for published event")
	}
}

func TestInMemoryTypedEventBus_PublishType(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := bus.SubscribeType(ctx, EventAlert)

	event := BusEvent{
		ID:        "evt-alert-1",
		Type:      EventAlert,
		Topic:     "alert:high",
		Payload:   map[string]interface{}{"severity": "high"},
		Timestamp: time.Now(),
	}

	bus.Publish(event)

	select {
	case received := <-ch:
		if received.ID != event.ID {
			t.Errorf("ID = %q, want %q", received.ID, event.ID)
		}
		if received.Type != event.Type {
			t.Errorf("Type = %q, want %q", received.Type, event.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for published event")
	}
}

func TestInMemoryTypedEventBus_PublishBothTopicAndType(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	topicCh := bus.Subscribe(ctx, "deploy:app-1")
	typeCh := bus.SubscribeType(ctx, EventDeploy)

	event := BusEvent{
		ID:        "evt-both",
		Type:      EventDeploy,
		Topic:     "deploy:app-1",
		Payload:   map[string]interface{}{"status": "running"},
		Timestamp: time.Now(),
	}

	bus.Publish(event)

	timeout := time.After(2 * time.Second)

	for i, ch := range []<-chan BusEvent{topicCh, typeCh} {
		select {
		case received := <-ch:
			if received.ID != event.ID {
				t.Errorf("channel %d: ID = %q, want %q", i, received.ID, event.ID)
			}
		case <-timeout:
			t.Fatalf("channel %d: timed out waiting for event", i)
		}
	}
}

func TestInMemoryTypedEventBus_PublishNoSubscribers(t *testing.T) {
	bus := NewInMemoryTypedEventBus()

	event := BusEvent{
		ID:        "evt-no-sub",
		Type:      EventSystem,
		Topic:     "system:start",
		Payload:   nil,
		Timestamp: time.Now(),
	}

	bus.Publish(event)
}

func TestInMemoryTypedEventBus_MultipleSubscribers(t *testing.T) {
	bus := NewInMemoryTypedEventBus()

	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	ctx3, cancel3 := context.WithCancel(context.Background())
	defer cancel3()

	ch1 := bus.Subscribe(ctx1, "multi-topic")
	ch2 := bus.Subscribe(ctx2, "multi-topic")
	ch3 := bus.SubscribeType(ctx3, EventDeploy)

	event := BusEvent{
		ID:        "evt-multi",
		Type:      EventDeploy,
		Topic:     "multi-topic",
		Payload:   nil,
		Timestamp: time.Now(),
	}

	bus.Publish(event)

	timeout := time.After(2 * time.Second)

	select {
	case received := <-ch1:
		if received.ID != event.ID {
			t.Errorf("ch1: ID = %q, want %q", received.ID, event.ID)
		}
	case <-timeout:
		t.Fatal("ch1: timed out waiting for event")
	}

	select {
	case received := <-ch2:
		if received.ID != event.ID {
			t.Errorf("ch2: ID = %q, want %q", received.ID, event.ID)
		}
	case <-timeout:
		t.Fatal("ch2: timed out waiting for event")
	}

	select {
	case received := <-ch3:
		if received.ID != event.ID {
			t.Errorf("ch3: ID = %q, want %q", received.ID, event.ID)
		}
	case <-timeout:
		t.Fatal("ch3: timed out waiting for event")
	}
}

func TestInMemoryTypedEventBus_UnsubscribeCleansUp(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	ctx, cancel := context.WithCancel(context.Background())

	ch := bus.Subscribe(ctx, "cleanup-topic")
	cancel()
	time.Sleep(50 * time.Millisecond)

	bus.mu.RLock()
	_, exists := bus.topicSubs["cleanup-topic"]
	bus.mu.RUnlock()
	if exists {
		t.Error("expected cleanup-topic to be removed after cancel")
	}

	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed after context cancel")
	}
}

func TestInMemoryTypedEventBus_UnsubscribeTypeCleansUp(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	ctx, cancel := context.WithCancel(context.Background())

	ch := bus.SubscribeType(ctx, EventAlert)
	cancel()
	time.Sleep(50 * time.Millisecond)

	bus.mu.RLock()
	_, exists := bus.typeSubs[EventAlert]
	bus.mu.RUnlock()
	if exists {
		t.Error("expected EventAlert type to be removed after cancel")
	}

	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed after context cancel")
	}
}

func TestInMemoryTypedEventBus_Close(t *testing.T) {
	bus := NewInMemoryTypedEventBus()

	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	ch1 := bus.Subscribe(ctx1, "close-topic")
	ch2 := bus.SubscribeType(ctx2, EventAlert)

	err := bus.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	bus.mu.RLock()
	topicCount := len(bus.topicSubs)
	typeCount := len(bus.typeSubs)
	bus.mu.RUnlock()

	if topicCount != 0 {
		t.Errorf("expected 0 topic subs after Close, got %d", topicCount)
	}
	if typeCount != 0 {
		t.Errorf("expected 0 type subs after Close, got %d", typeCount)
	}

	select {
	case _, ok := <-ch1:
		if ok {
			t.Error("expected ch1 to be closed after Close")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for ch1 to close")
	}

	select {
	case _, ok := <-ch2:
		if ok {
			t.Error("expected ch2 to be closed after Close")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for ch2 to close")
	}
}

func TestInMemoryTypedEventBus_ConcurrentPublish(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := bus.Subscribe(ctx, "concurrent-topic")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			bus.Publish(BusEvent{
				ID:        "evt-concurrent",
				Type:      EventSystem,
				Topic:     "concurrent-topic",
				Payload:   map[string]interface{}{"index": n},
				Timestamp: time.Now(),
			})
		}(i)
	}
	wg.Wait()

	received := 0
	timeout := time.After(2 * time.Second)
	for {
		select {
		case <-ch:
			received++
		case <-timeout:
			goto done
		}
	}
done:
	if received == 0 {
		t.Error("expected to receive at least some events from concurrent publish")
	}
}

func TestInMemoryTypedEventBus_FilteredByTopic(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := bus.Subscribe(ctx, "topic-a")

	bus.Publish(BusEvent{
		ID:    "evt-topic-b",
		Type:  EventDeploy,
		Topic: "topic-b",
	})

	bus.Publish(BusEvent{
		ID:    "evt-topic-a",
		Type:  EventDeploy,
		Topic: "topic-a",
	})

	select {
	case received := <-ch:
		if received.ID != "evt-topic-a" {
			t.Errorf("ID = %q, want %q", received.ID, "evt-topic-a")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for filtered event")
	}
}

func TestInMemoryTypedEventBus_FilteredByType(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := bus.SubscribeType(ctx, EventAlert)

	bus.Publish(BusEvent{
		ID:    "evt-deploy",
		Type:  EventDeploy,
		Topic: "deploy:app",
	})

	bus.Publish(BusEvent{
		ID:    "evt-alert",
		Type:  EventAlert,
		Topic: "alert:warning",
	})

	select {
	case received := <-ch:
		if received.ID != "evt-alert" {
			t.Errorf("ID = %q, want %q", received.ID, "evt-alert")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for filtered event")
	}
}

func TestGenerateBusEventID(t *testing.T) {
	id1 := GenerateBusEventID()
	id2 := GenerateBusEventID()

	if id1 == "" {
		t.Error("expected non-empty ID")
	}
	if id1 == id2 {
		t.Error("expected unique IDs for consecutive calls")
	}
	if len(id1) < 10 {
		t.Errorf("expected ID length > 10, got %d", len(id1))
	}
}

func TestInMemoryTypedEventBus_PublishToEmptyTopic(t *testing.T) {
	bus := NewInMemoryTypedEventBus()

	event := BusEvent{
		ID:        "evt-empty",
		Type:      EventSystem,
		Topic:     "nonexistent-topic",
		Payload:   nil,
		Timestamp: time.Now(),
	}

	bus.Publish(event)
}

func TestInMemoryTypedEventBus_PublishToEmptyType(t *testing.T) {
	bus := NewInMemoryTypedEventBus()

	event := BusEvent{
		ID:        "evt-empty-type",
		Type:      EventType("nonexistent"),
		Topic:     "test",
		Payload:   nil,
		Timestamp: time.Now(),
	}

	bus.Publish(event)
}
