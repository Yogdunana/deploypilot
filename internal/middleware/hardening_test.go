package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// --- SecurityEntrance tests ---

func TestSecurityEntrance_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityEntrance(""))
	r.GET("/dashboard", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/dashboard", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with empty entrance, got %d", w.Code)
	}
}

func TestSecurityEntrance_ValidPrefix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityEntrance("/my-secret-panel"))
	r.GET("/*path", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/my-secret-panel/dashboard", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for valid entrance prefix, got %d", w.Code)
	}
}

func TestSecurityEntrance_InvalidPrefix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityEntrance("/my-secret-panel"))
	r.GET("/*path", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/dashboard", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for invalid entrance prefix, got %d", w.Code)
	}
}

func TestSecurityEntrance_SkipsHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityEntrance("/my-secret-panel"))
	r.GET("/health", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for /health, got %d", w.Code)
	}
}

func TestSecurityEntrance_SkipsAPIPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityEntrance("/my-secret-panel"))
	r.GET("/api/apps", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/api/apps", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for /api/ path, got %d", w.Code)
	}
}

func TestSecurityEntrance_SkipsWebSocketPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityEntrance("/my-secret-panel"))
	r.GET("/ws/monitor", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/ws/monitor", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for /ws/ path, got %d", w.Code)
	}
}

func TestSecurityEntrance_SkipsStaticAssets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityEntrance("/my-secret-panel"))
	r.GET("/assets/app.js", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/assets/app.js", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for /assets/ path, got %d", w.Code)
	}
}

func TestSecurityEntrance_StripsPrefix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var receivedPath string
	r := gin.New()
	r.Use(SecurityEntrance("/secret"))
	r.GET("/*path", func(c *gin.Context) {
		receivedPath = c.Request.URL.Path
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/secret/dashboard", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if receivedPath != "/dashboard" {
		t.Errorf("expected path /dashboard, got %s", receivedPath)
	}
}

func TestSecurityEntrance_RootPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var receivedPath string
	r := gin.New()
	r.Use(SecurityEntrance("/secret"))
	r.GET("/*path", func(c *gin.Context) {
		receivedPath = c.Request.URL.Path
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/secret", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if receivedPath != "/" {
		t.Errorf("expected path /, got %s", receivedPath)
	}
}

// --- DomainBinding tests ---

func TestDomainBinding_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(DomainBinding(nil))
	r.GET("/test", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with no domain binding, got %d", w.Code)
	}
}

func TestDomainBinding_AllowedDomain(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(DomainBinding([]string{"example.com"}))
	r.GET("/test", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/test", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for allowed domain, got %d", w.Code)
	}
}

func TestDomainBinding_DeniedDomain(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(DomainBinding([]string{"example.com"}))
	r.GET("/test", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/test", nil)
	req.Host = "evil.com"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for denied domain, got %d", w.Code)
	}
}

func TestDomainBinding_PortStripped(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(DomainBinding([]string{"example.com"}))
	r.GET("/test", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/test", nil)
	req.Host = "example.com:8080"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when domain matches after stripping port, got %d", w.Code)
	}
}

func TestDomainBinding_CaseInsensitive(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(DomainBinding([]string{"Example.COM"}))
	r.GET("/test", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/test", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for case-insensitive match, got %d", w.Code)
	}
}

// --- IPWhitelist tests ---

func TestIPWhitelist_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(IPWhitelist(nil))
	r.GET("/test", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with empty whitelist, got %d", w.Code)
	}
}

func TestIPWhitelist_AllowedIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(IPWhitelist([]string{"192.168.1.1"}))
	r.GET("/test", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for allowed IP, got %d", w.Code)
	}
}

func TestIPWhitelist_DeniedIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(IPWhitelist([]string{"192.168.1.1"}))
	r.GET("/test", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for denied IP, got %d", w.Code)
	}
}

// --- isIPAllowed and matchCIDR unit tests ---

func TestIsIPAllowed_ExactMatch(t *testing.T) {
	if !isIPAllowed("192.168.1.1", []string{"192.168.1.1"}) {
		t.Error("expected exact IP match")
	}
}

func TestIsIPAllowed_NoMatch(t *testing.T) {
	if isIPAllowed("10.0.0.1", []string{"192.168.1.1"}) {
		t.Error("expected no match for different IP")
	}
}

func TestIsIPAllowed_CIDRMatch(t *testing.T) {
	if !isIPAllowed("192.168.1.50", []string{"192.168.1.0/24"}) {
		t.Error("expected CIDR match")
	}
}

func TestIsIPAllowed_CIDRNoMatch(t *testing.T) {
	if isIPAllowed("192.168.2.1", []string{"192.168.1.0/24"}) {
		t.Error("expected no CIDR match for out-of-range IP")
	}
}

func TestIsIPAllowed_EmptyEntry(t *testing.T) {
	if isIPAllowed("10.0.0.1", []string{""}) {
		t.Error("expected no match for empty entry")
	}
}

func TestMatchCIDR_Valid(t *testing.T) {
	tests := []struct {
		ip      string
		cidr    string
		want    bool
	}{
		{"192.168.1.1", "192.168.1.0/24", true},
		{"192.168.1.255", "192.168.1.0/24", true},
		{"192.168.2.1", "192.168.1.0/24", false},
		{"10.0.0.1", "10.0.0.0/8", true},
		{"11.0.0.1", "10.0.0.0/8", false},
		{"invalid", "192.168.1.0/24", false},
		{"192.168.1.1", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip+"_"+tt.cidr, func(t *testing.T) {
			if got := matchCIDR(tt.ip, tt.cidr); got != tt.want {
				t.Errorf("matchCIDR(%q, %q) = %v, want %v", tt.ip, tt.cidr, got, tt.want)
			}
		})
	}
}
