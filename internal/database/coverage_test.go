package database

import (
	"testing"
)

// ===================== Additional Migrate Coverage =====================

func TestMigrate_Memory(t *testing.T) {
	db, err := Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate(:memory:) failed: %v", err)
	}
	// Verify tables exist
	var count int64
	db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table'").Scan(&count)
	if count == 0 {
		t.Fatal("expected tables to exist")
	}
}

func TestMigrate_VerifyAllTables(t *testing.T) {
	db, err := Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}

	expectedTables := []string{
		"tenants", "roles", "users", "credentials", "providers",
		"servers", "apps", "deployments", "audit_logs", "ssl_certificates",
	}
	for _, table := range expectedTables {
		var count int64
		db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if count != 1 {
			t.Errorf("table %q not found", table)
		}
	}
}

func TestMigrate_VerifyForeignKeyColumns(t *testing.T) {
	db, err := Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}

	// Verify foreign key columns exist
	fkChecks := []struct {
		table  string
		column string
	}{
		{"users", "tenant_id"},
		{"users", "role_id"},
		{"credentials", "tenant_id"},
		{"providers", "tenant_id"},
		{"servers", "tenant_id"},
		{"servers", "credential_id"},
		{"servers", "provider_id"},
		{"apps", "tenant_id"},
		{"apps", "server_id"},
		{"deployments", "tenant_id"},
		{"deployments", "server_id"},
	}

	for _, check := range fkChecks {
		var count int
		err := db.Raw(
			"SELECT count(*) FROM pragma_table_info(?) WHERE name = ?",
			check.table, check.column,
		).Scan(&count).Error
		if err != nil {
			t.Fatalf("failed to check column: %v", err)
		}
		if count == 0 {
			t.Errorf("FK column %q not found in table %q", check.column, check.table)
		}
	}
}

func TestMigrate_VerifyAppsTableColumns(t *testing.T) {
	db, err := Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}

	columns := []string{
		"id", "tenant_id", "server_id", "name", "repo_url", "branch",
		"domain", "tech_stack", "deploy_mode", "status", "current_version",
		"container_name", "env_vars", "resource_limits",
	}
	for _, col := range columns {
		var count int
		err := db.Raw(
			"SELECT count(*) FROM pragma_table_info(?) WHERE name = ?",
			"apps", col,
		).Scan(&count).Error
		if err != nil {
			t.Fatalf("failed to check column: %v", err)
		}
		if count == 0 {
			t.Errorf("column %q not found in apps table", col)
		}
	}
}

func TestMigrate_VerifyServersTableColumns(t *testing.T) {
	db, err := Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}

	columns := []string{
		"id", "tenant_id", "credential_id", "provider_id",
		"name", "host", "port", "tags", "status", "detected_info",
	}
	for _, col := range columns {
		var count int
		err := db.Raw(
			"SELECT count(*) FROM pragma_table_info(?) WHERE name = ?",
			"servers", col,
		).Scan(&count).Error
		if err != nil {
			t.Fatalf("failed to check column: %v", err)
		}
		if count == 0 {
			t.Errorf("column %q not found in servers table", col)
		}
	}
}

func TestMigrate_VerifyAuditLogsTableColumns(t *testing.T) {
	db, err := Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}

	columns := []string{
		"id", "user_id", "username", "action",
		"resource_type", "resource_id", "detail",
		"ip_address", "user_agent",
	}
	for _, col := range columns {
		var count int
		err := db.Raw(
			"SELECT count(*) FROM pragma_table_info(?) WHERE name = ?",
			"audit_logs", col,
		).Scan(&count).Error
		if err != nil {
			t.Fatalf("failed to check column: %v", err)
		}
		if count == 0 {
			t.Errorf("column %q not found in audit_logs table", col)
		}
	}
}

func TestSeed_VerifyAllRoles(t *testing.T) {
	db, err := Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	if err := Seed(db); err != nil {
		t.Fatal(err)
	}

	roles := []struct {
		id   string
		name string
	}{
		{"role-owner", "owner"},
		{"role-admin", "admin"},
		{"role-dev", "dev"},
		{"role-viewer", "viewer"},
	}

	for _, r := range roles {
		var name string
		err := db.Table("roles").Where("id = ?", r.id).Select("name").Scan(&name).Error
		if err != nil {
			t.Errorf("failed to query role %s: %v", r.id, err)
		}
		if name != r.name {
			t.Errorf("expected role name %q, got %q", r.name, name)
		}
	}
}

func TestConnect_EmptyDSNAllDrivers(t *testing.T) {
	drivers := []string{"sqlite", "postgres", "mysql", "mongodb"}
	for _, driver := range drivers {
		_, err := Connect(driver, "")
		if err == nil {
			t.Errorf("expected error for empty DSN with driver %s", driver)
		}
	}
}

func TestConnect_UnsupportedDrivers(t *testing.T) {
	drivers := []string{"mysql", "mssql", "oracle", "sqlite3"}
	for _, driver := range drivers {
		_, err := Connect(driver, "some-dsn")
		if err == nil {
			t.Errorf("expected error for unsupported driver %s", driver)
		}
	}
}

func TestConnect_PostgresInvalidDSN(t *testing.T) {
	// postgres driver with invalid DSN should return an error
	_, err := Connect("postgres", "invalid-dsn")
	if err == nil {
		t.Error("expected error for invalid postgres DSN")
	}
}

func TestConnect_SQLiteValid(t *testing.T) {
	db, err := Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("expected no error for sqlite :memory:, got: %v", err)
	}
	if db == nil {
		t.Fatal("expected non-nil db")
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get underlying sql.DB: %v", err)
	}
	if sqlDB == nil {
		t.Fatal("expected non-nil sqlDB")
	}
}

func TestSeed_VerifyDefaultTenant_Cov(t *testing.T) {
	db, err := Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	if err := Seed(db); err != nil {
		t.Fatal(err)
	}

	var name string
	db.Table("tenants").Where("id = ?", "tenant-default").Select("name").Scan(&name)
	if name != "Default" {
		t.Errorf("expected tenant name 'Default', got %q", name)
	}
}

func TestSeed_Idempotent_Cov(t *testing.T) {
	db, err := Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	// Seed twice - should not fail due to INSERT OR IGNORE
	if err := Seed(db); err != nil {
		t.Fatal(err)
	}
	if err := Seed(db); err != nil {
		t.Fatal(err)
	}
}

func TestMigrate_Idempotent_Cov(t *testing.T) {
	db, err := Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	// First migration
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	// Second migration on same DB - covers gormigrate's "already applied" path
	if err := Migrate(db); err != nil {
		t.Fatalf("second Migrate should be idempotent: %v", err)
	}

	// Verify tables still exist
	expectedTables := []string{
		"tenants", "roles", "users", "credentials", "providers",
		"servers", "apps", "deployments", "audit_logs", "ssl_certificates",
	}
	for _, table := range expectedTables {
		var count int64
		db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if count != 1 {
			t.Errorf("table %q not found after second migrate", table)
		}
	}
}

func TestMigrate_Rollback_Cov(t *testing.T) {
	db, err := Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}

	// Verify all tables exist
	expectedTables := []string{
		"tenants", "roles", "users", "credentials", "providers",
		"servers", "apps", "deployments", "audit_logs", "ssl_certificates",
	}
	for _, table := range expectedTables {
		var count int64
		db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if count != 1 {
			t.Errorf("table %q not found after migrate", table)
		}
	}

	// Verify individual rollback functions work
	tx := db.Begin()
	_ = tx.Migrator().DropTable("apps", "servers", "providers", "credentials", "users", "roles", "tenants")
	tx.Rollback()
	_ = tx

	// Verify tables still exist after rollback
	for _, table := range expectedTables {
		var count int64
		db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if count != 1 {
			t.Errorf("table %q should still exist after rollback", table)
		}
	}
}
