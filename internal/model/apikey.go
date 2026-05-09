package model

import "time"

// APIKey represents a programmable API key for authentication.
// The raw key is only shown once at creation time; only the SHA-256 hash is stored.
type APIKey struct {
	ID         string     `gorm:"primaryKey" json:"id"`
	TenantID   string     `gorm:"index" json:"tenant_id"`
	UserID     string     `gorm:"index" json:"user_id"`
	Name       string     `gorm:"not null;size:100" json:"name"`
	KeyHash    string     `gorm:"uniqueIndex;not null;size:64" json:"-"`
	KeyPrefix  string     `gorm:"size:10;not null" json:"key_prefix"`
	Scopes     string     `gorm:"type:text" json:"scopes"`
	AllowedIPs string     `gorm:"type:text" json:"allowed_ips,omitempty"` // JSON array of IPs/CIDRs
	ExpiresAt  *time.Time `gorm:"index" json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	UsageCount int64      `gorm:"default:0" json:"usage_count"` // total request count
	CreatedAt  time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	Tenant Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	User   User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (APIKey) TableName() string { return "api_keys" }
