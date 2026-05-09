package service

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"gorm.io/gorm"
)

// CheckResourceAccess checks if a user has access to a specific resource.
// Resource types: "app", "server", "credential", "cluster"
// owner and admin roles can access all resources.
// viewer and dev roles can only access resources they created (user_id match),
// except for clusters which are tenant-level resources (tenant_id match).
func CheckResourceAccess(db *gorm.DB, resourceType, resourceID, role, userID string) bool {
	// owner and admin can access all resources
	if role == "owner" || role == "admin" {
		return true
	}

	// viewer and dev can only access their own resources
	var count int64
	switch resourceType {
	case "app":
		db.Table("apps").Where("id = ? AND user_id = ?", resourceID, userID).Count(&count)
	case "server":
		db.Table("servers").Where("id = ? AND user_id = ?", resourceID, userID).Count(&count)
	case "credential":
		db.Table("credentials").Where("id = ? AND user_id = ?", resourceID, userID).Count(&count)
	case "cluster":
		// Clusters are tenant-level resources: check tenant_id match
		var tenantID string
		if err := db.Table("users").Where("id = ?", userID).Pluck("tenant_id", &tenantID).Error; err != nil {
			return false
		}
		db.Table("clusters").Where("id = ? AND tenant_id = ?", resourceID, tenantID).Count(&count)
	default:
		return false
	}
	return count > 0
}

// CheckResourceAccessCached checks resource access with caching.
// It uses the Bridge's Cache if available, falling back to direct DB query.
// Cache key format: perm:{userID}:{resourceType}:{resourceID}, TTL: 5 minutes.
func (b *Bridge) CheckResourceAccessCached(ctx context.Context, resourceType, resourceID, role, userID string) bool {
	// owner and admin can access all resources (no need to cache)
	if role == "owner" || role == "admin" {
		return true
	}

	// Try cache first
	if b.Cache != nil {
		cacheKey := fmt.Sprintf("perm:%s:%s:%s", userID, resourceType, resourceID)
		cached, err := b.Cache.Get(ctx, cacheKey)
		if err == nil {
			result, parseErr := strconv.ParseBool(cached)
			if parseErr == nil {
				return result
			}
		}
		if err != nil && err != ErrCacheMiss {
			slog.Warn("cache get error, falling back to DB", "error", err)
		}

		// Cache miss: query DB
		result := CheckResourceAccess(b.DB, resourceType, resourceID, role, userID)

		// Store result in cache (fire-and-forget)
		if cacheErr := b.Cache.Set(ctx, cacheKey, strconv.FormatBool(result), 5*time.Minute); cacheErr != nil {
			slog.Warn("cache set error", "error", cacheErr)
		}

		return result
	}

	// No cache available, query directly
	return CheckResourceAccess(b.DB, resourceType, resourceID, role, userID)
}