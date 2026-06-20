package service

import (
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/database"
	"github.com/Yogdunana/deploypilot/internal/model"
)

func setupAlertEngineTest(t *testing.T) (*AlertEngine, *NotificationService) {
	t.Helper()
	db, err := database.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}

	notifySvc := &NotificationService{}
	engine := NewAlertEngine(db, notifySvc)

	return engine, notifySvc
}

func TestNewAlertEngine(t *testing.T) {
	engine, _ := setupAlertEngineTest(t)
	if engine == nil {
		t.Fatal("NewAlertEngine returned nil")
	}
	if engine.stopCh == nil {
		t.Fatal("expected non-nil stopCh")
	}
}

func TestAlertEngine_IsSilenced_NoDB(t *testing.T) {
	engine := &AlertEngine{db: nil}
	silenced, _ := engine.IsSilenced("critical", "server-1")
	if silenced {
		t.Error("expected not silenced when DB is nil")
	}
}

func TestAlertEngine_IsSilenced_NoSilences(t *testing.T) {
	engine, _ := setupAlertEngineTest(t)
	silenced, _ := engine.IsSilenced("critical", "server-1")
	if silenced {
		t.Error("expected not silenced when no silences exist")
	}
}

func TestAlertEngine_IsSilenced_ActiveSilence(t *testing.T) {
	engine, _ := setupAlertEngineTest(t)

	now := time.Now()
	silence := model.AlertSilence{
		ID:      "silence-1",
		Name:    "Test Silence",
		StartsAt: now.Add(-1 * time.Hour),
		EndsAt:   now.Add(1 * time.Hour),
		Matchers: "",
	}
	engine.db.Create(&silence)

	silenced, name := engine.IsSilenced("critical", "server-1")
	if !silenced {
		t.Error("expected silenced")
	}
	if name != "Test Silence" {
		t.Errorf("expected silence name 'Test Silence', got %q", name)
	}
}

func TestAlertEngine_IsSilenced_SilenceExpired(t *testing.T) {
	engine, _ := setupAlertEngineTest(t)

	now := time.Now()
	silence := model.AlertSilence{
		ID:      "silence-expired",
		Name:    "Expired Silence",
		StartsAt: now.Add(-2 * time.Hour),
		EndsAt:   now.Add(-1 * time.Hour),
		Matchers: "",
	}
	engine.db.Create(&silence)

	silenced, _ := engine.IsSilenced("critical", "server-1")
	if silenced {
		t.Error("expected not silenced for expired silence")
	}
}

func TestAlertEngine_IsSilenced_NotStarted(t *testing.T) {
	engine, _ := setupAlertEngineTest(t)

	now := time.Now()
	silence := model.AlertSilence{
		ID:      "silence-future",
		Name:    "Future Silence",
		StartsAt: now.Add(1 * time.Hour),
		EndsAt:   now.Add(2 * time.Hour),
		Matchers: "",
	}
	engine.db.Create(&silence)

	silenced, _ := engine.IsSilenced("critical", "server-1")
	if silenced {
		t.Error("expected not silenced for future silence")
	}
}

func TestAlertEngine_IsSilenced_WithMatchers(t *testing.T) {
	engine, _ := setupAlertEngineTest(t)

	now := time.Now()
	silence := model.AlertSilence{
		ID:      "silence-matcher",
		Name:    "Matcher Silence",
		StartsAt: now.Add(-1 * time.Hour),
		EndsAt:   now.Add(1 * time.Hour),
		Matchers: `{"severity": "critical", "server_id": "server-1"}`,
	}
	engine.db.Create(&silence)

	silenced, _ := engine.IsSilenced("critical", "server-1")
	if !silenced {
		t.Error("expected silenced for matching severity and server")
	}

	silenced, _ = engine.IsSilenced("warning", "server-1")
	if silenced {
		t.Error("expected not silenced for non-matching severity")
	}

	silenced, _ = engine.IsSilenced("critical", "server-2")
	if silenced {
		t.Error("expected not silenced for non-matching server")
	}
}

func TestAlertEngine_IsSilenced_SeverityList(t *testing.T) {
	engine, _ := setupAlertEngineTest(t)

	now := time.Now()
	silence := model.AlertSilence{
		ID:      "silence-list",
		Name:    "List Silence",
		StartsAt: now.Add(-1 * time.Hour),
		EndsAt:   now.Add(1 * time.Hour),
		Matchers: `{"severity": ["critical", "warning"]}`,
	}
	engine.db.Create(&silence)

	silenced, _ := engine.IsSilenced("critical", "server-1")
	if !silenced {
		t.Error("expected silenced for critical")
	}

	silenced, _ = engine.IsSilenced("warning", "server-1")
	if !silenced {
		t.Error("expected silenced for warning")
	}

	silenced, _ = engine.IsSilenced("info", "server-1")
	if silenced {
		t.Error("expected not silenced for info")
	}
}

func TestAlertEngine_CreateSilence(t *testing.T) {
	engine, _ := setupAlertEngineTest(t)

	silence := &model.AlertSilence{
		Name:    "New Silence",
		StartsAt: time.Now(),
		EndsAt:   time.Now().Add(1 * time.Hour),
	}

	err := engine.CreateSilence(silence)
	if err != nil {
		t.Fatalf("CreateSilence failed: %v", err)
	}

	if silence.ID == "" {
		t.Error("expected ID to be set")
	}
}

func TestAlertEngine_CreateSilence_WithID(t *testing.T) {
	engine, _ := setupAlertEngineTest(t)

	silence := &model.AlertSilence{
		ID:      "custom-id",
		Name:    "Custom Silence",
		StartsAt: time.Now(),
		EndsAt:   time.Now().Add(1 * time.Hour),
	}

	err := engine.CreateSilence(silence)
	if err != nil {
		t.Fatalf("CreateSilence failed: %v", err)
	}

	if silence.ID != "custom-id" {
		t.Errorf("expected ID 'custom-id', got %q", silence.ID)
	}
}

func TestAlertEngine_ListSilences(t *testing.T) {
	engine, _ := setupAlertEngineTest(t)

	now := time.Now()
	engine.db.Create(&model.AlertSilence{
		ID:      "silence-1",
		Name:    "Silence 1",
		StartsAt: now.Add(-1 * time.Hour),
		EndsAt:   now.Add(1 * time.Hour),
	})
	engine.db.Create(&model.AlertSilence{
		ID:      "silence-2",
		Name:    "Silence 2",
		StartsAt: now.Add(-2 * time.Hour),
		EndsAt:   now.Add(-1 * time.Hour),
	})

	silences, err := engine.ListSilences()
	if err != nil {
		t.Fatalf("ListSilences failed: %v", err)
	}

	if len(silences) != 1 {
		t.Errorf("expected 1 active silence, got %d", len(silences))
	}
}

func TestAlertEngine_DeleteSilence(t *testing.T) {
	engine, _ := setupAlertEngineTest(t)

	now := time.Now()
	silence := model.AlertSilence{
		ID:      "silence-delete",
		Name:    "Delete Silence",
		StartsAt: now.Add(-1 * time.Hour),
		EndsAt:   now.Add(1 * time.Hour),
	}
	engine.db.Create(&silence)

	err := engine.DeleteSilence("silence-delete")
	if err != nil {
		t.Fatalf("DeleteSilence failed: %v", err)
	}

	var count int64
	engine.db.Model(&model.AlertSilence{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 silences, got %d", count)
	}
}

func TestAlertEngine_DeleteSilence_NotFound(t *testing.T) {
	engine, _ := setupAlertEngineTest(t)

	err := engine.DeleteSilence("nonexistent")
	if err != nil {
		t.Fatalf("DeleteSilence should not error for nonexistent silence: %v", err)
	}
}

func TestAlertEngine_CreateEscalation(t *testing.T) {
	engine, _ := setupAlertEngineTest(t)

	escalation := &model.AlertEscalation{
		Name: "Test Escalation",
		Steps: `[{"after_minutes": 5, "severity": "critical", "channels": ["webhook"]}]`,
		Enabled: true,
	}

	err := engine.CreateEscalation(escalation)
	if err != nil {
		t.Fatalf("CreateEscalation failed: %v", err)
	}

	if escalation.ID == "" {
		t.Error("expected ID to be set")
	}
}

func TestAlertEngine_CreateEscalation_WithID(t *testing.T) {
	engine, _ := setupAlertEngineTest(t)

	escalation := &model.AlertEscalation{
		ID: "esc-custom",
		Name: "Custom Escalation",
		Steps: `[{"after_minutes": 10, "severity": "warning"}]`,
		Enabled: true,
	}

	err := engine.CreateEscalation(escalation)
	if err != nil {
		t.Fatalf("CreateEscalation failed: %v", err)
	}

	if escalation.ID != "esc-custom" {
		t.Errorf("expected ID 'esc-custom', got %q", escalation.ID)
	}
}

func TestAlertEngine_ListEscalations(t *testing.T) {
	engine, _ := setupAlertEngineTest(t)

	engine.db.Create(&model.AlertEscalation{
		ID: "esc-1",
		Name: "Escalation 1",
		Steps: `[{"after_minutes": 5}]`,
	})
	engine.db.Create(&model.AlertEscalation{
		ID: "esc-2",
		Name: "Escalation 2",
		Steps: `[{"after_minutes": 10}]`,
	})

	escalations, err := engine.ListEscalations()
	if err != nil {
		t.Fatalf("ListEscalations failed: %v", err)
	}

	if len(escalations) != 2 {
		t.Errorf("expected 2 escalations, got %d", len(escalations))
	}
}

func TestAlertEngine_DeleteEscalation(t *testing.T) {
	engine, _ := setupAlertEngineTest(t)

	escalation := model.AlertEscalation{
		ID: "esc-delete",
		Name: "Delete Escalation",
		Steps: `[{"after_minutes": 5}]`,
	}
	engine.db.Create(&escalation)

	err := engine.DeleteEscalation("esc-delete")
	if err != nil {
		t.Fatalf("DeleteEscalation failed: %v", err)
	}

	var count int64
	engine.db.Model(&model.AlertEscalation{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 escalations, got %d", count)
	}
}

func TestAlertEngine_DeleteEscalation_NotFound(t *testing.T) {
	engine, _ := setupAlertEngineTest(t)

	err := engine.DeleteEscalation("nonexistent")
	if err != nil {
		t.Fatalf("DeleteEscalation should not error for nonexistent escalation: %v", err)
	}
}

func TestAlertEngine_GroupAlert(t *testing.T) {
	engine, _ := setupAlertEngineTest(t)

	group, err := engine.GroupAlert("rule-1", "group-key-1", "critical")
	if err != nil {
		t.Fatalf("GroupAlert failed: %v", err)
	}

	if group == nil {
		t.Fatal("expected non-nil group")
	}
	if group.GroupKey != "group-key-1" {
		t.Errorf("expected group key 'group-key-1', got %q", group.GroupKey)
	}
	if group.Severity != "critical" {
		t.Errorf("expected severity 'critical', got %q", group.Severity)
	}
	if group.AlertCount != 1 {
		t.Errorf("expected alert count 1, got %d", group.AlertCount)
	}
}

func TestAlertEngine_GroupAlert_Duplicate(t *testing.T) {
	engine, _ := setupAlertEngineTest(t)

	_, err := engine.GroupAlert("rule-1", "group-key-dup", "critical")
	if err != nil {
		t.Fatalf("GroupAlert failed: %v", err)
	}

	group, err := engine.GroupAlert("rule-1", "group-key-dup", "critical")
	if err != nil {
		t.Fatalf("GroupAlert failed on duplicate: %v", err)
	}

	if group.AlertCount != 2 {
		t.Errorf("expected alert count 2 for duplicate, got %d", group.AlertCount)
	}
}

func TestAlertEngine_GroupAlert_NoDB(t *testing.T) {
	engine := NewAlertEngine(nil, nil)

	group, err := engine.GroupAlert("rule-1", "group-key", "critical")
	if err != nil {
		t.Fatalf("GroupAlert should not error with nil DB: %v", err)
	}
	if group != nil {
		t.Error("expected nil group when DB is nil")
	}
}

func TestAlertEngine_ResolveGroup(t *testing.T) {
	engine, _ := setupAlertEngineTest(t)

	engine.GroupAlert("rule-1", "group-resolve", "critical")

	err := engine.ResolveGroup("rule-1", "group-resolve")
	if err != nil {
		t.Fatalf("ResolveGroup failed: %v", err)
	}

	var group model.AlertGroup
	engine.db.Where("group_key = ?", "group-resolve").First(&group)
	if group.Status != "resolved" {
		t.Errorf("expected status 'resolved', got %q", group.Status)
	}
}

func TestAlertEngine_ResolveGroup_NoDB(t *testing.T) {
	engine := NewAlertEngine(nil, nil)

	err := engine.ResolveGroup("rule-1", "group-key")
	if err != nil {
		t.Fatalf("ResolveGroup should not error with nil DB: %v", err)
	}
}

func TestAlertEngine_ResolveGroup_NotFound(t *testing.T) {
	engine, _ := setupAlertEngineTest(t)

	err := engine.ResolveGroup("rule-nonexistent", "group-nonexistent")
	if err != nil {
		t.Fatalf("ResolveGroup should not error for nonexistent group: %v", err)
	}
}

func TestAlertEngine_ListActiveGroups(t *testing.T) {
	engine, _ := setupAlertEngineTest(t)

	engine.GroupAlert("rule-1", "group-active-1", "critical")
	engine.GroupAlert("rule-1", "group-active-2", "warning")
	engine.GroupAlert("rule-1", "group-resolved", "info")
	engine.ResolveGroup("rule-1", "group-resolved")

	groups, err := engine.ListActiveGroups()
	if err != nil {
		t.Fatalf("ListActiveGroups failed: %v", err)
	}

	if len(groups) != 2 {
		t.Errorf("expected 2 active groups, got %d", len(groups))
	}
}

func TestAlertEngine_StartStop(t *testing.T) {
	engine, _ := setupAlertEngineTest(t)

	engine.Start()
	time.Sleep(100 * time.Millisecond)
	engine.Stop()
}

func TestAlertEngine_StringSliceContains(t *testing.T) {
	engine, _ := setupAlertEngineTest(t)

	slice := []string{"a", "b", "c"}

	if !engine.stringSliceContains(slice, "b") {
		t.Error("expected 'b' to be found")
	}

	if engine.stringSliceContains(slice, "d") {
		t.Error("expected 'd' to not be found")
	}

	if engine.stringSliceContains(nil, "a") {
		t.Error("expected nil slice to return false")
	}

	if engine.stringSliceContains([]string{}, "a") {
		t.Error("expected empty slice to return false")
	}
}

func TestAlertEngine_MatchesSilence_EmptyMatchers(t *testing.T) {
	engine, _ := setupAlertEngineTest(t)

	silence := model.AlertSilence{
		Matchers: "",
	}

	result := engine.matchesSilence(silence, "critical", "server-1")
	if !result {
		t.Error("expected empty matchers to match all")
	}
}

func TestAlertEngine_MatchesSilence_InvalidJSON(t *testing.T) {
	engine, _ := setupAlertEngineTest(t)

	silence := model.AlertSilence{
		Matchers: "not-valid-json",
	}

	result := engine.matchesSilence(silence, "critical", "server-1")
	if result {
		t.Error("expected invalid JSON matchers to return false")
	}
}

func TestAlertEngine_CheckEscalations_NoDB(t *testing.T) {
	engine := NewAlertEngine(nil, nil)

	engine.CheckEscalations(nil)
}

func TestAlertEngine_CheckEscalations_NoPolicies(t *testing.T) {
	engine, _ := setupAlertEngineTest(t)

	engine.CheckEscalations(nil)
}

func TestAlertEngine_CheckEscalations_NoGroups(t *testing.T) {
	engine, _ := setupAlertEngineTest(t)

	engine.db.Create(&model.AlertEscalation{
		ID: "esc-no-groups",
		Name: "No Groups",
		Steps: `[{"after_minutes": 0, "severity": "critical"}]`,
		Enabled: true,
	})

	engine.CheckEscalations(nil)
}

func TestAlertEngine_CheckEscalations_InvalidSteps(t *testing.T) {
	engine, _ := setupAlertEngineTest(t)

	engine.db.Create(&model.AlertEscalation{
		ID: "esc-invalid",
		Name: "Invalid Steps",
		Steps: "not-valid-json",
		Enabled: true,
	})

	engine.GroupAlert("rule-esc", "group-esc", "warning")

	engine.CheckEscalations(nil)
}