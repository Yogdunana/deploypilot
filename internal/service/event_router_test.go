package service

import (
	"testing"
	"time"
)

// ===================== extractSeverity Tests =====================

func TestExtractSeverity(t *testing.T) {
	tests := []struct {
		name     string
		payload  interface{}
		expected string
	}{
		{
			name:     "valid severity in map",
			payload:  map[string]interface{}{"severity": "high", "message": "test"},
			expected: "high",
		},
		{
			name:     "valid severity medium",
			payload:  map[string]interface{}{"severity": "medium"},
			expected: "medium",
		},
		{
			name:     "valid severity critical",
			payload:  map[string]interface{}{"severity": "critical"},
			expected: "critical",
		},
		{
			name:     "empty severity",
			payload:  map[string]interface{}{"severity": ""},
			expected: "",
		},
		{
			name:     "severity field missing",
			payload:  map[string]interface{}{"message": "test"},
			expected: "",
		},
		{
			name:     "severity wrong type - int",
			payload:  map[string]interface{}{"severity": 1},
			expected: "",
		},
		{
			name:     "nil payload",
			payload:  nil,
			expected: "",
		},
		{
			name:     "non-map payload",
			payload:  "string payload",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractSeverity(tc.payload)
			if result != tc.expected {
				t.Errorf("extractSeverity() = %q, want %q", result, tc.expected)
			}
		})
	}
}

// ===================== matchConditions Tests =====================

func TestMatchConditions(t *testing.T) {
	tests := []struct {
		name        string
		conditions  string
		payload     interface{}
		expectMatch bool
	}{
		{
			name:        "exact string match",
			conditions:  `{"action": "login"}`,
			payload:     map[string]interface{}{"action": "login"},
			expectMatch: true,
		},
		{
			name:        "string mismatch",
			conditions:  `{"action": "login"}`,
			payload:     map[string]interface{}{"action": "logout"},
			expectMatch: false,
		},
		{
			name:        "missing key in payload",
			conditions:  `{"action": "login"}`,
			payload:     map[string]interface{}{"status": "active"},
			expectMatch: false,
		},
		{
			name:        "extra key in payload - OK",
			conditions:  `{"action": "login"}`,
			payload:     map[string]interface{}{"action": "login", "extra": "value"},
			expectMatch: true,
		},
		{
			name:        "numeric equality via string conversion",
			conditions:  `{"count": "5"}`,
			payload:     map[string]interface{}{"count": 5},
			expectMatch: true,
		},
		{
			name:        "invalid JSON conditions",
			conditions:  `{invalid}`,
			payload:     map[string]interface{}{"action": "login"},
			expectMatch: false,
		},
		{
			name:        "empty conditions",
			conditions:  `{}`,
			payload:     map[string]interface{}{"action": "login"},
			expectMatch: true,
		},
		{
			name:        "multiple conditions all match",
			conditions:  `{"action": "login", "success": "true"}`,
			payload:     map[string]interface{}{"action": "login", "success": true},
			expectMatch: true,
		},
		{
			name:        "multiple conditions one mismatch",
			conditions:  `{"action": "login", "success": "true"}`,
			payload:     map[string]interface{}{"action": "login", "success": false},
			expectMatch: false,
		},
		{
			name:        "struct payload converted to map",
			conditions:  `{"action": "login"}`,
			payload:     UserEventPayload{Action: "login"},
			expectMatch: true,
		},
		{
			name:        "struct payload mismatch",
			conditions:  `{"action": "login"}`,
			payload:     UserEventPayload{Action: "logout"},
			expectMatch: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := matchConditions(tc.conditions, tc.payload)
			if result != tc.expectMatch {
				t.Errorf("matchConditions(%q, %v) = %v, want %v", tc.conditions, tc.payload, result, tc.expectMatch)
			}
		})
	}
}

// ===================== payloadToMap Tests =====================

func TestPayloadToMap(t *testing.T) {
	tests := []struct {
		name      string
		payload   interface{}
		expectErr bool
		expectMap map[string]interface{}
	}{
		{
			name:      "already a map",
			payload:   map[string]interface{}{"key": "value", "num": 42},
			expectErr: false,
			expectMap: map[string]interface{}{"key": "value", "num": 42},
		},
		{
			name:      "struct converts to map",
			payload:   UserEventPayload{UserID: "u1", Username: "testuser", Action: "login"},
			expectErr: false,
			expectMap: map[string]interface{}{"user_id": "u1", "username": "testuser", "action": "login"},
		},
		{
			name:      "nil payload",
			payload:   nil,
			expectErr: false,
			expectMap: map[string]interface{}{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := payloadToMap(tc.payload)
			if (err != nil) != tc.expectErr {
				t.Errorf("payloadToMap() error = %v, expectErr %v", err, tc.expectErr)
				return
			}
			if !tc.expectErr {
				for key, expectedVal := range tc.expectMap {
					if actualVal, ok := result[key]; !ok || actualVal != expectedVal {
						t.Errorf("payloadToMap()[%q] = %v, want %v", key, actualVal, expectedVal)
					}
				}
			}
		})
	}
}

// ===================== EventRouter Rule Management Tests =====================

func TestEventRouter_SetRules(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	rules := []EventRouteRule{
		{ID: "rule-1", Name: "Rule One", Enabled: true, EventType: EventDeploy},
		{ID: "rule-2", Name: "Rule Two", Enabled: false, EventType: EventAlert},
	}

	router.SetRules(rules)

	got := router.GetRules()
	if len(got) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(got))
	}
	if got[0].ID != "rule-1" || got[1].ID != "rule-2" {
		t.Errorf("unexpected rule IDs: %v", got)
	}
}

func TestEventRouter_GetRules_Immutability(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	router.SetRules([]EventRouteRule{{ID: "original"}})

	// Modify returned slice should not affect internal state
	rules := router.GetRules()
	rules[0].ID = "modified"

	rules = router.GetRules()
	if rules[0].ID != "original" {
		t.Error("GetRules should return immutable copy")
	}
}

func TestEventRouter_AddRule(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	router.AddRule(EventRouteRule{ID: "rule-1", Name: "First"})
	router.AddRule(EventRouteRule{ID: "rule-2", Name: "Second"})

	rules := router.GetRules()
	if len(rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(rules))
	}
}

func TestEventRouter_RemoveRule(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	router.AddRule(EventRouteRule{ID: "rule-1", Name: "First"})
	router.AddRule(EventRouteRule{ID: "rule-2", Name: "Second"})
	router.AddRule(EventRouteRule{ID: "rule-3", Name: "Third"})

	router.RemoveRule("rule-2")

	rules := router.GetRules()
	if len(rules) != 2 {
		t.Errorf("expected 2 rules after removal, got %d", len(rules))
	}
	for _, r := range rules {
		if r.ID == "rule-2" {
			t.Error("rule-2 should have been removed")
		}
	}
}

func TestEventRouter_RemoveRule_NotFound(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	router.AddRule(EventRouteRule{ID: "rule-1"})
	router.RemoveRule("nonexistent")

	rules := router.GetRules()
	if len(rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(rules))
	}
}

// ===================== EventRouter matchRule Tests =====================

func TestEventRouter_MatchRule(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	router.SetRules([]EventRouteRule{
		{
			ID:          "rule-deploy",
			EventType:   EventDeploy,
			TopicPrefix: "deploy:app-",
			Severity:    "",
			Conditions:  "",
			Enabled:     true,
		},
	})

	event := BusEvent{
		ID:    "evt-1",
		Type:  EventDeploy,
		Topic: "deploy:app-1",
		Payload: map[string]interface{}{
			"app_name": "myapp",
			"status":   "success",
		},
		Timestamp: time.Now(),
	}

	rules := router.GetRules()
	if !router.matchRule(rules[0], event) {
		t.Error("expected event to match rule")
	}
}

func TestEventRouter_MatchRule_TypeMismatch(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	router.SetRules([]EventRouteRule{
		{
			ID:        "rule-deploy",
			EventType: EventDeploy,
			Enabled:   true,
		},
	})

	event := BusEvent{
		ID:        "evt-1",
		Type:      EventAlert, // wrong type
		Topic:     "alert:server-1",
		Payload:   nil,
		Timestamp: time.Now(),
	}

	rules := router.GetRules()
	if router.matchRule(rules[0], event) {
		t.Error("expected event NOT to match rule due to type mismatch")
	}
}

func TestEventRouter_MatchRule_TopicPrefixMismatch(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	router.SetRules([]EventRouteRule{
		{
			ID:          "rule-deploy",
			EventType:   EventDeploy,
			TopicPrefix: "deploy:app-",
			Enabled:     true,
		},
	})

	event := BusEvent{
		ID:        "evt-1",
		Type:      EventDeploy,
		Topic:     "deploy:web-1", // wrong prefix
		Payload:   nil,
		Timestamp: time.Now(),
	}

	rules := router.GetRules()
	if router.matchRule(rules[0], event) {
		t.Error("expected event NOT to match rule due to topic prefix mismatch")
	}
}

func TestEventRouter_MatchRule_SeverityMismatch(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	router.SetRules([]EventRouteRule{
		{
			ID:       "rule-alert",
			EventType: EventAlert,
			Severity:  "critical",
			Enabled:   true,
		},
	})

	event := BusEvent{
		ID:    "evt-1",
		Type:  EventAlert,
		Topic: "alert:test",
		Payload: map[string]interface{}{
			"severity": "low", // doesn't match "critical"
		},
		Timestamp: time.Now(),
	}

	rules := router.GetRules()
	if router.matchRule(rules[0], event) {
		t.Error("expected event NOT to match rule due to severity mismatch")
	}
}

func TestEventRouter_MatchRule_SeverityMatch(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	router.SetRules([]EventRouteRule{
		{
			ID:       "rule-alert",
			EventType: EventAlert,
			Severity:  "critical",
			Enabled:   true,
		},
	})

	event := BusEvent{
		ID:    "evt-1",
		Type:  EventAlert,
		Topic: "alert:test",
		Payload: map[string]interface{}{
			"severity": "critical",
		},
		Timestamp: time.Now(),
	}

	rules := router.GetRules()
	if !router.matchRule(rules[0], event) {
		t.Error("expected event to match rule with matching severity")
	}
}

// ===================== EventRouter Lifecycle Tests =====================

func TestEventRouter_StartStop(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	// Should not panic on start/stop
	router.Start()
	router.Stop()
}

func TestEventRouter_NewEventRouter(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	notifySvc := &NotificationService{}
	router := NewEventRouter(bus, notifySvc)

	if router == nil {
		t.Fatal("NewEventRouter returned nil")
	}
	if router.bus != bus {
		t.Error("bus not set correctly")
	}
	if router.notifySvc != notifySvc {
		t.Error("notifySvc not set correctly")
	}
	if len(router.rules) != 0 {
		t.Errorf("expected empty rules, got %d", len(router.rules))
	}
}
