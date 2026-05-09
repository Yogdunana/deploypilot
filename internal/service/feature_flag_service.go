package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/Yogdunana/deploypilot/internal/license"
	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/gorm"
)

// FeatureFlagCache is an in-memory cache of feature flags with TTL.
type FeatureFlagCache struct {
	mu      sync.RWMutex
	loadMu  sync.Mutex
	flags   map[string]*model.FeatureFlag
	overrides map[string]map[string]*model.FeatureFlagOverride // flagKey -> tenantID -> override
	loadedAt time.Time
	ttl      time.Duration
}

// NewFeatureFlagCache creates a new cache with the given TTL.
func NewFeatureFlagCache(ttl time.Duration) *FeatureFlagCache {
	return &FeatureFlagCache{
		flags:     make(map[string]*model.FeatureFlag),
		overrides: make(map[string]map[string]*model.FeatureFlagOverride),
		ttl:       ttl,
	}
}

// Get returns a cached flag by key, or nil if not found/expired.
func (c *FeatureFlagCache) Get(key string) *model.FeatureFlag {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if time.Since(c.loadedAt) > c.ttl {
		return nil
	}
	return c.flags[key]
}

// GetAll returns all cached flags, or nil if expired.
func (c *FeatureFlagCache) GetAll() []*model.FeatureFlag {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if time.Since(c.loadedAt) > c.ttl {
		return nil
	}
	result := make([]*model.FeatureFlag, 0, len(c.flags))
	for _, f := range c.flags {
		result = append(result, f)
	}
	return result
}

// GetOverride returns a cached override for a flag+tenant, or nil.
func (c *FeatureFlagCache) GetOverride(flagKey, tenantID string) *model.FeatureFlagOverride {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if time.Since(c.loadedAt) > c.ttl {
		return nil
	}
	if tenants, ok := c.overrides[flagKey]; ok {
		return tenants[tenantID]
	}
	return nil
}

// Set replaces the entire cache contents.
func (c *FeatureFlagCache) Set(flags []*model.FeatureFlag, overrides []*model.FeatureFlagOverride) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.flags = make(map[string]*model.FeatureFlag, len(flags))
	for _, f := range flags {
		cp := f
		c.flags[f.Key] = cp
	}
	c.overrides = make(map[string]map[string]*model.FeatureFlagOverride)
	for _, o := range overrides {
		if c.overrides[o.FlagKey] == nil {
			c.overrides[o.FlagKey] = make(map[string]*model.FeatureFlagOverride)
		}
		cp := o
		c.overrides[o.FlagKey][o.TenantID] = cp
	}
	c.loadedAt = time.Now()
}

// Invalidate clears the cache.
func (c *FeatureFlagCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.flags = nil
	c.overrides = nil
	c.loadedAt = time.Time{}
}

// InitFeatureFlags seeds the feature_flags table with default flags from the license engine.
// This should be called once at startup.
func (b *Bridge) InitFeatureFlags(ctx context.Context) error {
	var count int64
	if err := b.DB.Model(&model.FeatureFlag{}).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to count feature flags: %w", err)
	}
	if count > 0 {
		slog.Info("feature flags already initialized", "count", count)
		return nil
	}

	// Seed from license.AllFeatures
	now := time.Now()
	flags := make([]model.FeatureFlag, 0, len(license.AllFeatures))
	for _, f := range license.AllFeatures {
		flags = append(flags, model.FeatureFlag{
			ID:             fmt.Sprintf("ff-%d", now.UnixNano()+int64(len(flags))),
			Key:            string(f),
			Name:           strings.ReplaceAll(string(f), "_", " "),
			Description:    fmt.Sprintf("Feature flag for %s", strings.ReplaceAll(string(f), "_", " ")),
			Status:         model.FeatureFlagEnabled,
			DefaultEnabled: true,
			Category:       categorizeFeature(f),
		})
	}

	if err := b.DB.CreateInBatches(flags, 50).Error; err != nil {
		return fmt.Errorf("failed to seed feature flags: %w", err)
	}

	slog.Info("feature flags seeded", "count", len(flags))
	return nil
}

// categorizeFeature assigns a category to a license feature.
func categorizeFeature(f license.Feature) string {
	switch f {
	case license.FeatureSSL, license.FeatureBackup:
		return "infrastructure"
	case license.FeatureMonitoring, license.FeatureAlerting, license.FeatureDashboardTV:
		return "monitoring"
	case license.FeaturePlugins, license.FeatureWebhooks, license.FeatureGrafana:
		return "integration"
	case license.FeatureOAuth2, license.Feature2FA, license.FeatureIPWhitelist,
		license.FeatureDeviceBinding, license.FeatureCodeSigning,
		license.FeatureSSO, license.FeatureLDAP, license.FeatureMultiTenant, license.FeatureFederation:
		return "security"
	case license.FeatureAPIKeys, license.FeatureAuditExport, license.FeatureSLA:
		return "management"
	case license.FeatureCluster, license.FeatureRegistry, license.FeatureBatchOperations:
		return "infrastructure"
	case license.FeatureToolbox, license.FeatureSSHKeyManagement:
		return "tools"
	case license.FeatureCustomBranding, license.FeaturePrioritySupport:
		return "commercial"
	default:
		return "general"
	}
}

// EvaluateFeature checks if a feature is enabled, combining license engine evaluation
// with dynamic feature flag overrides.
func (b *Bridge) EvaluateFeature(ctx context.Context, featureKey string, tenantID string) (bool, error) {
	// 1. Check admin override first (highest priority)
	if tenantID != "" {
		override := b.featureFlagCache.GetOverride(featureKey, tenantID)
		if override == nil {
			// Cache miss: acquire loadMu to prevent cache stampede
			b.featureFlagCache.loadMu.Lock()
			// double-check cache after acquiring lock
			override = b.featureFlagCache.GetOverride(featureKey, tenantID)
			if override == nil {
				var dbOverride model.FeatureFlagOverride
				if err := b.DB.Where("flag_key = ? AND tenant_id = ?", featureKey, tenantID).First(&dbOverride).Error; err == nil {
					override = &dbOverride
				}
			}
			b.featureFlagCache.loadMu.Unlock()
		}
		if override != nil {
			return override.Enabled, nil
		}
	}

	// 2. Check dynamic feature flag from DB
	flag := b.featureFlagCache.Get(featureKey)
	if flag == nil {
		// Cache miss: acquire loadMu to prevent cache stampede
		b.featureFlagCache.loadMu.Lock()
		// double-check cache after acquiring lock
		flag = b.featureFlagCache.Get(featureKey)
		if flag == nil {
			var dbFlag model.FeatureFlag
			if err := b.DB.Where("key = ?", featureKey).First(&dbFlag).Error; err == nil {
				flag = &dbFlag
			}
		}
		b.featureFlagCache.loadMu.Unlock()
	}
	if flag != nil {
		if flag.Status != model.FeatureFlagEnabled {
			return false, nil
		}
		// Check tier/use_type restrictions
		if flag.RequiredTier != "" && b.LicenseEngine != nil {
			currentTier := b.LicenseEngine.GetTier()
			if license.Tier(flag.RequiredTier) != currentTier {
				// Check if the current tier is higher than required
				tierOrder := map[license.Tier]int{
					license.TierCommunity:  0,
					license.TierTeam:       1,
					license.TierPro:        2,
					license.TierEnterprise: 3,
				}
				if tierOrder[currentTier] < tierOrder[license.Tier(flag.RequiredTier)] {
					return false, nil
				}
			}
		}
		if flag.RequiredUseType != "" && b.LicenseEngine != nil {
			if b.LicenseEngine.GetUseType() != license.UseType(flag.RequiredUseType) {
				return false, nil
			}
		}
		return true, nil
	}

	// 3. Fallback to license engine static evaluation
	if b.LicenseEngine != nil {
		return b.LicenseEngine.IsFeatureEnabled(license.Feature(featureKey)), nil
	}

	// 4. No license engine, no flag = default enabled (community has all features)
	return true, nil
}

// ListFeatureFlags returns all feature flags with their current evaluation status.
func (b *Bridge) ListFeatureFlags(ctx context.Context) (interface{}, error) {
	flags := b.featureFlagCache.GetAll()
	if flags == nil {
		// Cache miss, load from DB
		var dbFlags []model.FeatureFlag
		if err := b.DB.Order("category, key").Find(&dbFlags).Error; err != nil {
			return nil, fmt.Errorf("failed to list feature flags: %w", err)
		}
		flags = make([]*model.FeatureFlag, len(dbFlags))
		for i := range dbFlags {
			flags[i] = &dbFlags[i]
		}
		b.featureFlagCache.Set(flags, nil)
	}

	result := make([]map[string]interface{}, 0, len(flags))
	for _, f := range flags {
		enabled, err := b.EvaluateFeature(ctx, f.Key, "")
		if err != nil {
			slog.Warn("failed to evaluate feature flag", "key", f.Key, "error", err)
			enabled = false
		}
		result = append(result, map[string]interface{}{
			"id":               f.ID,
			"key":              f.Key,
			"name":             f.Name,
			"description":      f.Description,
			"status":           f.Status,
			"default_enabled":  f.DefaultEnabled,
			"required_tier":    f.RequiredTier,
			"required_use_type": f.RequiredUseType,
			"category":         f.Category,
			"enabled":          enabled,
		})
	}
	return map[string]interface{}{
		"flags":   result,
		"total":   len(result),
	}, nil
}

// GetFeatureFlag returns a single feature flag by key.
func (b *Bridge) GetFeatureFlag(ctx context.Context, key string) (interface{}, error) {
	var flag model.FeatureFlag
	if err := b.DB.Where("key = ?", key).First(&flag).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("feature flag '%s' not found", key)
		}
		return nil, fmt.Errorf("failed to get feature flag: %w", err)
	}

	enabled, err := b.EvaluateFeature(ctx, flag.Key, "")
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":               flag.ID,
		"key":              flag.Key,
		"name":             flag.Name,
		"description":      flag.Description,
		"status":           flag.Status,
		"default_enabled":  flag.DefaultEnabled,
		"required_tier":    flag.RequiredTier,
		"required_use_type": flag.RequiredUseType,
		"category":         flag.Category,
		"overridden_by":    flag.OverriddenBy,
		"enabled":          enabled,
	}, nil
}

// UpdateFeatureFlag updates a feature flag's properties (admin only).
func (b *Bridge) UpdateFeatureFlag(ctx context.Context, key string, params map[string]interface{}) (interface{}, error) {
	var flag model.FeatureFlag
	if err := b.DB.Where("key = ?", key).First(&flag).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("feature flag '%s' not found", key)
		}
		return nil, fmt.Errorf("failed to get feature flag: %w", err)
	}

	updates := make(map[string]interface{})
	if status, ok := params["status"].(string); ok {
		updates["status"] = status
	}
	if name, ok := params["name"].(string); ok {
		updates["name"] = name
	}
	if desc, ok := params["description"].(string); ok {
		updates["description"] = desc
	}
	if de, ok := params["default_enabled"].(bool); ok {
		updates["default_enabled"] = de
	}
	if tier, ok := params["required_tier"].(string); ok {
		updates["required_tier"] = tier
	}
	if useType, ok := params["required_use_type"].(string); ok {
		updates["required_use_type"] = useType
	}
	if category, ok := params["category"].(string); ok {
		updates["category"] = category
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	if err := b.DB.Model(&flag).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update feature flag: %w", err)
	}

	// Invalidate cache
	b.featureFlagCache.Invalidate()

	slog.Info("feature flag updated", "key", key, "updates", updates)
	return b.GetFeatureFlag(ctx, key)
}

// SetFeatureFlagOverride creates or updates an admin override for a feature flag.
func (b *Bridge) SetFeatureFlagOverride(ctx context.Context, flagKey, tenantID string, enabled bool, reason, overriddenBy string) (interface{}, error) {
	var flag model.FeatureFlag
	if err := b.DB.Where("key = ?", flagKey).First(&flag).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("feature flag '%s' not found", flagKey)
		}
		return nil, fmt.Errorf("failed to get feature flag: %w", err)
	}

	var override model.FeatureFlagOverride
	result := b.DB.Where("flag_key = ? AND tenant_id = ?", flagKey, tenantID).First(&override)
	if result.Error == gorm.ErrRecordNotFound {
		override = model.FeatureFlagOverride{
			ID:           fmt.Sprintf("ffo-%d", time.Now().UnixNano()),
			FlagKey:      flagKey,
			TenantID:     tenantID,
			Enabled:      enabled,
			Reason:       reason,
			OverriddenBy: overriddenBy,
		}
		if err := b.DB.Create(&override).Error; err != nil {
			return nil, fmt.Errorf("failed to create feature flag override: %w", err)
		}
	} else if result.Error != nil {
		return nil, fmt.Errorf("failed to check existing override: %w", result.Error)
	} else {
		if err := b.DB.Model(&override).Updates(map[string]interface{}{
			"enabled":       enabled,
			"reason":        reason,
			"overridden_by": overriddenBy,
		}).Error; err != nil {
			return nil, fmt.Errorf("failed to update feature flag override: %w", err)
		}
	}

	// Update the flag's overridden_by field
	if err := b.DB.Model(&flag).Update("overridden_by", overriddenBy).Error; err != nil {
		slog.Error("failed to update feature flag overridden_by", "flag", flagKey, "error", err)
	}

	// Invalidate cache
	b.featureFlagCache.Invalidate()

	slog.Info("feature flag override set", "flag", flagKey, "tenant", tenantID, "enabled", enabled)
	return map[string]interface{}{
		"id":            override.ID,
		"flag_key":      flagKey,
		"tenant_id":     tenantID,
		"enabled":       enabled,
		"reason":        reason,
		"overridden_by": overriddenBy,
	}, nil
}

// DeleteFeatureFlagOverride removes an admin override.
func (b *Bridge) DeleteFeatureFlagOverride(ctx context.Context, flagKey, tenantID string) (interface{}, error) {
	result := b.DB.Where("flag_key = ? AND tenant_id = ?", flagKey, tenantID).Delete(&model.FeatureFlagOverride{})
	if result.Error != nil {
		return nil, fmt.Errorf("failed to delete feature flag override: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("no override found for flag '%s' and tenant '%s'", flagKey, tenantID)
	}

	// Clear overridden_by on the flag
	if err := b.DB.Model(&model.FeatureFlag{}).Where("key = ?", flagKey).Update("overridden_by", "").Error; err != nil {
		slog.Error("failed to clear feature flag overridden_by", "flag", flagKey, "error", err)
	}

	// Invalidate cache
	b.featureFlagCache.Invalidate()

	slog.Info("feature flag override deleted", "flag", flagKey, "tenant", tenantID)
	return map[string]interface{}{"message": "override deleted"}, nil
}

// ListFeatureFlagOverrides returns all overrides for a given flag key.
func (b *Bridge) ListFeatureFlagOverrides(ctx context.Context, flagKey string) (interface{}, error) {
	var overrides []model.FeatureFlagOverride
	if err := b.DB.Where("flag_key = ?", flagKey).Order("created_at DESC").Find(&overrides).Error; err != nil {
		return nil, fmt.Errorf("failed to list feature flag overrides: %w", err)
	}

	result := make([]map[string]interface{}, len(overrides))
	for i, o := range overrides {
		result[i] = map[string]interface{}{
			"id":            o.ID,
			"flag_key":      o.FlagKey,
			"tenant_id":     o.TenantID,
			"enabled":       o.Enabled,
			"reason":        o.Reason,
			"overridden_by": o.OverriddenBy,
			"created_at":    o.CreatedAt,
			"updated_at":    o.UpdatedAt,
		}
	}
	return map[string]interface{}{
		"overrides": result,
		"total":     len(result),
	}, nil
}

// GetFeatureFlagsForTenant returns all feature flags with their evaluation status for a specific tenant.
func (b *Bridge) GetFeatureFlagsForTenant(ctx context.Context, tenantID string) (interface{}, error) {
	flags := b.featureFlagCache.GetAll()
	if flags == nil {
		var dbFlags []model.FeatureFlag
		if err := b.DB.Order("category, key").Find(&dbFlags).Error; err != nil {
			return nil, fmt.Errorf("failed to list feature flags: %w", err)
		}
		flags = make([]*model.FeatureFlag, len(dbFlags))
		for i := range dbFlags {
			flags[i] = &dbFlags[i]
		}
		b.featureFlagCache.Set(flags, nil)
	}

	result := make([]map[string]interface{}, 0, len(flags))
	for _, f := range flags {
		enabled, err := b.EvaluateFeature(ctx, f.Key, tenantID)
		if err != nil {
			slog.Warn("failed to evaluate feature flag for tenant", "key", f.Key, "tenant", tenantID, "error", err)
			enabled = false
		}

		entry := map[string]interface{}{
			"key":     f.Key,
			"name":    f.Name,
			"enabled": enabled,
		}

		// Include override info if exists
		override := b.featureFlagCache.GetOverride(f.Key, tenantID)
		if override != nil {
			entry["overridden"] = true
			entry["override_enabled"] = override.Enabled
			entry["override_reason"] = override.Reason
		}

		result = append(result, entry)
	}
	return map[string]interface{}{
		"flags":     result,
		"tenant_id": tenantID,
		"total":     len(result),
	}, nil
}
