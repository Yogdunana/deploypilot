package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/model"
)

func TestEventRouter_SetAndGetRules(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	rules := []EventRouteRule{
		{
			ID:        "rule-1",
			Name:      "Critical alerts",
			Enabled:   true,
			EventType: EventAlert,
			Severity:  "critical",
			Channels:  []string{"slack", "email"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "rule-2",
			Name:      "Deploy notifications",
			Enabled:   true,
			EventType: EventDeploy,
			Channels:  []string{"slack"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	router.SetRules(rules)

	got := router.GetRules()
	if len(got) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(got))
	}

	if got[0].ID != "rule-1" {
		t.Errorf("expected rule ID 'rule-1', got %q", got[0].ID)
	}

	if got[0].Severity != "critical" {
		t.Errorf("expected severity 'critical', got %q", got[0].Severity)
	}

	if len(got[0].Channels) != 2 {
		t.Errorf("expected 2 channels, got %d", len(got[0].Channels))
	}
}

func TestEventRouter_AddAndRemoveRule(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	rule1 := EventRouteRule{
		ID:        "rule-1",
		Name:      "Initial rule",
		Enabled:   true,
		EventType: EventAlert,
		Channels:  []string{"slack"},
	}
	router.AddRule(rule1)

	if len(router.GetRules()) != 1 {
		t.Fatal("expected 1 rule after add")
	}

	rule2 := EventRouteRule{
		ID:        "rule-2",
		Name:      "Second rule",
		Enabled:   true,
		EventType: EventDeploy,
		Channels:  []string{"email"},
	}
	router.AddRule(rule2)

	if len(router.GetRules()) != 2 {
		t.Fatal("expected 2 rules after second add")
	}

	router.RemoveRule("rule-1")
	rules := router.GetRules()
	if len(rules) != 1 {
		t.Fatal("expected 1 rule after remove")
	}
	if rules[0].ID != "rule-2" {
		t.Errorf("expected remaining rule ID 'rule-2', got %q", rules[0].ID)
	}

	router.RemoveRule("non-existent")
	if len(router.GetRules()) != 1 {
		t.Error("expected no change for non-existent rule removal")
	}
}

func TestMatchConditions_ValidConditions(t *testing.T) {
	payload := map[string]interface{}{
		"action":  "deploy",
		"status":  "success",
		"app_id":  "app-001",
		"severity": "info",
	}

	conditions := `{"action": "deploy", "status": "success"}`
	if !matchConditions(conditions, payload) {
		t.Error("expected conditions to match")
	}

	conditionsPartial := `{"action": "deploy"}`
	if !matchConditions(conditionsPartial, payload) {
		t.Error("expected partial conditions to match")
	}
}

func TestMatchConditions_MissingField(t *testing.T) {
	payload := map[string]interface{}{
		"action": "deploy",
	}

	conditions := `{"action": "deploy", "status": "success"}`
	if matchConditions(conditions, payload) {
		t.Error("expected conditions NOT to match due to missing field")
	}
}

func TestMatchConditions_WrongValue(t *testing.T) {
	payload := map[string]interface{}{
		"action": "deploy",
		"status": "failed",
	}

	conditions := `{"action": "deploy", "status": "success"}`
	if matchConditions(conditions, payload) {
		t.Error("expected conditions NOT to match due to wrong value")
	}
}

func TestMatchConditions_InvalidJSON(t *testing.T) {
	payload := map[string]interface{}{
		"action": "deploy",
	}

	conditions := `{invalid json`
	if matchConditions(conditions, payload) {
		t.Error("expected conditions NOT to match due to invalid JSON")
	}
}

func TestMatchConditions_TypeCoercion(t *testing.T) {
	payload := map[string]interface{}{
		"count": float64(42),
	}

	conditions := `{"count": 42}`
	if !matchConditions(conditions, payload) {
		t.Error("expected numeric conditions to match with type coercion")
	}
}

func TestExtractSeverity(t *testing.T) {
	tests := []struct {
		name     string
		payload  interface{}
		expected string
	}{
		{
			name:     "direct severity field",
			payload:  map[string]interface{}{"severity": "critical"},
			expected: "critical",
		},
		{
			name:     "nested severity field",
			payload: map[string]interface{}{
				"data": map[string]interface{}{"severity": "high"},
			},
			expected: "",
		},
		{
			name:     "empty payload",
			payload:  map[string]interface{}{},
			expected: "",
		},
		{
			name:     "non-string severity",
			payload:  map[string]interface{}{"severity": 123},
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
			result := extractSeverity(tt.payload)
			if result != tt.expected {
				t.Errorf("extractSeverity() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestPayloadToMap(t *testing.T) {
	t.Run("already a map", func(t *testing.T) {
		payload := map[string]interface{}{
			"key1": "value1",
			"key2": float64(42),
		}

		result, err := payloadToMap(payload)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result["key1"] != "value1" {
			t.Errorf("expected key1='value1', got %v", result["key1"])
		}
	})

	t.Run("struct payload", func(t *testing.T) {
		type testPayload struct {
			Action string `json:"action"`
			Status string `json:"status"`
		}
		payload := testPayload{Action: "deploy", Status: "success"}

		result, err := payloadToMap(payload)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result["action"] != "deploy" {
			t.Errorf("expected action='deploy', got %v", result["action"])
		}
		if result["status"] != "success" {
			t.Errorf("expected status='success', got %v", result["status"])
		}
	})

	t.Run("json marshaling error", func(t *testing.T) {
		ch := make(chan struct{})
		_, err := payloadToMap(ch)
		if err == nil {
			t.Error("expected error for non-marshallable type")
		}
	})
}

func TestEventRouteRule_Structure(t *testing.T) {
	rule := EventRouteRule{
		ID:          "rule-001",
		Name:        "Test Rule",
		Enabled:     true,
		EventType:   EventAlert,
		TopicPrefix: "alert:",
		Severity:    "critical",
		Channels:    []string{"slack", "email"},
		Conditions:  `{"action": "deploy"}`,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if rule.ID != "rule-001" {
		t.Errorf("expected ID 'rule-001', got %q", rule.ID)
	}
	if rule.EventType != EventAlert {
		t.Errorf("expected EventType EventAlert, got %v", rule.EventType)
	}
	if rule.TopicPrefix != "alert:" {
		t.Errorf("expected TopicPrefix 'alert:', got %q", rule.TopicPrefix)
	}
	if rule.Severity != "critical" {
		t.Errorf("expected Severity 'critical', got %q", rule.Severity)
	}
	if len(rule.Channels) != 2 {
		t.Errorf("expected 2 channels, got %d", len(rule.Channels))
	}
}

func TestMatchRule_BasicMatching(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	event := BusEvent{
		ID:     "event-001",
		Type:   EventAlert,
		Topic:  "alert:server-down",
		Source: "monitor",
		Payload: map[string]interface{}{
			"severity": "critical",
			"message":  "Server is down",
		},
		Timestamp: time.Now(),
	}

	t.Run("matches all criteria", func(t *testing.T) {
		rule := EventRouteRule{
			ID:        "rule-1",
			EventType: EventAlert,
			Severity:  "critical",
			Enabled:   true,
		}
		if !router.matchRule(rule, event) {
			t.Error("expected rule to match event")
		}
	})

	t.Run("wrong event type", func(t *testing.T) {
		rule := EventRouteRule{
			ID:        "rule-2",
			EventType: EventDeploy,
			Severity:  "critical",
			Enabled:   true,
		}
		if router.matchRule(rule, event) {
			t.Error("expected rule NOT to match due to wrong event type")
		}
	})

	t.Run("wrong severity", func(t *testing.T) {
		rule := EventRouteRule{
			ID:        "rule-3",
			EventType: EventAlert,
			Severity:  "low",
			Enabled:   true,
		}
		if router.matchRule(rule, event) {
			t.Error("expected rule NOT to match due to wrong severity")
		}
	})

	t.Run("topic prefix mismatch", func(t *testing.T) {
		rule := EventRouteRule{
			ID:          "rule-4",
			EventType:   EventAlert,
			TopicPrefix: "metric:",
			Enabled:     true,
		}
		if router.matchRule(rule, event) {
			t.Error("expected rule NOT to match due to topic prefix mismatch")
		}
	})

	t.Run("topic prefix matches", func(t *testing.T) {
		rule := EventRouteRule{
			ID:          "rule-5",
			EventType:   EventAlert,
			TopicPrefix: "alert:",
			Enabled:     true,
		}
		if !router.matchRule(rule, event) {
			t.Error("expected rule to match with correct topic prefix")
		}
	})

	t.Run("conditions mismatch", func(t *testing.T) {
		rule := EventRouteRule{
			ID:         "rule-6",
			EventType:  EventAlert,
			Conditions: `{"severity": "low"}`,
			Enabled:    true,
		}
		if router.matchRule(rule, event) {
			t.Error("expected rule NOT to match due to conditions mismatch")
		}
	})

	t.Run("disabled rule", func(t *testing.T) {
		rule := EventRouteRule{
			ID:        "rule-7",
			EventType: EventAlert,
			Enabled:   false,
		}
		if router.matchRule(rule, event) {
			t.Error("expected disabled rule NOT to match")
		}
	})

	t.Run("empty event type matches all", func(t *testing.T) {
		rule := EventRouteRule{
			ID:      "rule-8",
			Enabled: true,
		}
		if !router.matchRule(rule, event) {
			t.Error("expected rule with empty event type to match")
		}
	})
}

func TestEventTypes(t *testing.T) {
	if EventUser != "user" {
		t.Errorf("expected EventUser='user', got %q", EventUser)
	}
	if EventServer != "server" {
		t.Errorf("expected EventServer='server', got %q", EventServer)
	}
	if EventSecurity != "security" {
		t.Errorf("expected EventSecurity='security', got %q", EventSecurity)
	}
	if EventAudit != "audit" {
		t.Errorf("expected EventAudit='audit', got %q", EventAudit)
	}
	if EventBackup != "backup" {
		t.Errorf("expected EventBackup='backup', got %q", EventBackup)
	}
}

func TestUserEventPayload(t *testing.T) {
	payload := UserEventPayload{
		UserID:    "user-001",
		Username:  "testuser",
		Action:    "login",
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0",
		Success:   true,
		Message:   "Login successful",
	}

	if payload.UserID != "user-001" {
		t.Errorf("expected UserID 'user-001', got %q", payload.UserID)
	}
	if payload.Action != "login" {
		t.Errorf("expected Action 'login', got %q", payload.Action)
	}
	if !payload.Success {
		t.Error("expected Success=true")
	}
}

func TestDeployEventPayload(t *testing.T) {
	payload := DeployEventPayload{
		AppID:      "app-001",
		AppName:    "test-app",
		ServerID:   "server-001",
		ServerName: "prod-server",
		Action:     "success",
		Status:     "deployed",
		Duration:   30000,
		Message:    "Deployment completed",
	}

	if payload.AppID != "app-001" {
		t.Errorf("expected AppID 'app-001', got %q", payload.AppID)
	}
	if payload.Duration != 30000 {
		t.Errorf("expected Duration 30000, got %d", payload.Duration)
	}
}

func TestServerEventPayload(t *testing.T) {
	payload := ServerEventPayload{
		ServerID:   "server-001",
		ServerName: "web-server",
		Action:     "cpu_high",
		Metric:     "cpu_usage",
		Value:      95.5,
		Threshold:  80.0,
		Message:    "CPU usage exceeded threshold",
	}

	if payload.Action != "cpu_high" {
		t.Errorf("expected Action 'cpu_high', got %q", payload.Action)
	}
	if payload.Value != 95.5 {
		t.Errorf("expected Value 95.5, got %f", payload.Value)
	}
}

func TestSecurityEventPayload(t *testing.T) {
	payload := SecurityEventPayload{
		Action:    "brute_force",
		IPAddress: "192.168.1.100",
		Username:  "admin",
		Detail:    "Multiple failed login attempts",
		Severity:  "high",
	}

	if payload.Action != "brute_force" {
		t.Errorf("expected Action 'brute_force', got %q", payload.Action)
	}
	if payload.Severity != "high" {
		t.Errorf("expected Severity 'high', got %q", payload.Severity)
	}
}

func TestBackupEventPayload(t *testing.T) {
	payload := BackupEventPayload{
		AppID:    "app-001",
		AppName:  "test-app",
		Action:   "success",
		Location: "s3",
		Size:     1024000,
		Message:  "Backup completed successfully",
	}

	if payload.Action != "success" {
		t.Errorf("expected Action 'success', got %q", payload.Action)
	}
	if payload.Location != "s3" {
		t.Errorf("expected Location 's3', got %q", payload.Location)
	}
	if payload.Size != 1024000 {
		t.Errorf("expected Size 1024000, got %d", payload.Size)
	}
}

func TestEventRouter_Stop(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	router.AddRule(EventRouteRule{ID: "before-stop"})
	router.Stop()

	if len(router.GetRules()) != 1 {
		t.Error("expected 1 rule after stop")
	}
}

func TestMatchConditions_WithNestedPayload(t *testing.T) {
	payload := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": "value",
		},
	}

	conditions := `{"level1.level2": "value"}`
	result := matchConditions(conditions, payload)

	if result {
		t.Error("expected conditions NOT to match (nested path not supported)")
	}
}

func TestPayloadToMap_WithJSONString(t *testing.T) {
	jsonStr := `{"action": "test", "count": 42}`

	var rawPayload interface{}
	if err := json.Unmarshal([]byte(jsonStr), &rawPayload); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	result, err := payloadToMap(rawPayload)
	if err != nil {
		t.Fatalf("payloadToMap failed: %v", err)
	}

	if result["action"] != "test" {
		t.Errorf("expected action='test', got %v", result["action"])
	}
}

func TestEventRouter_GetRules_Immutability(t *testing.T) {
	bus := NewInMemoryTypedEventBus()
	router := NewEventRouter(bus, nil)

	rules := []EventRouteRule{
		{ID: "original", Name: "Original", Channels: []string{"slack"}},
	}
	router.SetRules(rules)

	returned := router.GetRules()
	returned[0].ID = "modified"

	current := router.GetRules()
	if current[0].ID != "original" {
		t.Error("GetRules should return a copy, not the original")
	}
}

func TestAlertEscalation_Structure(t *testing.T) {
	escalation := model.AlertEscalation{
		ID:        "esc-001",
		Name:      "Critical escalation",
		Enabled:   true,
		Steps:     `[{"after_minutes": 5, "severity": "high", "channels": ["slack"]}]`,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if escalation.ID != "esc-001" {
		t.Errorf("expected ID 'esc-001', got %q", escalation.ID)
	}
	if !escalation.Enabled {
		t.Error("expected Enabled=true")
	}
}

func TestAlertGroup_Structure(t *testing.T) {
	group := model.AlertGroup{
		ID:        "group-001",
		GroupKey:  "fingerprint-hash",
		RuleID:    "rule-001",
		Severity:  "critical",
		Status:    "firing",
	}

	if group.ID != "group-001" {
		t.Errorf("expected ID 'group-001', got %q", group.ID)
	}
	if group.Status != "firing" {
		t.Errorf("expected Status 'firing', got %q", group.Status)
	}
}
