package service

import (
	"context"
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAlertEngineTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	db.AutoMigrate(&model.AlertSilence{}, &model.AlertEscalation{}, &model.AlertGroup{})
	return db
}

func newTestAlertEngine(db *gorm.DB) *AlertEngine {
	return NewAlertEngine(db, nil)
}

// ========== Silence Period Tests ==========

func TestIsSilenced_NoSilences(t *testing.T) {
	db := setupAlertEngineTestDB(t)
	engine := newTestAlertEngine(db)

	silenced, name := engine.IsSilenced("critical", "server-1")
	if silenced {
		t.Error("expected not silenced when no silence rules exist")
	}
	if name != "" {
		t.Errorf("expected empty name, got %q", name)
	}
}

func TestIsSilenced_ActiveSilence(t *testing.T) {
	db := setupAlertEngineTestDB(t)
	engine := newTestAlertEngine(db)

	silence := &model.AlertSilence{
		ID:      "silence-1",
		Name:    "Test Silence",
		StartsAt: time.Now().Add(-1 * time.Hour),
		EndsAt:   time.Now().Add(1 * time.Hour),
		Matchers: "", // empty matchers = match all
	}
	if err := db.Create(silence).Error; err != nil {
		t.Fatalf("failed to create silence: %v", err)
	}

	silenced, name := engine.IsSilenced("critical", "server-1")
	if !silenced {
		t.Error("expected silenced when active silence exists")
	}
	if name != "Test Silence" {
		t.Errorf("expected name 'Test Silence', got %q", name)
	}
}

func TestIsSilenced_InactiveSilence(t *testing.T) {
	db := setupAlertEngineTestDB(t)
	engine := newTestAlertEngine(db)

	silence := &model.AlertSilence{
		ID:      "silence-1",
		Name:    "Expired Silence",
		StartsAt: time.Now().Add(-2 * time.Hour),
		EndsAt:   time.Now().Add(-1 * time.Hour),
		Matchers: "",
	}
	if err := db.Create(silence).Error; err != nil {
		t.Fatalf("failed to create silence: %v", err)
	}

	silenced, _ := engine.IsSilenced("critical", "server-1")
	if silenced {
		t.Error("expected not silenced for expired silence")
	}
}

func TestIsSilenced_SeverityMatcher(t *testing.T) {
	db := setupAlertEngineTestDB(t)
	engine := newTestAlertEngine(db)

	silence := &model.AlertSilence{
		ID:      "silence-1",
		Name:    "Critical Only",
		StartsAt: time.Now().Add(-1 * time.Hour),
		EndsAt:   time.Now().Add(1 * time.Hour),
		Matchers: `{"severity": "critical"}`,
	}
	if err := db.Create(silence).Error; err != nil {
		t.Fatalf("failed to create silence: %v", err)
	}

	// Should match critical
	silenced, _ := engine.IsSilenced("critical", "server-1")
	if !silenced {
		t.Error("expected silenced for critical severity")
	}

	// Should not match warning
	silenced, _ = engine.IsSilenced("warning", "server-1")
	if silenced {
		t.Error("expected not silenced for warning severity")
	}
}

func TestIsSilenced_SeverityListMatcher(t *testing.T) {
	db := setupAlertEngineTestDB(t)
	engine := newTestAlertEngine(db)

	silence := &model.AlertSilence{
		ID:      "silence-1",
		Name:    "Multiple Severities",
		StartsAt: time.Now().Add(-1 * time.Hour),
		EndsAt:   time.Now().Add(1 * time.Hour),
		Matchers: `{"severity": ["critical", "warning"]}`,
	}
	if err := db.Create(silence).Error; err != nil {
		t.Fatalf("failed to create silence: %v", err)
	}

	silenced, _ := engine.IsSilenced("critical", "server-1")
	if !silenced {
		t.Error("expected silenced for critical severity in list")
	}

	silenced, _ = engine.IsSilenced("warning", "server-1")
	if !silenced {
		t.Error("expected silenced for warning severity in list")
	}

	silenced, _ = engine.IsSilenced("info", "server-1")
	if silenced {
		t.Error("expected not silenced for info severity not in list")
	}
}

func TestIsSilenced_ServerIDMatcher(t *testing.T) {
	db := setupAlertEngineTestDB(t)
	engine := newTestAlertEngine(db)

	silence := &model.AlertSilence{
		ID:      "silence-1",
		Name:    "Server Specific",
		StartsAt: time.Now().Add(-1 * time.Hour),
		EndsAt:   time.Now().Add(1 * time.Hour),
		Matchers: `{"server_id": "server-1"}`,
	}
	if err := db.Create(silence).Error; err != nil {
		t.Fatalf("failed to create silence: %v", err)
	}

	silenced, _ := engine.IsSilenced("critical", "server-1")
	if !silenced {
		t.Error("expected silenced for matching server")
	}

	silenced, _ = engine.IsSilenced("critical", "server-2")
	if silenced {
		t.Error("expected not silenced for non-matching server")
	}
}

func TestIsSilenced_InvalidMatchersJSON(t *testing.T) {
	db := setupAlertEngineTestDB(t)
	engine := newTestAlertEngine(db)

	silence := &model.AlertSilence{
		ID:      "silence-1",
		Name:    "Invalid Matchers",
		StartsAt: time.Now().Add(-1 * time.Hour),
		EndsAt:   time.Now().Add(1 * time.Hour),
		Matchers: `not-valid-json`,
	}
	if err := db.Create(silence).Error; err != nil {
		t.Fatalf("failed to create silence: %v", err)
	}

	silenced, _ := engine.IsSilenced("critical", "server-1")
	if silenced {
		t.Error("expected not silenced when matchers JSON is invalid")
	}
}

func TestIsSilenced_NilDB(t *testing.T) {
	engine := NewAlertEngine(nil, nil)
	silenced, name := engine.IsSilenced("critical", "server-1")
	if silenced {
		t.Error("expected not silenced when DB is nil")
	}
	if name != "" {
		t.Errorf("expected empty name, got %q", name)
	}
}

func TestCreateSilence(t *testing.T) {
	db := setupAlertEngineTestDB(t)
	engine := newTestAlertEngine(db)

	silence := &model.AlertSilence{
		Name:    "Test",
		StartsAt: time.Now().Add(-1 * time.Hour),
		EndsAt:   time.Now().Add(1 * time.Hour),
	}

	if err := engine.CreateSilence(silence); err != nil {
		t.Fatalf("CreateSilence failed: %v", err)
	}
	if silence.ID == "" {
		t.Error("expected auto-generated ID")
	}

	var count int64
	db.Model(&model.AlertSilence{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 silence, got %d", count)
	}
}

func TestListSilences(t *testing.T) {
	db := setupAlertEngineTestDB(t)
	engine := newTestAlertEngine(db)

	db.Create(&model.AlertSilence{
		ID:      "s1",
		Name:    "Active",
		StartsAt: time.Now().Add(-1 * time.Hour),
		EndsAt:   time.Now().Add(1 * time.Hour),
	})
	db.Create(&model.AlertSilence{
		ID:      "s2",
		Name:    "Expired",
		StartsAt: time.Now().Add(-2 * time.Hour),
		EndsAt:   time.Now().Add(-1 * time.Hour),
	})

	silences, err := engine.ListSilences()
	if err != nil {
		t.Fatalf("ListSilences failed: %v", err)
	}
	if len(silences) != 1 {
		t.Errorf("expected 1 active silence, got %d", len(silences))
	}
	if silences[0].Name != "Active" {
		t.Errorf("expected 'Active', got %q", silences[0].Name)
	}
}

func TestDeleteSilence(t *testing.T) {
	db := setupAlertEngineTestDB(t)
	engine := newTestAlertEngine(db)

	db.Create(&model.AlertSilence{ID: "s1", Name: "ToDelete"})

	if err := engine.DeleteSilence("s1"); err != nil {
		t.Fatalf("DeleteSilence failed: %v", err)
	}

	var count int64
	db.Model(&model.AlertSilence{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 silences, got %d", count)
	}
}

// ========== Escalation Tests ==========

func TestCreateEscalation(t *testing.T) {
	db := setupAlertEngineTestDB(t)
	engine := newTestAlertEngine(db)

	esc := &model.AlertEscalation{
		Name:    "Test Escalation",
		Enabled: true,
		Steps:   `[{"after_minutes": 5, "severity": "critical", "channels": ["slack"]}]`,
	}

	if err := engine.CreateEscalation(esc); err != nil {
		t.Fatalf("CreateEscalation failed: %v", err)
	}
	if esc.ID == "" {
		t.Error("expected auto-generated ID")
	}
}

func TestListEscalations(t *testing.T) {
	db := setupAlertEngineTestDB(t)
	engine := newTestAlertEngine(db)

	db.Create(&model.AlertEscalation{ID: "e1", Name: "First", Enabled: true})
	db.Create(&model.AlertEscalation{ID: "e2", Name: "Second", Enabled: false})

	escalations, err := engine.ListEscalations()
	if err != nil {
		t.Fatalf("ListEscalations failed: %v", err)
	}
	if len(escalations) != 2 {
		t.Errorf("expected 2 escalations, got %d", len(escalations))
	}
}

func TestDeleteEscalation(t *testing.T) {
	db := setupAlertEngineTestDB(t)
	engine := newTestAlertEngine(db)

	db.Create(&model.AlertEscalation{ID: "e1", Name: "ToDelete"})

	if err := engine.DeleteEscalation("e1"); err != nil {
		t.Fatalf("DeleteEscalation failed: %v", err)
	}

	var count int64
	db.Model(&model.AlertEscalation{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 escalations, got %d", count)
	}
}

func TestCheckEscalations_NilDB(t *testing.T) {
	engine := NewAlertEngine(nil, nil)
	// Should not panic
	engine.CheckEscalations(context.Background())
}

func TestCheckEscalations_NoPolicies(t *testing.T) {
	db := setupAlertEngineTestDB(t)
	engine := newTestAlertEngine(db)

	// Should not panic
	engine.CheckEscalations(context.Background())
}

func TestCheckEscalations_InvalidStepsJSON(t *testing.T) {
	db := setupAlertEngineTestDB(t)
	engine := newTestAlertEngine(db)

	db.Create(&model.AlertEscalation{
		ID:      "e1",
		Name:    "Bad Escalation",
		Enabled: true,
		Steps:   `not-valid-json`,
	})

	// Should not panic on invalid JSON
	engine.CheckEscalations(context.Background())
}

// ========== Alert Grouping Tests ==========

func TestGroupAlert_NewGroup(t *testing.T) {
	db := setupAlertEngineTestDB(t)
	engine := newTestAlertEngine(db)

	group, err := engine.GroupAlert("rule-1", "group-1", "critical")
	if err != nil {
		t.Fatalf("GroupAlert failed: %v", err)
	}
	if group == nil {
		t.Error("expected non-nil group")
	}
	if group.GroupKey != "group-1" {
		t.Errorf("expected group key 'group-1', got %q", group.GroupKey)
	}
	if group.AlertCount != 1 {
		t.Errorf("expected alert count 1, got %d", group.AlertCount)
	}
	if group.Status != "firing" {
		t.Errorf("expected status 'firing', got %q", group.Status)
	}
}

func TestGroupAlert_ExistingGroup(t *testing.T) {
	db := setupAlertEngineTestDB(t)
	engine := newTestAlertEngine(db)

	// Create initial group
	group1, _ := engine.GroupAlert("rule-1", "group-1", "critical")

	// Call again with same key
	group2, err := engine.GroupAlert("rule-1", "group-1", "critical")
	if err != nil {
		t.Fatalf("GroupAlert failed: %v", err)
	}

	if group1.ID != group2.ID {
		t.Error("expected same group ID")
	}

	var count int64
	db.Model(&model.AlertGroup{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 group, got %d", count)
	}
}

func TestGroupAlert_NilDB(t *testing.T) {
	engine := NewAlertEngine(nil, nil)
	group, err := engine.GroupAlert("rule-1", "group-1", "critical")
	if group != nil {
		t.Error("expected nil group when DB is nil")
	}
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestResolveGroup(t *testing.T) {
	db := setupAlertEngineTestDB(t)
	engine := newTestAlertEngine(db)

	// Create a group first
	engine.GroupAlert("rule-1", "group-1", "critical")

	if err := engine.ResolveGroup("rule-1", "group-1"); err != nil {
		t.Fatalf("ResolveGroup failed: %v", err)
	}

	var group model.AlertGroup
	db.Where("group_key = ?", "group-1").First(&group)
	if group.Status != "resolved" {
		t.Errorf("expected status 'resolved', got %q", group.Status)
	}
}

func TestResolveGroup_NilDB(t *testing.T) {
	engine := NewAlertEngine(nil, nil)
	err := engine.ResolveGroup("rule-1", "group-1")
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestListActiveGroups(t *testing.T) {
	db := setupAlertEngineTestDB(t)
	engine := newTestAlertEngine(db)

	engine.GroupAlert("rule-1", "group-1", "critical")
	engine.GroupAlert("rule-2", "group-2", "warning")
	engine.ResolveGroup("rule-2", "group-2")

	groups, err := engine.ListActiveGroups()
	if err != nil {
		t.Fatalf("ListActiveGroups failed: %v", err)
	}
	if len(groups) != 1 {
		t.Errorf("expected 1 active group, got %d", len(groups))
	}
	if groups[0].GroupKey != "group-1" {
		t.Errorf("expected group-1, got %q", groups[0].GroupKey)
	}
}

// ========== Background Worker Tests ==========

func TestAlertEngine_StartStop(t *testing.T) {
	db := setupAlertEngineTestDB(t)
	engine := newTestAlertEngine(db)

	// Start the engine
	engine.Start()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Stop should not block or panic
	engine.Stop()
}

// ========== Helper Tests ==========

func TestStringSliceContains(t *testing.T) {
	engine := NewAlertEngine(nil, nil)

	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{"contains", []string{"a", "b", "c"}, "b", true},
		{"not contains", []string{"a", "b", "c"}, "d", false},
		{"empty slice", []string{}, "a", false},
		{"nil slice", nil, "a", false},
		{"empty item", []string{"a", "b"}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.stringSliceContains(tt.slice, tt.item)
			if got != tt.expected {
				t.Errorf("stringSliceContains(%v, %q) = %v, want %v", tt.slice, tt.item, got, tt.expected)
			}
		})
	}
}
