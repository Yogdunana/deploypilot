package database

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DriverFactory creates a *gorm.DB from a DSN and config.
type DriverFactory func(dsn string, config *gorm.Config) (*gorm.DB, error)

var drivers = map[string]DriverFactory{}

// RegisterDriver registers a database driver factory.
func RegisterDriver(name string, factory DriverFactory) {
	drivers[name] = factory
}

func init() {
	RegisterDriver("sqlite", func(dsn string, cfg *gorm.Config) (*gorm.DB, error) {
		return gorm.Open(sqlite.Open(dsn), cfg)
	})
	RegisterDriver("postgres", func(dsn string, cfg *gorm.Config) (*gorm.DB, error) {
		return gorm.Open(postgres.Open(dsn), cfg)
	})
}

// Connect establishes a database connection based on the driver type.
// Supported drivers: "sqlite", "postgres".
func Connect(driver, dsn string) (*gorm.DB, error) {
	if dsn == "" {
		return nil, errors.New("database DSN must not be empty")
	}

	var db *gorm.DB
	var err error

	factory, ok := drivers[driver]
	if !ok {
		return nil, fmt.Errorf("unsupported database driver: %s", driver)
	}
	db, err = factory(dsn, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})

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

// ignoreDuplicateColumnError ignores "duplicate column" and "already exists" errors
// that occur when ALTER TABLE ADD COLUMN is run on a column that already exists.
// This makes migrations idempotent across both SQLite and PostgreSQL.
func ignoreDuplicateColumnError(tx *gorm.DB, err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "duplicate column") ||
		strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "Duplicate column") {
		return nil
	}
	return err
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
				if err := ignoreDuplicateColumnError(tx, tx.Exec(`ALTER TABLE credentials ADD COLUMN expires_at DATETIME`).Error); err != nil {
					return err
				}
				if err := ignoreDuplicateColumnError(tx, tx.Exec(`ALTER TABLE credentials ADD COLUMN last_rotated DATETIME`).Error); err != nil {
					return err
				}
				if err := ignoreDuplicateColumnError(tx, tx.Exec(`ALTER TABLE credentials ADD COLUMN rotation_days INTEGER DEFAULT 90`).Error); err != nil {
					return err
				}
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
		// 202604250004: Create clusters table
		{
			ID: "202604250004",
			Migrate: func(tx *gorm.DB) error {
				return tx.Exec(`CREATE TABLE IF NOT EXISTS clusters (
					id TEXT PRIMARY KEY,
					tenant_id TEXT REFERENCES tenants(id),
					name TEXT NOT NULL,
					description TEXT,
					provider TEXT NOT NULL DEFAULT 'kubernetes',
					api_server TEXT NOT NULL,
					kube_config TEXT,
					kube_config_path TEXT,
					context TEXT,
					namespace TEXT DEFAULT 'default',
					token TEXT,
					ca_data TEXT,
					status TEXT DEFAULT 'unknown',
					version TEXT,
					node_count INTEGER DEFAULT 0,
					tags TEXT,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				)`).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("clusters")
			},
		},
		// 202604250005: Create plugins table
		{
			ID: "202604250005",
			Migrate: func(tx *gorm.DB) error {
				return tx.Exec(`CREATE TABLE IF NOT EXISTS plugins (
					id TEXT PRIMARY KEY,
					tenant_id TEXT REFERENCES tenants(id),
					name TEXT NOT NULL,
					display_name TEXT,
					version TEXT DEFAULT '1.0.0',
					description TEXT,
					author TEXT,
					provider TEXT NOT NULL,
					type TEXT NOT NULL,
					config TEXT,
					enabled INTEGER DEFAULT 1,
					priority INTEGER DEFAULT 0,
					status TEXT DEFAULT 'active',
					error_msg TEXT,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					UNIQUE(tenant_id, name)
				)`).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("plugins")
			},
		},
		// 202604270001: Enhance deployments table for rollback support
		{
			ID: "202604270001",
			Migrate: func(tx *gorm.DB) error {
				// Add new columns for rollback enhancement (safe: ignores if column exists)
				if err := ignoreDuplicateColumnError(tx, tx.Exec(`ALTER TABLE deployments ADD COLUMN app_id TEXT`).Error); err != nil {
					return err
				}
				if err := ignoreDuplicateColumnError(tx, tx.Exec(`ALTER TABLE deployments ADD COLUMN previous_image TEXT`).Error); err != nil {
					return err
				}
				if err := ignoreDuplicateColumnError(tx, tx.Exec(`ALTER TABLE deployments ADD COLUMN deploy_type TEXT DEFAULT 'deploy'`).Error); err != nil {
					return err
				}
				if err := ignoreDuplicateColumnError(tx, tx.Exec(`ALTER TABLE deployments ADD COLUMN config_snapshot TEXT`).Error); err != nil {
					return err
				}
				// Create indexes for common queries
				tx.Exec(`CREATE INDEX IF NOT EXISTS idx_deployments_container_name ON deployments(container_name)`)
				tx.Exec(`CREATE INDEX IF NOT EXISTS idx_deployments_app_id ON deployments(app_id)`)
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				// SQLite: recreate table without new columns
				tx.Exec(`CREATE TABLE deployments_backup AS SELECT id, tenant_id, server_id, app_name, container_name, image, status, preflight_code, preflight_message, preflight_checks, error_message, created_at, updated_at FROM deployments`)
				tx.Exec(`DROP TABLE deployments`)
				tx.Exec(`ALTER TABLE deployments_backup RENAME TO deployments`)
				return nil
			},
		},
		// 202604270002: Create backup_records table
		{
			ID: "202604270002",
			Migrate: func(tx *gorm.DB) error {
				return tx.Exec(`CREATE TABLE IF NOT EXISTS backup_records (
					id TEXT PRIMARY KEY,
					type TEXT NOT NULL DEFAULT 'database',
					app_id TEXT,
					status TEXT NOT NULL DEFAULT 'completed',
					file_path TEXT,
					file_size INTEGER DEFAULT 0,
					trigger TEXT DEFAULT 'manual',
					error TEXT,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP
				)`).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("backup_records")
			},
		},
		// 202604270003: Add trace_id column to audit_logs
		{
			ID: "202604270003",
			Migrate: func(tx *gorm.DB) error {
				return ignoreDuplicateColumnError(tx, tx.Exec("ALTER TABLE audit_logs ADD COLUMN trace_id TEXT DEFAULT ''").Error)
			},
			Rollback: func(tx *gorm.DB) error {
				// SQLite: drop column not natively supported, recreate approach too complex
				// Just ignore - column will be unused
				return nil
			},
		},
		// 202604280001: Add OAuth fields to users table and record_hash to audit_logs
		{
			ID: "202604280001",
			Migrate: func(tx *gorm.DB) error {
				if err := ignoreDuplicateColumnError(tx, tx.Exec(`ALTER TABLE users ADD COLUMN auth_provider TEXT DEFAULT ''`).Error); err != nil {
					return err
				}
				if err := ignoreDuplicateColumnError(tx, tx.Exec(`ALTER TABLE users ADD COLUMN auth_uid TEXT DEFAULT ''`).Error); err != nil {
					return err
				}
				if err := ignoreDuplicateColumnError(tx, tx.Exec(`ALTER TABLE users ADD COLUMN avatar_url TEXT DEFAULT ''`).Error); err != nil {
					return err
				}
				// Create indexes for OAuth lookups
				tx.Exec(`CREATE INDEX IF NOT EXISTS idx_users_auth_provider ON users(auth_provider)`)
				tx.Exec(`CREATE INDEX IF NOT EXISTS idx_users_auth_uid ON users(auth_uid)`)
				// Also add record_hash to audit_logs
				if err := ignoreDuplicateColumnError(tx, tx.Exec(`ALTER TABLE audit_logs ADD COLUMN record_hash TEXT DEFAULT ''`).Error); err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				// SQLite doesn't support DROP COLUMN easily
				return nil
			},
		},
		// 202604290001: Add environment column to apps table
		{
			ID: "202604290001",
			Migrate: func(tx *gorm.DB) error {
				if err := ignoreDuplicateColumnError(tx, tx.Exec(`ALTER TABLE apps ADD COLUMN environment TEXT DEFAULT 'production'`).Error); err != nil {
					return err
				}
				tx.Exec(`CREATE INDEX IF NOT EXISTS idx_apps_environment ON apps(environment)`)
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				// SQLite doesn't support DROP COLUMN easily
				return nil
			},
		},
		// 202604300001: Create metric_records and alert_histories tables
		{
			ID: "202604300001",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&model.MetricRecord{}, &model.AlertHistory{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("metric_records", "alert_histories")
			},
		},
		// 202604300002: Create scheduled_tasks and task_executions tables
		{
			ID: "202604300002",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&model.ScheduledTask{}, &model.TaskExecution{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("task_executions", "scheduled_tasks")
			},
		},
		// 202605010002: Add 2FA fields to users table
		{
			ID: "202605010002",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&model.User{})
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Migrator().DropColumn("users", "totp_secret"); err != nil {
					return err
				}
				if err := tx.Migrator().DropColumn("users", "totp_enabled"); err != nil {
					return err
				}
				return tx.Migrator().DropColumn("users", "backup_codes")
			},
		},
		// 202605010001: Create api_keys table
		{
			ID: "202605010001",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&model.APIKey{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("api_keys")
			},
		},
		// 202605010200: Add allowed_ips and usage_count to api_keys
		{
			ID: "202605010200",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE api_keys ADD COLUMN allowed_ips TEXT").Error; err != nil {
					// Column may already exist, check if it's a duplicate column error
					if !strings.Contains(err.Error(), "duplicate column") {
						return err
					}
				}
				if err := tx.Exec("ALTER TABLE api_keys ADD COLUMN usage_count BIGINT DEFAULT 0").Error; err != nil {
					if !strings.Contains(err.Error(), "duplicate column") {
						return err
					}
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				_ = tx.Migrator().DropColumn("api_keys", "allowed_ips")
				_ = tx.Migrator().DropColumn("api_keys", "usage_count")
				return nil
			},
		},
		// 202605020001: Create alert_rules table for persistent alert rule configuration
		{
			ID: "202605020001",
			Migrate: func(tx *gorm.DB) error {
				return tx.Exec(`CREATE TABLE IF NOT EXISTS alert_rules (
					id TEXT PRIMARY KEY,
					tenant_id TEXT,
					name TEXT NOT NULL,
					metric_type TEXT NOT NULL,
					condition TEXT NOT NULL,
					threshold REAL DEFAULT 0,
					severity TEXT NOT NULL DEFAULT 'warning',
					enabled INTEGER DEFAULT 1,
					cooldown_seconds INTEGER DEFAULT 900,
					notify_channels TEXT,
					server_id TEXT,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				)`).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("alert_rules")
			},
		},
		// 202605020002: Add cloud storage columns to backup_records
		{
			ID: "202605020002",
			Migrate: func(tx *gorm.DB) error {
				if err := ignoreDuplicateColumnError(tx, tx.Exec("ALTER TABLE backup_records ADD COLUMN storage_type TEXT DEFAULT 'local'").Error); err != nil {
					return err
				}
				if err := ignoreDuplicateColumnError(tx, tx.Exec("ALTER TABLE backup_records ADD COLUMN storage_path TEXT DEFAULT ''").Error); err != nil {
					return err
				}
				if err := ignoreDuplicateColumnError(tx, tx.Exec("ALTER TABLE backup_records ADD COLUMN storage_bucket TEXT DEFAULT ''").Error); err != nil {
					return err
				}
				if err := ignoreDuplicateColumnError(tx, tx.Exec("ALTER TABLE backup_records ADD COLUMN file_checksum TEXT DEFAULT ''").Error); err != nil {
					return err
				}
				return ignoreDuplicateColumnError(tx, tx.Exec("ALTER TABLE backup_records ADD COLUMN encrypted INTEGER DEFAULT 0").Error)
			},
			Rollback: func(tx *gorm.DB) error {
				// SQLite doesn't support DROP COLUMN easily; table rebuild needed
				return nil
			},
		},
		// 202605010300: Add log_type, archived, archived_at to audit_logs
		{
			ID: "202605010300",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE audit_logs ADD COLUMN log_type TEXT DEFAULT 'operation'").Error; err != nil {
					if !strings.Contains(err.Error(), "duplicate column") {
						return err
					}
				}
				if err := tx.Exec("ALTER TABLE audit_logs ADD COLUMN archived BOOLEAN DEFAULT false").Error; err != nil {
					if !strings.Contains(err.Error(), "duplicate column") {
						return err
					}
				}
				if err := tx.Exec("ALTER TABLE audit_logs ADD COLUMN archived_at DATETIME").Error; err != nil {
					if !strings.Contains(err.Error(), "duplicate column") {
						return err
					}
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				_ = tx.Migrator().DropColumn("audit_logs", "log_type")
				_ = tx.Migrator().DropColumn("audit_logs", "archived")
				_ = tx.Migrator().DropColumn("audit_logs", "archived_at")
				return nil
			},
		},
		// 202605010400: Add onboarding_completed and last_login_at to users
		{
			ID: "202605010400",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec("ALTER TABLE users ADD COLUMN onboarding_completed BOOLEAN DEFAULT false").Error; err != nil {
					if !strings.Contains(err.Error(), "duplicate column") {
						return err
					}
				}
				if err := tx.Exec("ALTER TABLE users ADD COLUMN last_login_at DATETIME").Error; err != nil {
					if !strings.Contains(err.Error(), "duplicate column") {
						return err
					}
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				_ = tx.Migrator().DropColumn("users", "onboarding_completed")
				_ = tx.Migrator().DropColumn("users", "last_login_at")
				return nil
			},
		},
		// 202605010500: Create event_logs table
		{
			ID: "202605010500",
			Migrate: func(tx *gorm.DB) error {
				return tx.Exec(`CREATE TABLE IF NOT EXISTS event_logs (
					id TEXT PRIMARY KEY, tenant_id TEXT,
					event_type TEXT NOT NULL, topic TEXT,
					source TEXT, payload TEXT,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP
				)`).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Exec("DROP TABLE IF EXISTS event_logs").Error
			},
		},
		// 202605010600: Create alert_silences, alert_escalations, alert_groups tables
		{
			ID: "202605010600",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec(`CREATE TABLE IF NOT EXISTS alert_silences (
					id TEXT PRIMARY KEY, tenant_id TEXT,
					name TEXT, reason TEXT, matchers TEXT,
					starts_at DATETIME, ends_at DATETIME,
					created_by TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP
				)`).Error; err != nil {
					return err
				}
				if err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_silences_ends_at ON alert_silences(ends_at)`).Error; err != nil {
					return err
				}
				if err := tx.Exec(`CREATE TABLE IF NOT EXISTS alert_escalations (
					id TEXT PRIMARY KEY, tenant_id TEXT,
					name TEXT, rule_ids TEXT, steps TEXT,
					repeat_interval INTEGER DEFAULT 60,
					enabled BOOLEAN DEFAULT 1,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				)`).Error; err != nil {
					return err
				}
				return tx.Exec(`CREATE TABLE IF NOT EXISTS alert_groups (
					id TEXT PRIMARY KEY, tenant_id TEXT,
					group_key TEXT, rule_id TEXT, severity TEXT,
					alert_count INTEGER DEFAULT 1,
					first_alert_at DATETIME, last_alert_at DATETIME,
					status TEXT DEFAULT 'firing',
					resolved_at DATETIME,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				)`).Error
			},
			Rollback: func(tx *gorm.DB) error {
				tx.Exec("DROP TABLE IF EXISTS alert_groups")
				tx.Exec("DROP TABLE IF EXISTS alert_escalations")
				return tx.Exec("DROP TABLE IF EXISTS alert_silences").Error
			},
		},
		// 202605010700: Create ssh_key_pairs and ssh_authorizations tables
		{
			ID: "202605010700",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec(`CREATE TABLE IF NOT EXISTS ssh_key_pairs (
					id TEXT PRIMARY KEY, name TEXT,
					public_key TEXT, private_key TEXT,
					fingerprint TEXT, key_type TEXT DEFAULT 'rsa',
					key_bits INTEGER DEFAULT 4096,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP
				)`).Error; err != nil {
					return err
				}
				return tx.Exec(`CREATE TABLE IF NOT EXISTS ssh_authorizations (
					id TEXT PRIMARY KEY, key_pair_id TEXT,
					server_id TEXT, user TEXT DEFAULT 'root',
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP
				)`).Error
			},
			Rollback: func(tx *gorm.DB) error {
				tx.Exec("DROP TABLE IF EXISTS ssh_authorizations")
				return tx.Exec("DROP TABLE IF EXISTS ssh_key_pairs").Error
			},
		},
		// 202605010800: Create process_rules table
		{
			ID: "202605010800",
			Migrate: func(tx *gorm.DB) error {
				return tx.Exec(`CREATE TABLE IF NOT EXISTS process_rules (
					id TEXT PRIMARY KEY,
					tenant_id TEXT,
					server_id TEXT,
					name TEXT NOT NULL,
					process_pattern TEXT NOT NULL,
					restart_command TEXT,
					auto_restart INTEGER DEFAULT 0,
					max_restarts INTEGER DEFAULT 5,
					restart_count INTEGER DEFAULT 0,
					enabled INTEGER DEFAULT 1,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				)`).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Exec("DROP TABLE IF EXISTS process_rules").Error
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
