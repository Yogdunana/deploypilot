package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// sslTestDB creates a test DB with the ssl_certificates table.
func sslTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupTestDB(t)
	// Create ssl_certificates table
	db.Exec(`CREATE TABLE IF NOT EXISTS ssl_certificates (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		domain TEXT UNIQUE NOT NULL,
		email TEXT NOT NULL,
		provider TEXT NOT NULL DEFAULT 'cloudflare',
		status TEXT NOT NULL DEFAULT 'pending',
		cert_path TEXT,
		key_path TEXT,
		issued_at DATETIME,
		expires_at DATETIME,
		auto_renew INTEGER DEFAULT 1,
		last_renewed DATETIME,
		retry_count INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	return db
}

// sslTestRouter creates a router with SSL routes registered.
func sslTestRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})

	api := r.Group("/api/v1")
	ssl := api.Group("/ssl")
	{
		ssl.GET("/certificates", ListSSLCertificates(db))
		ssl.POST("/certificates", RequestSSLCertificate(db))
		ssl.DELETE("/certificates/:id", DeleteSSLCertificate(db))
		ssl.POST("/certificates/:id/renew", RenewSSLCertificate(db))
	}
	return r
}

func TestListSSLCertificates_Empty(t *testing.T) {
	db := sslTestDB(t)
	defer db.Exec("VACUUM")

	r := sslTestRouter(db)
	w := makeRequest(r, "GET", "/api/v1/ssl/certificates", nil, "")

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

func TestListSSLCertificates_WithData(t *testing.T) {
	db := sslTestDB(t)
	defer db.Exec("VACUUM")

	// Insert a certificate
	db.Create(&model.SSLCertificate{
		Domain:   "example.com",
		Email:    "admin@example.com",
		Provider: "cloudflare",
		Status:   "active",
	})

	r := sslTestRouter(db)
	w := makeRequest(r, "GET", "/api/v1/ssl/certificates", nil, "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	if len(data) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(data))
	}
	cert := data[0].(map[string]interface{})
	if cert["domain"] != "example.com" {
		t.Errorf("expected domain example.com, got %v", cert["domain"])
	}
}

func TestRequestSSLCertificate_Success(t *testing.T) {
	db := sslTestDB(t)
	defer db.Exec("VACUUM")

	r := sslTestRouter(db)
	w := makeRequest(r, "POST", "/api/v1/ssl/certificates", map[string]string{
		"domain":   "new.example.com",
		"email":    "admin@example.com",
		"provider": "cloudflare",
	}, "")

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["domain"] != "new.example.com" {
		t.Errorf("expected domain new.example.com, got %v", data["domain"])
	}
	if data["status"] != "pending" {
		t.Errorf("expected status pending, got %v", data["status"])
	}
}

func TestRequestSSLCertificate_MissingFields(t *testing.T) {
	db := sslTestDB(t)
	defer db.Exec("VACUUM")

	r := sslTestRouter(db)
	// Missing email
	w := makeRequest(r, "POST", "/api/v1/ssl/certificates", map[string]string{
		"domain": "new.example.com",
	}, "")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteSSLCertificate_Success(t *testing.T) {
	db := sslTestDB(t)
	defer db.Exec("VACUUM")

	cert := model.SSLCertificate{
		Domain:   "delete.example.com",
		Email:    "admin@example.com",
		Provider: "cloudflare",
		Status:   "active",
	}
	db.Create(&cert)

	r := sslTestRouter(db)
	w := makeRequest(r, "DELETE", "/api/v1/ssl/certificates/1", nil, "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify it's gone
	var count int64
	db.Model(&model.SSLCertificate{}).Where("domain = ?", "delete.example.com").Count(&count)
	if count != 0 {
		t.Error("expected certificate to be deleted")
	}
}

func TestDeleteSSLCertificate_NotFound(t *testing.T) {
	db := sslTestDB(t)
	defer db.Exec("VACUUM")

	r := sslTestRouter(db)
	w := makeRequest(r, "DELETE", "/api/v1/ssl/certificates/999", nil, "")

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRenewSSLCertificate_Success(t *testing.T) {
	db := sslTestDB(t)
	defer db.Exec("VACUUM")

	cert := model.SSLCertificate{
		Domain:     "renew.example.com",
		Email:      "admin@example.com",
		Provider:   "cloudflare",
		Status:     "active",
		RetryCount: 0,
	}
	db.Create(&cert)

	r := sslTestRouter(db)
	w := makeRequest(r, "POST", "/api/v1/ssl/certificates/1/renew", nil, "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	inner := data["data"].(map[string]interface{})
	if inner["status"] != "renewing" {
		t.Errorf("expected status renewing, got %v", inner["status"])
	}
	if inner["retry_count"].(float64) != 1 {
		t.Errorf("expected retry_count 1, got %v", inner["retry_count"])
	}
}

func TestRenewSSLCertificate_NotFound(t *testing.T) {
	db := sslTestDB(t)
	defer db.Exec("VACUUM")

	r := sslTestRouter(db)
	w := makeRequest(r, "POST", "/api/v1/ssl/certificates/999/renew", nil, "")

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}
