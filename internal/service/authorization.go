package service

import (
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