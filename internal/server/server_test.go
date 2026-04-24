package server

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNew(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	executor := &testLocalExecutor{}
	bridge := service.NewBridge(db, executor, []byte("test-key-1234567890abcdef"))

	srv := New("0.0.0.0:0", db, bridge)
	if srv == nil {
		t.Fatal("New() returned nil")
	}
	if srv.addr != "0.0.0.0:0" {
		t.Errorf("addr = %q, want %q", srv.addr, "0.0.0.0:0")
	}
	if srv.db != db {
		t.Error("db not set correctly")
	}
	if srv.bridge != bridge {
		t.Error("bridge not set correctly")
	}
	if srv.Router() == nil {
		t.Error("Router() returned nil")
	}
}

func TestCorsMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Test OPTIONS request
	r := gin.New()
	r.Use(corsMiddleware())
	r.GET("/test", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 204 {
		t.Errorf("OPTIONS status = %d, want 204", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing CORS origin header")
	}

	// Test GET request
	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("GET status = %d, want 200", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing CORS origin header for GET")
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("missing CORS methods header for GET")
	}
}

type testLocalExecutor struct{}

func (e *testLocalExecutor) RunCommand(ctx context.Context, cmd string) (string, error) {
	return "", nil
}
