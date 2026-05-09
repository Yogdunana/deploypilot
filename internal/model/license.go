package model

import (
	"time"
)

// Tier defines the license tier level.
type Tier string

const (
	TierCommunity  Tier = "community"
	TierTeam       Tier = "team"
	TierPro        Tier = "pro"
	TierEnterprise Tier = "enterprise"
)

// UseType defines the license usage type.
type UseType string

const (
	UseTypeNonCommercial UseType = "non_commercial"
	UseTypeCommercial     UseType = "commercial"
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
	Tier          Tier           `gorm:"size:20;not null;default:community" json:"tier"`
	UseType       UseType        `gorm:"size:20;not null;default:non_commercial" json:"use_type"`
	Status        LicenseStatus  `gorm:"size:20;not null;default:active" json:"status"`
	Features      string         `gorm:"type:text" json:"features"` // JSON array of enabled features
	Limits        string         `gorm:"type:text" json:"limits"`   // JSON object: {"max_servers":10,"max_apps":50,"max_users":5}
	MaxServers    int            `gorm:"default:3" json:"max_servers"`
	MaxApps       int            `gorm:"default:10" json:"max_apps"`
	MaxUsers      int            `gorm:"default:5" json:"max_users"`
	IssuerRole    string         `gorm:"size:20;not null;default:user" json:"issuer_role"`
	IssuedTo      string         `gorm:"size:64" json:"issued_to,omitempty"`
	MaxIssued     int            `gorm:"default:0" json:"max_issued"`
	IssuedCount   int            `gorm:"default:0" json:"issued_count"`
	Addons        string         `gorm:"type:text" json:"addons"` // JSON array of Addon structs
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
