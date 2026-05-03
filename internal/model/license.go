package model

import (
	"time"
)

// LicenseType defines the type of license.
type LicenseType string

const (
	LicenseTypeCommunity  LicenseType = "community"
	LicenseTypePro        LicenseType = "pro"
	LicenseTypeEnterprise LicenseType = "enterprise"
)

// LicenseStatus defines the status of a license.
type LicenseStatus string

const (
	LicenseStatusActive    LicenseStatus = "active"
	LicenseStatusExpired   LicenseStatus = "expired"
	LicenseStatusRevoked   LicenseStatus = "revoked"
	LicenseStatusSuspended LicenseStatus = "suspended"
)

// License represents a software license key.
type License struct {
	ID            string         `gorm:"primaryKey" json:"id"`
	TenantID      string         `gorm:"index" json:"tenant_id"`
	LicenseKey    string         `gorm:"uniqueIndex;not null;size:512" json:"license_key"`
	LicenseType   LicenseType    `gorm:"size:20;not null;default:community" json:"license_type"`
	Status        LicenseStatus  `gorm:"size:20;not null;default:active" json:"status"`
	Features      string         `gorm:"type:text" json:"features"` // JSON array of enabled features
	Limits        string         `gorm:"type:text" json:"limits"`   // JSON object: {"max_servers":10,"max_apps":50,"max_users":5}
	MaxServers    int            `gorm:"default:3" json:"max_servers"`
	MaxApps       int            `gorm:"default:10" json:"max_apps"`
	MaxUsers      int            `gorm:"default:3" json:"max_users"`
	IssuedAt      time.Time      `gorm:"autoCreateTime" json:"issued_at"`
	ExpiresAt     *time.Time     `json:"expires_at"`
	GraceDays     int            `gorm:"default:7" json:"grace_days"`
	LastCheckedAt *time.Time     `json:"last_checked_at"`
	ActivatedAt   *time.Time     `json:"activated_at"`
	MachineID     string         `gorm:"size:64" json:"machine_id"` // fingerprint of activated machine
	RevokedReason string         `gorm:"size:255" json:"revoked_reason,omitempty"`
	RevokedAt     *time.Time     `json:"revoked_at,omitempty"`
	CreatedBy     string         `gorm:"size:100" json:"created_by"`
	CreatedAt     time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

func (License) TableName() string { return "licenses" }
