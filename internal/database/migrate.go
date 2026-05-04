package database

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/gorm"
)

// RunMigrations runs golang-migrate SQL migrations from the migrations/ directory.
// This is the primary migration system for new installations.
func RunMigrations(dsn, driver string) error {
	dsnURL := buildMigrateURL(dsn, driver)

	m, err := migrate.New("file://migrations", dsnURL)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer func() {
		if _, closeErr := m.Close(); closeErr != nil {
			slog.Warn("failed to close migrate instance", "error", closeErr)
		}
	}()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %w", err)
	}

	version, dirty, verr := m.Version()
	if verr == nil {
		slog.Info("database migrations completed", "version", version, "dirty", dirty)
	} else {
		slog.Info("database migrations completed (no version tracking)")
	}

	return nil
}

// RunMigrationsDown rolls back the last migration.
func RunMigrationsDown(dsn, driver string) error {
	dsnURL := buildMigrateURL(dsn, driver)

	m, err := migrate.New("file://migrations", dsnURL)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer func() {
		if _, closeErr := m.Close(); closeErr != nil {
			slog.Warn("failed to close migrate instance", "error", closeErr)
		}
	}()

	if err := m.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration rollback failed: %w", err)
	}

	version, dirty, verr := m.Version()
	if verr == nil {
		slog.Info("database rollback completed", "version", version, "dirty", dirty)
	} else {
		slog.Info("database rollback completed (no version tracking)")
	}

	return nil
}

// MigrationStatus returns the current migration version and dirty state.
func MigrationStatus(dsn, driver string) (uint, bool, error) {
	dsnURL := buildMigrateURL(dsn, driver)

	m, err := migrate.New("file://migrations", dsnURL)
	if err != nil {
		return 0, false, fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer func() {
		if _, closeErr := m.Close(); closeErr != nil {
			slog.Warn("failed to close migrate instance", "error", closeErr)
		}
	}()

	version, dirty, err := m.Version()
	if err != nil {
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}

	return version, dirty, nil
}

// buildMigrateURL constructs the database URL for golang-migrate.
func buildMigrateURL(dsn, driver string) string {
	switch driver {
	case "postgres":
		return convertPostgresDSN(dsn)
	case "sqlite":
		return fmt.Sprintf("sqlite3://%s", dsn)
	default:
		return dsn
	}
}

// convertPostgresDSN converts a key=value format DSN to a postgres:// URL format
// required by golang-migrate. If the DSN is already a URL, it is returned as-is.
func convertPostgresDSN(dsn string) string {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		return dsn
	}
	params := make(map[string]string)
	for _, part := range strings.Fields(dsn) {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			params[kv[0]] = kv[1]
		}
	}
	host := params["host"]
	port := params["port"]
	user := params["user"]
	password := params["password"]
	dbname := params["dbname"]
	sslmode := params["sslmode"]
	if sslmode == "" {
		sslmode = "disable"
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, password, host, port, dbname, sslmode)
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

// MigrateLegacy runs all database migrations using gormigrate.
// This is the fallback migration system for existing databases that were
// already set up with gormigrate. Migrations are idempotent.
func MigrateLegacy(db *gorm.DB) error {
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
		// 202605010900: Create system_snapshots table
		{
			ID: "202605010900",
			Migrate: func(tx *gorm.DB) error {
				return tx.Exec(`CREATE TABLE IF NOT EXISTS system_snapshots (
					id TEXT PRIMARY KEY,
					tenant_id TEXT,
					server_id TEXT,
					name TEXT NOT NULL,
					description TEXT,
					file_count INTEGER DEFAULT 0,
					total_size INTEGER DEFAULT 0,
					checksum TEXT,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP
				)`).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Exec("DROP TABLE IF EXISTS system_snapshots").Error
			},
		},
		// 202605011000: Create toolbox_scripts table
		{
			ID: "202605011000",
			Migrate: func(tx *gorm.DB) error {
				return tx.Exec(`CREATE TABLE IF NOT EXISTS toolbox_scripts (
					id TEXT PRIMARY KEY,
					tenant_id TEXT,
					name TEXT NOT NULL,
					description TEXT,
					category TEXT,
					script TEXT NOT NULL,
					is_built_in INTEGER DEFAULT 0,
					enabled INTEGER DEFAULT 1,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				)`).Error
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Exec("DROP TABLE IF EXISTS toolbox_scripts").Error
			},
		},
		// 202605011100: Create monitors, monitor_check_results, heartbeats tables
		{
			ID: "202605011100",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Exec(`CREATE TABLE IF NOT EXISTS monitors (
					id TEXT PRIMARY KEY,
					tenant_id TEXT,
					name TEXT NOT NULL,
					type TEXT NOT NULL,
					target TEXT NOT NULL,
					interval INTEGER DEFAULT 60,
					timeout INTEGER DEFAULT 10,
					retries INTEGER DEFAULT 3,
					status TEXT DEFAULT 'unknown',
					enabled INTEGER DEFAULT 1,
					last_check TEXT,
					last_status TEXT,
					uptime REAL DEFAULT 100,
					total_checks INTEGER DEFAULT 0,
					up_checks INTEGER DEFAULT 0,
					avg_latency REAL DEFAULT 0,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				)`).Error; err != nil {
					return err
				}
				if err := tx.Exec(`CREATE TABLE IF NOT EXISTS monitor_check_results (
					id TEXT PRIMARY KEY,
					monitor_id TEXT NOT NULL,
					status TEXT,
					status_code INTEGER DEFAULT 0,
					latency REAL DEFAULT 0,
					message TEXT,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP
				)`).Error; err != nil {
					return err
				}
				return tx.Exec(`CREATE TABLE IF NOT EXISTS heartbeats (
					id TEXT PRIMARY KEY,
					tenant_id TEXT,
					name TEXT NOT NULL,
					token TEXT NOT NULL UNIQUE,
					interval INTEGER DEFAULT 60,
					timeout INTEGER DEFAULT 120,
					status TEXT DEFAULT 'unknown',
					last_beat TEXT,
					enabled INTEGER DEFAULT 1,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				)`).Error
			},
			Rollback: func(tx *gorm.DB) error {
				tx.Exec("DROP TABLE IF EXISTS monitor_check_results")
				tx.Exec("DROP TABLE IF EXISTS monitors")
				return tx.Exec("DROP TABLE IF EXISTS heartbeats").Error
			},
		},
		// 202605020100: Create grafana_custom_dashboards and grafana_sync_logs tables
		{
			ID: "202605020100",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&model.GrafanaCustomDashboard{}, &model.GrafanaSyncLog{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("grafana_sync_logs", "grafana_custom_dashboards")
			},
		},
		// 202605020200: Create OAuth2 tables for API Open Platform
		{
			ID: "202605020200",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&model.OAuth2Client{}, &model.OAuth2Authorization{}, &model.OAuth2Token{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("oauth2_tokens", "oauth2_authorizations", "oauth2_clients")
			},
		},
		// 202605020300: Create plugin_configs table for event-driven plugin system
		{
			ID: "202605020300",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&model.PluginConfig{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("plugin_configs")
			},
		},
		// 202605030001: Create audit_hashes table for hash chain verification
		{
			ID: "202605030001",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&model.AuditHash{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("audit_hashes")
			},
		},
		// 202605030002: Create ip_whitelists table for per-user IP whitelist
		{
			ID: "202605030002",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&model.IPWhitelist{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("ip_whitelists")
			},
		},
		// 202605030003: Create devices table for device binding
		{
			ID: "202605030003",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&model.Device{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("devices")
			},
		},
		// 202605030004: Create signing_keys table for Ed25519 code signing
		{
			ID: "202605030004",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&model.SigningKey{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("signing_keys")
			},
		},
		// 202605030005: Create licenses table for license management
		{
			ID: "202605030005",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&model.License{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("licenses")
			},
		},
		// 202605030006: Add license v2 fields (use_type, tier, addons, issuer fields)
		{
			ID: "202605030006",
			Migrate: func(tx *gorm.DB) error {
				cols := []string{
					"ALTER TABLE licenses ADD COLUMN use_type VARCHAR(20) NOT NULL DEFAULT 'non_commercial'",
					"ALTER TABLE licenses ADD COLUMN tier VARCHAR(20) NOT NULL DEFAULT 'community'",
					"ALTER TABLE licenses ADD COLUMN addons TEXT",
					"ALTER TABLE licenses ADD COLUMN issuer_role VARCHAR(20) NOT NULL DEFAULT 'user'",
					"ALTER TABLE licenses ADD COLUMN issued_to VARCHAR(64)",
					"ALTER TABLE licenses ADD COLUMN max_issued INTEGER DEFAULT 0",
					"ALTER TABLE licenses ADD COLUMN issued_count INTEGER DEFAULT 0",
				}
				for _, col := range cols {
					if err := tx.Exec(col).Error; err != nil {
						if !strings.Contains(err.Error(), "duplicate column") {
							return err
						}
					}
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("licenses")
			},
		},
		// 202605030007: Create feature_flags table
		{
			ID: "202605030007",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&model.FeatureFlag{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("feature_flags")
			},
		},
		// 202605030008: Create feature_flag_overrides table
		{
			ID: "202605030008",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&model.FeatureFlagOverride{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("feature_flag_overrides")
			},
		},
		// 202605030009: Create trial_periods table
		{
			ID: "202605030009",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&model.TrialPeriod{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("trial_periods")
			},
		},
		// 202605030010: Create degradation_audits table
		{
			ID: "202605030010",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&model.DegradationAudit{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("degradation_audits")
			},
		},
		// 202605040001: Create license_signing_keys table
		{
			ID: "202605040001",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&model.LicenseSigningKey{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("license_signing_keys")
			},
		},
	})

	// Use InitSchema for initial creation (faster than Migrate)
	if err := m.Migrate(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// tableExists checks if a table exists in the database.
func tableExists(db *gorm.DB, tableName string) (bool, error) {
	var count int64
	err := db.Raw(
		"SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?",
		tableName,
	).Scan(&count).Error
	if err != nil {
		// Try PostgreSQL syntax if SQLite syntax fails
		err = db.Raw(
			"SELECT count(*) FROM information_schema.tables WHERE table_name=?",
			tableName,
		).Scan(&count).Error
		if err != nil {
			return false, fmt.Errorf("failed to check table existence: %w", err)
		}
	}
	return count > 0, nil
}

// Migrate is the primary migration entry point. It detects which migration
// system was previously used (golang-migrate or gormigrate) and applies
// migrations accordingly:
//   - If schema_migrations table exists: database uses golang-migrate, run it.
//   - If migrations table exists: database uses gormigrate, run legacy migrations.
//   - If neither exists: fresh install, use golang-migrate.
func Migrate(db *gorm.DB, optionalDriverAndDSN ...string) error {
	// Extract optional driver and dsn for golang-migrate
	var driver, dsn string
	if len(optionalDriverAndDSN) >= 2 {
		driver = optionalDriverAndDSN[0]
		dsn = optionalDriverAndDSN[1]
	}

	// Check if golang-migrate's tracking table exists
	golangMigrateExists, err := tableExists(db, "schema_migrations")
	if err != nil {
		return fmt.Errorf("failed to check for golang-migrate table: %w", err)
	}

	if golangMigrateExists && driver != "" && dsn != "" {
		slog.Info("detected golang-migrate schema, running golang-migrate migrations")
		return RunMigrations(dsn, driver)
	}

	// Check if gormigrate's tracking table exists
	gormigrateExists, err := tableExists(db, "migrations")
	if err != nil {
		return fmt.Errorf("failed to check for gormigrate table: %w", err)
	}

	if gormigrateExists {
		slog.Info("detected gormigrate schema, running legacy migrations")
		if err := MigrateLegacy(db); err != nil {
			return fmt.Errorf("legacy migration failed: %w", err)
		}
		slog.Info("legacy migrations completed successfully")
		return nil
	}

	// Fresh install — use golang-migrate if driver/dsn provided, otherwise gormigrate
	if driver != "" && dsn != "" {
		slog.Info("fresh database detected, running golang-migrate migrations")
		return RunMigrations(dsn, driver)
	}

	// Fallback to gormigrate for backward compatibility
	slog.Info("running gormigrate migrations (no driver/dsn provided)")
	return MigrateLegacy(db)
}

// Seed inserts default data required for the application to function.
// This should be called after Migrate().
