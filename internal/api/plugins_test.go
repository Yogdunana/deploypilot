package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Yogdunana/deploypilot/internal/crypto"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupPluginTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	// Create tables
	db.Exec(`CREATE TABLE IF NOT EXISTS tenants (
		id TEXT PRIMARY KEY, name TEXT NOT NULL, slug TEXT UNIQUE NOT NULL,
		plan TEXT DEFAULT 'free', max_servers INTEGER DEFAULT 5, max_apps INTEGER DEFAULT 20,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS roles (
		id TEXT PRIMARY KEY, name TEXT UNIQUE NOT NULL, permissions TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY, tenant_id TEXT, role_id TEXT,
		username TEXT UNIQUE NOT NULL, email TEXT UNIQUE NOT NULL, password_hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS plugins (
		id TEXT PRIMARY KEY, tenant_id TEXT, name TEXT NOT NULL,
		display_name TEXT, version TEXT DEFAULT '1.0.0', description TEXT,
		author TEXT, provider TEXT NOT NULL, type TEXT NOT NULL,
		config TEXT, enabled INTEGER DEFAULT 1, priority INTEGER DEFAULT 0,
		status TEXT DEFAULT 'active', error_msg TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(tenant_id, name)
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER, username TEXT,
		action TEXT, resource_type TEXT, resource_id TEXT, detail TEXT,
		ip_address TEXT, user_agent TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	// Seed roles
	db.Exec(`INSERT INTO roles (id, name, permissions) VALUES
		('role-owner', 'owner', '{"admin":true}'),
		('role-admin', 'admin', '{"admin":false}'),
		('role-dev', 'dev', '{"deploy":true}'),
		('role-viewer', 'viewer', '{"deploy":false}')`)

	// Seed default tenant
	db.Exec(`INSERT INTO tenants (id, name, slug) VALUES ('tenant-default', 'Default', 'default')`)

	// Initialize model global DB so model functions (getDB) work
	encKey := crypto.NewEncryptionKey()
	model.InitDB(db, encKey)

	return db
}

func setupPluginTestRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})

	api := r.Group("/api/v1")

	// Public auth routes
	authGroup := api.Group("/auth")
	{
		authGroup.POST("/register", Register(db))
		authGroup.POST("/login", Login(db, nil))
	}

	// Protected routes
	protected := api.Group("")
	protected.Use(authMiddlewareForTest())
	{
		pluginHandler := NewPluginHandler(nil)
		plugins := protected.Group("/plugins")
		{
			plugins.GET("", pluginHandler.ListPlugins())
			plugins.POST("", pluginHandler.CreatePlugin())
			plugins.GET("/:id", pluginHandler.GetPlugin())
			plugins.PUT("/:id", pluginHandler.UpdatePlugin())
			plugins.DELETE("/:id", pluginHandler.DeletePlugin())
			plugins.POST("/:id/enable", pluginHandler.EnablePlugin())
			plugins.POST("/:id/disable", pluginHandler.DisablePlugin())
			plugins.POST("/:id/reload", pluginHandler.ReloadPlugin())
		}
	}

	return r
}

// authMiddlewareForTest is a simplified auth middleware for testing.
func authMiddlewareForTest() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip auth for testing - just set a user context
		c.Set("user_id", "test-user")
		c.Set("user_role", "admin")
		c.Next()
	}
}

func makePluginRequest(r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var bodyReader io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(data)
	}

	req, _ := http.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestListPlugins(t *testing.T) {
	db := setupPluginTestDB(t)

	// Seed plugins
	db.Exec(`INSERT INTO plugins (id, tenant_id, name, provider, type, enabled, status) VALUES
		('plg-1', 'tenant-default', 'dns-cf', 'dns', 'cloudflare', 1, 'active'),
		('plg-2', 'tenant-default', 'notify-webhook', 'notify', 'webhook', 1, 'active')`)

	router := setupPluginTestRouter(db)

	w := makePluginRequest(router, "GET", "/api/v1/plugins", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "success" {
		t.Errorf("status = %v, want success", resp["status"])
	}
	data, ok := resp["data"].([]interface{})
	if !ok || len(data) != 2 {
		t.Errorf("expected 2 plugins, got %v", resp["data"])
	}
}

func TestListPluginsWithFilter(t *testing.T) {
	db := setupPluginTestDB(t)

	db.Exec(`INSERT INTO plugins (id, tenant_id, name, provider, type, enabled, status) VALUES
		('plg-1', 'tenant-default', 'dns-cf', 'dns', 'cloudflare', 1, 'active'),
		('plg-2', 'tenant-default', 'notify-webhook', 'notify', 'webhook', 1, 'active')`)

	router := setupPluginTestRouter(db)

	w := makePluginRequest(router, "GET", "/api/v1/plugins?provider=dns", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data, ok := resp["data"].([]interface{})
	if !ok || len(data) != 1 {
		t.Errorf("expected 1 plugin with provider=dns, got %v", resp["data"])
	}
}

func TestCreatePlugin(t *testing.T) {
	db := setupPluginTestDB(t)

	router := setupPluginTestRouter(db)

	body := map[string]interface{}{
		"name":        "test-plugin",
		"display_name": "Test Plugin",
		"provider":    "dns",
		"type":        "cloudflare",
		"description": "A test plugin",
	}

	w := makePluginRequest(router, "POST", "/api/v1/plugins", body)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "success" {
		t.Errorf("status = %v, want success", resp["status"])
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data should be a map")
	}
	if data["name"] != "test-plugin" {
		t.Errorf("name = %v, want test-plugin", data["name"])
	}
	if data["provider"] != "dns" {
		t.Errorf("provider = %v, want dns", data["provider"])
	}
}

func TestCreatePluginMissingFields(t *testing.T) {
	db := setupPluginTestDB(t)

	router := setupPluginTestRouter(db)

	// Missing provider and type
	body := map[string]interface{}{
		"name": "incomplete-plugin",
	}

	w := makePluginRequest(router, "POST", "/api/v1/plugins", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetPlugin(t *testing.T) {
	db := setupPluginTestDB(t)

	db.Exec(`INSERT INTO plugins (id, tenant_id, name, provider, type, enabled, status) VALUES
		('plg-get', 'tenant-default', 'dns-cf', 'dns', 'cloudflare', 1, 'active')`)

	router := setupPluginTestRouter(db)

	w := makePluginRequest(router, "GET", "/api/v1/plugins/plg-get", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data should be a map")
	}
	if data["id"] != "plg-get" {
		t.Errorf("id = %v, want plg-get", data["id"])
	}
}

func TestGetPluginNotFound(t *testing.T) {
	db := setupPluginTestDB(t)

	router := setupPluginTestRouter(db)

	w := makePluginRequest(router, "GET", "/api/v1/plugins/nonexistent", nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdatePlugin(t *testing.T) {
	db := setupPluginTestDB(t)

	db.Exec(`INSERT INTO plugins (id, tenant_id, name, provider, type, enabled, status) VALUES
		('plg-update', 'tenant-default', 'dns-cf', 'dns', 'cloudflare', 1, 'active')`)

	router := setupPluginTestRouter(db)

	body := map[string]interface{}{
		"description": "Updated description",
		"priority":    10,
	}

	w := makePluginRequest(router, "PUT", "/api/v1/plugins/plg-update", body)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data should be a map")
	}
	if data["description"] != "Updated description" {
		t.Errorf("description = %v, want 'Updated description'", data["description"])
	}
}

func TestDeletePlugin(t *testing.T) {
	db := setupPluginTestDB(t)

	db.Exec(`INSERT INTO plugins (id, tenant_id, name, provider, type, enabled, status) VALUES
		('plg-delete', 'tenant-default', 'dns-cf', 'dns', 'cloudflare', 1, 'active')`)

	router := setupPluginTestRouter(db)

	w := makePluginRequest(router, "DELETE", "/api/v1/plugins/plg-delete", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data should be a map")
	}
	if data["id"] != "plg-delete" {
		t.Errorf("id = %v, want plg-delete", data["id"])
	}

	// Verify deleted
	var count int64
	db.Table("plugins").Where("id = ?", "plg-delete").Count(&count)
	if count != 0 {
		t.Error("plugin should be deleted from DB")
	}
}

func TestEnablePluginNoManager(t *testing.T) {
	db := setupPluginTestDB(t)

	router := setupPluginTestRouter(db) // nil lifecycle manager

	w := makePluginRequest(router, "POST", "/api/v1/plugins/plg-enable/enable", nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when no lifecycle manager, got %d", w.Code)
	}
}

func TestDisablePluginNoManager(t *testing.T) {
	db := setupPluginTestDB(t)

	router := setupPluginTestRouter(db) // nil lifecycle manager

	w := makePluginRequest(router, "POST", "/api/v1/plugins/plg-disable/disable", nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when no lifecycle manager, got %d", w.Code)
	}
}

func TestReloadPluginNoManager(t *testing.T) {
	db := setupPluginTestDB(t)

	router := setupPluginTestRouter(db) // nil lifecycle manager

	w := makePluginRequest(router, "POST", "/api/v1/plugins/plg-reload/reload", nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when no lifecycle manager, got %d", w.Code)
	}
}
