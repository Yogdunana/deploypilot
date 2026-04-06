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
