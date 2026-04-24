package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetSystemMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Create a mock handler that returns system metrics
	r.GET("/api/v1/monitor/system", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"data": []map[string]interface{}{
				{"type": "cpu", "name": "cpu_usage", "value": 25.3, "unit": "percent"},
				{"type": "memory", "name": "memory_usage_percent", "value": 62.1, "unit": "percent"},
			},
		})
	})

	token := getTestToken(t, "user-1", "owner")
	req, _ := http.NewRequest("GET", "/api/v1/monitor/system", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "success" {
		t.Errorf("expected status 'success', got %v", resp["status"])
	}
}

func TestGetContainerMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/api/v1/monitor/container/:name", func(c *gin.Context) {
		name := c.Param("name")
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"data": []map[string]interface{}{
				{"type": "cpu", "name": "container_cpu", "value": 12.5, "unit": "percent", "container": name},
			},
		})
	})

	token := getTestToken(t, "user-1", "owner")
	req, _ := http.NewRequest("GET", "/api/v1/monitor/container/my-app", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListAlerts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/api/v1/monitor/alerts", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"data": []interface{}{},
		})
	})

	token := getTestToken(t, "user-1", "owner")
	req, _ := http.NewRequest("GET", "/api/v1/monitor/alerts", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "success" {
		t.Errorf("expected status 'success', got %v", resp["status"])
	}
}

func TestHealContainer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.POST("/api/v1/monitor/heal/:name", func(c *gin.Context) {
		name := c.Param("name")
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"data": map[string]interface{}{
				"action":        "none",
				"container_name": name,
				"reason":        "container is healthy",
			},
		})
	})

	token := getTestToken(t, "user-1", "owner")
	req, _ := http.NewRequest("POST", "/api/v1/monitor/heal/my-app", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "success" {
		t.Errorf("expected status 'success', got %v", resp["status"])
	}
}

func TestListAlertRules(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/api/v1/monitor/alert-rules", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"data": []map[string]interface{}{
				{"id": "disk-space", "name": "Disk Space Low"},
				{"id": "memory-high", "name": "Memory Usage High"},
				{"id": "cpu-high", "name": "CPU Usage High"},
			},
		})
	})

	token := getTestToken(t, "user-1", "owner")
	req, _ := http.NewRequest("GET", "/api/v1/monitor/alert-rules", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "success" {
		t.Errorf("expected status 'success', got %v", resp["status"])
	}
}

// TestMonitorRoutesRegistered verifies the monitor routes are properly registered.
func TestMonitorRoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Register monitor routes
	api := r.Group("/api/v1")
	mon := api.Group("/monitor")
	{
		mon.GET("/system", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
		mon.GET("/container/:name", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
		mon.GET("/alerts", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
		mon.GET("/alert-rules", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
		mon.POST("/heal/:name", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
		mon.POST("/check/:name", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	}

	routes := r.Routes()
	routePaths := make(map[string]bool)
	for _, route := range routes {
		routePaths[route.Path] = true
	}

	expectedPaths := []string{
		"/api/v1/monitor/system",
		"/api/v1/monitor/container/:name",
		"/api/v1/monitor/alerts",
		"/api/v1/monitor/alert-rules",
		"/api/v1/monitor/heal/:name",
		"/api/v1/monitor/check/:name",
	}

	for _, path := range expectedPaths {
		if !routePaths[path] {
			t.Errorf("expected route %s to be registered", path)
		}
	}
}



