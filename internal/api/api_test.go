package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/crypto"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) *gorm.DB {
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
	db.Exec(`CREATE TABLE IF NOT EXISTS credentials (
		id TEXT PRIMARY KEY, tenant_id TEXT, name TEXT NOT NULL,
		type TEXT NOT NULL, encrypted_value TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS providers (
		id TEXT PRIMARY KEY, tenant_id TEXT, type TEXT NOT NULL,
		name TEXT NOT NULL, config TEXT, enabled INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS servers (
		id TEXT PRIMARY KEY, tenant_id TEXT, credential_id TEXT, provider_id TEXT,
		name TEXT NOT NULL, host TEXT NOT NULL, port INTEGER DEFAULT 22,
		tags TEXT, status TEXT DEFAULT 'unknown', detected_info TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS apps (
		id TEXT PRIMARY KEY, tenant_id TEXT, server_id TEXT,
		name TEXT NOT NULL, repo_url TEXT NOT NULL, branch TEXT DEFAULT 'main',
		domain TEXT, tech_stack TEXT DEFAULT 'docker', deploy_mode TEXT DEFAULT 'api',
		status TEXT DEFAULT 'pending', current_version TEXT, container_name TEXT,
		env_vars TEXT, resource_limits TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS deployments (
		id TEXT PRIMARY KEY, tenant_id TEXT, server_id TEXT,
		app_name TEXT, container_name TEXT, image TEXT,
		status TEXT DEFAULT 'deploying', preflight_code TEXT,
		preflight_message TEXT, preflight_checks TEXT, error_message TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	// Seed roles
	db.Exec(`INSERT INTO roles (id, name, permissions) VALUES
		('role-owner', 'owner', '{"admin":true}'),
		('role-admin', 'admin', '{"admin":false}'),
		('role-dev', 'dev', '{"deploy":true}'),
		('role-viewer', 'viewer', '{"deploy":false}')`)

	// Seed default tenant
	db.Exec(`INSERT INTO tenants (id, name, slug) VALUES ('tenant-default', 'Default', 'default')`)

	return db
}

// setupTestRouter creates a Gin engine with test routes.
func setupTestRouter(db *gorm.DB) *gin.Engine {
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
		authGroup.POST("/login", Login(db))
	}

	// Protected routes
	protected := api.Group("")
	protected.Use(auth.AuthMiddleware())
	{
		apps := protected.Group("/apps")
		{
			apps.POST("", CreateApp(db))
			apps.GET("", ListApps(db))
			apps.GET("/:id", GetApp(db))
			apps.PUT("/:id", UpdateApp(db))
		}

		protected.GET("/users/me", GetCurrentUser)
		protected.GET("/users", auth.RoleRequired("owner", "admin"), ListUsers(db))

		protected.GET("/system/version", GetVersion)
		protected.GET("/system/health", SystemHealth(db))

		deployments := protected.Group("/deployments")
		{
			deployments.GET("", ListDeployments(db))
			deployments.GET("/:id", GetDeployment(db))
		}
	}

	return r
}

// makeRequest is a test helper to make HTTP requests.
func makeRequest(r *gin.Engine, method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	var bodyReader *bytes.Buffer
	if body != nil {
		data, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(data)
	} else {
		bodyReader = bytes.NewBuffer([]byte("{}"))
	}

	req, _ := http.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// createTestUser creates a test user and returns the user and token.
func createTestUser(t *testing.T, db *gorm.DB, username, email, password, roleID string) (*model.User, string) {
	t.Helper()
	hash, _ := crypto.HashPassword(password)
	user := &model.User{
		ID:           uuid.New().String(),
		TenantID:     "tenant-default",
		RoleID:       roleID,
		Username:     username,
		Email:        email,
		PasswordHash: hash,
	}
	db.Create(user)

	roleName := "viewer"
	var role model.Role
	if db.Where("id = ?", roleID).First(&role).Error == nil {
		roleName = role.Name
	}
	token, _ := auth.GenerateToken(user.ID, roleName)
	return user, token
}

// --- Auth Tests ---

func TestRegister_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Exec("VACUUM")

	r := setupTestRouter(db)
	w := makeRequest(r, "POST", "/api/v1/auth/register", map[string]string{
		"username": "testuser",
		"email":    "test@example.com",
		"password": "password123",
	}, "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "success" {
		t.Fatalf("expected success status, got: %v", resp["status"])
	}

	data := resp["data"].(map[string]interface{})
	user := data["user"].(map[string]interface{})
	if user["username"] != "testuser" {
		t.Errorf("expected username testuser, got %v", user["username"])
	}
	if data["token"] == nil || data["token"] == "" {
		t.Error("expected token in response")
	}
}

func TestRegister_DuplicateUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Exec("VACUUM")

	r := setupTestRouter(db)

	body := map[string]string{
		"username": "dupuser",
		"email":    "dup@example.com",
		"password": "password123",
	}

	// First registration should succeed
	w1 := makeRequest(r, "POST", "/api/v1/auth/register", body, "")
	if w1.Code != http.StatusOK {
		t.Fatalf("first registration failed: %d", w1.Code)
	}

	// Second registration should fail with 409
	w2 := makeRequest(r, "POST", "/api/v1/auth/register", body, "")
	if w2.Code != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestLogin_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Exec("VACUUM")

	r := setupTestRouter(db)

	// Register first
	makeRequest(r, "POST", "/api/v1/auth/register", map[string]string{
		"username": "loginuser",
		"email":    "login@example.com",
		"password": "mypassword",
	}, "")

	// Login
	w := makeRequest(r, "POST", "/api/v1/auth/login", map[string]string{
		"username": "loginuser",
		"password": "mypassword",
	}, "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["token"] == nil || data["token"] == "" {
		t.Error("expected token in login response")
	}
}

func TestLogin_InvalidPassword(t *testing.T) {
	db := setupTestDB(t)
	defer db.Exec("VACUUM")

	r := setupTestRouter(db)

	// Register first
	makeRequest(r, "POST", "/api/v1/auth/register", map[string]string{
		"username": "wrongpassuser",
		"email":    "wrongpass@example.com",
		"password": "correctpassword",
	}, "")

	// Login with wrong password
	w := makeRequest(r, "POST", "/api/v1/auth/login", map[string]string{
		"username": "wrongpassuser",
		"password": "wrongpassword",
	}, "")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

// --- App Tests ---

func TestCRUDApps(t *testing.T) {
	db := setupTestDB(t)
	defer db.Exec("VACUUM")

	r := setupTestRouter(db)
	_, token := createTestUser(t, db, "appuser", "app@example.com", "password", "role-owner")

	// Create
	createBody := map[string]interface{}{
		"name":     "myapp",
		"repo_url": "https://github.com/example/app",
	}
	w := makeRequest(r, "POST", "/api/v1/apps", createBody, token)
	if w.Code != http.StatusOK {
		t.Fatalf("create app failed: %d: %s", w.Code, w.Body.String())
	}

	var createResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	appData := createResp["data"].(map[string]interface{})
	appID := appData["id"].(string)

	// Get
	w = makeRequest(r, "GET", "/api/v1/apps/"+appID, nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("get app failed: %d: %s", w.Code, w.Body.String())
	}

	// List
	w = makeRequest(r, "GET", "/api/v1/apps", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("list apps failed: %d: %s", w.Code, w.Body.String())
	}

	// Update
	w = makeRequest(r, "PUT", "/api/v1/apps/"+appID, map[string]interface{}{
		"domain": "myapp.example.com",
	}, token)
	if w.Code != http.StatusOK {
		t.Fatalf("update app failed: %d: %s", w.Code, w.Body.String())
	}
}

func TestListApps(t *testing.T) {
	db := setupTestDB(t)
	defer db.Exec("VACUUM")

	r := setupTestRouter(db)
	_, token := createTestUser(t, db, "listuser", "list@example.com", "password", "role-viewer")

	w := makeRequest(r, "GET", "/api/v1/apps", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("list apps failed: %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	// Should be empty initially
	if len(data) != 0 {
		t.Errorf("expected 0 apps, got %d", len(data))
	}
}

// --- System Tests ---

func TestGetVersion(t *testing.T) {
	db := setupTestDB(t)
	defer db.Exec("VACUUM")

	r := setupTestRouter(db)
	_, token := createTestUser(t, db, "versionuser", "ver@example.com", "password", "role-viewer")

	w := makeRequest(r, "GET", "/api/v1/system/version", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("get version failed: %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["version"] != "0.6.0" {
		t.Errorf("expected version 0.6.0, got %v", data["version"])
	}
}

func TestSystemHealth(t *testing.T) {
	db := setupTestDB(t)
	defer db.Exec("VACUUM")

	r := setupTestRouter(db)
	_, token := createTestUser(t, db, "healthuser", "health@example.com", "password", "role-viewer")

	w := makeRequest(r, "GET", "/api/v1/system/health", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("system health failed: %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["status"] != "healthy" {
		t.Errorf("expected healthy status, got %v", data["status"])
	}
}

// --- Auth Middleware Tests ---

func TestAuthMiddleware_ValidToken(t *testing.T) {
	db := setupTestDB(t)
	defer db.Exec("VACUUM")

	r := setupTestRouter(db)
	_, token := createTestUser(t, db, "authuser", "auth@example.com", "password", "role-viewer")

	// Access protected route with valid token
	w := makeRequest(r, "GET", "/api/v1/system/version", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid token, got %d", w.Code)
	}
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
	db := setupTestDB(t)
	defer db.Exec("VACUUM")

	r := setupTestRouter(db)

	// Access protected route without token
	w := makeRequest(r, "GET", "/api/v1/system/version", nil, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", w.Code)
	}
}

func TestRoleRequired(t *testing.T) {
	db := setupTestDB(t)
	defer db.Exec("VACUUM")

	r := setupTestRouter(db)

	// Viewer should NOT be able to list users
	_, viewerToken := createTestUser(t, db, "viewer1", "viewer@example.com", "password", "role-viewer")
	w := makeRequest(r, "GET", "/api/v1/users", nil, viewerToken)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for viewer, got %d: %s", w.Code, w.Body.String())
	}

	// Owner should be able to list users
	_, ownerToken := createTestUser(t, db, "owner1", "owner@example.com", "password", "role-owner")
	w = makeRequest(r, "GET", "/api/v1/users", nil, ownerToken)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for owner, got %d: %s", w.Code, w.Body.String())
	}

	// Admin should be able to list users
	_, adminToken := createTestUser(t, db, "admin1", "admin@example.com", "password", "role-admin")
	w = makeRequest(r, "GET", "/api/v1/users", nil, adminToken)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Deployment Tests ---

func TestListDeployments(t *testing.T) {
	db := setupTestDB(t)
	defer db.Exec("VACUUM")

	r := setupTestRouter(db)
	_, token := createTestUser(t, db, "depuser", "dep@example.com", "password", "role-viewer")

	w := makeRequest(r, "GET", "/api/v1/deployments", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("list deployments failed: %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	if len(data) != 0 {
		t.Errorf("expected 0 deployments, got %d", len(data))
	}
}

// --- JWT Tests ---

func TestGenerateAndParseToken(t *testing.T) {
	token, err := auth.GenerateToken("user-123", "admin")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := auth.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}
	if claims.UserID != "user-123" {
		t.Errorf("expected user ID user-123, got %s", claims.UserID)
	}
	if claims.Role != "admin" {
		t.Errorf("expected role admin, got %s", claims.Role)
	}
}

func TestParseToken_Invalid(t *testing.T) {
	_, err := auth.ParseToken("invalid-token")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

// --- Response Helpers ---

func TestRespondSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		respondSuccess(c, map[string]string{"key": "value"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "success" {
		t.Errorf("expected success, got %v", resp["status"])
	}
}

func TestRespondError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		respondError(c, http.StatusBadRequest, "something went wrong")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "error" {
		t.Errorf("expected error, got %v", resp["status"])
	}
	if resp["message"] != "something went wrong" {
		t.Errorf("expected 'something went wrong', got %v", resp["message"])
	}
}

// --- GetCurrentUser ---

func TestGetCurrentUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Exec("VACUUM")

	r := setupTestRouter(db)
	user, token := createTestUser(t, db, "meuser", "me@example.com", "password", "role-dev")

	w := makeRequest(r, "GET", "/api/v1/users/me", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("get current user failed: %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["id"] != user.ID {
		t.Errorf("expected user ID %s, got %v", user.ID, data["id"])
	}
	if data["username"] != "meuser" {
		t.Errorf("expected username meuser, got %v", data["username"])
	}
}

// --- Context key test ---

func TestContextKeys(t *testing.T) {
	// Verify context keys are properly typed
	if auth.UserIDKey != "userID" {
		t.Errorf("expected UserIDKey 'userID', got %q", auth.UserIDKey)
	}
	if auth.RoleKey != "role" {
		t.Errorf("expected RoleKey 'role', got %q", auth.RoleKey)
	}
}

// --- Optional Auth ---

func TestOptionalAuth_NoToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(auth.OptionalAuth())
	r.GET("/test", func(c *gin.Context) {
		_, exists := c.Get(string(auth.UserIDKey))
		if exists {
			t.Error("expected no userID in context without token")
		}
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestOptionalAuth_WithToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(auth.OptionalAuth())
	r.GET("/test", func(c *gin.Context) {
		userID, exists := c.Get(string(auth.UserIDKey))
		if !exists {
			t.Fatal("expected userID in context with valid token")
		}
		if userID != "test-user-id" {
			t.Errorf("expected test-user-id, got %v", userID)
		}
		c.JSON(200, gin.H{"ok": true})
	})

	token, _ := auth.GenerateToken("test-user-id", "viewer")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// Ensure unused import is consumed (context import needed for tests)
var _ = context.Background
