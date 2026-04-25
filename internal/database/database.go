package database

import (
	"errors"
	"fmt"

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
		// 202604080001: Create deployments table
		{
			ID: "202604080001",
			Migrate: func(tx *gorm.DB) error {
				return tx.Exec(`CREATE TABLE IF NOT EXISTS deployments (
					id TEXT PRIMARY KEY,
					tenant_id TEXT REFERENCES tenants(id),
					server_id TEXT,
					app_name TEXT,
					container_name TEXT,
					image TEXT,
					status TEXT DEFAULT 'deploying',
					preflight_code TEXT,
					preflight_message TEXT,
					preflight_checks TEXT,
					error_message TEXT,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				)`).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("deployments")
			},
		},
		// 202604240001: Create audit_logs table
		{
			ID: "202604240001",
			Migrate: func(tx *gorm.DB) error {
				return tx.Exec(`CREATE TABLE IF NOT EXISTS audit_logs (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					user_id INTEGER,
					username TEXT,
					action TEXT,
					resource_type TEXT,
					resource_id TEXT,
					detail TEXT,
					ip_address TEXT,
					user_agent TEXT,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP
				)`).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("audit_logs")
			},
		},
		// 202604250001: Create ssl_certificates table
		{
			ID: "202604250001",
			Migrate: func(tx *gorm.DB) error {
				return tx.Exec(`CREATE TABLE IF NOT EXISTS ssl_certificates (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					domain TEXT UNIQUE NOT NULL,
					email TEXT NOT NULL,
					provider TEXT NOT NULL DEFAULT 'cloudflare',
					status TEXT NOT NULL DEFAULT 'pending',
					cert_path TEXT,
					key_path TEXT,
					issued_at DATETIME,
					expires_at DATETIME,
					auto_renew INTEGER DEFAULT 1,
					last_renewed DATETIME,
					retry_count INTEGER DEFAULT 0,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				)`).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("ssl_certificates")
			},
		},
		// 202604250002: Add credential lifecycle fields
		{
			ID: "202604250002",
			Migrate: func(tx *gorm.DB) error {
				// Add lifecycle columns to credentials table (safe to run if columns already exist)
				tx.Exec(`ALTER TABLE credentials ADD COLUMN expires_at DATETIME`)
				tx.Exec(`ALTER TABLE credentials ADD COLUMN last_rotated DATETIME`)
				tx.Exec(`ALTER TABLE credentials ADD COLUMN rotation_days INTEGER DEFAULT 90`)
				// Create index on expires_at for expiry queries
				tx.Exec(`CREATE INDEX IF NOT EXISTS idx_credentials_expires_at ON credentials(expires_at)`)
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				// SQLite does not support DROP COLUMN easily, so we recreate the table
				tx.Exec(`CREATE TABLE credentials_backup AS SELECT id, tenant_id, name, type, encrypted_value, created_at, updated_at FROM credentials`)
				tx.Exec(`DROP TABLE credentials`)
				tx.Exec(`ALTER TABLE credentials_backup RENAME TO credentials`)
				return nil
			},
		},
		// 202604250003: Create registries table
		{
			ID: "202604250003",
			Migrate: func(tx *gorm.DB) error {
				return tx.Exec(`CREATE TABLE IF NOT EXISTS registries (
					id TEXT PRIMARY KEY,
					tenant_id TEXT REFERENCES tenants(id),
					name TEXT NOT NULL,
					provider TEXT NOT NULL,
					url TEXT NOT NULL,
					username TEXT,
					password TEXT,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				)`).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("registries")
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
