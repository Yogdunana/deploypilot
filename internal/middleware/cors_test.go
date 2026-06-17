package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
)

func newCORSServer(t *testing.T, cfg CORSConfig) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS(cfg))
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	r.POST("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	return r
}

func TestCORS_EmptyAllowedOrigins_SetsNoHeaders(t *testing.T) {
	r := newCORSServer(t, CORSConfig{})
	req := httptest.NewRequest("GET", "/ping", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no Allow-Origin header, got %q", got)
	}
}

func TestCORS_AllowedOriginEchoedBack(t *testing.T) {
	r := newCORSServer(t, CORSConfig{
		AllowedOrigins: []string{"https://app.example.com"},
	})
	req := httptest.NewRequest("GET", "/ping", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("expected origin echoed, got %q", got)
	}
	if got := w.Header().Get("Vary"); got != "Origin" {
		t.Errorf("expected Vary=Origin when echoing back, got %q", got)
	}
}

func TestCORS_DisallowedOrigin_NoHeaders(t *testing.T) {
	r := newCORSServer(t, CORSConfig{
		AllowedOrigins: []string{"https://app.example.com"},
	})
	req := httptest.NewRequest("GET", "/ping", nil)
	req.Header.Set("Origin", "https://attacker.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no Allow-Origin for disallowed origin, got %q", got)
	}
}

func TestCORS_WildcardWithoutCredentials(t *testing.T) {
	r := newCORSServer(t, CORSConfig{
		AllowedOrigins: []string{"*"},
	})
	req := httptest.NewRequest("GET", "/ping", nil)
	req.Header.Set("Origin", "https://anything.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("expected wildcard Allow-Origin, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Errorf("expected no Allow-Credentials with wildcard, got %q", got)
	}
}

func TestCORS_WildcardWithCredentials_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for wildcard + AllowCredentials=true")
		}
	}()
	_ = CORS(CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
	})
}

func TestCORS_AllowCredentialsEchoesOrigin(t *testing.T) {
	r := newCORSServer(t, CORSConfig{
		AllowedOrigins:   []string{"https://app.example.com"},
		AllowCredentials: true,
	})
	req := httptest.NewRequest("GET", "/ping", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("expected origin echo, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("expected Allow-Credentials=true, got %q", got)
	}
}

func TestCORS_Preflight_AbortsWith204(t *testing.T) {
	r := newCORSServer(t, CORSConfig{
		AllowedOrigins: []string{"https://app.example.com"},
		AllowedMethods: []string{"GET", "POST", "DELETE"},
		AllowedHeaders: []string{"Content-Type", "X-Custom"},
		MaxAge:         600,
	})
	req := httptest.NewRequest("OPTIONS", "/ping", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", "DELETE")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 for preflight, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("expected Allow-Methods to be set on preflight")
	}
	if got := w.Header().Get("Access-Control-Max-Age"); got != strconv.Itoa(600) {
		t.Errorf("expected Max-Age=600, got %q", got)
	}
}

func TestCORS_DefaultsApplied(t *testing.T) {
	// Empty AllowedMethods / AllowedHeaders / MaxAge should fall back to
	// safe defaults rather than emit empty headers.
	r := newCORSServer(t, CORSConfig{
		AllowedOrigins: []string{"*"},
	})
	req := httptest.NewRequest("OPTIONS", "/ping", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("expected default methods to be set")
	}
	if got := w.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Error("expected default headers to be set")
	}
	if got := w.Header().Get("Access-Control-Max-Age"); got != "86400" {
		t.Errorf("expected default Max-Age=86400, got %q", got)
	}
}

func TestCORS_ExposeHeaders(t *testing.T) {
	r := newCORSServer(t, CORSConfig{
		AllowedOrigins: []string{"*"},
		ExposeHeaders:  []string{"X-Total-Count", "X-Request-Id"},
	})
	req := httptest.NewRequest("GET", "/ping", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := w.Header().Get("Access-Control-Expose-Headers")
	if got != "X-Total-Count, X-Request-Id" {
		t.Errorf("expected expose-headers list, got %q", got)
	}
}

func TestCORS_NoOriginHeader_NoAllowOrigin(t *testing.T) {
	// A same-origin request without an Origin header should not get an
	// Access-Control-Allow-Origin response header.
	r := newCORSServer(t, CORSConfig{
		AllowedOrigins: []string{"https://app.example.com"},
	})
	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no Allow-Origin for same-origin request, got %q", got)
	}
}
