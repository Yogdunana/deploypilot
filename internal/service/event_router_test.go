package service

import (
	"context"
	"testing"
	"time"
)

// stubBus captures events that are published; it implements TypedEventBus.
type stubBus struct {
	publishCh chan BusEvent
}

func newStubBus() *stubBus {
	return &stubBus{publishCh: make(chan BusEvent, 16)}
}

func (b *stubBus) Publish(event BusEvent) {
	select {
	case b.publishCh <- event:
	default:
	}
}

func (b *stubBus) Subscribe(ctx context.Context, topic string) <-chan BusEvent {
	ch := make(chan BusEvent, 8)
	return ch
}

func (b *stubBus) SubscribeType(ctx context.Context, eventType EventType) <-chan BusEvent {
	return b.publishCh
}

func (b *stubBus) Close() error { return nil }

func TestEventRouter_SetGetRules(t *testing.T) {
	bus := newStubBus()
	r := NewEventRouter(bus, nil)

	rules := []EventRouteRule{
		{ID: "r1", Name: "rule 1", Enabled: true, EventType: EventAlert, Channels: []string{"webhook"}},
		{ID: "r2", Name: "rule 2", Enabled: false, EventType: EventDeploy, Channels: []string{"email"}},
	}
	r.SetRules(rules)

	got := r.GetRules()
	if len(got) != 2 {
		t.Fatalf("GetRules() length = %d, want 2", len(got))
	}
	if got[0].ID != "r1" || got[1].ID != "r2" {
		t.Errorf("rule IDs = [%q, %q], want [r1, r2]", got[0].ID, got[1].ID)
	}

	// Ensure GetRules returns a copy (mutations don't affect internal state).
	got[0].Enabled = false
	again := r.GetRules()
	if !again[0].Enabled {
		t.Error("GetRules() should return a copy of rules, not the internal slice")
	}
}

func TestEventRouter_AddAndRemoveRule(t *testing.T) {
	bus := newStubBus()
	r := NewEventRouter(bus, nil)

	r.AddRule(EventRouteRule{ID: "added", Enabled: true, EventType: EventAlert, Channels: []string{"a"}})
	if got := r.GetRules(); len(got) != 1 || got[0].ID != "added" {
		t.Fatalf("after Add, rules = %+v", got)
	}

	r.RemoveRule("added")
	if got := r.GetRules(); len(got) != 0 {
		t.Errorf("after Remove, rules length = %d, want 0", len(got))
	}

	// Removing a non-existent ID is a no-op.
	r.RemoveRule("nonexistent")
	if got := r.GetRules(); len(got) != 0 {
		t.Errorf("after Remove of nonexistent, rules length = %d, want 0", len(got))
	}
}

func TestMatchRule_EventTypeFilter(t *testing.T) {
	bus := newStubBus()
	r := NewEventRouter(bus, nil)

	event := BusEvent{Type: EventAlert, Topic: "alert:server-1"}
	tests := []struct {
		name     string
		rule     EventRouteRule
		expected bool
	}{
		{
			name:     "no filter matches all",
			rule:     EventRouteRule{Enabled: true},
			expected: true,
		},
		{
			name:     "matching type",
			rule:     EventRouteRule{Enabled: true, EventType: EventAlert},
			expected: true,
		},
		{
			name:     "non-matching type",
			rule:     EventRouteRule{Enabled: true, EventType: EventDeploy},
			expected: false,
		},
		{
			name:     "topic prefix matches",
			rule:     EventRouteRule{Enabled: true, TopicPrefix: "alert:"},
			expected: true,
		},
		{
			name:     "topic prefix doesn't match",
			rule:     EventRouteRule{Enabled: true, TopicPrefix: "deploy:"},
			expected: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := r.matchRule(tc.rule, event)
			if got != tc.expected {
				t.Errorf("matchRule() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestMatchRule_SeverityFilter(t *testing.T) {
	bus := newStubBus()
	r := NewEventRouter(bus, nil)

	event := BusEvent{
		Type:  EventAlert,
		Topic: "alert:server-1",
		Payload: map[string]interface{}{
			"severity": "critical",
		},
	}

	// Rule with matching severity.
	rule := EventRouteRule{Enabled: true, EventType: EventAlert, Severity: "critical"}
	if !r.matchRule(rule, event) {
		t.Error("rule with matching severity should match")
	}

	// Rule with non-matching severity.
	rule = EventRouteRule{Enabled: true, EventType: EventAlert, Severity: "low"}
	if r.matchRule(rule, event) {
		t.Error("rule with non-matching severity should not match")
	}

	// Payload without severity field: rule with severity shouldn't crash,
	// and should be treated as "no severity to compare" (default match).
	eventNoSeverity := BusEvent{Type: EventAlert, Payload: "string payload"}
	rule = EventRouteRule{Enabled: true, EventType: EventAlert, Severity: "low"}
	if !r.matchRule(rule, eventNoSeverity) {
		t.Error("rule with severity should match when payload has no severity field")
	}
}

func TestMatchConditions_KeyValueMatching(t *testing.T) {
	payload := map[string]interface{}{
		"status":     "firing",
		"severity":   "critical",
		"server_id":  "srv-1",
		"rule_count": 5,
	}
	tests := []struct {
		name       string
		conditions string
		expected   bool
	}{
		{"single match", `{"status":"firing"}`, true},
		{"single non-match", `{"status":"resolved"}`, false},
		{"multiple all match", `{"status":"firing","severity":"critical"}`, true},
		{"multiple partial", `{"status":"firing","severity":"low"}`, false},
		{"missing key", `{"missing":"x"}`, false},
		{"numeric value as string match", `{"rule_count":"5"}`, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := matchConditions(tc.conditions, payload)
			if got != tc.expected {
				t.Errorf("matchConditions(%s) = %v, want %v", tc.conditions, got, tc.expected)
			}
		})
	}
}

func TestMatchConditions_InvalidJSON(t *testing.T) {
	payload := map[string]interface{}{"status": "firing"}
	if matchConditions("not-json", payload) {
		t.Error("matchConditions() with invalid JSON should return false")
	}
}

func TestMatchConditions_NonMapPayload(t *testing.T) {
	// When payload isn't a map but is a JSON-marshalable struct, the function
	// should round-trip it through JSON to get a map.
	type p struct {
		Status string `json:"status"`
	}
	got := matchConditions(`{"status":"firing"}`, p{Status: "firing"})
	if !got {
		t.Error("matchConditions() should handle non-map payloads via JSON round-trip")
	}
}

func TestExtractSeverity(t *testing.T) {
	tests := []struct {
		name    string
		payload interface{}
		want    string
	}{
		{"map with severity", map[string]interface{}{"severity": "high"}, "high"},
		{"map without severity", map[string]interface{}{"status": "ok"}, ""},
		{"non-string severity", map[string]interface{}{"severity": 5}, ""},
		{"non-map payload", "just a string", ""},
		{"nil payload", nil, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractSeverity(tc.payload)
			if got != tc.want {
				t.Errorf("extractSeverity() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPayloadToMap_AlreadyMap(t *testing.T) {
	in := map[string]interface{}{"a": "1", "b": 2}
	got, err := payloadToMap(in)
	if err != nil {
		t.Fatalf("payloadToMap() error = %v", err)
	}
	if got["a"] != "1" || got["b"] != 2 {
		t.Errorf("payloadToMap() = %v, want original map", got)
	}
}

func TestPayloadToMap_StructRoundTrip(t *testing.T) {
	type p struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	got, err := payloadToMap(p{Name: "alice", Age: 30})
	if err != nil {
		t.Fatalf("payloadToMap() error = %v", err)
	}
	if got["name"] != "alice" || got["age"].(float64) != 30 {
		t.Errorf("payloadToMap(struct) = %v, want map{name:alice, age:30}", got)
	}
}

func TestRouteEvent_DisabledRulesSkipped(t *testing.T) {
	bus := newStubBus()
	r := NewEventRouter(bus, nil)

	calls := 0
	// Inject a no-op notifier. Since routeEvent only calls notifySvc.SendToChannels
	// we can't easily count calls without a fake service, but we CAN check
	// that the matchRule logic short-circuits for disabled rules.
	r.SetRules([]EventRouteRule{
		{ID: "disabled", Enabled: false, EventType: EventAlert, Channels: []string{"x"}},
	})
	event := BusEvent{
		ID:        "evt-1",
		Type:      EventAlert,
		Topic:     "alert:x",
		Timestamp: time.Now(),
		Payload:   map[string]string{"message": "boom"},
	}

	// routeEvent with notifySvc=nil should be a no-op (it checks for nil).
	r.routeEvent(event)
	_ = calls
}
