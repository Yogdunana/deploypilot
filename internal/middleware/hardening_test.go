package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupHardeningTestRouter(middleware gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware)
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "home") })
	r.GET("/dashboard", func(c *gin.Context) { c.String(http.StatusOK, "dashboard") })
	r.GET("/api/v1/apps", func(c *gin.Context) { c.String(http.StatusOK, "api") })
	r.GET("/ws/monitor", func(c *gin.Context) { c.String(http.StatusOK, "ws") })
	r.GET("/health", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	r.GET("/assets/app.js", func(c *gin.Context) { c.String(http.StatusOK, "js") })
	r.GET("/icon.svg", func(c *gin.Context) { c.String(http.StatusOK, "icon") })
	return r
}

func TestSecurityEntrance_Empty(t *testing.T) {
	r := setupHardeningTestRouter(SecurityEntrance(""))

	req := httptest.NewRequest("GET", "/dashboard", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("with empty entrance, all paths should pass, got status %d", w.Code)
	}
}

func TestSecurityEntrance_WithPrefix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityEntrance("/my-secret-panel"))
	r.GET("/api/v1/apps", func(c *gin.Context) { c.String(http.StatusOK, "api") })
	r.GET("/health", func(c *gin.Context) { c.String(http.StatusOK, "health") })
	r.NoRoute(func(c *gin.Context) {
		c.String(http.StatusOK, "spa:"+c.Request.URL.Path)
	})

	tests := []struct {
		path     string
		code     int
		contains string
	}{
		{"/my-secret-panel/dashboard", http.StatusOK, "spa:/dashboard"},
		{"/my-secret-panel/", http.StatusOK, "spa:/"},
		{"/my-secret-panel", http.StatusOK, "spa:/"},
		{"/dashboard", http.StatusNotFound, ""},
		{"/", http.StatusNotFound, ""},
		{"/api/v1/apps", http.StatusOK, "api"},
		{"/health", http.StatusOK, "health"},
	}
	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != tt.code {
			t.Errorf("path %q: expected status %d, got %d (body: %s)", tt.path, tt.code, w.Code, w.Body.String())
		}
		if tt.contains != "" && !strings.Contains(w.Body.String(), tt.contains) {
			t.Errorf("path %q: expected body to contain %q, got %q", tt.path, tt.contains, w.Body.String())
		}
	}
}

func TestSecurityEntrance_APIPathsExempt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityEntrance("/my-secret-panel"))
	r.GET("/api/v1/apps", func(c *gin.Context) { c.String(http.StatusOK, "api") })

	req := httptest.NewRequest("GET", "/api/v1/apps", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("API paths should be exempt from security entrance, got status %d", w.Code)
	}
}

func TestSecurityEntrance_WebSocketPathsExempt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityEntrance("/my-secret-panel"))
	r.GET("/ws/monitor", func(c *gin.Context) { c.String(http.StatusOK, "ws") })

	req := httptest.NewRequest("GET", "/ws/monitor", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("WebSocket paths should be exempt from security entrance, got status %d", w.Code)
	}
}

func TestSecurityEntrance_HealthCheckExempt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityEntrance("/my-secret-panel"))
	r.GET("/health", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("health check should be exempt, got status %d", w.Code)
	}
}

func TestSecurityEntrance_StaticAssetsExempt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityEntrance("/my-secret-panel"))
	r.GET("/assets/app.js", func(c *gin.Context) { c.String(http.StatusOK, "js") })
	r.GET("/icon.svg", func(c *gin.Context) { c.String(http.StatusOK, "icon") })

	tests := []string{
		"/assets/app.js",
		"/icon.svg",
	}
	for _, path := range tests {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("static asset %q should be exempt, got status %d", path, w.Code)
		}
	}
}

func TestSecurityEntrance_PrefixNormalization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityEntrance("my-secret-panel/"))
	r.NoRoute(func(c *gin.Context) {
		c.String(http.StatusOK, "spa:"+c.Request.URL.Path)
	})

	req := httptest.NewRequest("GET", "/my-secret-panel/dashboard", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("prefix should be normalized (leading slash added, trailing removed), got status %d (body: %s)", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "spa:/dashboard") {
		t.Errorf("path should be stripped correctly, got body: %s", w.Body.String())
	}
}

func TestSecurityEntrance_Returns404Not403(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityEntrance("/secret"))
	r.NoRoute(func(c *gin.Context) {
		c.String(http.StatusOK, "spa")
	})

	req := httptest.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("should return 404 to hide panel existence, got status %d", w.Code)
	}
}

func TestDomainBinding_Empty(t *testing.T) {
	r := setupHardeningTestRouter(DomainBinding(nil))

	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "evil.com"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("with empty allowed domains, all hosts should pass, got status %d", w.Code)
	}
}

func TestDomainBinding_Allowed(t *testing.T) {
	r := setupHardeningTestRouter(DomainBinding([]string{"example.com", "app.example.com"}))

	tests := []struct {
		host string
		code int
	}{
		{"example.com", http.StatusOK},
		{"app.example.com", http.StatusOK},
		{"evil.com", http.StatusForbidden},
		{"other.com", http.StatusForbidden},
	}
	for _, tt := range tests {
		req := httptest.NewRequest("GET", "/", nil)
		req.Host = tt.host
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != tt.code {
			t.Errorf("host %q: expected status %d, got %d", tt.host, tt.code, w.Code)
		}
	}
}

func TestDomainBinding_CaseInsensitive(t *testing.T) {
	r := setupHardeningTestRouter(DomainBinding([]string{"Example.COM"}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("domain comparison should be case-insensitive, got status %d", w.Code)
	}
}

func TestDomainBinding_WithPort(t *testing.T) {
	r := setupHardeningTestRouter(DomainBinding([]string{"example.com"}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "example.com:8080"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("port should be stripped before comparison, got status %d", w.Code)
	}
}

func TestDomainBinding_TrimsSpaces(t *testing.T) {
	r := setupHardeningTestRouter(DomainBinding([]string{"  example.com  "}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("domains should be trimmed, got status %d", w.Code)
	}
}

func TestDomainBinding_SkipsEmpty(t *testing.T) {
	r := setupHardeningTestRouter(DomainBinding([]string{"", "  ", "example.com"}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("empty domain entries should be skipped, got status %d", w.Code)
	}
}

func TestIPWhitelistMiddleware_Empty(t *testing.T) {
	r := setupHardeningTestRouter(IPWhitelist(nil))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("with empty whitelist, all IPs should pass, got status %d", w.Code)
	}
}

func TestIPWhitelistMiddleware_ExactMatch(t *testing.T) {
	r := setupHardeningTestRouter(IPWhitelist([]string{"192.168.1.100", "10.0.0.1"}))

	tests := []struct {
		ip   string
		code int
	}{
		{"192.168.1.100", http.StatusOK},
		{"10.0.0.1", http.StatusOK},
		{"192.168.1.101", http.StatusForbidden},
		{"10.0.0.2", http.StatusForbidden},
	}
	for _, tt := range tests {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = tt.ip + ":12345"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != tt.code {
			t.Errorf("IP %q: expected status %d, got %d", tt.ip, tt.code, w.Code)
		}
	}
}

func TestIPWhitelistMiddleware_CIDRMatch(t *testing.T) {
	r := setupHardeningTestRouter(IPWhitelist([]string{"192.168.1.0/24", "10.0.0.0/8"}))

	tests := []struct {
		ip   string
		code int
	}{
		{"192.168.1.1", http.StatusOK},
		{"192.168.1.254", http.StatusOK},
		{"10.1.2.3", http.StatusOK},
		{"192.168.2.1", http.StatusForbidden},
		{"172.16.0.1", http.StatusForbidden},
	}
	for _, tt := range tests {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = tt.ip + ":12345"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != tt.code {
			t.Errorf("IP %q: expected status %d, got %d", tt.ip, tt.code, w.Code)
		}
	}
}

func TestIPWhitelistMiddleware_Mixed(t *testing.T) {
	r := setupHardeningTestRouter(IPWhitelist([]string{"192.168.1.100", "10.0.0.0/8"}))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Error("should match exact IP")
	}

	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "10.1.2.3:12345"
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Error("should match CIDR")
	}
}

func TestIPWhitelistMiddleware_TrimsSpaces(t *testing.T) {
	r := setupHardeningTestRouter(IPWhitelist([]string{"  192.168.1.100  "}))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("IP entries should be trimmed, got status %d", w.Code)
	}
}

func TestIsIPAllowed(t *testing.T) {
	tests := []struct {
		ip      string
		allowed []string
		want    bool
	}{
		{"192.168.1.1", []string{"192.168.1.1"}, true},
		{"192.168.1.2", []string{"192.168.1.1"}, false},
		{"192.168.1.5", []string{"192.168.1.0/24"}, true},
		{"192.168.2.5", []string{"192.168.1.0/24"}, false},
		{"10.0.0.1", []string{"192.168.1.0/24", "10.0.0.0/8"}, true},
		{"192.168.1.1", []string{}, false},
		{"192.168.1.1", nil, false},
		{"192.168.1.1", []string{"", "  "}, false},
	}
	for _, tt := range tests {
		got := isIPAllowed(tt.ip, tt.allowed)
		if got != tt.want {
			t.Errorf("isIPAllowed(%q, %v) = %v, want %v", tt.ip, tt.allowed, got, tt.want)
		}
	}
}

func TestMatchCIDR(t *testing.T) {
	tests := []struct {
		ip   string
		cidr string
		want bool
	}{
		{"192.168.1.1", "192.168.1.0/24", true},
		{"192.168.1.255", "192.168.1.0/24", true},
		{"192.168.2.1", "192.168.1.0/24", false},
		{"10.0.0.1", "10.0.0.0/8", true},
		{"invalid-ip", "192.168.1.0/24", false},
		{"192.168.1.1", "invalid-cidr", false},
		{"192.168.1.1", "", false},
	}
	for _, tt := range tests {
		got := matchCIDR(tt.ip, tt.cidr)
		if got != tt.want {
			t.Errorf("matchCIDR(%q, %q) = %v, want %v", tt.ip, tt.cidr, got, tt.want)
		}
	}
}

func TestDomainBinding_ReturnsJSONError(t *testing.T) {
	r := setupHardeningTestRouter(DomainBinding([]string{"example.com"}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "evil.com"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("should return JSON error, got content-type: %s", contentType)
	}
	if !strings.Contains(w.Body.String(), "domain not allowed") {
		t.Errorf("response should mention domain not allowed, got: %s", w.Body.String())
	}
}

func TestIPWhitelistMiddleware_ReturnsJSONError(t *testing.T) {
	r := setupHardeningTestRouter(IPWhitelist([]string{"192.168.1.0/24"}))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("should return JSON error, got content-type: %s", contentType)
	}
	if !strings.Contains(w.Body.String(), "IP not allowed") {
		t.Errorf("response should mention IP not allowed, got: %s", w.Body.String())
	}
}
