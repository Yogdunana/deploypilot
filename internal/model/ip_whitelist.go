package model

import (
	"net"

	"time"

	"gorm.io/gorm"
)

// IPWhitelist represents a per-user IP whitelist entry.
type IPWhitelist struct {
	ID          string    `gorm:"primaryKey" json:"id"`
	TenantID    string    `gorm:"index" json:"tenant_id"`
	UserID      string    `gorm:"index;not null" json:"user_id"`
	Description string    `gorm:"size:200" json:"description"`
	CIDR        string    `gorm:"column:cidr;size:45;not null" json:"cidr"`
	CreatedBy   string    `gorm:"size:100" json:"created_by"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (IPWhitelist) TableName() string { return "ip_whitelists" }

// CreateIPWhitelist inserts a new IP whitelist entry into the database.
func CreateIPWhitelist(db *gorm.DB, entry *IPWhitelist) error {
	return db.Create(entry).Error
}

// GetUserWhitelists returns all whitelist entries for a given user.
func GetUserWhitelists(db *gorm.DB, userID string) ([]IPWhitelist, error) {
	var entries []IPWhitelist
	if err := db.Where("user_id = ?", userID).Order("created_at DESC").Find(&entries).Error; err != nil {
		return nil, err
	}
	return entries, nil
}

// DeleteIPWhitelist removes a whitelist entry by ID, enforcing owner-only deletion.
func DeleteIPWhitelist(db *gorm.DB, id, userID string) error {
	result := db.Where("id = ? AND user_id = ?", id, userID).Delete(&IPWhitelist{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// IsIPWhitelisted checks whether a given IP address matches any of the user's
// whitelist CIDR entries. Returns true if the user has no entries (not enforced).
func IsIPWhitelisted(db *gorm.DB, userID, ip string) bool {
	var count int64
	db.Model(&IPWhitelist{}).Where("user_id = ?", userID).Count(&count)
	if count == 0 {
		return true
	}

	var entries []IPWhitelist
	if err := db.Where("user_id = ?", userID).Find(&entries).Error; err != nil {
		return false
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	for _, entry := range entries {
		_, cidr, err := net.ParseCIDR(entry.CIDR)
		if err != nil {
			continue
		}
		if cidr.Contains(parsedIP) {
			return true
		}
	}
	return false
}
