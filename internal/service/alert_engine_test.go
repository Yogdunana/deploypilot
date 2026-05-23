package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/model"
)

func TestAlertEngine_IsSilenced_NilDB(t *testing.T) {
	engine := NewAlertEngine(nil, nil)
	silenced, name := engine.IsSilenced("critical", "server-1")
	if silenced {
		t.Error("expected not silenced with nil DB")
	}
	if name != "" {
		t.Errorf("expected empty name, got %q", name)
	}
}

func TestAlertEngine_IsSilenced_NoActiveSilences(t *testing.T) {
	db := setupTestDB(t)
	engine := NewAlertEngine(db, nil)

	silenced, name := engine.IsSilenced("warning", "server-1")
	if silenced {
		t.Error("expected not silenced when no silences exist")
	}
	if name != "" {
		t.Errorf("expected empty name, got %q", name)
	}
}

func TestAlertEngine_IsSilenced_EmptyMatchers(t *testing.T) {
	db := setupTestDB(t)
	engine := NewAlertEngine(db, nil)

	now := time.Now()
	silence := &model.AlertSilence{
		ID:        "silence-1",
		Name:      "Mute All",
		Matchers:  "",
		StartsAt:  now.Add(-1 * time.Hour),
		EndsAt:    now.Add(1 * time.Hour),
		CreatedBy: "admin",
	}
	if err := engine.CreateSilence(silence); err != nil {
		t.Fatalf("CreateSilence failed: %v", err)
	}

	silenced, name := engine.IsSilenced("critical", "any-server")
	if !silenced {
		t.Error("expected silenced with empty matchers")
	}
	if name != "Mute All" {
		t.Errorf("expected name 'Mute All', got %q", name)
	}
}

func TestAlertEngine_IsSilenced_SeverityMatch(t *testing.T) {
	db := setupTestDB(t)
	engine := NewAlertEngine(db, nil)

	now := time.Now()
	matchers := map[string]interface{}{"severity": "critical"}
	matchersJSON, _ := json.Marshal(matchers)

	silence := &model.AlertSilence{
		ID:        "silence-2",
		Name:      "Mute Critical",
		Matchers:  string(matchersJSON),
		StartsAt:  now.Add(-1 * time.Hour),
		EndsAt:    now.Add(1 * time.Hour),
		CreatedBy: "admin",
	}
	if err := engine.CreateSilence(silence); err != nil {
		t.Fatalf("CreateSilence failed: %v", err)
	}

	silenced, _ := engine.IsSilenced("critical", "server-1")
	if !silenced {
		t.Error("expected silenced for critical severity")
	}

	silenced, _ = engine.IsSilenced("warning", "server-1")
	if silenced {
		t.Error("expected not silenced for warning severity")
	}
}

func TestAlertEngine_IsSilenced_SeverityArrayMatch(t *testing.T) {
	db := setupTestDB(t)
	engine := NewAlertEngine(db, nil)

	now := time.Now()
	matchers := map[string]interface{}{"severity": []interface{}{"critical", "high"}}
	matchersJSON, _ := json.Marshal(matchers)

	silence := &model.AlertSilence{
		ID:        "silence-3",
		Name:      "Mute Critical and High",
		Matchers:  string(matchersJSON),
		StartsAt:  now.Add(-1 * time.Hour),
		EndsAt:    now.Add(1 * time.Hour),
		CreatedBy: "admin",
	}
	if err := engine.CreateSilence(silence); err != nil {
		t.Fatalf("CreateSilence failed: %v", err)
	}

	silenced, _ := engine.IsSilenced("critical", "server-1")
	if !silenced {
		t.Error("expected silenced for critical")
	}

	silenced, _ = engine.IsSilenced("high", "server-1")
	if !silenced {
		t.Error("expected silenced for high")
	}

	silenced, _ = engine.IsSilenced("warning", "server-1")
	if silenced {
		t.Error("expected not silenced for warning")
	}
}

func TestAlertEngine_IsSilenced_ServerIDMatch(t *testing.T) {
	db := setupTestDB(t)
	engine := NewAlertEngine(db, nil)

	now := time.Now()
	matchers := map[string]interface{}{"server_id": "target-server"}
	matchersJSON, _ := json.Marshal(matchers)

	silence := &model.AlertSilence{
		ID:        "silence-4",
		Name:      "Mute Target Server",
		Matchers:  string(matchersJSON),
		StartsAt:  now.Add(-1 * time.Hour),
		EndsAt:    now.Add(1 * time.Hour),
		CreatedBy: "admin",
	}
	if err := engine.CreateSilence(silence); err != nil {
		t.Fatalf("CreateSilence failed: %v", err)
	}

	silenced, _ := engine.IsSilenced("critical", "target-server")
	if !silenced {
		t.Error("expected silenced for target-server")
	}

	silenced, _ = engine.IsSilenced("critical", "other-server")
	if silenced {
		t.Error("expected not silenced for other-server")
	}
}

func TestAlertEngine_IsSilenced_ExpiredSilence(t *testing.T) {
	db := setupTestDB(t)
	engine := NewAlertEngine(db, nil)

	now := time.Now()
	silence := &model.AlertSilence{
		ID:        "silence-5",
		Name:      "Expired Silence",
		Matchers:  "",
		StartsAt:  now.Add(-2 * time.Hour),
		EndsAt:    now.Add(-1 * time.Hour),
		CreatedBy: "admin",
	}
	if err := engine.CreateSilence(silence); err != nil {
		t.Fatalf("CreateSilence failed: %v", err)
	}

	silenced, _ := engine.IsSilenced("critical", "any-server")
	if silenced {
		t.Error("expected not silenced for expired silence")
	}
}

func TestAlertEngine_ListSilences(t *testing.T) {
	db := setupTestDB(t)
	engine := NewAlertEngine(db, nil)

	now := time.Now()
	silences := []*model.AlertSilence{
		{ID: "list-1", Name: "Active", StartsAt: now.Add(-1 * time.Hour), EndsAt: now.Add(1 * time.Hour)},
		{ID: "list-2", Name: "Future", StartsAt: now.Add(1 * time.Hour), EndsAt: now.Add(2 * time.Hour)},
		{ID: "list-3", Name: "Expired", StartsAt: now.Add(-2 * time.Hour), EndsAt: now.Add(-1 * time.Hour)},
	}

	for _, s := range silences {
		if err := engine.CreateSilence(s); err != nil {
			t.Fatalf("CreateSilence failed: %v", err)
		}
	}

	list, err := engine.ListSilences()
	if err != nil {
		t.Fatalf("ListSilences failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 active/upcoming silences, got %d", len(list))
	}
}

func TestAlertEngine_DeleteSilence(t *testing.T) {
	db := setupTestDB(t)
	engine := NewAlertEngine(db, nil)

	now := time.Now()
	silence := &model.AlertSilence{
		ID:       "delete-1",
		Name:     "To Delete",
		Matchers: "",
		StartsAt: now.Add(-1 * time.Hour),
		EndsAt:   now.Add(1 * time.Hour),
	}
	if err := engine.CreateSilence(silence); err != nil {
		t.Fatalf("CreateSilence failed: %v", err)
	}

	if err := engine.DeleteSilence("delete-1"); err != nil {
		t.Fatalf("DeleteSilence failed: %v", err)
	}

	list, _ := engine.ListSilences()
	for _, s := range list {
		if s.ID == "delete-1" {
			t.Error("silence should have been deleted")
		}
	}
}

func TestAlertEngine_GroupAlert_CreateNew(t *testing.T) {
	db := setupTestDB(t)
	engine := NewAlertEngine(db, nil)

	group, err := engine.GroupAlert("rule-1", "server:cpu-high", "critical")
	if err != nil {
		t.Fatalf("GroupAlert failed: %v", err)
	}
	if group == nil {
		t.Fatal("expected non-nil group")
	}
	if group.RuleID != "rule-1" {
		t.Errorf("expected rule_id=rule-1, got %q", group.RuleID)
	}
	if group.Severity != "critical" {
		t.Errorf("expected severity=critical, got %q", group.Severity)
	}
	if group.AlertCount != 1 {
		t.Errorf("expected alert_count=1, got %d", group.AlertCount)
	}
	if group.Status != "firing" {
		t.Errorf("expected status=firing, got %q", group.Status)
	}
}

func TestAlertEngine_GroupAlert_UpdateExisting(t *testing.T) {
	db := setupTestDB(t)
	engine := NewAlertEngine(db, nil)

	engine.GroupAlert("rule-1", "server:mem-high", "warning")

	group, err := engine.GroupAlert("rule-1", "server:mem-high", "warning")
	if err != nil {
		t.Fatalf("GroupAlert failed: %v", err)
	}

	groups, _ := engine.ListActiveGroups()
	if len(groups) != 1 {
		t.Errorf("expected 1 group, got %d", len(groups))
	}
	if groups[0].AlertCount != 2 {
		t.Errorf("expected alert_count=2, got %d", groups[0].AlertCount)
	}

	_ = group
}

func TestAlertEngine_GroupAlert_NilDB(t *testing.T) {
	engine := NewAlertEngine(nil, nil)
	group, err := engine.GroupAlert("rule-1", "test", "warning")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if group != nil {
		t.Error("expected nil group with nil DB")
	}
}

func TestAlertEngine_ResolveGroup(t *testing.T) {
	db := setupTestDB(t)
	engine := NewAlertEngine(db, nil)

	engine.GroupAlert("rule-1", "server:disk-full", "critical")

	if err := engine.ResolveGroup("rule-1", "server:disk-full"); err != nil {
		t.Fatalf("ResolveGroup failed: %v", err)
	}

	groups, _ := engine.ListActiveGroups()
	for _, g := range groups {
		if g.GroupKey == "server:disk-full" {
			t.Error("expected resolved group not in active groups")
		}
	}
}

func TestAlertEngine_ResolveGroup_NilDB(t *testing.T) {
	engine := NewAlertEngine(nil, nil)
	if err := engine.ResolveGroup("rule-1", "test"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAlertEngine_ListActiveGroups(t *testing.T) {
	db := setupTestDB(t)
	engine := NewAlertEngine(db, nil)

	engine.GroupAlert("rule-1", "active-1", "warning")
	engine.GroupAlert("rule-2", "active-2", "critical")
	engine.GroupAlert("rule-3", "resolved-1", "high")
	engine.ResolveGroup("rule-3", "resolved-1")

	groups, err := engine.ListActiveGroups()
	if err != nil {
		t.Fatalf("ListActiveGroups failed: %v", err)
	}

	if len(groups) != 2 {
		t.Errorf("expected 2 active groups, got %d", len(groups))
	}
}

func TestAlertEngine_CreateEscalation(t *testing.T) {
	db := setupTestDB(t)
	engine := NewAlertEngine(db, nil)

	esc := &model.AlertEscalation{
		ID:      "esc-1",
		Name:    "Test Escalation",
		Steps:   `[{"after_minutes":30,"severity":"critical","channels":["telegram"]}]`,
		Enabled: true,
	}

	if err := engine.CreateEscalation(esc); err != nil {
		t.Fatalf("CreateEscalation failed: %v", err)
	}

	list, err := engine.ListEscalations()
	if err != nil {
		t.Fatalf("ListEscalations failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 escalation, got %d", len(list))
	}
}

func TestAlertEngine_DeleteEscalation(t *testing.T) {
	db := setupTestDB(t)
	engine := NewAlertEngine(db, nil)

	esc := &model.AlertEscalation{
		ID:      "esc-del",
		Name:    "To Delete",
		Steps:   `[]`,
		Enabled: true,
	}
	engine.CreateEscalation(esc)

	if err := engine.DeleteEscalation("esc-del"); err != nil {
		t.Fatalf("DeleteEscalation failed: %v", err)
	}

	list, _ := engine.ListEscalations()
	for _, e := range list {
		if e.ID == "esc-del" {
			t.Error("escalation should have been deleted")
		}
	}
}

func TestAlertEngine_CheckEscalations_NilDB(t *testing.T) {
	engine := NewAlertEngine(nil, nil)
	engine.CheckEscalations(context.Background())
}

func TestAlertEngine_StartStop(t *testing.T) {
	db := setupTestDB(t)
	engine := NewAlertEngine(db, nil)

	engine.Start()
	engine.Stop()
}

func TestAlertEngine_stringSliceContains(t *testing.T) {
	db := setupTestDB(t)
	engine := NewAlertEngine(db, nil)

	slice := []string{"a", "b", "c"}
	if !engine.stringSliceContains(slice, "b") {
		t.Error("expected contains b")
	}
	if engine.stringSliceContains(slice, "d") {
		t.Error("expected not contains d")
	}
	if engine.stringSliceContains(nil, "a") {
		t.Error("expected not contains in nil slice")
	}
}
