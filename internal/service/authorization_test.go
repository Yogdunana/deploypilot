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
	// Multi-tenant schema: resources are scoped by tenant_id (not user_id).
	db.Exec(`CREATE TABLE users (id TEXT PRIMARY KEY, tenant_id TEXT)`)
	db.Exec(`CREATE TABLE apps (id TEXT PRIMARY KEY, name TEXT, tenant_id TEXT)`)
	db.Exec(`CREATE TABLE servers (id TEXT PRIMARY KEY, name TEXT, tenant_id TEXT)`)
	db.Exec(`CREATE TABLE credentials (id TEXT PRIMARY KEY, name TEXT, tenant_id TEXT)`)
	db.Exec(`CREATE TABLE clusters (id TEXT PRIMARY KEY, name TEXT, tenant_id TEXT)`)
	db.Exec(`CREATE TABLE registries (id TEXT PRIMARY KEY, name TEXT, tenant_id TEXT)`)
	return db
}

func seedTestData(db *gorm.DB) {
	// Two tenants, each with users.
	db.Exec("INSERT INTO users (id, tenant_id) VALUES ('user-t1-a', 'tenant-1')")
	db.Exec("INSERT INTO users (id, tenant_id) VALUES ('user-t1-b', 'tenant-1')")
	db.Exec("INSERT INTO users (id, tenant_id) VALUES ('user-t2-a', 'tenant-2')")
	db.Exec("INSERT INTO users (id, tenant_id) VALUES ('user-orphan', '')")

	// Resources scoped to tenant-1 and tenant-2.
	db.Exec("INSERT INTO apps (id, name, tenant_id) VALUES ('app-t1', 'AppT1', 'tenant-1')")
	db.Exec("INSERT INTO apps (id, name, tenant_id) VALUES ('app-t2', 'AppT2', 'tenant-2')")
	db.Exec("INSERT INTO servers (id, name, tenant_id) VALUES ('srv-t1', 'SrvT1', 'tenant-1')")
	db.Exec("INSERT INTO servers (id, name, tenant_id) VALUES ('srv-t2', 'SrvT2', 'tenant-2')")
	db.Exec("INSERT INTO credentials (id, name, tenant_id) VALUES ('cred-t1', 'CredT1', 'tenant-1')")
	db.Exec("INSERT INTO credentials (id, name, tenant_id) VALUES ('cred-t2', 'CredT2', 'tenant-2')")
	db.Exec("INSERT INTO clusters (id, name, tenant_id) VALUES ('cls-t1', 'ClsT1', 'tenant-1')")
	db.Exec("INSERT INTO clusters (id, name, tenant_id) VALUES ('cls-t2', 'ClsT2', 'tenant-2')")
	db.Exec("INSERT INTO registries (id, name, tenant_id) VALUES ('reg-t1', 'RegT1', 'tenant-1')")
	db.Exec("INSERT INTO registries (id, name, tenant_id) VALUES ('reg-t2', 'RegT2', 'tenant-2')")
}

func TestCheckResourceAccess_OwnerAdminBypassTenantCheck(t *testing.T) {
	db := setupAuthTestDB(t)
	seedTestData(db)

	if !CheckResourceAccess(db, "app", "app-t1", "owner", "user-t2-a") {
		t.Error("owner should bypass tenant-scoped checks")
	}
	if !CheckResourceAccess(db, "app", "app-t2", "admin", "user-t1-a") {
		t.Error("admin should bypass tenant-scoped checks")
	}
}

func TestCheckResourceAccess_DevViewerAccessOwnTenantResources(t *testing.T) {
	db := setupAuthTestDB(t)
	seedTestData(db)

	if !CheckResourceAccess(db, "app", "app-t1", "dev", "user-t1-a") {
		t.Error("dev should access tenant-1 app")
	}
	if !CheckResourceAccess(db, "app", "app-t1", "viewer", "user-t1-a") {
		t.Error("viewer should access tenant-1 app")
	}
}

func TestCheckResourceAccess_DevViewerBlockedFromOtherTenant(t *testing.T) {
	db := setupAuthTestDB(t)
	seedTestData(db)

	if CheckResourceAccess(db, "app", "app-t1", "dev", "user-t2-a") {
		t.Error("dev from tenant-2 should NOT access tenant-1 app")
	}
	if CheckResourceAccess(db, "server", "srv-t1", "viewer", "user-t2-a") {
		t.Error("viewer from tenant-2 should NOT access tenant-1 server")
	}
	if CheckResourceAccess(db, "credential", "cred-t1", "dev", "user-t2-a") {
		t.Error("dev from tenant-2 should NOT access tenant-1 credential")
	}
}

func TestCheckResourceAccess_NonexistentResource(t *testing.T) {
	db := setupAuthTestDB(t)
	seedTestData(db)

	if CheckResourceAccess(db, "app", "nonexistent", "dev", "user-t1-a") {
		t.Error("should return false for nonexistent resource")
	}
}

func TestCheckResourceAccess_AllResourceTypes(t *testing.T) {
	db := setupAuthTestDB(t)
	seedTestData(db)

	cases := []struct {
		resourceType string
		resourceID   string
	}{
		{"app", "app-t1"},
		{"server", "srv-t1"},
		{"credential", "cred-t1"},
		{"cluster", "cls-t1"},
		{"registry", "reg-t1"},
	}
	for _, c := range cases {
		if !CheckResourceAccess(db, c.resourceType, c.resourceID, "dev", "user-t1-a") {
			t.Errorf("dev should access tenant-1 %s (id=%s)", c.resourceType, c.resourceID)
		}
		if CheckResourceAccess(db, c.resourceType, c.resourceID, "dev", "user-t2-a") {
			t.Errorf("dev from tenant-2 should NOT access tenant-1 %s (id=%s)", c.resourceType, c.resourceID)
		}
	}
}

func TestCheckResourceAccess_UnknownType(t *testing.T) {
	db := setupAuthTestDB(t)
	if CheckResourceAccess(db, "unknown", "id-1", "dev", "user-t1-a") {
		t.Error("unknown resource type should return false")
	}
}

func TestCheckResourceAccess_EmptyRoleOrUserID(t *testing.T) {
	db := setupAuthTestDB(t)
	seedTestData(db)

	if CheckResourceAccess(db, "app", "app-t1", "", "user-t1-a") {
		t.Error("empty role should deny access")
	}
	if CheckResourceAccess(db, "app", "app-t1", "dev", "") {
		t.Error("empty userID should deny access")
	}
}

func TestCheckResourceAccess_UserWithNoTenant(t *testing.T) {
	db := setupAuthTestDB(t)
	seedTestData(db)

	if CheckResourceAccess(db, "app", "app-t1", "dev", "user-orphan") {
		t.Error("user with empty tenant_id should NOT access any resource")
	}
}

func TestCheckResourceAccess_NonexistentUser(t *testing.T) {
	db := setupAuthTestDB(t)
	seedTestData(db)

	if CheckResourceAccess(db, "app", "app-t1", "dev", "user-nobody") {
		t.Error("nonexistent user should NOT access any resource")
	}
}