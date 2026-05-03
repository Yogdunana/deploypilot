package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestExtractAPIVersion(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/api/v1/apps", "v1"},
		{"/api/v1/servers/1", "v1"},
		{"/api/v2/anything", "v2"},
		{"/api/v10/deep/nested", "v10"},
		{"/health", "v1"},           // no version in path -> default
		{"/", "v1"},                 // root -> default
		{"/api/no-version", "v1"},   // missing v prefix -> default
		{"/other/v1/path", "v1"},    // not under /api -> default
	}

	for _, tt := range tests {
		got := extractAPIVersion(tt.path)
		if got != tt.want {
			t.Errorf("extractAPIVersion(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestIsVersionSupported(t *testing.T) {
	if !isVersionSupported("v1") {
		t.Error("expected v1 to be supported")
	}
	if isVersionSupported("v2") {
		t.Error("expected v2 to not be supported")
	}
	if isVersionSupported("") {
		t.Error("expected empty string to not be supported")
	}
}

func TestAPIVersionMiddleware_SetsHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(APIVersionMiddleware())
	r.GET("/api/v1/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if got := w.Header().Get("X-API-Version"); got != "v1" {
		t.Errorf("X-API-Version = %q, want %q", got, "v1")
	}
}

func TestAPIVersionMiddleware_UnsupportedVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(APIVersionMiddleware())
	r.GET("/api/v99/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/api/v99/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// The middleware should not block; it just sets a warning header.
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if got := w.Header().Get("X-API-Version"); got != "v99" {
		t.Errorf("X-API-Version = %q, want %q", got, "v99")
	}
	acceptVer := w.Header().Get("Accept-Version")
	if acceptVer == "" {
		t.Error("expected Accept-Version header to be set for unsupported version")
	}
}

func TestAPIVersionMiddleware_DefaultVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(APIVersionMiddleware())
	r.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if got := w.Header().Get("X-API-Version"); got != "v1" {
		t.Errorf("X-API-Version = %q, want %q (default)", got, "v1")
	}
}
