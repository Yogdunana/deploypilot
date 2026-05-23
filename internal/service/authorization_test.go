package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

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

func TestCheckResourceAccess_ClusterTenantLevel(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	db.Exec(`CREATE TABLE users (id TEXT PRIMARY KEY, tenant_id TEXT)`)
	db.Exec(`CREATE TABLE clusters (id TEXT PRIMARY KEY, name TEXT, tenant_id TEXT)`)

	db.Exec("INSERT INTO users (id, tenant_id) VALUES ('user-1', 'tenant-1')")
	db.Exec("INSERT INTO users (id, tenant_id) VALUES ('user-2', 'tenant-2')")
	db.Exec("INSERT INTO clusters (id, name, tenant_id) VALUES ('cluster-1', 'Cluster1', 'tenant-1')")
	db.Exec("INSERT INTO clusters (id, name, tenant_id) VALUES ('cluster-2', 'Cluster2', 'tenant-2')")

	if !CheckResourceAccess(db, "cluster", "cluster-1", "dev", "user-1") {
		t.Error("dev should access cluster in same tenant")
	}
	if !CheckResourceAccess(db, "cluster", "cluster-1", "viewer", "user-1") {
		t.Error("viewer should access cluster in same tenant")
	}
	if CheckResourceAccess(db, "cluster", "cluster-2", "dev", "user-1") {
		t.Error("dev should not access cluster in different tenant")
	}
	if CheckResourceAccess(db, "cluster", "cluster-2", "viewer", "user-1") {
		t.Error("viewer should not access cluster in different tenant")
	}
}

func TestCheckResourceAccess_ClusterOwnerAdmin(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	db.Exec(`CREATE TABLE users (id TEXT PRIMARY KEY, tenant_id TEXT)`)
	db.Exec(`CREATE TABLE clusters (id TEXT PRIMARY KEY, name TEXT, tenant_id TEXT)`)

	db.Exec("INSERT INTO users (id, tenant_id) VALUES ('user-1', 'tenant-1')")
	db.Exec("INSERT INTO clusters (id, name, tenant_id) VALUES ('cluster-1', 'Cluster1', 'tenant-2')")

	if !CheckResourceAccess(db, "cluster", "cluster-1", "owner", "user-1") {
		t.Error("owner should access any cluster")
	}
	if !CheckResourceAccess(db, "cluster", "cluster-1", "admin", "user-1") {
		t.Error("admin should access any cluster")
	}
}

func TestCheckResourceAccess_ClusterNonexistentUser(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	db.Exec(`CREATE TABLE users (id TEXT PRIMARY KEY, tenant_id TEXT)`)
	db.Exec(`CREATE TABLE clusters (id TEXT PRIMARY KEY, name TEXT, tenant_id TEXT)`)

	if CheckResourceAccess(db, "cluster", "cluster-1", "dev", "nonexistent-user") {
		t.Error("should return false for nonexistent user")
	}
}

func TestCheckResourceAccessCached_OwnerAdminNoCache(t *testing.T) {
	bridge := &Bridge{}

	result := bridge.CheckResourceAccessCached(context.Background(), "app", "app-1", "owner", "user-1")
	if !result {
		t.Error("owner should have access")
	}

	result = bridge.CheckResourceAccessCached(context.Background(), "app", "app-1", "admin", "user-1")
	if !result {
		t.Error("admin should have access")
	}
}

func TestCheckResourceAccessCached_WithMockCache(t *testing.T) {
	db := setupAuthTestDB(t)
	seedTestData(db)

	cache := &mockCache{store: make(map[string]string)}
	cache.store["perm:user-1:app:app-1"] = "true"
	cache.store["perm:user-1:app:app-2"] = "false"

	bridge := &Bridge{
		DB:    db,
		Cache: cache,
	}

	if !bridge.CheckResourceAccessCached(context.Background(), "app", "app-1", "dev", "user-1") {
		t.Error("should return cached true for own app")
	}

	if bridge.CheckResourceAccessCached(context.Background(), "app", "app-2", "dev", "user-1") {
		t.Error("should return cached false for other user's app")
	}

	hasAccess := bridge.CheckResourceAccessCached(context.Background(), "app", "app-1", "dev", "user-1")
	if !hasAccess {
		t.Error("should return cached true (set by first call) for own app")
	}

	noAccess := bridge.CheckResourceAccessCached(context.Background(), "app", "app-2", "dev", "user-1")
	if noAccess {
		t.Error("should return cached false for other user's app")
	}
}

func TestCheckResourceAccessCached_CacheMissFallsBackToDB(t *testing.T) {
	db := setupAuthTestDB(t)
	seedTestData(db)

	cache := &mockCache{store: make(map[string]string)}

	bridge := &Bridge{
		DB:    db,
		Cache: cache,
	}

	if !bridge.CheckResourceAccessCached(context.Background(), "app", "app-1", "dev", "user-1") {
		t.Error("should access own app via DB fallback")
	}

	if bridge.CheckResourceAccessCached(context.Background(), "app", "app-2", "dev", "user-1") {
		t.Error("should not access other user's app via DB fallback")
	}

	if len(cache.store) == 0 {
		t.Error("cache should have been populated after DB fallback")
	}
}

func TestCheckResourceAccessCached_InvalidCacheValue(t *testing.T) {
	db := setupAuthTestDB(t)
	seedTestData(db)

	cache := &mockCache{store: make(map[string]string)}
	cache.store["perm:user-1:app:app-1"] = "invalid"

	bridge := &Bridge{
		DB:    db,
		Cache: cache,
	}

	if !bridge.CheckResourceAccessCached(context.Background(), "app", "app-1", "dev", "user-1") {
		t.Error("should fall back to DB for invalid cache value")
	}
}

type mockCache struct {
	store map[string]string
}

func (m *mockCache) Get(ctx context.Context, key string) (string, error) {
	if v, ok := m.store[key]; ok {
		return v, nil
	}
	return "", ErrCacheMiss
}

func (m *mockCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	m.store[key] = value
	return nil
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	delete(m.store, key)
	return nil
}

func (m *mockCache) GetJSON(ctx context.Context, key string, dest interface{}) error {
	v, err := m.Get(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(v), dest)
}

func (m *mockCache) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return m.Set(ctx, key, string(data), ttl)
}

func (m *mockCache) Close() error {
	return nil
}