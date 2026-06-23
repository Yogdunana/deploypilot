package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestIsValidScope(t *testing.T) {
	tests := []struct {
		scope  string
		expect bool
	}{
		{ScopeRead, true},
		{ScopeWrite, true},
		{ScopeDelete, true},
		{ScopeDeploy, true},
		{ScopeAdmin, true},
		{ScopeMonitorRead, true},
		{ScopeServerRead, true},
		{ScopeCredentialRead, true},
		{ScopeDNSWrite, true},
		{ScopeSSLWrite, true},
		{ScopeBackupRead, true},
		{ScopeWebhookManage, true},
		{ScopeGrafanaManage, true},
		// Invalid scopes
		{"", false},
		{"unknown", false},
		{"Read", false},        // case-sensitive
		{"monitor:writez", false}, // typo
		{"server:", false},
		{"' OR 1=1 --", false},
	}
	for _, tc := range tests {
		got := IsValidScope(tc.scope)
		if got != tc.expect {
			t.Errorf("IsValidScope(%q) = %v, want %v", tc.scope, got, tc.expect)
		}
	}
}

func TestAllScopesContainEveryConstant(t *testing.T) {
	// AllScopes must include every defined scope constant.
	all := make(map[string]bool, len(AllScopes))
	for _, s := range AllScopes {
		all[s] = true
	}
	constants := []string{
		ScopeRead, ScopeWrite, ScopeDelete, ScopeDeploy, ScopeAdmin,
		ScopeMonitorRead, ScopeMonitorWrite,
		ScopeServerRead, ScopeServerExec,
		ScopeCredentialRead, ScopeCredentialWrite,
		ScopeDNSWrite, ScopeSSLWrite,
		ScopeBackupRead, ScopeBackupWrite,
		ScopeWebhookManage, ScopeGrafanaManage,
	}
	for _, c := range constants {
		if !all[c] {
			t.Errorf("AllScopes missing constant %q", c)
		}
	}
}

func TestValidateScopes_FiltersInvalid(t *testing.T) {
	in := []string{ScopeRead, "invalid", ScopeWrite, "", ScopeAdmin}
	got := ValidateScopes(in)
	want := []string{ScopeRead, ScopeWrite, ScopeAdmin}
	if len(got) != len(want) {
		t.Fatalf("ValidateScopes() returned %d scopes, want %d: %v", len(got), len(want), got)
	}
	for i, s := range want {
		if got[i] != s {
			t.Errorf("ValidateScopes()[%d] = %q, want %q", i, got[i], s)
		}
	}
}

func TestValidateScopes_EmptyInput(t *testing.T) {
	got := ValidateScopes(nil)
	if got == nil {
		t.Error("ValidateScopes(nil) should return non-nil empty slice")
	}
	if len(got) != 0 {
		t.Errorf("ValidateScopes(nil) length = %d, want 0", len(got))
	}
}

func TestRequireScope_AllowsAPIKeyScopeMatch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(APIKeyScopesKey), []string{ScopeRead, ScopeMonitorRead})
		c.Next()
	})
	r.Use(RequireScope(ScopeMonitorRead))
	handlerCalled := false
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("handler should be called when scope matches")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRequireScope_AdminBypass(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(APIKeyScopesKey), []string{ScopeAdmin})
		c.Next()
	})
	r.Use(RequireScope(ScopeMonitorRead, ScopeServerExec)) // unrelated required scopes
	handlerCalled := false
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("admin scope should bypass all scope checks")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRequireScope_RejectsInsufficient(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(APIKeyScopesKey), []string{ScopeRead})
		c.Next()
	})
	r.Use(RequireScope(ScopeMonitorWrite)) // user has only ScopeRead
	handlerCalled := false
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if handlerCalled {
		t.Error("handler should not be called when scope is missing")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
	// Verify response is JSON with an error key.
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if body["status"] != "error" {
		t.Errorf("expected status=error in body, got %q", body["status"])
	}
}

func TestRequireScope_FallsBackToOAuth2Scopes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		// No API key scopes set, but OAuth2 scopes are available.
		c.Set(string(OAuth2ScopesKey), []string{ScopeServerRead})
		c.Next()
	})
	r.Use(RequireScope(ScopeServerRead))
	handlerCalled := false
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("handler should be called when OAuth2 scope matches")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRequireScope_NoScopesPassesThrough(t *testing.T) {
	// If no scopes are present (e.g., JWT auth path), the middleware should
	// pass through and let downstream RoleRequired middleware handle authz.
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(RequireScope(ScopeServerRead))
	handlerCalled := false
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("handler should be called when no scopes are present")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
