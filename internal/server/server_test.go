package server

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/config"
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
	bridge := service.NewBridge(db, executor, []byte("test-key-1234567890abcdef"), nil)
	cfg := config.DefaultConfig()

	srv := New("0.0.0.0:0", db, bridge, cfg)
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

	// Test OPTIONS request with wildcard origins
	r := gin.New()
	r.Use(corsMiddleware([]string{"*"}))
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

func TestCorsMiddleware_SpecificOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(corsMiddleware([]string{"https://example.com"}))
	r.GET("/test", func(c *gin.Context) { c.String(200, "ok") })

	// Request with matching origin
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("GET status = %d, want 200", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Errorf("CORS origin = %q, want %q", w.Header().Get("Access-Control-Allow-Origin"), "https://example.com")
	}

	// Request with non-matching origin
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("GET status = %d, want 200", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("CORS origin should be empty for non-matching origin, got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCorsMiddleware_EmptyOrigins(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(corsMiddleware([]string{}))
	r.GET("/test", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://any.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("GET status = %d, want 200", w.Code)
	}
	// Empty origins list should fall back to wildcard
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("CORS origin = %q, want %q", w.Header().Get("Access-Control-Allow-Origin"), "*")
	}
}

func TestCorsMiddleware_MaxAge(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(corsMiddleware([]string{"*"}))
	r.GET("/test", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Max-Age") != "86400" {
		t.Errorf("Max-Age = %q, want %q", w.Header().Get("Access-Control-Max-Age"), "86400")
	}
}

type testLocalExecutor struct{}

func (e *testLocalExecutor) RunCommand(ctx context.Context, cmd string) (string, error) {
	return "", nil
}

func TestRun(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	executor := &testLocalExecutor{}
	bridge := service.NewBridge(db, executor, []byte("test-key-1234567890abcdef"), nil)
	cfg := config.DefaultConfig()

	srv := New("127.0.0.1:0", db, bridge, cfg)

	done := make(chan error, 1)
	go func() {
		done <- srv.Run()
	}()

	// Give the server a moment to start
	time.Sleep(200 * time.Millisecond)

	// The server is running; we can't easily test HTTP requests here
	// since we don't know the exact port, but we verify it started without error.
	// We stop by sending a request to the server (it will panic on close, but
	// that's expected since gin.Run blocks until the server is closed).
	// Instead, just verify the goroutine is running.
	select {
	case err := <-done:
		// If we got here immediately, the server failed to start
		t.Fatalf("Run() returned immediately with error: %v", err)
	default:
		// Server is running, good
	}
}
