package service

import (
	"context"
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTrialTestDB(t *testing.T) (*gorm.DB, *Bridge) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&model.TrialPeriod{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	bridge := NewBridge(db, nil, nil, nil)
	return db, bridge
}

func TestGenerateMachineID(t *testing.T) {
	id1 := GenerateMachineID()
	id2 := GenerateMachineID()

	if id1 == "" {
		t.Error("machine ID should not be empty")
	}
	if len(id1) != 32 {
		t.Errorf("machine ID should be 32 chars, got %d", len(id1))
	}
	// Same machine should produce same ID
	if id1 != id2 {
		t.Error("same machine should produce same ID")
	}
}

func TestInitTrialPeriod_FirstRun(t *testing.T) {
	_, bridge := setupTrialTestDB(t)
	ctx := context.Background()

	if err := bridge.InitTrialPeriod(ctx); err != nil {
		t.Fatalf("InitTrialPeriod failed: %v", err)
	}

	var trial model.TrialPeriod
	if err := bridge.DB.First(&trial).Error; err != nil {
		t.Fatalf("failed to find trial: %v", err)
	}

	if trial.Status != model.TrialActive {
		t.Errorf("expected status active, got %s", trial.Status)
	}
	if trial.OriginalDays != 30 {
		t.Errorf("expected original_days 30, got %d", trial.OriginalDays)
	}
	if trial.MachineID == "" {
		t.Error("machine_id should not be empty")
	}
}

func TestInitTrialPeriod_AlreadyExists(t *testing.T) {
	_, bridge := setupTrialTestDB(t)
	ctx := context.Background()

	// First init
	if err := bridge.InitTrialPeriod(ctx); err != nil {
		t.Fatalf("first init failed: %v", err)
	}

	var count1 int64
	bridge.DB.Model(&model.TrialPeriod{}).Count(&count1)

	// Second init should not create duplicate
	if err := bridge.InitTrialPeriod(ctx); err != nil {
		t.Fatalf("second init failed: %v", err)
	}

	var count2 int64
	bridge.DB.Model(&model.TrialPeriod{}).Count(&count2)

	if count2 != count1 {
		t.Errorf("expected count to remain %d, got %d", count1, count2)
	}
}

func TestInitTrialPeriod_ExpiresOldTrial(t *testing.T) {
	db, bridge := setupTrialTestDB(t)
	ctx := context.Background()

	// Create an expired trial
	now := time.Now()
	trial := model.TrialPeriod{
		ID:           "trial-expired",
		MachineID:    GenerateMachineID(),
		Status:       model.TrialActive,
		StartedAt:    now.AddDate(0, 0, -60),
		ExpiresAt:    now.AddDate(0, 0, -30),
		OriginalDays: 30,
	}
	if err := db.Create(&trial).Error; err != nil {
		t.Fatalf("failed to create expired trial: %v", err)
	}

	if err := bridge.InitTrialPeriod(ctx); err != nil {
		t.Fatalf("InitTrialPeriod failed: %v", err)
	}

	var updated model.TrialPeriod
	db.Where("machine_id = ?", trial.MachineID).First(&updated)

	if updated.Status != model.TrialExpired {
		t.Errorf("expected status expired, got %s", updated.Status)
	}
}

func TestGetTrialStatus_NoTrial(t *testing.T) {
	_, bridge := setupTrialTestDB(t)
	ctx := context.Background()

	result, err := bridge.GetTrialStatus(ctx)
	if err != nil {
		t.Fatalf("GetTrialStatus failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	if resultMap["status"] != "none" {
		t.Errorf("expected status 'none', got %v", resultMap["status"])
	}
}

func TestGetTrialStatus_Active(t *testing.T) {
	_, bridge := setupTrialTestDB(t)
	ctx := context.Background()

	if err := bridge.InitTrialPeriod(ctx); err != nil {
		t.Fatalf("InitTrialPeriod failed: %v", err)
	}

	result, err := bridge.GetTrialStatus(ctx)
	if err != nil {
		t.Fatalf("GetTrialStatus failed: %v", err)
	}

	resultMap := result.(map[string]interface{})
	if resultMap["is_active"] != true {
		t.Error("expected is_active=true")
	}
	if resultMap["is_expired"] != false {
		t.Error("expected is_expired=false")
	}
	daysRemaining, ok := resultMap["days_remaining"].(int)
	if !ok || daysRemaining <= 0 {
		t.Errorf("expected days_remaining > 0, got %d", daysRemaining)
	}
}

func TestExtendTrial(t *testing.T) {
	_, bridge := setupTrialTestDB(t)
	ctx := context.Background()

	if err := bridge.InitTrialPeriod(ctx); err != nil {
		t.Fatalf("InitTrialPeriod failed: %v", err)
	}

	machineID := GenerateMachineID()
	result, err := bridge.ExtendTrial(ctx, machineID, 15, "evaluation extension")
	if err != nil {
		t.Fatalf("ExtendTrial failed: %v", err)
	}

	resultMap := result.(map[string]interface{})
	if resultMap["status"] != model.TrialExtended {
		t.Errorf("expected status extended, got %v", resultMap["status"])
	}

	// Verify the extension in DB
	var trial model.TrialPeriod
	bridge.DB.Where("machine_id = ?", machineID).First(&trial)
	if trial.ExtendedDays != 15 {
		t.Errorf("expected extended_days 15, got %d", trial.ExtendedDays)
	}
}

func TestExtendTrial_InvalidDays(t *testing.T) {
	_, bridge := setupTrialTestDB(t)
	ctx := context.Background()

	_, err := bridge.ExtendTrial(ctx, "some-machine", 0, "test")
	if err == nil {
		t.Error("expected error for 0 days")
	}

	_, err = bridge.ExtendTrial(ctx, "some-machine", 400, "test")
	if err == nil {
		t.Error("expected error for 400 days")
	}
}

func TestExtendTrial_NotFound(t *testing.T) {
	_, bridge := setupTrialTestDB(t)
	ctx := context.Background()

	_, err := bridge.ExtendTrial(ctx, "nonexistent-machine", 10, "test")
	if err == nil {
		t.Error("expected error for nonexistent machine")
	}
}

func TestConvertTrial(t *testing.T) {
	_, bridge := setupTrialTestDB(t)
	ctx := context.Background()

	if err := bridge.InitTrialPeriod(ctx); err != nil {
		t.Fatalf("InitTrialPeriod failed: %v", err)
	}

	if err := bridge.ConvertTrial(ctx); err != nil {
		t.Fatalf("ConvertTrial failed: %v", err)
	}

	var trial model.TrialPeriod
	bridge.DB.First(&trial)
	if trial.Status != model.TrialConverted {
		t.Errorf("expected status converted, got %s", trial.Status)
	}
	if trial.ConvertedAt == nil {
		t.Error("expected converted_at to be set")
	}
}

func TestIsTrialActive(t *testing.T) {
	_, bridge := setupTrialTestDB(t)
	ctx := context.Background()

	// No trial = not active
	if bridge.IsTrialActive(ctx) {
		t.Error("expected not active without trial")
	}

	// Create trial
	if err := bridge.InitTrialPeriod(ctx); err != nil {
		t.Fatalf("InitTrialPeriod failed: %v", err)
	}

	if !bridge.IsTrialActive(ctx) {
		t.Error("expected active after init")
	}
}

func TestIsTrialExpired(t *testing.T) {
	db, bridge := setupTrialTestDB(t)
	ctx := context.Background()

	// No trial = not expired
	if bridge.IsTrialExpired(ctx) {
		t.Error("expected not expired without trial")
	}

	// Create expired trial
	now := time.Now()
	trial := model.TrialPeriod{
		ID:           "trial-expired-2",
		MachineID:    GenerateMachineID(),
		Status:       model.TrialExpired,
		StartedAt:    now.AddDate(0, 0, -60),
		ExpiresAt:    now.AddDate(0, 0, -30),
		OriginalDays: 30,
	}
	if err := db.Create(&trial).Error; err != nil {
		t.Fatalf("failed to create expired trial: %v", err)
	}

	if !bridge.IsTrialExpired(ctx) {
		t.Error("expected expired")
	}
}

func TestCheckTrialOrLicense_NoTrialNoLicense(t *testing.T) {
	_, bridge := setupTrialTestDB(t)
	ctx := context.Background()

	// No trial, no license = community mode (allowed)
	err := bridge.CheckTrialOrLicense(ctx)
	if err != nil {
		t.Errorf("expected no error in community mode, got: %v", err)
	}
}

func TestCheckTrialOrLicense_ActiveTrial(t *testing.T) {
	_, bridge := setupTrialTestDB(t)
	ctx := context.Background()

	if err := bridge.InitTrialPeriod(ctx); err != nil {
		t.Fatalf("InitTrialPeriod failed: %v", err)
	}

	err := bridge.CheckTrialOrLicense(ctx)
	if err != nil {
		t.Errorf("expected no error with active trial, got: %v", err)
	}
}

func TestCheckTrialOrLicense_ExpiredTrial(t *testing.T) {
	db, bridge := setupTrialTestDB(t)
	ctx := context.Background()

	now := time.Now()
	trial := model.TrialPeriod{
		ID:           "trial-expired-3",
		MachineID:    GenerateMachineID(),
		Status:       model.TrialExpired,
		StartedAt:    now.AddDate(0, 0, -60),
		ExpiresAt:    now.AddDate(0, 0, -30),
		OriginalDays: 30,
	}
	if err := db.Create(&trial).Error; err != nil {
		t.Fatalf("failed to create expired trial: %v", err)
	}

	err := bridge.CheckTrialOrLicense(ctx)
	if err == nil {
		t.Error("expected error for expired trial")
	}
}

func TestListTrialPeriods(t *testing.T) {
	_, bridge := setupTrialTestDB(t)
	ctx := context.Background()

	if err := bridge.InitTrialPeriod(ctx); err != nil {
		t.Fatalf("InitTrialPeriod failed: %v", err)
	}

	result, err := bridge.ListTrialPeriods(ctx)
	if err != nil {
		t.Fatalf("ListTrialPeriods failed: %v", err)
	}

	resultMap := result.(map[string]interface{})
	total, ok := resultMap["total"].(int)
	if !ok || total == 0 {
		t.Errorf("expected total > 0, got %v", total)
	}
}
