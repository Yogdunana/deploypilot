package service

import (
	"context"
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupDegradationTestDB(t *testing.T) (*gorm.DB, *Bridge) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&model.DegradationAudit{}, &model.TrialPeriod{}, &model.FeatureFlag{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	bridge := NewBridge(db, nil, nil, nil)
	return db, bridge
}

func TestGetDegradationStatus_NoLicenseNoTrial(t *testing.T) {
	_, bridge := setupDegradationTestDB(t)
	ctx := context.Background()

	result, err := bridge.GetDegradationStatus(ctx)
	if err != nil {
		t.Fatalf("GetDegradationStatus failed: %v", err)
	}

	ds, ok := result.(DegradationStatus)
	if !ok {
		t.Fatal("expected DegradationStatus")
	}
	if ds.Level != DegradationNone {
		t.Errorf("expected level none, got %s", ds.Level)
	}
	if ds.LicenseStatus != "none" {
		t.Errorf("expected license_status none, got %s", ds.LicenseStatus)
	}
	if ds.Tier != "community" {
		t.Errorf("expected tier community, got %s", ds.Tier)
	}
}

func TestGetDegradationStatus_ExpiredTrial(t *testing.T) {
	db, bridge := setupDegradationTestDB(t)
	ctx := context.Background()

	// Create expired trial
	now := time.Now()
	trial := model.TrialPeriod{
		ID:           "trial-deg-1",
		MachineID:    GenerateMachineID(),
		Status:       model.TrialExpired,
		StartedAt:    now.AddDate(0, 0, -60),
		ExpiresAt:    now.AddDate(0, 0, -30),
		OriginalDays: 30,
	}
	if err := db.Create(&trial).Error; err != nil {
		t.Fatalf("failed to create trial: %v", err)
	}

	result, err := bridge.GetDegradationStatus(ctx)
	if err != nil {
		t.Fatalf("GetDegradationStatus failed: %v", err)
	}

	ds := result.(DegradationStatus)
	if ds.Level != DegradationReadOnly {
		t.Errorf("expected level readonly, got %s", ds.Level)
	}
	if ds.TrialStatus != "expired" {
		t.Errorf("expected trial_status expired, got %s", ds.TrialStatus)
	}
	if ds.ReadOnlyReason == "" {
		t.Error("expected read_only_reason to be set")
	}
}

func TestGetDegradationStatus_ActiveTrial(t *testing.T) {
	_, bridge := setupDegradationTestDB(t)
	ctx := context.Background()

	// Create active trial
	if err := bridge.InitTrialPeriod(ctx); err != nil {
		t.Fatalf("InitTrialPeriod failed: %v", err)
	}

	result, err := bridge.GetDegradationStatus(ctx)
	if err != nil {
		t.Fatalf("GetDegradationStatus failed: %v", err)
	}

	ds := result.(DegradationStatus)
	if ds.Level != DegradationNone {
		t.Errorf("expected level none for active trial, got %s", ds.Level)
	}
	if ds.TrialStatus != "active" {
		t.Errorf("expected trial_status active, got %s", ds.TrialStatus)
	}
}

func TestCheckReadOnly_NoRestriction(t *testing.T) {
	_, bridge := setupDegradationTestDB(t)
	ctx := context.Background()

	err := bridge.CheckReadOnly(ctx)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestCheckReadOnly_ExpiredTrial(t *testing.T) {
	db, bridge := setupDegradationTestDB(t)
	ctx := context.Background()

	now := time.Now()
	trial := model.TrialPeriod{
		ID:           "trial-ro-1",
		MachineID:    GenerateMachineID(),
		Status:       model.TrialExpired,
		StartedAt:    now.AddDate(0, 0, -60),
		ExpiresAt:    now.AddDate(0, 0, -30),
		OriginalDays: 30,
	}
	if err := db.Create(&trial).Error; err != nil {
		t.Fatalf("failed to create trial: %v", err)
	}

	err := bridge.CheckReadOnly(ctx)
	if err == nil {
		t.Error("expected error for expired trial")
	}
}

func TestAuditDegradation(t *testing.T) {
	_, bridge := setupDegradationTestDB(t)
	ctx := context.Background()

	bridge.AuditDegradation(ctx, model.DegradationActionFeatureGated, "ssl", "tier too low", "tenant-1", "user-1", "127.0.0.1")

	var audits []model.DegradationAudit
	if err := bridge.DB.Find(&audits).Error; err != nil {
		t.Fatalf("failed to query audits: %v", err)
	}
	if len(audits) != 1 {
		t.Fatalf("expected 1 audit, got %d", len(audits))
	}
	if audits[0].Action != model.DegradationActionFeatureGated {
		t.Errorf("expected action feature_gated, got %s", audits[0].Action)
	}
	if audits[0].Feature != "ssl" {
		t.Errorf("expected feature ssl, got %s", audits[0].Feature)
	}
}

func TestListDegradationAudits(t *testing.T) {
	_, bridge := setupDegradationTestDB(t)
	ctx := context.Background()

	// Create some audits
	bridge.AuditDegradation(ctx, model.DegradationActionFeatureGated, "ssl", "tier", "t1", "u1", "1.1.1.1")
	bridge.AuditDegradation(ctx, model.DegradationActionReadOnly, "POST /api/apps", "readonly", "t1", "u1", "1.1.1.1")

	result, err := bridge.ListDegradationAudits(ctx, 10)
	if err != nil {
		t.Fatalf("ListDegradationAudits failed: %v", err)
	}

	resultMap := result.(map[string]interface{})
	total, ok := resultMap["total"].(int)
	if !ok || total != 2 {
		t.Errorf("expected total 2, got %v", total)
	}
}

func TestExportDegradationSummary(t *testing.T) {
	_, bridge := setupDegradationTestDB(t)
	ctx := context.Background()

	result, err := bridge.ExportDegradationSummary(ctx)
	if err != nil {
		t.Fatalf("ExportDegradationSummary failed: %v", err)
	}

	summary, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	if _, ok := summary["apps"]; !ok {
		t.Error("expected apps key in summary")
	}
	if _, ok := summary["servers"]; !ok {
		t.Error("expected servers key in summary")
	}
	if _, ok := summary["exported_at"]; !ok {
		t.Error("expected exported_at key in summary")
	}
}
