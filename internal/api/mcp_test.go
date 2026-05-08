package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestMCPHandler(t *testing.T) (*MCPHTTPHandler, *gin.Engine, *gorm.DB) {
	gin.SetMode(gin.TestMode)

	// Setup test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	// Run migrations
	if err := db.AutoMigrate(&model.User{}, &model.App{}, &model.Server{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Create test user
	testUser := &model.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
	}
	db.Create(testUser)

	// Create bridge
	bridge := service.NewBridge(db, nil, []byte("test-encryption-key-32bytes!!"), nil)

	// Create handler
	handler := NewMCPHTTPHandler(bridge)

	// Setup router
	r := gin.New()
	api := r.Group("/api/v1")

	// Add auth middleware mock
	api.Use(func(c *gin.Context) {
		c.Set("userID", "1")
		c.Set("username", "testuser")
		c.Set("role", "owner")
		c.Next()
	})

	// Register MCP routes
	mcpGroup := api.Group("/mcp")
	{
		mcpGroup.GET("/sse", handler.HandleSSE)
		mcpGroup.POST("/message", handler.HandleMessage)
		mcpGroup.POST("", handler.HandleMessageDirect)
		mcpGroup.GET("/tools", handler.HandleListTools)
	}

	return handler, r, db
}

// Test 1: MCP tools endpoint returns list of tools
func TestMCPListTools(t *testing.T) {
	_, r, db := setupTestMCPHandler(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/mcp/tools", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Check JSON-RPC structure
	assert.Equal(t, "2.0", response["jsonrpc"])
	assert.NotNil(t, response["result"])

	// Check result contains tools
	result, ok := response["result"].(map[string]interface{})
	assert.True(t, ok)
	assert.NotNil(t, result["tools"])

	tools, ok := result["tools"].([]interface{})
	assert.True(t, ok)
	assert.Greater(t, len(tools), 0, "should have at least one tool")

	t.Logf("Found %d MCP tools", len(tools))
}

// Test 2: Direct JSON-RPC call works
func TestMCPDirectMessage(t *testing.T) {
	_, r, db := setupTestMCPHandler(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// Call tools/list via direct endpoint
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
	}
	body, _ := json.Marshal(request)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/mcp", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "2.0", response["jsonrpc"])
	assert.Equal(t, float64(1), response["id"])
	assert.NotNil(t, response["result"])
}

// Test 3: Invalid JSON returns error
func TestMCPInvalidJSON(t *testing.T) {
	_, r, db := setupTestMCPHandler(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/mcp", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Test 4: SSE endpoint returns correct headers
func TestMCPSSEHeaders(t *testing.T) {
	_, r, db := setupTestMCPHandler(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/mcp/sse", nil)
	r.ServeHTTP(w, req)

	// SSE should keep connection open, but in test it will close immediately
	// Check headers were set correctly
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	assert.NotEmpty(t, w.Header().Get("Mcp-Session-Id"))
}

// Test 5: Message endpoint requires session
func TestMCPMessageRequiresSession(t *testing.T) {
	_, r, db := setupTestMCPHandler(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
	}
	body, _ := json.Marshal(request)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/mcp/message", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Test 6: Verify specific tools exist
func TestMCPSpecificToolsExist(t *testing.T) {
	_, r, db := setupTestMCPHandler(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/mcp/tools", nil)
	r.ServeHTTP(w, req)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	result := response["result"].(map[string]interface{})
	tools := result["tools"].([]interface{})

	// Check for some expected tools
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolMap := tool.(map[string]interface{})
		name := toolMap["name"].(string)
		toolNames[name] = true
	}

	// Verify core tools exist
	expectedTools := []string{"list_servers", "list_apps", "deploy_app", "create_app"}
	for _, name := range expectedTools {
		assert.True(t, toolNames[name], "Tool %s should exist", name)
	}

	t.Logf("Verified %d tools including: %v", len(tools), expectedTools)
}
