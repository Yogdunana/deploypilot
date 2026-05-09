package model

import "time"

// FeatureFlagStatus represents the status of a feature flag.
type FeatureFlagStatus string

const (
	FeatureFlagEnabled  FeatureFlagStatus = "enabled"
	FeatureFlagDisabled FeatureFlagStatus = "disabled"
)

// FeatureFlag represents a dynamic feature flag that can override
// the static license-based feature evaluation.
type FeatureFlag struct {
	ID          string            `gorm:"primaryKey" json:"id"`
	Key         string            `gorm:"uniqueIndex;not null;size:100" json:"key"`
	Name        string            `gorm:"size:200;not null" json:"name"`
	Description string            `gorm:"type:text" json:"description"`
	Status      FeatureFlagStatus `gorm:"size:20;not null;default:enabled" json:"status"`
	// DefaultEnabled is the default state when no license override exists.
	DefaultEnabled bool   `gorm:"not null;default:false" json:"default_enabled"`
	// RequiredTier is the minimum license tier required (empty = any tier).
	RequiredTier string `gorm:"size:20;default:''" json:"required_tier"`
	// RequiredUseType restricts to non_commercial or commercial (empty = both).
	RequiredUseType string `gorm:"size:30;default:''" json:"required_use_type"`
	// Category groups related flags together (e.g., "monitoring", "security").
	Category string `gorm:"size:50;default:'general'" json:"category"`
	// OverriddenBy stores the tenant_id that manually overrode this flag.
	OverriddenBy string    `gorm:"size:100;default:''" json:"overridden_by"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (FeatureFlag) TableName() string { return "feature_flags" }
