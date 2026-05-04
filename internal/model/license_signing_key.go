package model

import "time"

// LicenseSigningKey represents a versioned license signing key pair.
type LicenseSigningKey struct {
	ID          string    `gorm:"primaryKey" json:"id"`
	KeyVersion  int       `gorm:"uniqueIndex;not null" json:"key_version"`
	PublicKey   string    `gorm:"type:text;not null" json:"public_key"`   // Base64 encoded ed25519 public key
	PrivateKey  string    `gorm:"type:text;not null" json:"-"`             // Base64 encoded ed25519 private key seed (never exposed via API)
	Fingerprint string    `gorm:"size:16;not null" json:"fingerprint"`     // SHA256 first 16 hex chars
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedBy   string    `gorm:"size:100" json:"created_by"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	RotatedAt   *time.Time `json:"rotated_at,omitempty"`
}

func (LicenseSigningKey) TableName() string { return "license_signing_keys" }
