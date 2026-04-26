package middleware

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAuditTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	db.Exec(`CREATE TABLE IF NOT EXISTS audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER, username TEXT,
		action TEXT, resource_type TEXT, resource_id TEXT, detail TEXT,
		ip_address TEXT, user_agent TEXT, record_hash TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	return db
}

func TestAuditMiddleware_RecordsMutation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupAuditTestDB(t)
	auditSvc := service.NewAuditService(db)

	r := gin.New()
	r.Use(AuditMiddleware(auditSvc))
	r.POST("/api/v1/apps", func(c *gin.Context) {
		c.Set("userID", "42")
		c.Set("username", "testuser")
		c.String(200, "ok")
	})

	req := httptest.NewRequest("POST", "/api/v1/apps", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Verify audit log was created
	logs, total, err := auditSvc.List(context.TODO(), service.AuditFilter{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("failed to list audit logs: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 audit log, got %d", total)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log entry, got %d", len(logs))
	}
	if logs[0].Action != "apps.create" {
		t.Errorf("action = %q, want %q", logs[0].Action, "apps.create")
	}
	if logs[0].ResourceType != "apps" {
		t.Errorf("resource_type = %q, want %q", logs[0].ResourceType, "apps")
	}
}

func TestAuditMiddleware_SkipsGET(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupAuditTestDB(t)
	auditSvc := service.NewAuditService(db)

	r := gin.New()
	r.Use(AuditMiddleware(auditSvc))
	r.GET("/api/v1/apps", func(c *gin.Context) {
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/api/v1/apps", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Verify no audit log was created
	_, total, err := auditSvc.List(context.TODO(), service.AuditFilter{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("failed to list audit logs: %v", err)
	}
	if total != 0 {
		t.Errorf("expected 0 audit logs for GET, got %d", total)
	}
}

func TestAuditMiddleware_SkipsHEAD(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupAuditTestDB(t)
	auditSvc := service.NewAuditService(db)

	r := gin.New()
	r.Use(AuditMiddleware(auditSvc))
	r.HEAD("/api/v1/apps", func(c *gin.Context) {
		c.String(200, "")
	})

	req := httptest.NewRequest("HEAD", "/api/v1/apps", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	_, total, _ := auditSvc.List(context.TODO(), service.AuditFilter{Page: 1, PageSize: 10})
	if total != 0 {
		t.Errorf("expected 0 audit logs for HEAD, got %d", total)
	}
}

func TestAuditMiddleware_SkipsOPTIONS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupAuditTestDB(t)
	auditSvc := service.NewAuditService(db)

	r := gin.New()
	r.Use(AuditMiddleware(auditSvc))
	r.OPTIONS("/api/v1/apps", func(c *gin.Context) {
		c.String(204, "")
	})

	req := httptest.NewRequest("OPTIONS", "/api/v1/apps", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	_, total, _ := auditSvc.List(context.TODO(), service.AuditFilter{Page: 1, PageSize: 10})
	if total != 0 {
		t.Errorf("expected 0 audit logs for OPTIONS, got %d", total)
	}
}

func TestMapMethodToAction(t *testing.T) {
	tests := []struct {
		method string
		path   string
		want   string
	}{
		{"POST", "/api/v1/apps", "apps.create"},
		{"PUT", "/api/v1/apps/123", "apps.update"},
		{"DELETE", "/api/v1/apps/123", "apps.delete"},
		{"POST", "/api/v1/apps/123/deploy", "apps.deploy"},
		{"POST", "/api/v1/servers", "servers.create"},
		{"DELETE", "/api/v1/servers/456", "servers.delete"},
		{"POST", "/api/v1/credentials", "credentials.create"},
		{"GET", "/api/v1/unknown", "unknown.get"},
	}

	for _, tt := range tests {
		got := mapMethodToAction(tt.method, tt.path)
		if got != tt.want {
			t.Errorf("mapMethodToAction(%q, %q) = %q, want %q", tt.method, tt.path, got, tt.want)
		}
	}
}

func TestExtractResourceType(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/api/v1/apps", "apps"},
		{"/api/v1/servers", "servers"},
		{"/api/v1/apps/123/deploy", "apps"},
		{"/api/v1", "unknown"},
		{"/", "unknown"},
	}

	for _, tt := range tests {
		got := extractResourceType(tt.path)
		if got != tt.want {
			t.Errorf("extractResourceType(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestExtractResourceID(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/api/v1/apps/123", "123"},
		{"/api/v1/servers/456", "456"},
		{"/api/v1/apps", ""},
		{"/api/v1", ""},
	}

	for _, tt := range tests {
		got := extractResourceID(tt.path)
		if got != tt.want {
			t.Errorf("extractResourceID(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestMethodToVerb(t *testing.T) {
	tests := []struct {
		method string
		want   string
	}{
		{"POST", "create"},
		{"PUT", "update"},
		{"DELETE", "delete"},
		{"PATCH", "patch"},
		{"GET", "get"},
	}

	for _, tt := range tests {
		got := methodToVerb(tt.method)
		if got != tt.want {
			t.Errorf("methodToVerb(%q) = %q, want %q", tt.method, got, tt.want)
		}
	}
}

func TestAuditMiddleware_DeleteAction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupAuditTestDB(t)
	auditSvc := service.NewAuditService(db)

	r := gin.New()
	r.Use(AuditMiddleware(auditSvc))
	r.DELETE("/api/v1/servers/srv-123", func(c *gin.Context) {
		c.Set("userID", "10")
		c.Set("username", "admin")
		c.String(200, "deleted")
	})

	req := httptest.NewRequest("DELETE", "/api/v1/servers/srv-123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	logs, _, _ := auditSvc.List(context.TODO(), service.AuditFilter{Page: 1, PageSize: 10})
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(logs))
	}
	if logs[0].Action != "servers.delete" {
		t.Errorf("action = %q, want %q", logs[0].Action, "servers.delete")
	}
	if logs[0].ResourceID != "srv-123" {
		t.Errorf("resource_id = %q, want %q", logs[0].ResourceID, "srv-123")
	}
}

func TestToUint(t *testing.T) {
	tests := []struct {
		input interface{}
		want  uint
	}{
		{nil, 0},
		{uint(42), 42},
		{uint64(42), 42},
		{int(42), 42},
		{int64(42), 42},
		{float64(42.5), 42},
		{"not-a-number", 0},
	}

	for _, tt := range tests {
		got := toUint(tt.input)
		if got != tt.want {
			t.Errorf("toUint(%v) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		input interface{}
		want  string
	}{
		{nil, ""},
		{"hello", "hello"},
		{42, ""},
		{true, ""},
	}

	for _, tt := range tests {
		got := toString(tt.input)
		if got != tt.want {
			t.Errorf("toString(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
