package service

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"gorm.io/gorm"
)

// resourceTables maps a resource type to the table holding it. Every table here
// carries a tenant_id, which is the boundary access is enforced on.
//
// ownerScoped marks tables that record a creator. Clusters and registries are
// shared by the whole tenant and have no user_id column at all.
var resourceTables = map[string]struct {
	table       string
	ownerScoped bool
}{
	"app":        {"apps", true},
	"server":     {"servers", true},
	"credential": {"credentials", true},
	"cluster":    {"clusters", false},
	"registry":   {"registries", false},
}

// CheckResourceAccess checks if a user has access to a specific resource.
// Resource types: "app", "server", "credential", "cluster", "registry".
//
// The tenant is a hard boundary that no role crosses, admin and owner included.
// Inside a tenant, owner and admin reach every resource; viewer and dev are
// additionally limited to resources they created. Rows with no recorded owner
// (user_id empty — everything written before that column existed) stay visible
// to the whole tenant rather than becoming unreachable.
func CheckResourceAccess(db *gorm.DB, resourceType, resourceID, role, userID string) bool {
	if resourceID == "" || userID == "" {
		return false
	}

	resource, ok := resourceTables[resourceType]
	if !ok {
		slog.Warn("resource access check on unknown resource type", "type", resourceType)
		return false
	}

	var tenantID string
	if err := db.Table("users").Where("id = ?", userID).Pluck("tenant_id", &tenantID).Error; err != nil {
		slog.Error("resource access check: could not resolve tenant", "user_id", userID, "error", err)
		return false
	}
	if tenantID == "" {
		slog.Warn("resource access check: user has no tenant", "user_id", userID)
		return false
	}

	query := db.Table(resource.table).Where("id = ? AND tenant_id = ?", resourceID, tenantID)
	if resource.ownerScoped && role != "owner" && role != "admin" {
		query = query.Where("(user_id IS NULL OR user_id = '' OR user_id = ?)", userID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		slog.Error("resource access check failed", "type", resourceType, "id", resourceID, "error", err)
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