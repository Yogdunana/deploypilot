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
	db.Exec(`CREATE TABLE users (id TEXT PRIMARY KEY, username TEXT, tenant_id TEXT)`)
	db.Exec(`CREATE TABLE apps (id TEXT PRIMARY KEY, name TEXT, user_id TEXT, tenant_id TEXT)`)
	db.Exec(`CREATE TABLE servers (id TEXT PRIMARY KEY, name TEXT, user_id TEXT, tenant_id TEXT)`)
	db.Exec(`CREATE TABLE credentials (id TEXT PRIMARY KEY, name TEXT, user_id TEXT, tenant_id TEXT)`)
	db.Exec(`CREATE TABLE clusters (id TEXT PRIMARY KEY, name TEXT, tenant_id TEXT)`)
	db.Exec(`CREATE TABLE registries (id TEXT PRIMARY KEY, name TEXT, tenant_id TEXT)`)
	return db
}

func seedTestData(db *gorm.DB) {
	db.Exec("INSERT INTO users (id, username, tenant_id) VALUES ('user-1', 'u1', 'tenant-default')")
	db.Exec("INSERT INTO users (id, username, tenant_id) VALUES ('user-2', 'u2', 'tenant-default')")
	db.Exec("INSERT INTO users (id, username, tenant_id) VALUES ('user-x', 'ux', 'tenant-other')")

	db.Exec("INSERT INTO apps (id, name, user_id, tenant_id) VALUES ('app-1', 'App1', 'user-1', 'tenant-default')")
	db.Exec("INSERT INTO apps (id, name, user_id, tenant_id) VALUES ('app-2', 'App2', 'user-2', 'tenant-default')")
	db.Exec("INSERT INTO servers (id, name, user_id, tenant_id) VALUES ('srv-1', 'Server1', 'user-1', 'tenant-default')")
	db.Exec("INSERT INTO credentials (id, name, user_id, tenant_id) VALUES ('cred-1', 'Cred1', 'user-1', 'tenant-default')")

	// A resource that belongs to another tenant entirely.
	db.Exec("INSERT INTO apps (id, name, user_id, tenant_id) VALUES ('app-x', 'AppX', 'user-x', 'tenant-other')")
	// A legacy row: no owner recorded, still inside the tenant.
	db.Exec("INSERT INTO apps (id, name, user_id, tenant_id) VALUES ('app-legacy', 'Legacy', '', 'tenant-default')")
}

// The tenant boundary must hold for every role, admin and owner included.
func TestCheckResourceAccess_CrossTenantDenied(t *testing.T) {
	db := setupAuthTestDB(t)
	seedTestData(db)

	for _, role := range []string{"viewer", "dev", "admin", "owner"} {
		if CheckResourceAccess(db, "app", "app-x", role, "user-1") {
			t.Errorf("%s from tenant-default should not reach app-x in tenant-other", role)
		}
	}
}

// Rows written before user_id existed keep an empty owner. Locking them out would
// make every pre-existing resource unreachable after upgrade.
func TestCheckResourceAccess_LegacyRowHasNoOwner(t *testing.T) {
	db := setupAuthTestDB(t)
	seedTestData(db)

	if !CheckResourceAccess(db, "app", "app-legacy", "dev", "user-2") {
		t.Error("dev should still reach a tenant resource that has no recorded owner")
	}
}

func TestCheckResourceAccess_TenantLevelResources(t *testing.T) {
	db := setupAuthTestDB(t)
	seedTestData(db)

	db.Exec("INSERT INTO clusters (id, name, tenant_id) VALUES ('cls-1', 'C1', 'tenant-default')")
	db.Exec("INSERT INTO registries (id, name, tenant_id) VALUES ('reg-1', 'R1', 'tenant-default')")

	if !CheckResourceAccess(db, "cluster", "cls-1", "dev", "user-2") {
		t.Error("clusters are tenant-level: any member of the tenant should reach them")
	}
	if !CheckResourceAccess(db, "registry", "reg-1", "viewer", "user-1") {
		t.Error("registries are tenant-level: any member of the tenant should reach them")
	}
	if CheckResourceAccess(db, "cluster", "cls-1", "owner", "user-x") {
		t.Error("owner of another tenant should not reach this cluster")
	}
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