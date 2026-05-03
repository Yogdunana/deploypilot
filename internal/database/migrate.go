package database

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
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
		if closeErr := m.Close(); closeErr != nil {
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
		if closeErr := m.Close(); closeErr != nil {
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
		if closeErr := m.Close(); closeErr != nil {
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
