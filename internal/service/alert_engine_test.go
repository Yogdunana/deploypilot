package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/database"
	"github.com/Yogdunana/deploypilot/internal/model"
)

func TestAlertEngine_IsSilenced(t *testing.T) {
	db, err := database.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	engine := NewAlertEngine(db, nil)
	now := time.Now()

	silence := &model.AlertSilence{
		ID:        "silence-001",
		Name:      "Test Silence",
		Matchers:  `{"severity": "critical"}`,
		StartsAt:  now.Add(-1 * time.Hour),
		EndsAt:    now.Add(1 * time.Hour),
	}
	if err := db.Create(silence).Error; err != nil {
		t.Fatalf("failed to create silence: %v", err)
	}

	t.Run("silenced alert matches", func(t *testing.T) {
		silenced, name := engine.IsSilenced("critical", "server-001")
		if !silenced {
			t.Error("expected alert to be silenced")
		}
		if name != "Test Silence" {
			t.Errorf("expected silence name 'Test Silence', got %q", name)
		}
	})

	t.Run("non-matching severity not silenced", func(t *testing.T) {
		silenced, _ := engine.IsSilenced("warning", "server-001")
		if silenced {
			t.Error("expected alert NOT to be silenced for non-matching severity")
		}
	})
}

func TestAlertEngine_IsSilenced_EmptyMatchers(t *testing.T) {
	db, err := database.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	engine := NewAlertEngine(db, nil)
	now := time.Now()

	blankMatchers := &model.AlertSilence{
		ID:       "silence-blank",
		Name:     "Blank Matchers",
		Matchers: "",
		StartsAt: now.Add(-1 * time.Hour),
		EndsAt:   now.Add(1 * time.Hour),
	}
	if err := db.Create(blankMatchers).Error; err != nil {
		t.Fatalf("failed to create blank matcher silence: %v", err)
	}

	silenced, _ := engine.IsSilenced("warning", "server-001")
	if !silenced {
		t.Error("expected empty matchers to silence all alerts")
	}
}

func TestAlertEngine_IsSilenced_ServerID(t *testing.T) {
	db, err := database.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	engine := NewAlertEngine(db, nil)
	now := time.Now()

	serverSpecific := &model.AlertSilence{
		ID:       "silence-server",
		Name:     "Server Specific",
		Matchers: `{"server_id": "server-001"}`,
		StartsAt: now.Add(-1 * time.Hour),
		EndsAt:   now.Add(1 * time.Hour),
	}
	if err := db.Create(serverSpecific).Error; err != nil {
		t.Fatalf("failed to create server-specific silence: %v", err)
	}

	silenced, _ := engine.IsSilenced("critical", "server-001")
	if !silenced {
		t.Error("expected alert to be silenced for matching server_id")
	}

	silenced, _ = engine.IsSilenced("critical", "server-002")
	if silenced {
		t.Error("expected alert NOT to be silenced for non-matching server_id")
	}
}

func TestAlertEngine_matchesSilence(t *testing.T) {
	engine := &AlertEngine{}

	t.Run("empty matchers matches everything", func(t *testing.T) {
		silence := model.AlertSilence{Matchers: ""}
		if !engine.matchesSilence(silence, "critical", "server-001") {
			t.Error("expected empty matchers to match all")
		}
	})

	t.Run("invalid JSON matchers", func(t *testing.T) {
		silence := model.AlertSilence{Matchers: "{invalid json}"}
		if engine.matchesSilence(silence, "critical", "server-001") {
			t.Error("expected invalid JSON matchers NOT to match")
		}
	})

	t.Run("severity string match", func(t *testing.T) {
		silence := model.AlertSilence{Matchers: `{"severity": "critical"}`}
		if !engine.matchesSilence(silence, "critical", "server-001") {
			t.Error("expected severity 'critical' to match")
		}
		if engine.matchesSilence(silence, "warning", "server-001") {
			t.Error("expected severity 'warning' NOT to match 'critical'")
		}
	})

	t.Run("severity array match", func(t *testing.T) {
		silence := model.AlertSilence{Matchers: `{"severity": ["critical", "high"]}`}
		if !engine.matchesSilence(silence, "critical", "server-001") {
			t.Error("expected severity 'critical' in array to match")
		}
		if !engine.matchesSilence(silence, "high", "server-001") {
			t.Error("expected severity 'high' in array to match")
		}
		if engine.matchesSilence(silence, "warning", "server-001") {
			t.Error("expected severity 'warning' NOT to match array")
		}
	})

	t.Run("server_id match", func(t *testing.T) {
		silence := model.AlertSilence{Matchers: `{"server_id": "server-001"}`}
		if !engine.matchesSilence(silence, "critical", "server-001") {
			t.Error("expected server_id 'server-001' to match")
		}
		if engine.matchesSilence(silence, "critical", "server-002") {
			t.Error("expected server_id 'server-002' NOT to match 'server-001'")
		}
	})

	t.Run("combined matchers", func(t *testing.T) {
		silence := model.AlertSilence{Matchers: `{"severity": "critical", "server_id": "server-001"}`}
		if !engine.matchesSilence(silence, "critical", "server-001") {
			t.Error("expected combined matchers to match")
		}
		if engine.matchesSilence(silence, "warning", "server-001") {
			t.Error("expected severity mismatch to fail")
		}
		if engine.matchesSilence(silence, "critical", "server-002") {
			t.Error("expected server_id mismatch to fail")
		}
	})
}

func TestAlertEngine_CreateAndListSilences(t *testing.T) {
	db, err := database.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	engine := NewAlertEngine(db, nil)
	now := time.Now()

	silence := &model.AlertSilence{
		Name:      "Test Silence",
		Matchers:  `{"severity": "critical"}`,
		StartsAt:  now,
		EndsAt:    now.Add(24 * time.Hour),
	}

	if err := engine.CreateSilence(silence); err != nil {
		t.Fatalf("CreateSilence failed: %v", err)
	}

	if silence.ID == "" {
		t.Error("expected auto-generated ID")
	}

	silences, err := engine.ListSilences()
	if err != nil {
		t.Fatalf("ListSilences failed: %v", err)
	}

	if len(silences) != 1 {
		t.Fatalf("expected 1 silence, got %d", len(silences))
	}

	if silences[0].Name != "Test Silence" {
		t.Errorf("expected name 'Test Silence', got %q", silences[0].Name)
	}
}

func TestAlertEngine_DeleteSilence(t *testing.T) {
	db, err := database.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	engine := NewAlertEngine(db, nil)
	now := time.Now()

	silence := &model.AlertSilence{
		ID:       "to-delete",
		Name:     "Delete Me",
		Matchers: "",
		StartsAt: now,
		EndsAt:   now.Add(24 * time.Hour),
	}
	if err := db.Create(silence).Error; err != nil {
		t.Fatalf("failed to create silence: %v", err)
	}

	if err := engine.DeleteSilence("to-delete"); err != nil {
		t.Fatalf("DeleteSilence failed: %v", err)
	}

	silences, _ := engine.ListSilences()
	if len(silences) != 0 {
		t.Error("expected no silences after delete")
	}
}

func TestAlertEngine_GroupAlert(t *testing.T) {
	db, err := database.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	engine := NewAlertEngine(db, nil)

	t.Run("create new group", func(t *testing.T) {
		group, err := engine.GroupAlert("rule-001", "server-down", "critical")
		if err != nil {
			t.Fatalf("GroupAlert failed: %v", err)
		}
		if group == nil {
			t.Fatal("expected non-nil group")
		}
		if group.AlertCount != 1 {
			t.Errorf("expected AlertCount=1, got %d", group.AlertCount)
		}
		if group.Status != "firing" {
			t.Errorf("expected Status='firing', got %q", group.Status)
		}
	})

	t.Run("different group key creates new group", func(t *testing.T) {
		group, err := engine.GroupAlert("rule-001", "disk-full", "warning")
		if err != nil {
			t.Fatalf("GroupAlert failed: %v", err)
		}
		if group.AlertCount != 1 {
			t.Errorf("expected AlertCount=1 for new group, got %d", group.AlertCount)
		}
	})
}

func TestAlertEngine_ResolveGroup(t *testing.T) {
	db, err := database.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	engine := NewAlertEngine(db, nil)

	group, _ := engine.GroupAlert("rule-001", "server-down", "critical")
	_ = group

	if err := engine.ResolveGroup("rule-001", "server-down"); err != nil {
		t.Fatalf("ResolveGroup failed: %v", err)
	}

	groups, _ := engine.ListActiveGroups()
	if len(groups) != 0 {
		t.Errorf("expected 0 active groups after resolve, got %d", len(groups))
	}
}

func TestAlertEngine_ListActiveGroups(t *testing.T) {
	db, err := database.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	engine := NewAlertEngine(db, nil)

	engine.GroupAlert("rule-001", "group-1", "critical")
	engine.GroupAlert("rule-001", "group-2", "warning")
	engine.GroupAlert("rule-002", "group-3", "high")

	groups, err := engine.ListActiveGroups()
	if err != nil {
		t.Fatalf("ListActiveGroups failed: %v", err)
	}

	if len(groups) != 3 {
		t.Errorf("expected 3 active groups, got %d", len(groups))
	}
}

func TestAlertEngine_CheckEscalations(t *testing.T) {
	db, err := database.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	engine := NewAlertEngine(db, nil)

	stepsJSON := `[{"after_minutes": 5, "severity": "high", "channels": ["slack"]}]`
	policy := &model.AlertEscalation{
		ID:       "esc-001",
		Name:     "Test Policy",
		Enabled:  true,
		Steps:    stepsJSON,
		RuleIDs:  `["rule-001"]`,
	}
	if err := db.Create(policy).Error; err != nil {
		t.Fatalf("failed to create policy: %v", err)
	}

	group := &model.AlertGroup{
		ID:           "ag-001",
		GroupKey:     "test-group",
		RuleID:       "rule-001",
		Severity:     "warning",
		AlertCount:   5,
		FirstAlertAt: time.Now().Add(-10 * time.Minute),
		LastAlertAt:  time.Now(),
		Status:       "firing",
	}
	if err := db.Create(group).Error; err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	engine.CheckEscalations(context.Background())
}

func TestAlertEngine_CreateAndListEscalations(t *testing.T) {
	db, err := database.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	engine := NewAlertEngine(db, nil)

	esc := &model.AlertEscalation{
		Name:    "Test Escalation",
		Enabled: true,
		Steps:   `[{"after_minutes": 5, "severity": "high", "channels": ["email"]}]`,
	}

	if err := engine.CreateEscalation(esc); err != nil {
		t.Fatalf("CreateEscalation failed: %v", err)
	}

	if esc.ID == "" {
		t.Error("expected auto-generated ID")
	}

	escs, err := engine.ListEscalations()
	if err != nil {
		t.Fatalf("ListEscalations failed: %v", err)
	}

	if len(escs) != 1 {
		t.Fatalf("expected 1 escalation, got %d", len(escs))
	}
}

func TestAlertEngine_DeleteEscalation(t *testing.T) {
	db, err := database.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	engine := NewAlertEngine(db, nil)

	esc := &model.AlertEscalation{
		ID:      "esc-to-delete",
		Name:    "Delete Me",
		Enabled: true,
		Steps:   `[]`,
	}
	if err := db.Create(esc).Error; err != nil {
		t.Fatalf("failed to create escalation: %v", err)
	}

	if err := engine.DeleteEscalation("esc-to-delete"); err != nil {
		t.Fatalf("DeleteEscalation failed: %v", err)
	}

	escs, _ := engine.ListEscalations()
	if len(escs) != 0 {
		t.Error("expected no escalations after delete")
	}
}

func TestAlertEngine_StartAndStop(t *testing.T) {
	db, err := database.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}

	engine := NewAlertEngine(db, nil)

	engine.Start()
	engine.Stop()
}

func TestAlertEngine_stringSliceContains(t *testing.T) {
	engine := &AlertEngine{}

	tests := []struct {
		slice   []string
		item    string
		matches bool
	}{
		{[]string{"a", "b", "c"}, "b", true},
		{[]string{"a", "b", "c"}, "d", false},
		{[]string{}, "a", false},
		{[]string{"only"}, "only", true},
	}

	for _, tt := range tests {
		result := engine.stringSliceContains(tt.slice, tt.item)
		if result != tt.matches {
			t.Errorf("stringSliceContains(%v, %q) = %v, want %v",
				tt.slice, tt.item, result, tt.matches)
		}
	}
}

func TestEscalationStep_Structure(t *testing.T) {
	step := EscalationStep{
		AfterMinutes: 5,
		Severity:    "high",
		Channels:    []string{"slack", "email"},
		Message:     "Escalated alert",
	}

	if step.AfterMinutes != 5 {
		t.Errorf("expected AfterMinutes=5, got %d", step.AfterMinutes)
	}
	if step.Severity != "high" {
		t.Errorf("expected Severity='high', got %q", step.Severity)
	}
	if len(step.Channels) != 2 {
		t.Errorf("expected 2 channels, got %d", len(step.Channels))
	}
}

func TestAlertEngine_NilDB_IsSilenced(t *testing.T) {
	engine := NewAlertEngine(nil, nil)

	silenced, _ := engine.IsSilenced("critical", "server-001")
	if silenced {
		t.Error("expected nil DB to return not silenced")
	}
}

func TestAlertEngine_NilDB_GroupAlert(t *testing.T) {
	engine := NewAlertEngine(nil, nil)

	group, err := engine.GroupAlert("rule-001", "key", "critical")
	if err != nil {
		t.Error("expected nil GroupAlert to return nil group without error")
	}
	if group != nil {
		t.Error("expected nil GroupAlert to return nil group")
	}
}

func TestAlertEngine_NilDB_ResolveGroup(t *testing.T) {
	engine := NewAlertEngine(nil, nil)

	err := engine.ResolveGroup("rule-001", "key")
	if err != nil {
		t.Error("expected nil ResolveGroup to return nil error")
	}
}

func TestAlertEngine_CheckEscalations_DisabledPolicy(t *testing.T) {
	db, err := database.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	engine := NewAlertEngine(db, nil)

	policy := &model.AlertEscalation{
		ID:      "esc-disabled",
		Name:    "Disabled Policy",
		Enabled: false,
		Steps:   `[]`,
	}
	if err := db.Create(policy).Error; err != nil {
		t.Fatalf("failed to create policy: %v", err)
	}

	group := &model.AlertGroup{
		ID:           "ag-disabled",
		GroupKey:     "test-group",
		RuleID:       "rule-001",
		Severity:     "warning",
		FirstAlertAt: time.Now().Add(-10 * time.Minute),
		Status:       "firing",
	}
	if err := db.Create(group).Error; err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	engine.CheckEscalations(context.Background())
}

func TestAlertSilence_Model(t *testing.T) {
	silence := model.AlertSilence{
		ID:        "silence-001",
		Name:      "Weekend maintenance",
		Matchers:  `{"severity": "critical", "server_id": "server-001"}`,
		StartsAt: time.Now(),
		EndsAt:    time.Now().Add(48 * time.Hour),
	}

	if silence.ID == "" {
		t.Error("expected non-empty ID")
	}
	if silence.Name != "Weekend maintenance" {
		t.Errorf("expected name 'Weekend maintenance', got %q", silence.Name)
	}

	var matchers map[string]interface{}
	if err := json.Unmarshal([]byte(silence.Matchers), &matchers); err != nil {
		t.Fatalf("invalid matchers JSON: %v", err)
	}
	if matchers["severity"] != "critical" {
		t.Error("expected severity 'critical' in matchers")
	}
}

func TestAlertEscalation_Model(t *testing.T) {
	esc := model.AlertEscalation{
		ID:       "esc-001",
		Name:     "Critical escalation",
		Enabled:  true,
		Steps:    `[{"after_minutes": 5, "severity": "high", "channels": ["slack"]}]`,
		RuleIDs:  `["rule-001", "rule-002"]`,
	}

	if esc.ID != "esc-001" {
		t.Errorf("expected ID 'esc-001', got %q", esc.ID)
	}
	if !esc.Enabled {
		t.Error("expected Enabled=true")
	}

	var steps []EscalationStep
	if err := json.Unmarshal([]byte(esc.Steps), &steps); err != nil {
		t.Fatalf("invalid steps JSON: %v", err)
	}
	if len(steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(steps))
	}
}

func TestAlertGroup_Model(t *testing.T) {
	group := model.AlertGroup{
		ID:           "ag-001",
		GroupKey:     "fingerprint-hash",
		RuleID:       "rule-001",
		Severity:     "critical",
		AlertCount:   10,
		FirstAlertAt: time.Now().Add(-1 * time.Hour),
		LastAlertAt:  time.Now(),
		Status:       "firing",
	}

	if group.ID != "ag-001" {
		t.Errorf("expected ID 'ag-001', got %q", group.ID)
	}
	if group.Status != "firing" {
		t.Errorf("expected Status 'firing', got %q", group.Status)
	}
	if group.AlertCount != 10 {
		t.Errorf("expected AlertCount 10, got %d", group.AlertCount)
	}
}

func TestPreflightErrorCode(t *testing.T) {
	codes := []PreflightErrorCode{
		PreflightTCPUnreachable,
		PreflightSSHAuthFailed,
		PreflightDockerUnavailable,
		PreflightPortInUse,
		PreflightUnknownError,
	}

	expected := []string{
		"SSH_TCP_UNREACHABLE",
		"SSH_AUTH_FAILED",
		"REMOTE_DOCKER_UNAVAILABLE",
		"PORT_ALREADY_IN_USE",
		"PREFLIGHT_UNKNOWN_ERROR",
	}

	for i, code := range codes {
		if string(code) != expected[i] {
			t.Errorf("expected code %q, got %q", expected[i], code)
		}
	}
}

func TestPreflightResult_Structure(t *testing.T) {
	result := &PreflightResult{
		Passed: true,
		Checks: []PreflightCheck{
			{
				Name:       "TCP Check",
				Passed:     true,
				Message:    "Connection successful",
				Suggestion: "",
				Category:   "network",
				Severity:   "info",
			},
		},
	}

	if !result.Passed {
		t.Error("expected Passed=true")
	}
	if len(result.Checks) != 1 {
		t.Errorf("expected 1 check, got %d", len(result.Checks))
	}
	if result.Checks[0].Name != "TCP Check" {
		t.Errorf("expected check name 'TCP Check', got %q", result.Checks[0].Name)
	}
}

func TestPreflightCheck_Structure(t *testing.T) {
	check := PreflightCheck{
		Name:       "Docker Check",
		Passed:     false,
		Message:    "Docker is not running",
		Suggestion: "Start Docker daemon",
		Category:   "docker",
		Severity:   "error",
	}

	if check.Passed {
		t.Error("expected Passed=false")
	}
	if check.Message == "" {
		t.Error("expected non-empty message")
	}
	if check.Suggestion == "" {
		t.Error("expected non-empty suggestion")
	}
}
