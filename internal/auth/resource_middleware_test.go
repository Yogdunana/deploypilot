package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	db.Exec(`CREATE TABLE apps (id TEXT PRIMARY KEY, name TEXT, user_id TEXT, tenant_id TEXT)`)
	db.Exec(`CREATE TABLE servers (id TEXT PRIMARY KEY, name TEXT, user_id TEXT, tenant_id TEXT)`)
	db.Exec(`CREATE TABLE credentials (id TEXT PRIMARY KEY, name TEXT, user_id TEXT, tenant_id TEXT)`)
	db.Exec(`CREATE TABLE users (id TEXT PRIMARY KEY, tenant_id TEXT)`)
	db.Exec(`CREATE TABLE clusters (id TEXT PRIMARY KEY, name TEXT, tenant_id TEXT)`)
	return db
}

func seedResourceData(db *gorm.DB) {
	db.Exec("INSERT INTO apps (id, name, user_id, tenant_id) VALUES ('app-1', 'App1', 'user-1', 'tenant-1')")
	db.Exec("INSERT INTO apps (id, name, user_id, tenant_id) VALUES ('app-2', 'App2', 'user-2', 'tenant-1')")
	db.Exec("INSERT INTO servers (id, name, user_id, tenant_id) VALUES ('srv-1', 'Server1', 'user-1', 'tenant-1')")
	db.Exec("INSERT INTO credentials (id, name, user_id, tenant_id) VALUES ('cred-1', 'Cred1', 'user-1', 'tenant-1')")
	db.Exec("INSERT INTO users (id, tenant_id) VALUES ('user-1', 'tenant-1')")
	db.Exec("INSERT INTO users (id, tenant_id) VALUES ('user-2', 'tenant-1')")
	db.Exec("INSERT INTO clusters (id, name, tenant_id) VALUES ('cluster-1', 'Cluster1', 'tenant-1')")
}

func TestRequireResourceAccess_MissingResourceID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)

	handler := RequireResourceAccess(db, "app", "id")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/apps/", nil) // no id param
	c.Set(string(UserIDKey), "user-1")
	c.Set(string(RoleKey), "dev")

	handler(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
	if !c.IsAborted() {
		t.Error("context should be aborted for missing resource ID")
	}
}

func TestRequireResourceAccess_OwnerCanAccessAny(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	seedResourceData(db)

	handler := RequireResourceAccess(db, "app", "id")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/apps/app-1", nil)
	c.Params = []gin.Param{{Key: "id", Value: "app-1"}}
	c.Set(string(UserIDKey), "user-2") // different user
	c.Set(string(RoleKey), "owner")

	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if c.IsAborted() {
		t.Error("owner should be able to access any resource")
	}
}

func TestRequireResourceAccess_AdminCanAccessAny(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	seedResourceData(db)

	handler := RequireResourceAccess(db, "app", "id")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/apps/app-1", nil)
	c.Params = []gin.Param{{Key: "id", Value: "app-1"}}
	c.Set(string(UserIDKey), "user-2")
	c.Set(string(RoleKey), "admin")

	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if c.IsAborted() {
		t.Error("admin should be able to access any resource")
	}
}

func TestRequireResourceAccess_DevCanAccessOwnResource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	seedResourceData(db)

	handler := RequireResourceAccess(db, "app", "id")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/apps/app-1", nil)
	c.Params = []gin.Param{{Key: "id", Value: "app-1"}}
	c.Set(string(UserIDKey), "user-1") // owner of app-1
	c.Set(string(RoleKey), "dev")

	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if c.IsAborted() {
		t.Error("dev should be able to access own resource")
	}
}

func TestRequireResourceAccess_DevCannotAccessOthersResource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	seedResourceData(db)

	handler := RequireResourceAccess(db, "app", "id")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/apps/app-2", nil)
	c.Params = []gin.Param{{Key: "id", Value: "app-2"}}
	c.Set(string(UserIDKey), "user-1") // not owner of app-2
	c.Set(string(RoleKey), "dev")

	handler(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
	if !c.IsAborted() {
		t.Error("dev should not be able to access others' resources")
	}
}

func TestRequireResourceAccess_ViewerCanAccessOwnResource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	seedResourceData(db)

	handler := RequireResourceAccess(db, "credential", "id")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/credentials/cred-1", nil)
	c.Params = []gin.Param{{Key: "id", Value: "cred-1"}}
	c.Set(string(UserIDKey), "user-1")
	c.Set(string(RoleKey), "viewer")

	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestRequireResourceAccess_ViewerCannotAccessOthersResource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	seedResourceData(db)

	handler := RequireResourceAccess(db, "credential", "id")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/credentials/cred-1", nil)
	c.Params = []gin.Param{{Key: "id", Value: "cred-1"}}
	c.Set(string(UserIDKey), "user-2") // not owner of cred-1
	c.Set(string(RoleKey), "viewer")

	handler(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestRequireResourceAccess_ServerResourceType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	seedResourceData(db)

	handler := RequireResourceAccess(db, "server", "id")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/servers/srv-1", nil)
	c.Params = []gin.Param{{Key: "id", Value: "srv-1"}}
	c.Set(string(UserIDKey), "user-1")
	c.Set(string(RoleKey), "viewer")

	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestRequireResourceAccess_NonexistentResource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	seedResourceData(db)

	handler := RequireResourceAccess(db, "app", "id")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/apps/nonexistent", nil)
	c.Params = []gin.Param{{Key: "id", Value: "nonexistent"}}
	c.Set(string(UserIDKey), "user-1")
	c.Set(string(RoleKey), "dev")

	handler(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
	if !c.IsAborted() {
		t.Error("access to nonexistent resource should be denied")
	}
}

func TestRequireResourceAccess_UnknownResourceType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	seedResourceData(db)

	// Use a dev role which will hit the default case (unknown type -> return false)
	handler := RequireResourceAccess(db, "unknown", "id")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/unknown/id", nil)
	c.Params = []gin.Param{{Key: "id", Value: "some-id"}}
	c.Set(string(UserIDKey), "user-1")
	c.Set(string(RoleKey), "dev")

	handler(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
	if !c.IsAborted() {
		t.Error("access to unknown resource type should be denied for dev role")
	}
}

func TestRequireResourceAccess_OwnerCanAccessAnyResourceType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	seedResourceData(db)

	// Owner/admin bypasses resource type checks - can access even unknown types
	handler := RequireResourceAccess(db, "unknown", "id")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/unknown/id", nil)
	c.Params = []gin.Param{{Key: "id", Value: "some-id"}}
	c.Set(string(UserIDKey), "user-1")
	c.Set(string(RoleKey), "owner")

	handler(c)

	// Note: owner gets true from CheckResourceAccess before resource type validation
	// This is by design - owner/admin can access any resource
	if c.IsAborted() {
		t.Error("owner should be able to access any resource type (by design)")
	}
}

func TestRequireResourceAccess_NoUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	seedResourceData(db)

	handler := RequireResourceAccess(db, "app", "id")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/apps/app-1", nil)
	c.Params = []gin.Param{{Key: "id", Value: "app-1"}}
	// No UserID or Role set

	handler(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}
