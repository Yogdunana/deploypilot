package service

import (
	"context"
	"testing"
	"time"
)

// Tests for EventRouter using the actual InMemoryTypedEventBus implementation

func TestEventRouter_New(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	defer bus.Close()

	// EventRouter requires a NotificationService which has complex dependencies
	// For basic initialization test, we verify the bus works
	if bus == nil {
		t.Error("NewInMemoryTypedEventBus returned nil")
	}
}

func TestInMemoryTypedEventBus_PublishSubscribe(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	defer bus.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Subscribe to event type
	ch := bus.SubscribeType(ctx, EventAlert)

	// Publish an event
	event := BusEvent{
		ID:      "event-1",
		Type:    EventAlert,
		Topic:   "alert:test",
		Payload: map[string]interface{}{"message": "test"},
	}

	bus.Publish(event)

	// Receive the event
	select {
	case received := <-ch:
		if received.ID != event.ID {
			t.Errorf("Received event ID = %q, want %q", received.ID, event.ID)
		}
		if received.Type != event.Type {
			t.Errorf("Received event type = %v, want %v", received.Type, event.Type)
		}
	case <-time.After(1 * time.Second):
		t.Error("Did not receive event within timeout")
	}
}

func TestInMemoryTypedEventBus_MultipleSubscribers(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	defer bus.Close()

	ctx := context.Background()

	// Subscribe multiple times to same type
	ch1 := bus.SubscribeType(ctx, EventDeploy)
	ch2 := bus.SubscribeType(ctx, EventDeploy)

	// Publish an event
	event := BusEvent{
		ID:   "event-1",
		Type: EventDeploy,
	}

	bus.Publish(event)

	// Both subscribers should receive
	select {
	case <-ch1:
		// OK
	case <-time.After(100 * time.Millisecond):
		t.Error("First subscriber did not receive event")
	}

	select {
	case <-ch2:
		// OK
	case <-time.After(100 * time.Millisecond):
		t.Error("Second subscriber did not receive event")
	}
}

func TestInMemoryTypedEventBus_DifferentTypes(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	defer bus.Close()

	ctx := context.Background()

	chAlert := bus.SubscribeType(ctx, EventAlert)
	chDeploy := bus.SubscribeType(ctx, EventDeploy)

	// Publish alert event
	bus.Publish(BusEvent{Type: EventAlert, ID: "alert-1"})

	// Only alert subscriber should receive
	select {
	case e := <-chAlert:
		if e.Type != EventAlert {
			t.Errorf("Alert subscriber received wrong type: %v", e.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Alert subscriber did not receive event")
	}

	// Deploy subscriber should not receive alert
	select {
	case <-chDeploy:
		t.Error("Deploy subscriber should not receive alert event")
	default:
		// OK - no event received
	}
}

func TestInMemoryTypedEventBus_Close(t *testing.T) {
	bus := NewInMemoryTypedEventBus()

	ctx := context.Background()
	_ = bus.SubscribeType(ctx, EventAlert) // ch is unused but needed for test

	// Close the bus
	err := bus.Close()
	if err != nil {
		t.Errorf("Close() error: %v", err)
	}

	// Channel should be closed after bus close
	// Give some time for cleanup
	time.Sleep(50 * time.Millisecond)

	// Publishing after close should not panic
	bus.Publish(BusEvent{Type: EventAlert})
}

func TestInMemoryTypedEventBus_NonBlocking(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	defer bus.Close()

	ctx := context.Background()
	// Small buffer channel
	ch := bus.SubscribeType(ctx, EventAlert)

	// Publish multiple events (should not block even if channel full)
	for i := 0; i < 20; i++ {
		bus.Publish(BusEvent{Type: EventAlert, ID: string(rune('a' + i))})
	}

	// Should complete without blocking
	// Read a few events to verify
	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			break
		}
		if count >= 10 {
			break
		}
	}
}

func TestBusEvent_JSONPayload(t *testing.T) {
	// Test that BusEvent with JSON payload can be serialized
	event := BusEvent{
		ID:      "event-1",
		Type:    EventDeploy,
		Topic:   "deploy:success",
		Payload: DeployEventPayload{
			AppID:   "app-1",
			AppName: "myapp",
			Status:  "success",
		},
	}

	bus := NewInMemoryTypedEventBus()
	defer bus.Close()

	ctx := context.Background()
	ch := bus.SubscribeType(ctx, EventDeploy)

	bus.Publish(event)

	select {
	case received := <-ch:
		if received.ID != event.ID {
			t.Errorf("Event ID mismatch")
		}
		// Payload should be preserved
		if received.Payload == nil {
			t.Error("Payload should not be nil")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Did not receive event")
	}
}

func TestEventTypeConstants(t *testing.T) {
	// Verify all event type constants are defined
	types := []EventType{
		EventDeploy,
		EventAlert,
		EventNotify,
		EventSystem,
		EventUser,
		EventServer,
		EventSecurity,
		EventAudit,
		EventBackup,
	}

	for _, et := range types {
		if et == EventType("") {
			t.Errorf("Event type constant should not be empty")
		}
	}
}

func TestEventRouteRule_Struct(t *testing.T) {
	// Test EventRouteRule structure
	rule := EventRouteRule{
		ID:          "rule-1",
		Name:        "Test Rule",
		Enabled:     true,
		EventType:   EventAlert,
		TopicPrefix: "alert:",
		Channels:    []string{"slack", "email"},
		Conditions:  `{"severity":"high"}`,
	}

	if rule.ID != "rule-1" {
		t.Error("ID mismatch")
	}
	if len(rule.Channels) != 2 {
		t.Errorf("Channels count = %d, want 2", len(rule.Channels))
	}
}

func TestEventPayloads(t *testing.T) {
	// Test all event payload structures
	t.Run("DeployEventPayload", func(t *testing.T) {
		payload := DeployEventPayload{
			AppID:    "app-1",
			AppName:  "myapp",
			Action:   "deploy",
			Status:   "success",
			Duration: 5000,
		}
		if payload.AppID != "app-1" {
			t.Error("AppID mismatch")
		}
	})

	t.Run("AlertEventPayload", func(t *testing.T) {
		payload := AlertEventPayload{
			AlertID:  "alert-1",
			Severity: "high",
			Status:   "firing",
		}
		if payload.AlertID != "alert-1" {
			t.Error("AlertID mismatch")
		}
	})

	t.Run("UserEventPayload", func(t *testing.T) {
		payload := UserEventPayload{
			UserID:    "user-1",
			Username:  "testuser",
			Action:    "login",
			IPAddress: "192.168.1.1",
			Success:   true,
		}
		if payload.UserID != "user-1" {
			t.Error("UserID mismatch")
		}
	})

	t.Run("ServerEventPayload", func(t *testing.T) {
		payload := ServerEventPayload{
			ServerID:   "server-1",
			ServerName: "myserver",
			Action:     "health_check",
			Metric:     "cpu",
			Value:      80.5,
		}
		if payload.ServerID != "server-1" {
			t.Error("ServerID mismatch")
		}
	})

	t.Run("SecurityEventPayload", func(t *testing.T) {
		payload := SecurityEventPayload{
			Action:    "brute_force",
			IPAddress: "10.0.0.1",
			Severity:  "high",
		}
		if payload.Action != "brute_force" {
			t.Error("Action mismatch")
		}
	})

	t.Run("BackupEventPayload", func(t *testing.T) {
		payload := BackupEventPayload{
			AppID:    "app-1",
			Action:   "success",
			Location: "s3",
			Size:     1024000,
		}
		if payload.Action != "success" {
			t.Error("Action mismatch")
		}
	})
}