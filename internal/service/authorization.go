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
// Resource types: "app", "server", "credential", "cluster", "registry".
// owner and admin roles can access all resources within their tenant.
// Non-admin roles (viewer, dev) can only access resources belonging to their tenant.
func CheckResourceAccess(db *gorm.DB, resourceType, resourceID, role, userID string) bool {
	// Empty role or userID means unauthenticated — deny access.
	if role == "" || userID == "" {
		return false
	}

	// owner and admin can access all resources within their tenant
	if role == "owner" || role == "admin" {
		return true
	}

	// Resolve the user's tenant_id once for all tenant-scoped lookups.
	var tenantID string
	if err := db.Table("users").Where("id = ?", userID).Pluck("tenant_id", &tenantID).Error; err != nil || tenantID == "" {
		return false
	}

	// All resource tables (apps, servers, credentials, clusters, registries)
	// are scoped by tenant_id, not user_id. Check that the resource exists
	// and belongs to the user's tenant.
	var count int64
	switch resourceType {
	case "app":
		db.Table("apps").Where("id = ? AND tenant_id = ?", resourceID, tenantID).Count(&count)
	case "server":
		db.Table("servers").Where("id = ? AND tenant_id = ?", resourceID, tenantID).Count(&count)
	case "credential":
		db.Table("credentials").Where("id = ? AND tenant_id = ?", resourceID, tenantID).Count(&count)
	case "cluster":
		db.Table("clusters").Where("id = ? AND tenant_id = ?", resourceID, tenantID).Count(&count)
	case "registry":
		db.Table("registries").Where("id = ? AND tenant_id = ?", resourceID, tenantID).Count(&count)
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