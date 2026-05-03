package model

import "time"

// FeatureFlagOverride represents a per-instance admin override for a feature flag.
type FeatureFlagOverride struct {
	ID          string            `gorm:"primaryKey" json:"id"`
	FlagKey     string            `gorm:"uniqueIndex:idx_flag_tenant;not null;size:100" json:"flag_key"`
	TenantID    string            `gorm:"uniqueIndex:idx_flag_tenant;not null;size:100" json:"tenant_id"`
	Enabled     bool              `gorm:"not null" json:"enabled"`
	Reason      string            `gorm:"size:500;default:''" json:"reason"`
	OverriddenBy string           `gorm:"size:100;not null" json:"overridden_by"`
	CreatedAt   time.Time         `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time         `gorm:"autoUpdateTime" json:"updated_at"`
}

func (FeatureFlagOverride) TableName() string { return "feature_flag_overrides" }
