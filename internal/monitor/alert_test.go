package monitor

import (
	"testing"
	"time"
)

func TestDefaultRules(t *testing.T) {
	rules := DefaultRules()
	if len(rules) != 3 {
		t.Fatalf("expected 3 default rules, got %d", len(rules))
	}

	found := map[string]bool{}
	for _, r := range rules {
		found[r.ID] = true
		if !r.Enabled {
			t.Errorf("rule %s should be enabled", r.ID)
		}
	}

	for _, id := range []string{"disk-space", "memory-high", "cpu-high"} {
		if !found[id] {
			t.Errorf("missing rule %s", id)
		}
	}
}

func TestEvaluate_NoAlerts(t *testing.T) {
	am := NewAlertManager()
	metrics := []Metric{
		{Type: MetricCPU, Name: "cpu_usage", Value: 50.0, Unit: "percent"},
		{Type: MetricMemory, Name: "memory_usage_percent", Value: 60.0, Unit: "percent"},
		{Type: MetricDisk, Name: "disk_usage_percent", Value: 50.0, Unit: "percent"},
	}

	alerts := am.Evaluate(metrics)
	if len(alerts) != 0 {
		t.Errorf("expected no alerts, got %d", len(alerts))
	}
}

func TestEvaluate_DiskLow(t *testing.T) {
	am := NewAlertManager()
	metrics := []Metric{
		{Type: MetricDisk, Name: "disk_available_percent", Value: 3.0, Unit: "percent"},
	}

	alerts := am.Evaluate(metrics)
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	if alerts[0].RuleID != "disk-space" {
		t.Errorf("expected rule 'disk-space', got %q", alerts[0].RuleID)
	}
	if alerts[0].Severity != SeverityWarning {
		t.Errorf("expected severity 'warning', got %q", alerts[0].Severity)
	}
	if alerts[0].Status != "firing" {
		t.Errorf("expected status 'firing', got %q", alerts[0].Status)
	}
}

func TestEvaluate_MemoryHigh(t *testing.T) {
	am := NewAlertManager()
	metrics := []Metric{
		{Type: MetricMemory, Name: "memory_usage_percent", Value: 95.0, Unit: "percent"},
	}

	alerts := am.Evaluate(metrics)
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	if alerts[0].RuleID != "memory-high" {
		t.Errorf("expected rule 'memory-high', got %q", alerts[0].RuleID)
	}
}

func TestEvaluate_CPUHigh(t *testing.T) {
	am := NewAlertManager()
	metrics := []Metric{
		{Type: MetricCPU, Name: "cpu_usage", Value: 98.0, Unit: "percent"},
	}

	alerts := am.Evaluate(metrics)
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	if alerts[0].RuleID != "cpu-high" {
		t.Errorf("expected rule 'cpu-high', got %q", alerts[0].RuleID)
	}
}

func TestEvaluate_Cooldown(t *testing.T) {
	am := NewAlertManager()
	// Set a very short cooldown for testing
	am.rules[0].Cooldown = 1 * time.Hour

	metrics := []Metric{
		{Type: MetricDisk, Name: "disk_available_percent", Value: 2.0, Unit: "percent"},
	}

	// First evaluation should fire
	alerts1 := am.Evaluate(metrics)
	if len(alerts1) != 1 {
		t.Fatalf("expected 1 alert on first evaluation, got %d", len(alerts1))
	}

	// Second evaluation should be suppressed by cooldown
	alerts2 := am.Evaluate(metrics)
	if len(alerts2) != 0 {
		t.Errorf("expected 0 alerts due to cooldown, got %d", len(alerts2))
	}
}

func TestEvaluate_NoDuplicateAlerts(t *testing.T) {
	am := NewAlertManager()
	metrics := []Metric{
		{Type: MetricMemory, Name: "memory_usage_percent", Value: 95.0, Unit: "percent"},
	}

	// First evaluation fires
	alerts1 := am.Evaluate(metrics)
	if len(alerts1) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts1))
	}

	// Second evaluation should not fire again (already firing)
	alerts2 := am.Evaluate(metrics)
	if len(alerts2) != 0 {
		t.Errorf("expected 0 new alerts (already firing), got %d", len(alerts2))
	}
}

func TestResolveAlert(t *testing.T) {
	am := NewAlertManager()
	metrics := []Metric{
		{Type: MetricMemory, Name: "memory_usage_percent", Value: 95.0, Unit: "percent"},
	}

	// Fire alert
	am.Evaluate(metrics)

	// Verify it's active
	active := am.GetActiveAlerts()
	if len(active) != 1 {
		t.Fatalf("expected 1 active alert, got %d", len(active))
	}

	// Resolve it
	am.ResolveAlert("memory-high")

	// Verify no active alerts
	active = am.GetActiveAlerts()
	if len(active) != 0 {
		t.Errorf("expected 0 active alerts after resolve, got %d", len(active))
	}
}

func TestGetActiveAlerts_Empty(t *testing.T) {
	am := NewAlertManager()
	alerts := am.GetActiveAlerts()
	if len(alerts) != 0 {
		t.Errorf("expected 0 active alerts, got %d", len(alerts))
	}
}

func TestGetRules(t *testing.T) {
	am := NewAlertManager()
	rules := am.GetRules()
	if len(rules) != 3 {
		t.Errorf("expected 3 rules, got %d", len(rules))
	}
}

func TestAddRule(t *testing.T) {
	am := NewAlertManager()
	am.AddRule(AlertRule{
		ID:         "custom-rule",
		Name:       "Custom Rule",
		MetricType: MetricCPU,
		Condition:  "gt",
		Threshold:  80.0,
		Severity:   SeverityCritical,
		Enabled:    true,
		Cooldown:   5 * time.Minute,
	})

	rules := am.GetRules()
	if len(rules) != 4 {
		t.Errorf("expected 4 rules after add, got %d", len(rules))
	}
}

func TestEvaluate_DisabledRule(t *testing.T) {
	am := NewAlertManager()
	am.rules[0].Enabled = false

	metrics := []Metric{
		{Type: MetricDisk, Name: "disk_available_percent", Value: 1.0, Unit: "percent"},
	}

	alerts := am.Evaluate(metrics)
	if len(alerts) != 0 {
		t.Errorf("expected no alerts for disabled rule, got %d", len(alerts))
	}
}

func TestEvaluateCondition(t *testing.T) {
	am := NewAlertManager()

	tests := []struct {
		value     float64
		condition string
		threshold float64
		want      bool
	}{
		{10.0, "gt", 5.0, true},
		{3.0, "gt", 5.0, false},
		{3.0, "lt", 5.0, true},
		{10.0, "lt", 5.0, false},
		{5.0, "eq", 5.0, true},
		{5.0, "eq", 3.0, false},
		{5.0, "neq", 3.0, true},
		{5.0, "neq", 5.0, false},
		{5.0, "gte", 5.0, true},
		{4.0, "gte", 5.0, false},
		{5.0, "lte", 5.0, true},
		{6.0, "lte", 5.0, false},
	}

	for _, tt := range tests {
		got := am.evaluateCondition(tt.value, tt.condition, tt.threshold)
		if got != tt.want {
			t.Errorf("evaluateCondition(%.1f, %s, %.1f) = %v, want %v",
				tt.value, tt.condition, tt.threshold, got, tt.want)
		}
	}
}
