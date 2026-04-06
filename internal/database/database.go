package database

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect establishes a database connection based on the driver type.
// Supported drivers: "sqlite", "postgres".
func Connect(driver, dsn string) (*gorm.DB, error) {
	if dsn == "" {
		return nil, errors.New("database DSN must not be empty")
	}

	var db *gorm.DB
	var err error

	switch driver {
	case "sqlite":
		// Ensure parent directory exists
		dir := filepath.Dir(dsn)
		if dir != "" && dir != "." {
			// Let GORM/sqlite handle directory creation implicitly
		}
		db, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
	case "postgres":
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", driver)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database (%s): %w", driver, err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50)

	return db, nil
}

// Migrate runs all database migrations using gormigrate.
// Migrations are idempotent — running them multiple times is safe.
func Migrate(db *gorm.DB) error {
	m := gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		// 202604060001: Create core tables
		{
			ID: "202604060001",
			Migrate: func(tx *gorm.DB) error {
				// tenants
				if err := tx.Exec(`CREATE TABLE IF NOT EXISTS tenants (
					id TEXT PRIMARY KEY,
					name TEXT NOT NULL,
					slug TEXT UNIQUE NOT NULL,
					plan TEXT DEFAULT 'free',
					max_servers INTEGER DEFAULT 5,
					max_apps INTEGER DEFAULT 20,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				)`).Error; err != nil {
					return err
				}

				// roles
				if err := tx.Exec(`CREATE TABLE IF NOT EXISTS roles (
					id TEXT PRIMARY KEY,
					name TEXT UNIQUE NOT NULL,
					permissions TEXT,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP
				)`).Error; err != nil {
					return err
				}

				// users
				if err := tx.Exec(`CREATE TABLE IF NOT EXISTS users (
					id TEXT PRIMARY KEY,
					tenant_id TEXT REFERENCES tenants(id),
					role_id TEXT REFERENCES roles(id),
					username TEXT UNIQUE NOT NULL,
					email TEXT UNIQUE NOT NULL,
					password_hash TEXT NOT NULL,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				)`).Error; err != nil {
					return err
				}

				// credentials (must be before servers due to FK)
				if err := tx.Exec(`CREATE TABLE IF NOT EXISTS credentials (
					id TEXT PRIMARY KEY,
					tenant_id TEXT REFERENCES tenants(id),
					name TEXT NOT NULL,
					type TEXT NOT NULL,
					encrypted_value TEXT NOT NULL,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				)`).Error; err != nil {
					return err
				}

				// providers (must be before servers due to FK)
				if err := tx.Exec(`CREATE TABLE IF NOT EXISTS providers (
					id TEXT PRIMARY KEY,
					tenant_id TEXT REFERENCES tenants(id),
					type TEXT NOT NULL,
					name TEXT NOT NULL,
					config TEXT,
					enabled INTEGER DEFAULT 1,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				)`).Error; err != nil {
					return err
				}

				// servers
				if err := tx.Exec(`CREATE TABLE IF NOT EXISTS servers (
					id TEXT PRIMARY KEY,
					tenant_id TEXT REFERENCES tenants(id),
					credential_id TEXT REFERENCES credentials(id),
					provider_id TEXT REFERENCES providers(id),
					name TEXT NOT NULL,
					host TEXT NOT NULL,
					port INTEGER DEFAULT 22,
					tags TEXT,
					status TEXT DEFAULT 'unknown',
					detected_info TEXT,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				)`).Error; err != nil {
					return err
				}

				// apps
				if err := tx.Exec(`CREATE TABLE IF NOT EXISTS apps (
					id TEXT PRIMARY KEY,
					tenant_id TEXT REFERENCES tenants(id),
					server_id TEXT REFERENCES servers(id),
					name TEXT NOT NULL,
					repo_url TEXT NOT NULL,
					branch TEXT DEFAULT 'main',
					domain TEXT,
					tech_stack TEXT DEFAULT 'docker',
					deploy_mode TEXT DEFAULT 'api',
					status TEXT DEFAULT 'pending',
					current_version TEXT,
					container_name TEXT,
					env_vars TEXT,
					resource_limits TEXT,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				)`).Error; err != nil {
					return err
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable(
					"apps", "servers", "providers", "credentials",
					"users", "roles", "tenants",
				)
			},
		},
	})

	// Use InitSchema for initial creation (faster than Migrate)
	if err := m.Migrate(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// Seed inserts default data required for the application to function.
// This should be called after Migrate().
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
		"tenant-default", "Default", "default", "free",
	)
	if result.Error != nil {
		return fmt.Errorf("failed to seed default tenant: %w", result.Error)
	}

	return nil
}
