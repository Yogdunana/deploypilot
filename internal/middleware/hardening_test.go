package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// ===================== isIPAllowed (pure function) =====================

func TestIsIPAllowed_EmptyList(t *testing.T) {
	if isIPAllowed("10.0.0.1", nil) {
		t.Error("empty allow list should reject")
	}
}

func TestIsIPAllowed_ExactMatch(t *testing.T) {
	allowed := []string{"10.0.0.1", "192.168.1.10"}
	if !isIPAllowed("10.0.0.1", allowed) {
		t.Error("expected 10.0.0.1 to be allowed by exact match")
	}
	if isIPAllowed("10.0.0.2", allowed) {
		t.Error("expected 10.0.0.2 to be rejected")
	}
}

func TestIsIPAllowed_CIDRMatch(t *testing.T) {
	allowed := []string{"192.168.1.0/24"}
	if !isIPAllowed("192.168.1.42", allowed) {
		t.Error("expected 192.168.1.42 to be allowed by CIDR")
	}
	if isIPAllowed("192.168.2.1", allowed) {
		t.Error("expected 192.168.2.1 to be rejected by CIDR")
	}
}

func TestIsIPAllowed_InvalidIPRejected(t *testing.T) {
	allowed := []string{"10.0.0.1"}
	if isIPAllowed("not-an-ip", allowed) {
		t.Error("invalid IP should be rejected")
	}
}

func TestIsIPAllowed_InvalidCIDRIgnored(t *testing.T) {
	allowed := []string{"not-a-cidr/24", "10.0.0.0/8"}
	// The malformed entry should be skipped, the valid one should still match.
	if !isIPAllowed("10.5.6.7", allowed) {
		t.Error("valid CIDR should still match when other entries are malformed")
	}
}

func TestIsIPAllowed_WhitespaceAndEmptyEntries(t *testing.T) {
	allowed := []string{"", "  ", "10.0.0.1"}
	if !isIPAllowed("10.0.0.1", allowed) {
		t.Error("expected 10.0.0.1 to be allowed despite empty/whitespace entries")
	}
}

func TestIsIPAllowed_IPv6NotMatchedByIPv4CIDR(t *testing.T) {
	// net.ParseCIDR("10.0.0.0/8") returns an IPv4 network; ::1 should not match.
	allowed := []string{"10.0.0.0/8"}
	if isIPAllowed("::1", allowed) {
		t.Error("IPv6 address should not match an IPv4 CIDR")
	}
}

// ===================== IPWhitelist middleware =====================

func newIPWhitelistServer(t *testing.T, allowed []string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(IPWhitelist(allowed))
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	return r
}

func TestIPWhitelist_AllowsListed(t *testing.T) {
	r := newIPWhitelistServer(t, []string{"10.0.0.1"})
	req := httptest.NewRequest("GET", "/ping", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestIPWhitelist_RejectsUnlisted(t *testing.T) {
	r := newIPWhitelistServer(t, []string{"10.0.0.1"})
	req := httptest.NewRequest("GET", "/ping", nil)
	req.RemoteAddr = "203.0.113.5:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestIPWhitelist_AllowsCIDR(t *testing.T) {
	r := newIPWhitelistServer(t, []string{"192.168.1.0/24"})
	req := httptest.NewRequest("GET", "/ping", nil)
	req.RemoteAddr = "192.168.1.50:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for CIDR-allowed IP, got %d", w.Code)
	}
}

func TestIPWhitelist_EmptyListIsNoOp(t *testing.T) {
	r := newIPWhitelistServer(t, nil)
	req := httptest.NewRequest("GET", "/ping", nil)
	req.RemoteAddr = "203.0.113.5:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when no IP allowlist is configured, got %d", w.Code)
	}
}

// ===================== DomainBinding middleware =====================

func newDomainBindingServer(t *testing.T, allowed []string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(DomainBinding(allowed))
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	return r
}

func TestDomainBinding_AllowsListedDomain(t *testing.T) {
	r := newDomainBindingServer(t, []string{"app.example.com"})
	req := httptest.NewRequest("GET", "/ping", nil)
	req.Host = "app.example.com"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for allowed host, got %d", w.Code)
	}
}

func TestDomainBinding_RejectsUnlistedDomain(t *testing.T) {
	r := newDomainBindingServer(t, []string{"app.example.com"})
	req := httptest.NewRequest("GET", "/ping", nil)
	req.Host = "attacker.example.com"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for unlisted host, got %d", w.Code)
	}
}

func TestDomainBinding_CaseInsensitive(t *testing.T) {
	r := newDomainBindingServer(t, []string{"app.example.com"})
	req := httptest.NewRequest("GET", "/ping", nil)
	req.Host = "APP.EXAMPLE.COM"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for case-insensitive match, got %d", w.Code)
	}
}

func TestDomainBinding_StripsPort(t *testing.T) {
	r := newDomainBindingServer(t, []string{"app.example.com"})
	req := httptest.NewRequest("GET", "/ping", nil)
	req.Host = "app.example.com:8443"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for host with port, got %d", w.Code)
	}
}

func TestDomainBinding_EmptyListIsNoOp(t *testing.T) {
	r := newDomainBindingServer(t, nil)
	req := httptest.NewRequest("GET", "/ping", nil)
	req.Host = "anywhere.example.com"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when no domain allowlist is configured, got %d", w.Code)
	}
}

func TestDomainBinding_BlankEntriesIgnored(t *testing.T) {
	r := newDomainBindingServer(t, []string{"", "  ", "app.example.com"})
	req := httptest.NewRequest("GET", "/ping", nil)
	req.Host = "app.example.com"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for valid host despite blank entries, got %d", w.Code)
	}
}

// ===================== SecurityEntrance middleware =====================

// newEntranceServer returns a router where every URL is served by a single
// catch-all handler. This mirrors the production use case where the entrance
// guard sits in front of a static SPA, and the middleware's path-stripping
// behaviour is observable through the request that reaches the handler.
func newEntranceServer(t *testing.T, entrance string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityEntrance(entrance))
	r.NoRoute(func(c *gin.Context) { c.String(http.StatusOK, "root") })
	r.GET("/health", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	r.GET("/api/v1/ping", func(c *gin.Context) { c.String(http.StatusOK, "api") })
	r.GET("/ws/socket", func(c *gin.Context) { c.String(http.StatusOK, "ws") })
	r.GET("/assets/app.js", func(c *gin.Context) { c.String(http.StatusOK, "asset") })
	return r
}

func TestSecurityEntrance_EmptyIsNoOp(t *testing.T) {
	r := newEntranceServer(t, "")
	req := httptest.NewRequest("GET", "/dashboard", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when entrance is empty, got %d", w.Code)
	}
}

func TestSecurityEntrance_AllowsPrefixedPath(t *testing.T) {
	r := newEntranceServer(t, "/secret")
	req := httptest.NewRequest("GET", "/secret/dashboard", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for prefixed dashboard, got %d", w.Code)
	}
}

func TestSecurityEntrance_RejectsUnprefixedPath(t *testing.T) {
	r := newEntranceServer(t, "/secret")
	req := httptest.NewRequest("GET", "/dashboard", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unprefixed dashboard, got %d", w.Code)
	}
}

func TestSecurityEntrance_StripsPrefixForSPA(t *testing.T) {
	r := newEntranceServer(t, "/secret")

	// Use a NoRoute handler that records the (rewritten) path it sees.
	var seenPath string
	r.NoRoute(func(c *gin.Context) {
		seenPath = c.Request.URL.Path
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/secret/inside", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if seenPath != "/inside" {
		t.Errorf("expected path to be stripped to /inside, got %q", seenPath)
	}
}

func TestSecurityEntrance_StripsPrefixAtRoot(t *testing.T) {
	r := newEntranceServer(t, "/secret")

	// Hitting /secret/ exactly should rewrite to "/".
	var seenPath string
	r.NoRoute(func(c *gin.Context) {
		seenPath = c.Request.URL.Path
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/secret", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if seenPath != "/" {
		t.Errorf("expected path to be /, got %q", seenPath)
	}
}

func TestSecurityEntrance_AllowsAPIPaths(t *testing.T) {
	r := newEntranceServer(t, "/secret")
	req := httptest.NewRequest("GET", "/api/v1/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for API path, got %d", w.Code)
	}
}

func TestSecurityEntrance_AllowsWebSocketPaths(t *testing.T) {
	r := newEntranceServer(t, "/secret")
	req := httptest.NewRequest("GET", "/ws/socket", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for ws path, got %d", w.Code)
	}
}

func TestSecurityEntrance_AllowsStaticAssets(t *testing.T) {
	r := newEntranceServer(t, "/secret")
	req := httptest.NewRequest("GET", "/assets/app.js", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for static assets path, got %d", w.Code)
	}
}

func TestSecurityEntrance_AllowsHealthCheck(t *testing.T) {
	r := newEntranceServer(t, "/secret")
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for /health, got %d", w.Code)
	}
}

func TestSecurityEntrance_NormalizesLeadingTrailingSlashes(t *testing.T) {
	r := newEntranceServer(t, "secret/") // user provided trailing slash
	req := httptest.NewRequest("GET", "/secret/dashboard", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when entrance is normalized, got %d", w.Code)
	}
}
