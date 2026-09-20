package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAddServer_InvalidPort(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/servers", AddServer(nil))

	body, _ := json.Marshal(map[string]interface{}{
		"name": "bad",
		"host": "10.0.0.1",
		"port": 70000,
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/servers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAddServer_NegativePort(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/servers", AddServer(nil))

	body, _ := json.Marshal(map[string]interface{}{
		"name": "bad",
		"host": "10.0.0.1",
		"port": -1,
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/servers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBatchDeployHandler_InvalidStrategy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/batch", BatchDeployHandler(nil))

	body, _ := json.Marshal(map[string]interface{}{
		"apps":     []map[string]interface{}{{"name": "app"}},
		"strategy": "explode",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGenerateKeyPair_InvalidBits(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api := NewSSHAPI(nil)
	r := gin.New()
	r.POST("/keys", api.GenerateKeyPair)

	body, _ := json.Marshal(map[string]interface{}{
		"name": "laptop",
		"bits": 1024,
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/keys", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExportMonitorData_InvalidTime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/export", ExportMonitorData(nil))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/export?start=not-a-date", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGrafanaAPI_NoPanicWithoutDB(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api := NewGrafanaAPI(nil, nil)
	r := gin.New()
	r.GET("/status", api.GetStatus)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/status", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOutboundWebhookAPI_NoPanicWithoutDB(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api := NewOutboundWebhookAPI(nil)
	r := gin.New()
	r.GET("/webhooks", api.ListWebhooks)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/webhooks", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRotateLicenseKeys_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/rotate", RotateLicenseKeysHandler(nil))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/rotate", bytes.NewBufferString("{not-json"))
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = int64(len("{not-json"))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
