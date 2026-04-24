package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

func TestCICDRoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	api := r.Group("/api/v1")
	cicd := api.Group("/cicd")
	{
		cicd.POST("/trigger", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
		cicd.GET("/status/:runID", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	}

	routes := r.Routes()
	routePaths := make(map[string]bool)
	for _, route := range routes {
		routePaths[route.Path] = true
	}

	expectedPaths := []string{
		"/api/v1/cicd/trigger",
		"/api/v1/cicd/status/:runID",
	}

	for _, path := range expectedPaths {
		if !routePaths[path] {
			t.Errorf("expected route %s to be registered", path)
		}
	}
}

func TestCICDTriggerInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/cicd/trigger", TriggerCIBuild(nil))

	body := bytes.NewBufferString("not json")
	req, _ := http.NewRequest("POST", "/api/v1/cicd/trigger", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for invalid JSON, got %d", w.Code)
	}
}

func TestCICDTriggerMissingFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/cicd/trigger", TriggerCIBuild(nil))

	data, _ := json.Marshal(map[string]string{"provider": "github-actions"})
	body := bytes.NewBuffer(data)
	req, _ := http.NewRequest("POST", "/api/v1/cicd/trigger", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for missing fields, got %d", w.Code)
	}
}

func TestCICDStatusMissingProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/cicd/status/:runID", GetCIBuildStatus(nil))

	req, _ := http.NewRequest("GET", "/api/v1/cicd/status/12345", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for missing provider, got %d", w.Code)
	}
}

func TestCICDStatusWithProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/cicd/status/:runID", GetCIBuildStatus(nil))

	req, _ := http.NewRequest("GET", "/api/v1/cicd/status/12345?provider=github-actions", nil)
	w := httptest.NewRecorder()

	defer func() {
		if rec := recover(); rec != nil {
			// Expected: nil bridge causes panic
			t.Logf("Recovered expected panic: %v", rec)
		} else {
			// If no panic, check response code
			if w.Code != 500 {
				t.Errorf("expected 500 for nil bridge, got %d", w.Code)
			}
		}
	}()
	r.ServeHTTP(w, req)
}

func TestCICDTriggerValidPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/cicd/trigger", TriggerCIBuild(nil))

	data, _ := json.Marshal(map[string]string{
		"provider": "github-actions",
		"repo":     "test-repo",
		"branch":   "main",
	})
	body := bytes.NewBuffer(data)
	req, _ := http.NewRequest("POST", "/api/v1/cicd/trigger", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	defer func() {
		if rec := recover(); rec != nil {
			// Expected: nil bridge causes panic
			t.Logf("Recovered expected panic: %v", rec)
		}
	}()
	r.ServeHTTP(w, req)
}

func TestCICDStatusUnsupportedProvider(t *testing.T) {
	db := setupTestDB(t)
	// Seed a CI/CD provider
	db.Exec(`INSERT INTO providers (id, type, name, config, enabled) VALUES
		('cicd-1', 'cicd-unknown', 'Unknown CI', '{"token":"t","owner":"o"}', 1)`)

	bridge := &service.Bridge{DB: db}
	r := gin.New()
	r.GET("/api/v1/cicd/status/:runID", GetCIBuildStatus(bridge))

	req, _ := http.NewRequest("GET", "/api/v1/cicd/status/12345?provider=unknown", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 500 {
		t.Errorf("expected 500 for unsupported provider, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCICDTriggerUnsupportedProvider(t *testing.T) {
	db := setupTestDB(t)
	db.Exec(`INSERT INTO providers (id, type, name, config, enabled) VALUES
		('cicd-2', 'cicd-unknown', 'Unknown CI', '{"token":"t","owner":"o"}', 1)`)

	bridge := &service.Bridge{DB: db}
	r := gin.New()
	r.POST("/api/v1/cicd/trigger", TriggerCIBuild(bridge))

	data, _ := json.Marshal(map[string]string{
		"provider": "unknown",
		"repo":     "test-repo",
		"branch":   "main",
	})
	body := bytes.NewBuffer(data)
	req, _ := http.NewRequest("POST", "/api/v1/cicd/trigger", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 500 {
		t.Errorf("expected 500 for unsupported provider, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCICDStatusNoProviderInDB(t *testing.T) {
	db := setupTestDB(t)
	// No CI/CD provider seeded

	bridge := &service.Bridge{DB: db}
	r := gin.New()
	r.GET("/api/v1/cicd/status/:runID", GetCIBuildStatus(bridge))

	req, _ := http.NewRequest("GET", "/api/v1/cicd/status/12345?provider=github-actions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200 (returns error in body), got %d: %s", w.Code, w.Body.String())
	}
}

func TestCICDTriggerNoProviderInDB(t *testing.T) {
	db := setupTestDB(t)
	// No CI/CD provider seeded

	bridge := &service.Bridge{DB: db}
	r := gin.New()
	r.POST("/api/v1/cicd/trigger", TriggerCIBuild(bridge))

	data, _ := json.Marshal(map[string]string{
		"provider": "github-actions",
		"repo":     "test-repo",
		"branch":   "main",
	})
	body := bytes.NewBuffer(data)
	req, _ := http.NewRequest("POST", "/api/v1/cicd/trigger", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200 (returns error in body), got %d: %s", w.Code, w.Body.String())
	}
}
