package database

import (
	"fmt"

	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/gorm"
)

func Seed(db *gorm.DB) error {
	// Seed default roles
	roles := []struct {
		ID          string
		Name        string
		Permissions string
	}{
		{"role-owner", "owner", `{"admin": true, "deploy": true, "manage_servers": true, "manage_apps": true, "manage_users": true}`},
		{"role-admin", "admin", `{"admin": false, "deploy": true, "manage_servers": true, "manage_apps": true, "manage_users": true}`},
		{"role-dev", "dev", `{"admin": false, "deploy": true, "manage_servers": false, "manage_apps": true, "manage_users": false}`},
		{"role-viewer", "viewer", `{"admin": false, "deploy": false, "manage_servers": false, "manage_apps": false, "manage_users": false}`},
	}

	for _, r := range roles {
		result := db.Exec(
			`INSERT OR IGNORE INTO roles (id, name, permissions) VALUES (?, ?, ?)`,
			r.ID, r.Name, r.Permissions,
		)
		if result.Error != nil {
			return fmt.Errorf("failed to seed role %s: %w", r.Name, result.Error)
		}
	}

	// Seed default tenant
	result := db.Exec(
		`INSERT OR IGNORE INTO tenants (id, name, slug, plan) VALUES (?, ?, ?, ?)`,
		model.DefaultTenantID, "Default", "default", "free",
	)
	if result.Error != nil {
		return fmt.Errorf("failed to seed default tenant: %w", result.Error)
	}

	return nil
}

