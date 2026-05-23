package service

import (
	"context"
	"testing"
	"time"
)

type mockTypedEventBus struct {
	subscriptions []typedSub
}

type typedSub struct {
	eventType EventType
	ch        chan BusEvent
}

func (m *mockTypedEventBus) Publish(event BusEvent) {
	for _, sub := range m.subscriptions {
		if sub.eventType == event.Type || sub.eventType == "" {
			select {
			case sub.ch <- event:
			default:
			}
		}
	}
}

func (m *mockTypedEventBus) Subscribe(ctx context.Context, topic string) <-chan BusEvent {
	ch := make(chan BusEvent, 100)
	return ch
}

func (m *mockTypedEventBus) SubscribeType(ctx context.Context, eventType EventType) <-chan BusEvent {
	ch := make(chan BusEvent, 100)
	m.subscriptions = append(m.subscriptions, typedSub{eventType: eventType, ch: ch})
	go func() {
		<-ctx.Done()
		for i, sub := range m.subscriptions {
			if sub.ch == ch {
				m.subscriptions = append(m.subscriptions[:i], m.subscriptions[i+1:]...)
				close(ch)
				break
			}
		}
	}()
	return ch
}

func (m *mockTypedEventBus) Close() error {
	for _, sub := range m.subscriptions {
		close(sub.ch)
	}
	m.subscriptions = nil
	return nil
}

func TestNewEventRouter(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)
	if router == nil {
		t.Fatal("NewEventRouter returned nil")
	}
	if len(router.rules) != 0 {
		t.Errorf("expected empty rules, got %d", len(router.rules))
	}
}

func TestEventRouter_SetAndGetRules(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	rules := []EventRouteRule{
		{ID: "rule-1", Name: "Test Rule", Enabled: true, EventType: EventAlert},
		{ID: "rule-2", Name: "Test Rule 2", Enabled: false, EventType: EventDeploy},
	}

	router.SetRules(rules)

	got := router.GetRules()
	if len(got) != 2 {
		t.Errorf("expected 2 rules, got %d", len(got))
	}

	got[0].Name = "Modified"
	if router.GetRules()[0].Name != "Test Rule" {
		t.Error("GetRules should return a copy")
	}
}

func TestEventRouter_AddRule(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	rule := EventRouteRule{ID: "add-1", Name: "Added Rule", EventType: EventSecurity}
	router.AddRule(rule)

	if len(router.GetRules()) != 1 {
		t.Error("expected 1 rule after AddRule")
	}

	router.AddRule(EventRouteRule{ID: "add-2", Name: "Added Rule 2"})
	if len(router.GetRules()) != 2 {
		t.Error("expected 2 rules after second AddRule")
	}
}

func TestEventRouter_RemoveRule(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	router.SetRules([]EventRouteRule{
		{ID: "remove-1", Name: "To Remove"},
		{ID: "remove-2", Name: "To Keep"},
	})

	router.RemoveRule("remove-1")

	rules := router.GetRules()
	if len(rules) != 1 {
		t.Errorf("expected 1 rule after remove, got %d", len(rules))
	}
	if rules[0].ID != "remove-2" {
		t.Errorf("expected remaining rule to be remove-2, got %s", rules[0].ID)
	}
}

func TestEventRouter_RemoveRule_NotFound(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	router.SetRules([]EventRouteRule{
		{ID: "keep-1", Name: "Keep"},
	})

	router.RemoveRule("nonexistent")

	if len(router.GetRules()) != 1 {
		t.Error("expected rule count unchanged when removing nonexistent")
	}
}

func TestExtractSeverity(t *testing.T) {
	tests := []struct {
		name     string
		payload  interface{}
		expected string
	}{
		{
			name:     "nil payload",
			payload:  nil,
			expected: "",
		},
		{
			name:     "non-map payload",
			payload:  "string",
			expected: "",
		},
		{
			name:     "map without severity",
			payload:  map[string]interface{}{"message": "test"},
			expected: "",
		},
		{
			name:     "map with severity",
			payload:  map[string]interface{}{"severity": "critical", "message": "test"},
			expected: "critical",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSeverity(tt.payload)
			if got != tt.expected {
				t.Errorf("extractSeverity() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestMatchConditions(t *testing.T) {
	tests := []struct {
		name       string
		conditions string
		payload    interface{}
		expected   bool
	}{
		{
			name:       "invalid JSON conditions",
			conditions: "not json",
			payload:    map[string]interface{}{},
			expected:   false,
		},
		{
			name:       "payload not convertible",
			conditions: `{"key": "value"}`,
			payload:    123,
			expected:   false,
		},
		{
			name:       "key missing in payload",
			conditions: `{"key": "value"}`,
			payload:    map[string]interface{}{"other": "data"},
			expected:   false,
		},
		{
			name:       "key matches",
			conditions: `{"severity": "critical"}`,
			payload:    map[string]interface{}{"severity": "critical"},
			expected:   true,
		},
		{
			name:       "key mismatch",
			conditions: `{"severity": "critical"}`,
			payload:    map[string]interface{}{"severity": "warning"},
			expected:   false,
		},
		{
			name:       "multiple conditions all match",
			conditions: `{"severity": "critical", "status": "firing"}`,
			payload:    map[string]interface{}{"severity": "critical", "status": "firing"},
			expected:   true,
		},
		{
			name:       "multiple conditions partial match",
			conditions: `{"severity": "critical", "status": "firing"}`,
			payload:    map[string]interface{}{"severity": "critical", "status": "resolved"},
			expected:   false,
		},
		{
			name:       "struct payload conversion",
			conditions: `{"Action": "deploy"}`,
			payload:    struct{ Action string }{Action: "deploy"},
			expected:   true,
		},
		{
			name:       "struct with json tags",
			conditions: `{"action": "deploy"}`,
			payload:    struct{ Action string `json:"action"` }{Action: "deploy"},
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchConditions(tt.conditions, tt.payload)
			if got != tt.expected {
				t.Errorf("matchConditions() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPayloadToMap(t *testing.T) {
	tests := []struct {
		name     string
		payload  interface{}
		wantErr  bool
		expected map[string]interface{}
	}{
		{
			name:     "already map",
			payload:  map[string]interface{}{"key": "value"},
			wantErr:  false,
			expected: map[string]interface{}{"key": "value"},
		},
		{
			name:     "nil",
			payload:  nil,
			wantErr:  false,
			expected: map[string]interface{}{},
		},
		{
			name:     "struct to map",
			payload:  struct{ Name string }{Name: "test"},
			wantErr:  false,
			expected: map[string]interface{}{"Name": "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := payloadToMap(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("payloadToMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				for k, v := range tt.expected {
					if got[k] != v {
						t.Errorf("payloadToMap()[%q] = %v, want %v", k, got[k], v)
					}
				}
			}
		})
	}
}

func TestEventRouter_matchRule(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	tests := []struct {
		name     string
		rule     EventRouteRule
		event    BusEvent
		expected bool
	}{
		{
			name:     "event type mismatch",
			rule:     EventRouteRule{EventType: EventAlert},
			event:    BusEvent{Type: EventDeploy, Topic: "test"},
			expected: false,
		},
		{
			name:     "event type match",
			rule:     EventRouteRule{EventType: EventAlert},
			event:    BusEvent{Type: EventAlert, Topic: "test"},
			expected: true,
		},
		{
			name:     "empty rule type matches all",
			rule:     EventRouteRule{EventType: ""},
			event:    BusEvent{Type: EventDeploy, Topic: "test"},
			expected: true,
		},
		{
			name:     "topic prefix mismatch",
			rule:     EventRouteRule{EventType: EventAlert, TopicPrefix: "alert:"},
			event:    BusEvent{Type: EventAlert, Topic: "deploy:test"},
			expected: false,
		},
		{
			name:     "topic prefix match",
			rule:     EventRouteRule{EventType: EventAlert, TopicPrefix: "alert:"},
			event:    BusEvent{Type: EventAlert, Topic: "alert:cpu"},
			expected: true,
		},
		{
			name:     "severity mismatch",
			rule:     EventRouteRule{EventType: EventAlert, Severity: "critical"},
			event:    BusEvent{Type: EventAlert, Topic: "test", Payload: map[string]interface{}{"severity": "warning"}},
			expected: false,
		},
		{
			name:     "severity match",
			rule:     EventRouteRule{EventType: EventAlert, Severity: "critical"},
			event:    BusEvent{Type: EventAlert, Topic: "test", Payload: map[string]interface{}{"severity": "critical"}},
			expected: true,
		},
		{
			name:     "conditions mismatch",
			rule:     EventRouteRule{EventType: EventDeploy, Conditions: `{"action": "rollback"}`},
			event:    BusEvent{Type: EventDeploy, Topic: "test", Payload: map[string]interface{}{"action": "deploy"}},
			expected: false,
		},
		{
			name:     "conditions match",
			rule:     EventRouteRule{EventType: EventDeploy, Conditions: `{"action": "deploy"}`},
			event:    BusEvent{Type: EventDeploy, Topic: "test", Payload: map[string]interface{}{"action": "deploy"}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := router.matchRule(tt.rule, tt.event)
			if got != tt.expected {
				t.Errorf("matchRule() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEventRouter_StartStop(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	router.Start()
	router.Stop()
}

func TestEventRouter_DisabledRule(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	router.SetRules([]EventRouteRule{
		{ID: "disabled", Enabled: false, EventType: EventAlert, TopicPrefix: "alert:"},
	})

	event := BusEvent{
		ID:        "evt-1",
		Type:      EventAlert,
		Topic:     "alert:test",
		Payload:   map[string]interface{}{"severity": "critical"},
		Timestamp: time.Now(),
	}

	router.routeEvent(event)
}

func TestGenerateBusEventID(t *testing.T) {
	id1 := GenerateBusEventID()
	id2 := GenerateBusEventID()

	if id1 == "" {
		t.Error("GenerateBusEventID returned empty string")
	}
	if id1 == id2 {
		t.Error("GenerateBusEventID returned same ID twice")
	}
	if len(id1) < 10 {
		t.Errorf("ID seems too short: %s", id1)
	}
}

func TestEventRouter_routeEvent_NoChannels(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	router.SetRules([]EventRouteRule{
		{ID: "rule-1", Enabled: true, EventType: EventAlert, Channels: []string{"telegram"}},
	})

	event := BusEvent{
		ID:        "evt-nochan",
		Type:      EventAlert,
		Topic:     "alert:test",
		Payload:   map[string]interface{}{"severity": "critical", "message": "Test"},
		Timestamp: time.Now(),
	}

	router.routeEvent(event)
}
