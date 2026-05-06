package model

import (
	"crypto/sha256"
	"time"
)

// Device represents a bound device for a user with trust management.
type Device struct {
	ID            string     `gorm:"primaryKey" json:"id"`
	TenantID      string     `gorm:"index" json:"tenant_id"`
	UserID        string     `gorm:"index" json:"user_id"`
	DeviceID      string     `gorm:"uniqueIndex;size:16;not null" json:"device_id"`
	DeviceName    string     `gorm:"size:200" json:"device_name"`
	UserAgent     string     `gorm:"size:500" json:"user_agent"`
	IP            string     `gorm:"size:45" json:"ip"`
	LastIP        string     `gorm:"size:45" json:"last_ip"`
	Trusted       bool       `gorm:"default:false" json:"trusted"`
	TrustExpiresAt *time.Time `json:"trust_expires_at,omitempty"`
	LastSeenAt    *time.Time `json:"last_seen_at,omitempty"`
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"created_at"`
}

func (Device) TableName() string { return "devices" }

// GenerateDeviceID computes a stable device fingerprint from userAgent and IP.
// It returns the first 16 hex characters of SHA256(userAgent + ip).
func GenerateDeviceID(userAgent, ip string) string {
	h := sha256.Sum256([]byte(userAgent + ip))
	return hexEncode(h[:])[:16]
}

// hexEncode converts a byte slice to a lowercase hex string.
func hexEncode(b []byte) string {
	const hexChars = "0123456789abcdef"
	// Guard against integer overflow on very large inputs
	if len(b) > (1<<30) {
		b = b[:1<<30]
	}
	result := make([]byte, len(b)*2)
	for i, v := range b {
		result[i*2] = hexChars[v>>4]
		result[i*2+1] = hexChars[v&0x0f]
	}
	return string(result)
}
