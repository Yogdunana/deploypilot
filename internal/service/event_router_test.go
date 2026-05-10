package service

import (
	"encoding/json"
	"sync"
	"testing"
	"time"
)

func TestNewEventRouter(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)
	if router == nil {
		t.Fatal("NewEventRouter returned nil")
	}
	if router.bus != bus {
		t.Error("expected bus to be set correctly")
	}
	if router.rules == nil {
		t.Error("expected rules to be initialized")
	}
}

func TestEventRouter_SetAndGetRules(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	rules := []EventRouteRule{
		{
			ID:        "rule-1",
			Name:      "Test Rule",
			Enabled:   true,
			EventType: EventAlert,
			Channels:  []string{"email"},
		},
	}

	router.SetRules(rules)
	got := router.GetRules()

	if len(got) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(got))
	}
	if got[0].ID != "rule-1" {
		t.Errorf("ID = %q, want %q", got[0].ID, "rule-1")
	}
}

func TestEventRouter_AddRule(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	router.AddRule(EventRouteRule{
		ID:        "rule-new",
		Name:      "New Rule",
		Enabled:   true,
		EventType: EventDeploy,
		Channels:  []string{"slack"},
	})

	rules := router.GetRules()
	if len(rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(rules))
	}
}

func TestEventRouter_RemoveRule(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	router.AddRule(EventRouteRule{ID: "rule-1", Name: "Rule 1"})
	router.AddRule(EventRouteRule{ID: "rule-2", Name: "Rule 2"})

	router.RemoveRule("rule-1")

	rules := router.GetRules()
	if len(rules) != 1 {
		t.Errorf("expected 1 rule after removal, got %d", len(rules))
	}
	if rules[0].ID != "rule-2" {
		t.Errorf("expected remaining rule ID to be rule-2, got %q", rules[0].ID)
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

func TestEventRouter_StartStop(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	router.Start()
	time.Sleep(100 * time.Millisecond)

	router.Stop()
}

func TestMatchRule_MatchEventType(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	event := BusEvent{
		ID:    "evt-1",
		Type:  EventAlert,
		Topic: "alert:warning",
	}

	rule := EventRouteRule{
		EventType: EventAlert,
		Enabled:   true,
	}

	if !router.matchRule(rule, event) {
		t.Error("expected match for matching event type")
	}

	rule.EventType = EventDeploy
	if router.matchRule(rule, event) {
		t.Error("expected no match for non-matching event type")
	}
}

func TestMatchRule_MatchTopicPrefix(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	event := BusEvent{
		ID:    "evt-1",
		Type:  EventAlert,
		Topic: "alert:critical",
	}

	rule := EventRouteRule{
		TopicPrefix: "alert:",
		Enabled:     true,
	}

	if !router.matchRule(rule, event) {
		t.Error("expected match for matching topic prefix")
	}

	rule.TopicPrefix = "deploy:"
	if router.matchRule(rule, event) {
		t.Error("expected no match for non-matching topic prefix")
	}
}

func TestMatchRule_MatchSeverity(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	event := BusEvent{
		ID:      "evt-1",
		Type:    EventAlert,
		Topic:   "alert:high",
		Payload: map[string]interface{}{"severity": "high"},
	}

	rule := EventRouteRule{
		Severity: "high",
		Enabled:  true,
	}

	if !router.matchRule(rule, event) {
		t.Error("expected match for matching severity")
	}

	rule.Severity = "low"
	if router.matchRule(rule, event) {
		t.Error("expected no match for non-matching severity")
	}
}

func TestMatchRule_MatchConditions(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	event := BusEvent{
		ID:      "evt-1",
		Type:    EventAlert,
		Topic:   "alert:warning",
		Payload: map[string]interface{}{"status": "firing", "severity": "warning"},
	}

	rule := EventRouteRule{
		Conditions: `{"status": "firing"}`,
		Enabled:   true,
	}

	if !router.matchRule(rule, event) {
		t.Error("expected match for matching conditions")
	}

	rule.Conditions = `{"status": "resolved"}`
	if router.matchRule(rule, event) {
		t.Error("expected no match for non-matching conditions")
	}
}

func TestMatchRule_DisabledRule(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	event := BusEvent{
		ID:    "evt-1",
		Type:  EventAlert,
		Topic: "alert:warning",
	}

	rule := EventRouteRule{
		EventType: EventAlert,
		Enabled:   false,
	}

	if router.matchRule(rule, event) {
		t.Error("expected no match for disabled rule")
	}
}

func TestMatchConditions_InvalidJSON(t *testing.T) {
	result := matchConditions("not valid json", map[string]interface{}{})
	if result {
		t.Error("expected false for invalid JSON conditions")
	}
}

func TestMatchConditions_EmptyPayload(t *testing.T) {
	result := matchConditions(`{"key": "value"}`, nil)
	if result {
		t.Error("expected false for nil payload")
	}
}

func TestExtractSeverity(t *testing.T) {
	tests := []struct {
		name     string
		payload  interface{}
		expected string
	}{
		{
			name:     "map with severity",
			payload:  map[string]interface{}{"severity": "high"},
			expected: "high",
		},
		{
			name:     "map without severity",
			payload:  map[string]interface{}{"status": "ok"},
			expected: "",
		},
		{
			name:     "non-map payload",
			payload:  "string payload",
			expected: "",
		},
		{
			name:     "nil payload",
			payload:  nil,
			expected: "",
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

func TestPayloadToMap(t *testing.T) {
	original := map[string]interface{}{"key": "value", "num": 42}
	got, err := payloadToMap(original)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["key"] != "value" {
		t.Errorf("key = %v, want %v", got["key"], "value")
	}
}

func TestPayloadToMap_FromStruct(t *testing.T) {
	type testPayload struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	payload := testPayload{Name: "test", Age: 25}
	got, err := payloadToMap(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["name"] != "test" {
		t.Errorf("name = %v, want %v", got["name"], "test")
	}
}

func TestPayloadToMap_AlreadyMap(t *testing.T) {
	m := map[string]interface{}{"already": "map"}
	got, err := payloadToMap(m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["already"] != "map" {
		t.Errorf("already = %v, want %v", got["already"], "map")
	}
}

func TestPayloadToMap_InvalidJSON(t *testing.T) {
	type nonSerializable struct {
		Chan chan int
	}

	payload := nonSerializable{Chan: make(chan int)}
	_, err := payloadToMap(payload)
	if err == nil {
		t.Error("expected error for non-serializable payload")
	}
}

func TestMatchConditions_ComplexPayload(t *testing.T) {
	payload := map[string]interface{}{
		"action": "deploy",
		"status": "success",
		"app": map[string]interface{}{
			"name": "my-app",
			"env":  "production",
		},
	}

	conditions := `{"action": "deploy"}`
	if !matchConditions(conditions, payload) {
		t.Error("expected match for simple condition")
	}

	conditions = `{"status": "failed"}`
	if matchConditions(conditions, payload) {
		t.Error("expected no match for non-matching condition")
	}

	conditions = `{"app.name": "other-app"}`
	if matchConditions(conditions, payload) {
		t.Error("expected no match for non-existent nested key")
	}
}

func TestMatchConditions_FromJSONString(t *testing.T) {
	payloadJSON := `{"status": "active", "count": 5}`
	var payload interface{}
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

	conditions := `{"status": "active"}`
	if !matchConditions(conditions, payload) {
		t.Error("expected match for JSON String payload")
	}
}

func TestForwardToChannels_NilNotifySvc(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	event := BusEvent{
		ID:      "evt-no-notify",
		Type:    EventAlert,
		Topic:   "alert:test",
		Payload: map[string]interface{}{"message": "test"},
	}

	router.forwardToChannels(event, []string{"email"})
}

func TestForwardToChannels_EmptyChannels(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	event := BusEvent{
		ID:      "evt-empty-ch",
		Type:    EventAlert,
		Topic:   "alert:test",
		Payload: map[string]interface{}{"message": "test"},
	}

	router.forwardToChannels(event, []string{})
}

func TestEventRouter_Concurrent(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	router.Start()
	defer router.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			bus.Publish(BusEvent{
				ID:        "evt-concurrent",
				Type:      EventAlert,
				Topic:     "alert:test",
				Payload:   map[string]interface{}{"index": n},
				Timestamp: time.Now(),
			})
		}(i)
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)
}

func TestMatchConditions_MissingField(t *testing.T) {
	payload := map[string]interface{}{
		"status": "active",
	}

	conditions := `{"missing_key": "value"}`
	if matchConditions(conditions, payload) {
		t.Error("expected no match when payload is missing condition key")
	}
}

func TestMatchConditions_NumericComparison(t *testing.T) {
	payload := map[string]interface{}{
		"count": float64(10),
	}

	conditions := `{"count": "10"}`
	if !matchConditions(conditions, payload) {
		t.Error("expected match for numeric value with string condition")
	}
}

func TestExtractSeverity_NoSeverity(t *testing.T) {
	result := extractSeverity(map[string]interface{}{"status": "ok", "count": 5})
	if result != "" {
		t.Errorf("expected empty string for payload without severity, got %q", result)
	}
}

func TestMatchRule_MultipleConditions(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	event := BusEvent{
		ID:      "evt-multi",
		Type:    EventAlert,
		Topic:   "alert:warning",
		Payload: map[string]interface{}{"status": "firing", "severity": "warning", "count": float64(5)},
	}

	rule := EventRouteRule{
		Conditions: `{"status": "firing", "severity": "warning"}`,
		Enabled:   true,
	}

	if !router.matchRule(rule, event) {
		t.Error("expected match for multiple matching conditions")
	}

	rule.Conditions = `{"status": "firing", "severity": "critical"}`
	if router.matchRule(rule, event) {
		t.Error("expected no match when one condition fails")
	}
}

func TestEventRouter_GetRules_ReturnsCopy(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	router.AddRule(EventRouteRule{ID: "rule-1", Name: "Rule 1"})

	rules1 := router.GetRules()
	rules2 := router.GetRules()

	if len(rules1) != len(rules2) {
		t.Errorf("rule counts differ: %d vs %d", len(rules1), len(rules2))
	}

	for i := range rules1 {
		if rules1[i].ID == rules2[i].ID && &rules1 == &rules2 {
			t.Error("GetRules should return different slices")
		}
	}
}

func TestPayloadToMap_FromMapStringInterface(t *testing.T) {
	m := map[string]interface{}{
		"string_val": "test",
		"int_val":    float64(42),
		"bool_val":   true,
		"nil_val":    nil,
	}

	got, err := payloadToMap(m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got["string_val"] != "test" {
		t.Errorf("string_val = %v, want %v", got["string_val"], "test")
	}
	if got["int_val"] != float64(42) {
		t.Errorf("int_val = %v, want %v", got["int_val"], 42)
	}
	if got["bool_val"] != true {
		t.Errorf("bool_val = %v, want %v", got["bool_val"], true)
	}
}

func TestExtractSeverity_NestedPayload(t *testing.T) {
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"severity": "critical",
		},
	}
	result := extractSeverity(payload)
	if result != "" {
		t.Errorf("expected empty for nested severity, got %q", result)
	}
}
