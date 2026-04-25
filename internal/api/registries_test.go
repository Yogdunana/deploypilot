package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Yogdunana/deploypilot/internal/crypto"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// registryTestDB creates a test DB with the registries table.
func registryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupTestDB(t)
	db.Exec(`CREATE TABLE IF NOT EXISTS registries (
		id TEXT PRIMARY KEY,
		tenant_id TEXT,
		name TEXT NOT NULL,
		provider TEXT NOT NULL,
		url TEXT NOT NULL,
		username TEXT,
		password TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	// Initialize model DB for CRUD operations
	encKey := []byte("abcdefghijklmnopqrstuvwxyz123456")
	model.InitDB(db, encKey)
	return db
}

// registryTestRouter creates a router with registry routes registered (no auth middleware).
func registryTestRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})

	api := r.Group("/api/v1")
	registries := api.Group("/registries")
	{
		registries.GET("", ListRegistries())
		registries.POST("", CreateRegistry())
		registries.GET("/:id", GetRegistry())
		registries.PUT("/:id", UpdateRegistry())
		registries.DELETE("/:id", DeleteRegistry())
	}
	return r
}

func TestListRegistries_Empty(t *testing.T) {
	db := registryTestDB(t)
	defer db.Exec("VACUUM")

	r := registryTestRouter(db)
	w := makeRequest(r, "GET", "/api/v1/registries", nil, "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "success" {
		t.Fatalf("expected success status, got: %v", resp["status"])
	}
	data := resp["data"].([]interface{})
	if len(data) != 0 {
		t.Errorf("expected empty list, got %d items", len(data))
	}
}

func TestCreateRegistry_Success(t *testing.T) {
	db := registryTestDB(t)
	defer db.Exec("VACUUM")

	r := registryTestRouter(db)
	w := makeRequest(r, "POST", "/api/v1/registries", map[string]string{
		"name":     "my-docker-hub",
		"provider": "docker_hub",
		"url":      "https://registry.hub.docker.com/v2/",
		"username": "myuser",
		"password": "mypassword",
	}, "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})

	if data["name"] != "my-docker-hub" {
		t.Errorf("expected name my-docker-hub, got %v", data["name"])
	}
	if data["provider"] != "docker_hub" {
		t.Errorf("expected provider docker_hub, got %v", data["provider"])
	}
	if data["url"] != "https://registry.hub.docker.com/v2/" {
		t.Errorf("expected url https://registry.hub.docker.com/v2/, got %v", data["url"])
	}
	if data["username"] != "myuser" {
		t.Errorf("expected username myuser, got %v", data["username"])
	}
	// Password must not be exposed
	if _, ok := data["password"]; ok {
		t.Error("password should not be exposed in JSON response")
	}
	if data["id"] == "" {
		t.Error("expected non-empty id")
	}
}

func TestCreateRegistry_MissingFields(t *testing.T) {
	db := registryTestDB(t)
	defer db.Exec("VACUUM")

	r := registryTestRouter(db)
	// Missing provider and url
	w := makeRequest(r, "POST", "/api/v1/registries", map[string]string{
		"name": "incomplete-registry",
	}, "")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateRegistry_DefaultTenantID(t *testing.T) {
	db := registryTestDB(t)
	defer db.Exec("VACUUM")

	r := registryTestRouter(db)
	w := makeRequest(r, "POST", "/api/v1/registries", map[string]string{
		"name":     "default-tenant-reg",
		"provider": "ghcr",
		"url":      "https://ghcr.io/v2/",
	}, "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["tenant_id"] != "tenant-default" {
		t.Errorf("expected tenant_id tenant-default, got %v", data["tenant_id"])
	}
}

func TestGetRegistry_Success(t *testing.T) {
	db := registryTestDB(t)
	defer db.Exec("VACUUM")

	// Create a registry via model
	reg, err := model.CreateRegistry("tenant-default", "test-reg", "harbor", "https://harbor.example.com/v2/", "admin", "secret123")
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	r := registryTestRouter(db)
	w := makeRequest(r, "GET", "/api/v1/registries/"+reg.ID, nil, "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["name"] != "test-reg" {
		t.Errorf("expected name test-reg, got %v", data["name"])
	}
	if data["provider"] != "harbor" {
		t.Errorf("expected provider harbor, got %v", data["provider"])
	}
}

func TestGetRegistry_NotFound(t *testing.T) {
	db := registryTestDB(t)
	defer db.Exec("VACUUM")

	r := registryTestRouter(db)
	w := makeRequest(r, "GET", "/api/v1/registries/nonexistent-id", nil, "")

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListRegistries_WithData(t *testing.T) {
	db := registryTestDB(t)
	defer db.Exec("VACUUM")

	// Create two registries
	_, _ = model.CreateRegistry("tenant-default", "reg-1", "docker_hub", "https://registry.hub.docker.com/v2/", "user1", "pass1")
	_, _ = model.CreateRegistry("tenant-default", "reg-2", "ghcr", "https://ghcr.io/v2/", "user2", "pass2")

	r := registryTestRouter(db)
	w := makeRequest(r, "GET", "/api/v1/registries", nil, "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	if len(data) != 2 {
		t.Fatalf("expected 2 registries, got %d", len(data))
	}
}

func TestUpdateRegistry_Success(t *testing.T) {
	db := registryTestDB(t)
	defer db.Exec("VACUUM")

	reg, _ := model.CreateRegistry("tenant-default", "original-name", "docker_hub", "https://registry.hub.docker.com/v2/", "user", "pass")

	r := registryTestRouter(db)
	w := makeRequest(r, "PUT", "/api/v1/registries/"+reg.ID, map[string]string{
		"name":     "updated-name",
		"provider": "ghcr",
		"url":      "https://ghcr.io/v2/",
		"username": "newuser",
		"password": "newpass",
	}, "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["name"] != "updated-name" {
		t.Errorf("expected name updated-name, got %v", data["name"])
	}
	if data["provider"] != "ghcr" {
		t.Errorf("expected provider ghcr, got %v", data["provider"])
	}
	if data["username"] != "newuser" {
		t.Errorf("expected username newuser, got %v", data["username"])
	}
}

func TestDeleteRegistry_Success(t *testing.T) {
	db := registryTestDB(t)
	defer db.Exec("VACUUM")

	reg, _ := model.CreateRegistry("tenant-default", "to-delete", "acr", "https://myacr.azurecr.io/v2/", "user", "pass")

	r := registryTestRouter(db)
	w := makeRequest(r, "DELETE", "/api/v1/registries/"+reg.ID, nil, "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["message"] != "registry deleted" {
		t.Errorf("expected 'registry deleted' message, got %v", data["message"])
	}

	// Verify it's gone
	_, err := model.GetRegistry(reg.ID)
	if err == nil {
		t.Error("expected registry to be deleted")
	}
}

func TestRegistryPasswordEncryption(t *testing.T) {
	db := registryTestDB(t)
	defer db.Exec("VACUUM")

	// Create registry via model
	reg, err := model.CreateRegistry("tenant-default", "enc-test", "docker_hub", "https://registry.hub.docker.com/v2/", "user", "super-secret")
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Verify password is encrypted in DB
	var row struct {
		Password string
	}
	db.Raw("SELECT password FROM registries WHERE id = ?", reg.ID).Scan(&row)
	if row.Password == "super-secret" {
		t.Error("password should be encrypted in database, not stored as plaintext")
	}

	// Verify we can decrypt it back
	decrypted, err := crypto.Decrypt([]byte("abcdefghijklmnopqrstuvwxyz123456"), row.Password)
	if err != nil {
		t.Fatalf("failed to decrypt password: %v", err)
	}
	if decrypted != "super-secret" {
		t.Errorf("expected decrypted password 'super-secret', got '%s'", decrypted)
	}
}

func TestListRegistries_NoPasswordExposure(t *testing.T) {
	db := registryTestDB(t)
	defer db.Exec("VACUUM")

	_, _ = model.CreateRegistry("tenant-default", "secret-reg", "docker_hub", "https://registry.hub.docker.com/v2/", "user", "should-not-appear")

	r := registryTestRouter(db)
	w := makeRequest(r, "GET", "/api/v1/registries", nil, "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Ensure password is not in response body
	bodyStr := w.Body.String()
	if contains(bodyStr, "should-not-appear") {
		t.Error("password should not be exposed in list response")
	}
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && jsonContains(s, substr)
}

func jsonContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
