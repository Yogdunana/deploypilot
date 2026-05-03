package service

import (
	"context"
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

func TestCheckResourceAccess_ClusterResourceType(t *testing.T) {
	db := setupAuthTestDB(t)

	db.Exec(`CREATE TABLE clusters (id TEXT PRIMARY KEY, name TEXT, tenant_id TEXT)`)
	db.Exec(`CREATE TABLE users (id TEXT PRIMARY KEY, tenant_id TEXT)`)

	db.Exec("INSERT INTO clusters (id, name, tenant_id) VALUES ('cluster-1', 'Cluster1', 'tenant-default')")
	db.Exec("INSERT INTO clusters (id, name, tenant_id) VALUES ('cluster-2', 'Cluster2', 'tenant-other')")
	db.Exec("INSERT INTO users (id, tenant_id) VALUES ('user-1', 'tenant-default')")
	db.Exec("INSERT INTO users (id, tenant_id) VALUES ('user-2', 'tenant-other')")

	if !CheckResourceAccess(db, "cluster", "cluster-1", "dev", "user-1") {
		t.Error("dev should access cluster in same tenant")
	}
	if CheckResourceAccess(db, "cluster", "cluster-2", "dev", "user-1") {
		t.Error("dev should not access cluster in different tenant")
	}
	if !CheckResourceAccess(db, "cluster", "cluster-1", "viewer", "user-1") {
		t.Error("viewer should access cluster in same tenant")
	}

	db.Exec("INSERT INTO users (id, tenant_id) VALUES ('user-no-tenant', '')")
	if CheckResourceAccess(db, "cluster", "cluster-1", "dev", "user-no-tenant") {
		t.Error("user without tenant should not access any cluster")
	}
}

func setupBridgeWithCache(t *testing.T) (*Bridge, *mockCache) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	db.Exec(`CREATE TABLE apps (id TEXT PRIMARY KEY, name TEXT, user_id TEXT, tenant_id TEXT)`)
	db.Exec(`CREATE TABLE servers (id TEXT PRIMARY KEY, name TEXT, user_id TEXT, tenant_id TEXT)`)
	db.Exec(`CREATE TABLE credentials (id TEXT PRIMARY KEY, name TEXT, user_id TEXT, tenant_id TEXT)`)
	db.Exec("INSERT INTO apps (id, name, user_id, tenant_id) VALUES ('app-cache-1', 'AppCache1', 'user-1', 'tenant-default')")

	cache := &mockCache{data: make(map[string]string)}
	bridge := &Bridge{
		DB:    db,
		Cache: cache,
	}
	return bridge, cache
}

type mockCache struct {
	data map[string]string
}

func (m *mockCache) Get(_ context.Context, key string) (string, error) {
	if val, ok := m.data[key]; ok {
		return val, nil
	}
	return "", ErrCacheMiss
}

func (m *mockCache) Set(_ context.Context, key, value string, _ time.Duration) error {
	m.data[key] = value
	return nil
}

func (m *mockCache) Delete(_ context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *mockCache) GetJSON(_ context.Context, key string, dest interface{}) error {
	return ErrCacheMiss
}

func (m *mockCache) SetJSON(_ context.Context, key string, value interface{}, _ time.Duration) error {
	return nil
}

func (m *mockCache) Close() error {
	return nil
}

func TestCheckResourceAccessCached_OwnerBypassesCache(t *testing.T) {
	bridge, cache := setupBridgeWithCache(t)

	result := bridge.CheckResourceAccessCached(context.Background(), "app", "app-cache-1", "owner", "user-2")
	if !result {
		t.Error("owner should have access")
	}
	if len(cache.data) != 0 {
		t.Error("owner/admin access should not be cached")
	}
}

func TestCheckResourceAccessCached_AdminBypassesCache(t *testing.T) {
	bridge, cache := setupBridgeWithCache(t)

	result := bridge.CheckResourceAccessCached(context.Background(), "app", "app-cache-1", "admin", "user-2")
	if !result {
		t.Error("admin should have access")
	}
	if len(cache.data) != 0 {
		t.Error("owner/admin access should not be cached")
	}
}

func TestCheckResourceAccessCached_CacheHit(t *testing.T) {
	bridge, cache := setupBridgeWithCache(t)

	cache.data["perm:user-2:app:app-cache-1"] = "true"

	result := bridge.CheckResourceAccessCached(context.Background(), "app", "app-cache-1", "dev", "user-2")
	if !result {
		t.Error("expected cached result to be true")
	}
}

func TestCheckResourceAccessCached_CacheMiss_QueryDB(t *testing.T) {
	bridge, cache := setupBridgeWithCache(t)

	result := bridge.CheckResourceAccessCached(context.Background(), "app", "app-cache-1", "dev", "user-1")
	if !result {
		t.Error("dev should have access to own resource")
	}

	cacheKey := "perm:user-1:app:app-cache-1"
	if _, exists := cache.data[cacheKey]; !exists {
		t.Error("cache miss should query DB and cache the result")
	}
}

func TestCheckResourceAccessCached_CacheMiss_AccessDenied(t *testing.T) {
	bridge, cache := setupBridgeWithCache(t)

	result := bridge.CheckResourceAccessCached(context.Background(), "app", "app-cache-1", "dev", "user-2")
	if result {
		t.Error("dev should not have access to other user's resource")
	}

	cacheKey := "perm:user-2:app:app-cache-1"
	if _, exists := cache.data[cacheKey]; !exists {
		t.Error("cache miss should query DB and cache the denied result")
	}
}

func TestCheckResourceAccessCached_NoCache(t *testing.T) {
	bridge, _ := setupBridgeWithCache(t)
	bridge.Cache = nil

	result := bridge.CheckResourceAccessCached(context.Background(), "app", "app-cache-1", "dev", "user-1")
	if !result {
		t.Error("should fallback to DB when no cache")
	}
}