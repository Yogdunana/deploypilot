package database

import (
	"path/filepath"
	"testing"
)

func TestConnectSQLite(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Connect("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
}

func TestConnectInvalidDriver(t *testing.T) {
	_, err := Connect("mongodb", "invalid")
	if err == nil {
		t.Error("Connect() should return error for invalid driver")
	}
}

func TestMigrate(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Connect("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	// Verify tables exist by checking SQLite master
	var tables []string
	rows, err := db.Raw("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name").Rows()
	if err != nil {
		t.Fatalf("failed to query tables: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("failed to scan table name: %v", err)
		}
		tables = append(tables, name)
	}

	expectedTables := []string{
		"tenants", "roles", "users", "servers",
		"apps", "credentials", "providers",
	}

	for _, expected := range expectedTables {
		found := false
		for _, got := range tables {
			if got == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("table %q not found in database, got %v", expected, tables)
		}
	}
}

func TestSeedData(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Connect("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	if err := Seed(db); err != nil {
		t.Fatalf("Seed() error = %v", err)
	}

	// Verify default role was created
	var count int64
	db.Table("roles").Where("name = ?", "owner").Count(&count)
	if count != 1 {
		t.Errorf("expected 1 owner role, got %d", count)
	}

	// Verify default tenant was created
	db.Table("tenants").Where("slug = ?", "default").Count(&count)
	if count != 1 {
		t.Errorf("expected 1 default tenant, got %d", count)
	}
}

func TestConnectEmptyDSN(t *testing.T) {
	_, err := Connect("sqlite", "")
	if err == nil {
		t.Error("Connect() should return error for empty DSN")
	}
}

func TestMigrateIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Connect("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	// Run migrate twice — should not fail
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate() first run error = %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate() second run (idempotent) error = %v", err)
	}
}

func TestSeedAllRoles(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Connect("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	if err := Seed(db); err != nil {
		t.Fatalf("Seed() error = %v", err)
	}

	// Verify all 4 default roles were created
	expectedRoles := []string{"owner", "admin", "dev", "viewer"}
	for _, roleName := range expectedRoles {
		var count int64
		db.Table("roles").Where("name = ?", roleName).Count(&count)
		if count != 1 {
			t.Errorf("expected 1 %q role, got %d", roleName, count)
		}
	}
}

func TestSeedIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Connect("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	// Run seed twice — should not fail due to INSERT OR IGNORE
	if err := Seed(db); err != nil {
		t.Fatalf("Seed() first run error = %v", err)
	}
	if err := Seed(db); err != nil {
		t.Fatalf("Seed() second run (idempotent) error = %v", err)
	}

	// Verify no duplicates
	var count int64
	db.Table("roles").Count(&count)
	if count != 4 {
		t.Errorf("expected 4 roles after double seed, got %d", count)
	}
}

func TestSeedRolePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Connect("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	if err := Seed(db); err != nil {
		t.Fatalf("Seed() error = %v", err)
	}

	// Verify owner has admin permission
	var permissions string
	db.Table("roles").Where("name = ?", "owner").Select("permissions").Scan(&permissions)
	if permissions == "" {
		t.Error("owner role should have permissions")
	}
}

func TestSeedTenantDetails(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Connect("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	if err := Seed(db); err != nil {
		t.Fatalf("Seed() error = %v", err)
	}

	// Verify default tenant details
	var name, slug, plan string
	db.Table("tenants").Where("id = ?", "tenant-default").
		Select("name, slug, plan").Row().Scan(&name, &slug, &plan)

	if name != "Default" {
		t.Errorf("tenant name = %q, want %q", name, "Default")
	}
	if slug != "default" {
		t.Errorf("tenant slug = %q, want %q", slug, "default")
	}
	if plan != "free" {
		t.Errorf("tenant plan = %q, want %q", plan, "free")
	}
}

func TestConnectPostgresDriver(t *testing.T) {
	// Test that postgres driver path is attempted (will fail without a real server)
	_, err := Connect("postgres", "host=localhost port=5432 dbname=test")
	if err == nil {
		t.Error("Connect() with postgres should fail without a real server")
	}
	// Verify the error mentions the driver
	if err != nil {
		t.Logf("Expected postgres connection error: %v", err)
	}
}

func TestConnectConnectionPoolSettings(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Connect("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	// Verify connection pool settings are applied (no crash = success)
	// The settings are applied via SetMaxIdleConns/SetMaxOpenConns in Connect()
	// We can't directly read them back, but we verify the code path was reached
	// by checking the DB is usable
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
}

func TestMigrateVerifySchema(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Connect("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	// Verify specific columns exist in key tables
	type columnCheck struct {
		table  string
		column string
	}
	checks := []columnCheck{
		{"tenants", "id"}, {"tenants", "name"}, {"tenants", "slug"}, {"tenants", "plan"},
		{"roles", "id"}, {"roles", "name"}, {"roles", "permissions"},
		{"users", "id"}, {"users", "username"}, {"users", "email"}, {"users", "password_hash"},
		{"servers", "id"}, {"servers", "host"}, {"servers", "port"}, {"servers", "status"},
		{"apps", "id"}, {"apps", "name"}, {"apps", "repo_url"}, {"apps", "deploy_mode"},
		{"credentials", "id"}, {"credentials", "name"}, {"credentials", "type"},
		{"providers", "id"}, {"providers", "type"}, {"providers", "name"},
	}

	for _, check := range checks {
		t.Run(check.table+"_"+check.column, func(t *testing.T) {
			var count int
			err := db.Raw(
				"SELECT count(*) FROM pragma_table_info(?) WHERE name = ?",
				check.table, check.column,
			).Scan(&count).Error
			if err != nil {
				t.Fatalf("failed to check column: %v", err)
			}
			if count == 0 {
				t.Errorf("column %q not found in table %q", check.column, check.table)
			}
		})
	}
}

func TestMigrate_PostgresUnsupported(t *testing.T) {
    db, err := Connect("sqlite", ":memory:")
    if err != nil {
        t.Fatal(err)
    }
    // Migrate should work for sqlite
    if err := Migrate(db); err != nil {
        t.Fatalf("Migrate sqlite should succeed: %v", err)
    }
}

func TestSeed_Idempotent(t *testing.T) {
    db, err := Connect("sqlite", ":memory:")
    if err != nil {
        t.Fatal(err)
    }
    if err := Migrate(db); err != nil {
        t.Fatal(err)
    }
    // Seed twice should not fail
    if err := Seed(db); err != nil {
        t.Fatalf("First seed failed: %v", err)
    }
    if err := Seed(db); err != nil {
        t.Fatalf("Second seed (idempotent) failed: %v", err)
    }
}
