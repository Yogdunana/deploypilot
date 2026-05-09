package service

import (
	"context"
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/license"
	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupFeatureFlagTestDB(t *testing.T) (*gorm.DB, *Bridge) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&model.FeatureFlag{}, &model.FeatureFlagOverride{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	bridge := NewBridge(db, nil, nil, nil)
	return db, bridge
}

func TestInitFeatureFlags(t *testing.T) {
	db, bridge := setupFeatureFlagTestDB(t)
	ctx := context.Background()

	// First call should seed flags
	if err := bridge.InitFeatureFlags(ctx); err != nil {
		t.Fatalf("InitFeatureFlags failed: %v", err)
	}

	var count int64
	db.Model(&model.FeatureFlag{}).Count(&count)
	if count == 0 {
		t.Error("expected feature flags to be seeded")
	}

	// Second call should skip (already seeded)
	if err := bridge.InitFeatureFlags(ctx); err != nil {
		t.Fatalf("InitFeatureFlags second call failed: %v", err)
	}

	var count2 int64
	db.Model(&model.FeatureFlag{}).Count(&count2)
	if count2 != count {
		t.Errorf("expected count to remain %d, got %d", count, count2)
	}
}

func TestEvaluateFeature_NoLicense(t *testing.T) {
	_, bridge := setupFeatureFlagTestDB(t)
	ctx := context.Background()

	// Without license engine, all features should be enabled (community defaults)
	enabled, err := bridge.EvaluateFeature(ctx, "ssl", "")
	if err != nil {
		t.Fatalf("EvaluateFeature failed: %v", err)
	}
	if !enabled {
		t.Error("expected feature to be enabled without license engine")
	}
}

func TestEvaluateFeature_WithFlag(t *testing.T) {
	db, bridge := setupFeatureFlagTestDB(t)
	ctx := context.Background()

	// Create a disabled flag
	flag := model.FeatureFlag{
		ID:             "ff-test-1",
		Key:            "test_feature",
		Name:           "Test Feature",
		Description:    "A test feature",
		Status:         model.FeatureFlagDisabled,
		DefaultEnabled: false,
		Category:       "test",
	}
	if err := db.Create(&flag).Error; err != nil {
		t.Fatalf("failed to create flag: %v", err)
	}

	// Invalidate cache to force DB load
	bridge.featureFlagCache.Invalidate()

	enabled, err := bridge.EvaluateFeature(ctx, "test_feature", "")
	if err != nil {
		t.Fatalf("EvaluateFeature failed: %v", err)
	}
	if enabled {
		t.Error("expected disabled feature flag to return false")
	}
}

func TestEvaluateFeature_WithOverride(t *testing.T) {
	db, bridge := setupFeatureFlagTestDB(t)
	ctx := context.Background()

	// Create a disabled flag
	flag := model.FeatureFlag{
		ID:             "ff-test-2",
		Key:            "test_override",
		Name:           "Test Override",
		Description:    "A test feature for override",
		Status:         model.FeatureFlagDisabled,
		DefaultEnabled: false,
		Category:       "test",
	}
	if err := db.Create(&flag).Error; err != nil {
		t.Fatalf("failed to create flag: %v", err)
	}

	// Create an override that enables it
	override := model.FeatureFlagOverride{
		ID:           "ffo-test-1",
		FlagKey:      "test_override",
		TenantID:     "tenant-123",
		Enabled:      true,
		Reason:       "Testing override",
		OverriddenBy: "admin",
	}
	if err := db.Create(&override).Error; err != nil {
		t.Fatalf("failed to create override: %v", err)
	}

	// Invalidate cache
	bridge.featureFlagCache.Invalidate()

	// Without tenant: should be disabled
	enabled, err := bridge.EvaluateFeature(ctx, "test_override", "")
	if err != nil {
		t.Fatalf("EvaluateFeature failed: %v", err)
	}
	if enabled {
		t.Error("expected feature to be disabled without tenant override")
	}

	// With matching tenant: should be enabled via override
	enabled, err = bridge.EvaluateFeature(ctx, "test_override", "tenant-123")
	if err != nil {
		t.Fatalf("EvaluateFeature failed: %v", err)
	}
	if !enabled {
		t.Error("expected feature to be enabled with tenant override")
	}
}

func TestListFeatureFlags(t *testing.T) {
	_, bridge := setupFeatureFlagTestDB(t)
	ctx := context.Background()

	// Seed flags
	if err := bridge.InitFeatureFlags(ctx); err != nil {
		t.Fatalf("InitFeatureFlags failed: %v", err)
	}

	result, err := bridge.ListFeatureFlags(ctx)
	if err != nil {
		t.Fatalf("ListFeatureFlags failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	total, ok := resultMap["total"].(int)
	if !ok || total == 0 {
		t.Errorf("expected total > 0, got %v", total)
	}
}

func TestUpdateFeatureFlag(t *testing.T) {
	db, bridge := setupFeatureFlagTestDB(t)
	ctx := context.Background()

	// Create a flag
	flag := model.FeatureFlag{
		ID:             "ff-update-1",
		Key:            "test_update",
		Name:           "Test Update",
		Description:    "Original description",
		Status:         model.FeatureFlagEnabled,
		DefaultEnabled: true,
		Category:       "test",
	}
	if err := db.Create(&flag).Error; err != nil {
		t.Fatalf("failed to create flag: %v", err)
	}

	// Update it
	result, err := bridge.UpdateFeatureFlag(ctx, "test_update", map[string]interface{}{
		"description": "Updated description",
		"category":    "updated_category",
	})
	if err != nil {
		t.Fatalf("UpdateFeatureFlag failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	if resultMap["description"] != "Updated description" {
		t.Errorf("expected updated description, got %v", resultMap["description"])
	}
	if resultMap["category"] != "updated_category" {
		t.Errorf("expected updated category, got %v", resultMap["category"])
	}
}

func TestSetFeatureFlagOverride(t *testing.T) {
	db, bridge := setupFeatureFlagTestDB(t)
	ctx := context.Background()

	// Create a flag
	flag := model.FeatureFlag{
		ID:             "ff-override-1",
		Key:            "test_set_override",
		Name:           "Test Set Override",
		Description:    "Test",
		Status:         model.FeatureFlagEnabled,
		DefaultEnabled: true,
		Category:       "test",
	}
	if err := db.Create(&flag).Error; err != nil {
		t.Fatalf("failed to create flag: %v", err)
	}

	// Set override
	result, err := bridge.SetFeatureFlagOverride(ctx, "test_set_override", "tenant-456", false, "test reason", "admin")
	if err != nil {
		t.Fatalf("SetFeatureFlagOverride failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	if resultMap["enabled"] != false {
		t.Error("expected enabled=false")
	}

	// Verify override is applied
	enabled, err := bridge.EvaluateFeature(ctx, "test_set_override", "tenant-456")
	if err != nil {
		t.Fatalf("EvaluateFeature failed: %v", err)
	}
	if enabled {
		t.Error("expected feature to be disabled by override")
	}
}

func TestDeleteFeatureFlagOverride(t *testing.T) {
	db, bridge := setupFeatureFlagTestDB(t)
	ctx := context.Background()

	// Create flag + override
	flag := model.FeatureFlag{
		ID:             "ff-del-1",
		Key:            "test_delete_override",
		Name:           "Test Delete Override",
		Description:    "Test",
		Status:         model.FeatureFlagEnabled,
		DefaultEnabled: true,
		Category:       "test",
	}
	if err := db.Create(&flag).Error; err != nil {
		t.Fatalf("failed to create flag: %v", err)
	}
	override := model.FeatureFlagOverride{
		ID:           "ffo-del-1",
		FlagKey:      "test_delete_override",
		TenantID:     "tenant-789",
		Enabled:      false,
		Reason:       "test",
		OverriddenBy: "admin",
	}
	if err := db.Create(&override).Error; err != nil {
		t.Fatalf("failed to create override: %v", err)
	}

	// Delete override
	_, err := bridge.DeleteFeatureFlagOverride(ctx, "test_delete_override", "tenant-789")
	if err != nil {
		t.Fatalf("DeleteFeatureFlagOverride failed: %v", err)
	}

	// Verify override is gone
	enabled, err := bridge.EvaluateFeature(ctx, "test_delete_override", "tenant-789")
	if err != nil {
		t.Fatalf("EvaluateFeature failed: %v", err)
	}
	if !enabled {
		t.Error("expected feature to be enabled after override deletion")
	}
}

func TestFeatureFlagCache_TTL(t *testing.T) {
	cache := NewFeatureFlagCache(50 * time.Millisecond)

	// Set some data
	cache.Set([]*model.FeatureFlag{
		{ID: "ff-1", Key: "test_cache", Name: "Test Cache", Status: model.FeatureFlagEnabled, Category: "test"},
	}, nil)

	// Should be available immediately
	if cache.Get("test_cache") == nil {
		t.Error("expected cache hit immediately after set")
	}

	// Wait for TTL to expire
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	if cache.Get("test_cache") != nil {
		t.Error("expected cache miss after TTL expiry")
	}
}

func TestCategorizeFeature(t *testing.T) {
	tests := []struct {
		feature  license.Feature
		expected string
	}{
		{license.FeatureSSL, "infrastructure"},
		{license.FeatureMonitoring, "monitoring"},
		{license.FeaturePlugins, "integration"},
		{license.FeatureOAuth2, "security"},
		{license.FeatureAPIKeys, "management"},
		{license.FeatureCluster, "infrastructure"},
		{license.FeatureToolbox, "tools"},
		{license.FeatureCustomBranding, "commercial"},
		{"unknown_feature", "general"},
	}
	for _, tt := range tests {
		cat := categorizeFeature(tt.feature)
		if cat != tt.expected {
			t.Errorf("categorizeFeature(%q) = %q, want %q", tt.feature, cat, tt.expected)
		}
	}
}
