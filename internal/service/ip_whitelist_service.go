package service

import (
	"fmt"
	"net"

	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// IPWhitelistService provides per-user IP whitelist management.
type IPWhitelistService struct {
	db *gorm.DB
}

// NewIPWhitelistService creates a new IPWhitelistService.
func NewIPWhitelistService(db *gorm.DB) *IPWhitelistService {
	return &IPWhitelistService{db: db}
}

// Create adds a new whitelist entry for the given user.
// It validates the CIDR format before insertion.
func (s *IPWhitelistService) Create(userID, description, cidr, tenantID, createdBy string) (*model.IPWhitelist, error) {
	if _, _, err := net.ParseCIDR(cidr); err != nil {
		return nil, fmt.Errorf("invalid CIDR format: %s", cidr)
	}

	entry := &model.IPWhitelist{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		UserID:      userID,
		Description: description,
		CIDR:        cidr,
		CreatedBy:   createdBy,
	}

	if err := model.CreateIPWhitelist(s.db, entry); err != nil {
		return nil, fmt.Errorf("failed to create IP whitelist entry: %w", err)
	}

	return entry, nil
}

// List returns all whitelist entries for the given user.
func (s *IPWhitelistService) List(userID string) ([]model.IPWhitelist, error) {
	entries, err := model.GetUserWhitelists(s.db, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list IP whitelist entries: %w", err)
	}
	return entries, nil
}

// Delete removes a whitelist entry. Only the owner (user who created it) can delete.
func (s *IPWhitelistService) Delete(id, userID string) error {
	if err := model.DeleteIPWhitelist(s.db, id, userID); err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("whitelist entry not found or does not belong to user")
		}
		return fmt.Errorf("failed to delete IP whitelist entry: %w", err)
	}
	return nil
}

// Check checks whether a given IP matches any of the user's whitelist entries.
// If the user has no entries, it returns true (not enforced).
func (s *IPWhitelistService) Check(ip, userID string) bool {
	return model.IsIPWhitelisted(s.db, userID, ip)
}

// CheckGlobal checks whether a given IP is allowed by the global security config whitelist.
func (s *IPWhitelistService) CheckGlobal(ip string, allowedIPs []string) bool {
	if len(allowedIPs) == 0 {
		return true
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	for _, allowed := range allowedIPs {
		// Try as CIDR
		if containsSlash(allowed) {
			_, cidr, err := net.ParseCIDR(allowed)
			if err == nil && cidr.Contains(parsedIP) {
				return true
			}
			continue
		}
		// Try as plain IP
		if net.ParseIP(allowed) != nil && net.ParseIP(allowed).Equal(parsedIP) {
			return true
		}
	}
	return false
}

// IsEnforced returns true if the user has any whitelist entries configured.
// When enforced, only whitelisted IPs are allowed access.
func (s *IPWhitelistService) IsEnforced(userID string) bool {
	var count int64
	s.db.Model(&model.IPWhitelist{}).Where("user_id = ?", userID).Count(&count)
	return count > 0
}

// containsSlash is a simple helper to check for CIDR notation.
func containsSlash(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			return true
		}
	}
	return false
}
