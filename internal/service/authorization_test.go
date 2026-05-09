package service

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	// Create tables with user_id column for RBAC testing
	// (model structs don't include user_id, so AutoMigrate won't create it)
	db.Exec(`CREATE TABLE apps (id TEXT PRIMARY KEY, name TEXT, user_id TEXT, tenant_id TEXT)`)
	db.Exec(`CREATE TABLE servers (id TEXT PRIMARY KEY, name TEXT, user_id TEXT, tenant_id TEXT)`)
	db.Exec(`CREATE TABLE credentials (id TEXT PRIMARY KEY, name TEXT, user_id TEXT, tenant_id TEXT)`)
	return db
}

func seedTestData(db *gorm.DB) {
	db.Exec("INSERT INTO apps (id, name, user_id, tenant_id) VALUES ('app-1', 'App1', 'user-1', 'tenant-default')")
	db.Exec("INSERT INTO apps (id, name, user_id, tenant_id) VALUES ('app-2', 'App2', 'user-2', 'tenant-default')")
	db.Exec("INSERT INTO servers (id, name, user_id, tenant_id) VALUES ('srv-1', 'Server1', 'user-1', 'tenant-default')")
	db.Exec("INSERT INTO credentials (id, name, user_id, tenant_id) VALUES ('cred-1', 'Cred1', 'user-1', 'tenant-default')")
}

func TestCheckResourceAccess_OwnerCanAccessAll(t *testing.T) {
	db := setupAuthTestDB(t)
	seedTestData(db)

	if !CheckResourceAccess(db, "app", "app-1", "owner", "user-2") {
		t.Error("owner should access any app")
	}
	if !CheckResourceAccess(db, "app", "app-2", "admin", "user-1") {
		t.Error("admin should access any app")
	}
}

func TestCheckResourceAccess_OwnResources(t *testing.T) {
	db := setupAuthTestDB(t)
	seedTestData(db)

	if !CheckResourceAccess(db, "app", "app-1", "dev", "user-1") {
		t.Error("dev should access own app")
	}
	if CheckResourceAccess(db, "app", "app-2", "viewer", "user-1") {
		t.Error("viewer should not access other user's app")
	}
}

func TestCheckResourceAccess_NonexistentResource(t *testing.T) {
	db := setupAuthTestDB(t)
	seedTestData(db)

	if CheckResourceAccess(db, "app", "nonexistent", "dev", "user-1") {
		t.Error("should return false for nonexistent resource")
	}
}

func TestCheckResourceAccess_AllResourceTypes(t *testing.T) {
	db := setupAuthTestDB(t)
	seedTestData(db)

	if !CheckResourceAccess(db, "server", "srv-1", "dev", "user-1") {
		t.Error("dev should access own server")
	}
	if !CheckResourceAccess(db, "credential", "cred-1", "viewer", "user-1") {
		t.Error("viewer should access own credential")
	}
	if CheckResourceAccess(db, "credential", "cred-1", "viewer", "user-2") {
		t.Error("viewer should not access other user's credential")
	}
}

func TestCheckResourceAccess_UnknownType(t *testing.T) {
	db := setupAuthTestDB(t)
	if CheckResourceAccess(db, "unknown", "id-1", "dev", "user-1") {
		t.Error("unknown resource type should return false")
	}
}