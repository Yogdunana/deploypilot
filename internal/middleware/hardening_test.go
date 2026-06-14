package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// newHardeningCtx builds a gin.Context with the given request and an
// optional list of pre-set cookies. Use it to drive individual middleware
// functions in isolation.
func newHardeningCtx(t *testing.T, method, path string, cookies map[string]string, headers map[string]string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(method, path, nil)
	for name, val := range cookies {
		req.AddCookie(&http.Cookie{Name: name, Value: val, Path: "/"})
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	c.Request = req
	return c
}

// ========== SecurityEntrance ==========

func TestSecurityEntrance_EmptyPassesAll(t *testing.T) {
	mw := SecurityEntrance("")
	c := newHardeningCtx(t, "GET", "/anything", nil, nil)
	mw(c)
	if c.IsAborted() {
		t.Error("empty entrance should not abort")
	}
}

func TestSecurityEntrance_AllowedPath(t *testing.T) {
	mw := SecurityEntrance("/secret")
	c := newHardeningCtx(t, "GET", "/secret/dashboard", nil, nil)
	mw(c)
	if c.IsAborted() {
		t.Error("expected allowed path to pass")
	}
}

func TestSecurityEntrance_ForbiddenPath(t *testing.T) {
	mw := SecurityEntrance("/secret")
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(mw)
	r.GET("/*proxy", func(c *gin.Context) { c.Status(200) })
	req := httptest.NewRequest("GET", "/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 404 {
		t.Errorf("expected 404 to hide panel, got %d", w.Code)
	}
}

func TestSecurityEntrance_StripsPrefix(t *testing.T) {
	mw := SecurityEntrance("/secret")
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(mw)
	var seen string
	r.GET("/*proxy", func(c *gin.Context) {
		seen = c.Request.URL.Path
		c.Status(200)
	})
	req := httptest.NewRequest("GET", "/secret/dashboard", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if seen != "/dashboard" {
		t.Errorf("expected stripped path /dashboard, got %q", seen)
	}
}

func TestSecurityEntrance_StripsPrefixEmpty(t *testing.T) {
	mw := SecurityEntrance("/secret")
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(mw)
	var seen string
	r.GET("/*proxy", func(c *gin.Context) {
		seen = c.Request.URL.Path
		c.Status(200)
	})
	req := httptest.NewRequest("GET", "/secret", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// /secret stripped => empty => should be normalized to "/"
	if seen != "/" {
		t.Errorf("expected path /, got %q", seen)
	}
}

func TestSecurityEntrance_NormalizesSlashes(t *testing.T) {
	// Leading and trailing slashes should be normalized
	cases := []string{"/secret", "secret/", "/secret/", "secret"}
	for _, entrance := range cases {
		mw := SecurityEntrance(entrance)
		c := newHardeningCtx(t, "GET", "/secret/x", nil, nil)
		mw(c)
		if c.IsAborted() {
			t.Errorf("entrance %q should accept /secret/x", entrance)
		}
	}
}

func TestSecurityEntrance_BypassedPaths(t *testing.T) {
	mw := SecurityEntrance("/secret")
	// Paths that should not be hidden behind the security entrance.
	allowed := []string{
		"/health",
		"/api",        // exact match
		"/api/v1/apps", // /api/ prefix
		"/swagger",     // exact match (note: /swagger/index.html is NOT bypassed)
		"/ws/monitor",
		"/assets/main.js",
		"/icon.svg",
	}
	for _, p := range allowed {
		t.Run(p, func(t *testing.T) {
			c := newHardeningCtx(t, "GET", p, nil, nil)
			mw(c)
			if c.IsAborted() {
				t.Errorf("path %q should not be hidden", p)
			}
		})
	}
}

func TestSecurityEntrance_SwaggerSubpathHidden(t *testing.T) {
	// /swagger/index.html is NOT in the bypass list — it must be hidden.
	// This guards against accidentally over-broadening the bypass set.
	mw := SecurityEntrance("/secret")
	c := newHardeningCtx(t, "GET", "/swagger/index.html", nil, nil)
	mw(c)
	if !c.IsAborted() {
		t.Error("/swagger/index.html should be hidden (only exact /swagger bypasses)")
	}
}

// ========== DomainBinding ==========

func TestDomainBinding_EmptyAllowsAll(t *testing.T) {
	mw := DomainBinding(nil)
	c := newHardeningCtx(t, "GET", "/x", nil, nil)
	mw(c)
	if c.IsAborted() {
		t.Error("empty allowlist should permit all")
	}
}

func TestDomainBinding_AllowedHost(t *testing.T) {
	mw := DomainBinding([]string{"example.com", "Example.COM", "  other.com  "})
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(mw)
	called := false
	r.GET("/x", func(c *gin.Context) { called = true; c.Status(200) })
	req := httptest.NewRequest("GET", "/x", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if !called {
		t.Errorf("expected allowed host to pass (status=%d)", w.Code)
	}
}

func TestDomainBinding_DisallowedHost(t *testing.T) {
	mw := DomainBinding([]string{"example.com"})
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(mw)
	r.GET("/x", func(c *gin.Context) { c.Status(200) })
	req := httptest.NewRequest("GET", "/x", nil)
	req.Host = "attacker.com"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Errorf("expected 403 for disallowed host, got %d", w.Code)
	}
}

func TestDomainBinding_HostWithPort(t *testing.T) {
	mw := DomainBinding([]string{"example.com"})
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(mw)
	called := false
	r.GET("/x", func(c *gin.Context) { called = true; c.Status(200) })
	req := httptest.NewRequest("GET", "/x", nil)
	req.Host = "example.com:8443"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if !called {
		t.Errorf("expected port-bearing host to match, got status=%d", w.Code)
	}
}

func TestDomainBinding_CaseInsensitive(t *testing.T) {
	mw := DomainBinding([]string{"example.com"})
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(mw)
	called := false
	r.GET("/x", func(c *gin.Context) { called = true; c.Status(200) })
	req := httptest.NewRequest("GET", "/x", nil)
	req.Host = "EXAMPLE.COM"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if !called {
		t.Errorf("expected case-insensitive match, got status=%d", w.Code)
	}
}

func TestDomainBinding_EmptyEntriesIgnored(t *testing.T) {
	mw := DomainBinding([]string{"", "  ", "example.com"})
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(mw)
	called := false
	r.GET("/x", func(c *gin.Context) { called = true; c.Status(200) })
	req := httptest.NewRequest("GET", "/x", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if !called {
		t.Errorf("whitespace-only entries should be ignored, got status=%d", w.Code)
	}
}

// ========== IPWhitelist ==========

func TestIPWhitelist_EmptyAllowsAll(t *testing.T) {
	mw := IPWhitelist(nil)
	c := newHardeningCtx(t, "GET", "/x", nil, nil)
	mw(c)
	if c.IsAborted() {
		t.Error("empty IP list should permit all")
	}
}

func TestIPWhitelist_Allowed(t *testing.T) {
	mw := IPWhitelist([]string{"10.0.0.5"})
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(mw)
	r.GET("/x", func(c *gin.Context) { c.Status(200) })
	req := httptest.NewRequest("GET", "/x", nil)
	req.RemoteAddr = "10.0.0.5:54321"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("expected allowed IP to pass, got %d", w.Code)
	}
}

func TestIPWhitelist_Denied(t *testing.T) {
	mw := IPWhitelist([]string{"10.0.0.5"})
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(mw)
	r.GET("/x", func(c *gin.Context) { c.Status(200) })
	req := httptest.NewRequest("GET", "/x", nil)
	req.RemoteAddr = "10.0.0.6:54321"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Errorf("expected 403 for blocked IP, got %d", w.Code)
	}
}

func TestIPWhitelist_CIDR(t *testing.T) {
	mw := IPWhitelist([]string{"192.168.0.0/16"})
	cases := []struct {
		ip       string
		wantCode int
	}{
		{"192.168.1.1:1234", 200},
		{"192.169.0.1:1234", 403},
		{"10.0.0.1:1234", 403},
	}
	for _, tc := range cases {
		t.Run(tc.ip, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			r := gin.New()
			r.Use(mw)
			r.GET("/x", func(c *gin.Context) { c.Status(200) })
			req := httptest.NewRequest("GET", "/x", nil)
			req.RemoteAddr = tc.ip
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.wantCode {
				t.Errorf("IP %s: got %d, want %d", tc.ip, w.Code, tc.wantCode)
			}
		})
	}
}

func TestIPWhitelist_WhitespaceEntriesIgnored(t *testing.T) {
	mw := IPWhitelist([]string{"", "  ", "10.0.0.5"})
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(mw)
	called := false
	r.GET("/x", func(c *gin.Context) { called = true; c.Status(200) })
	req := httptest.NewRequest("GET", "/x", nil)
	req.RemoteAddr = "10.0.0.5:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if !called {
		t.Errorf("whitespace entries should be ignored, got status=%d", w.Code)
	}
}

// ========== CSRF ==========

func TestCSRF_SafeMethodsPassThrough(t *testing.T) {
	mw := CSRF()
	for _, method := range []string{"GET", "HEAD", "OPTIONS"} {
		t.Run(method, func(t *testing.T) {
			c := newHardeningCtx(t, method, "/x", nil, nil)
			mw(c)
			if c.IsAborted() {
				t.Errorf("%s should pass through CSRF", method)
			}
		})
	}
}

func TestCSRF_PostWithoutCookieForbidden(t *testing.T) {
	mw := CSRF()
	c := newHardeningCtx(t, "POST", "/x", nil, nil)
	mw(c)
	if !c.IsAborted() {
		t.Error("POST without CSRF cookie should be aborted")
	}
}

func TestCSRF_PostWithMatchingTokenAllowed(t *testing.T) {
	mw := CSRF()
	token := "deadbeef-cafe-token"
	c := newHardeningCtx(t, "POST", "/x",
		map[string]string{CSRFTokenCookie: token},
		map[string]string{CSRFTokenHeader: token})
	mw(c)
	if c.IsAborted() {
		t.Errorf("matching token should not abort; body=%q", c.Writer)
	}
}

func TestCSRF_PostWithMismatchedTokenForbidden(t *testing.T) {
	mw := CSRF()
	c := newHardeningCtx(t, "POST", "/x",
		map[string]string{CSRFTokenCookie: "cookie-token"},
		map[string]string{CSRFTokenHeader: "different-token"})
	mw(c)
	if !c.IsAborted() {
		t.Error("mismatched token should abort")
	}
}

func TestCSRF_GenerateTokenIsRandomAndNonEmpty(t *testing.T) {
	tok1 := generateCSRFToken()
	tok2 := generateCSRFToken()
	if tok1 == "" || tok2 == "" {
		t.Fatal("generated token should be non-empty")
	}
	if tok1 == tok2 {
		t.Errorf("two consecutive tokens should not collide; got %q twice", tok1)
	}
	// Token is hex-encoded 32 bytes -> 64 chars
	if len(tok1) != 64 {
		t.Errorf("expected 64-char token (32 bytes hex), got %d chars", len(tok1))
	}
}

func TestCSRF_CaseInsensitiveMatch(t *testing.T) {
	mw := CSRF()
	token := "AbCdEf0123456789"
	c := newHardeningCtx(t, "PUT", "/x",
		map[string]string{CSRFTokenCookie: token},
		map[string]string{CSRFTokenHeader: "ABCDEF0123456789"})
	mw(c)
	if c.IsAborted() {
		t.Error("case-insensitive token match should not abort")
	}
}

func TestCSRF_HeaderMissingButFormHasToken(t *testing.T) {
	mw := CSRF()
	token := "form-token-123"
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	form := strings.NewReader("csrf_token=" + token + "&other=field")
	req := httptest.NewRequest("POST", "/x", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: CSRFTokenCookie, Value: token, Path: "/"})
	c.Request = req
	mw(c)
	if c.IsAborted() {
		t.Error("form-field CSRF token should satisfy middleware")
	}
}

func TestCSRF_SkipBypass(t *testing.T) {
	mw := CSRF()
	c := newHardeningCtx(t, "POST", "/x", nil, nil)
	c.Set("csrf_skip", true)
	mw(c)
	if c.IsAborted() {
		t.Error("csrf_skip should bypass middleware")
	}
}

func TestCSRF_NotExposedAsHelperExport(t *testing.T) {
	// Sanity: the package-level helpers should be reachable from the test
	// file (i.e. not renamed or unexported) — this guards accidental breakage
	// of the public security contract.
	_ = CSRFTokenCookie
	_ = CSRFTokenHeader
}
