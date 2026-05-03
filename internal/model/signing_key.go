package model

import "time"

// SigningKey stores an Ed25519 key pair used for code signing.
type SigningKey struct {
	ID          string    `gorm:"primaryKey" json:"id"`
	KeyVersion  int       `gorm:"uniqueIndex;not null" json:"key_version"`
	PublicKey   string    `gorm:"type:text;not null" json:"public_key"`
	PrivateKey  string    `gorm:"type:text;not null" json:"-"`
	Fingerprint string    `gorm:"size:16" json:"fingerprint"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedBy   string    `gorm:"size:100" json:"created_by"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName returns the database table name for SigningKey.
func (SigningKey) TableName() string { return "signing_keys" }
