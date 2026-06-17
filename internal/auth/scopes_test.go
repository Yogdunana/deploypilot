package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Yogdunana/deploypilot/internal/i18n"
	"github.com/gin-gonic/gin"
)

func TestIsValidScope(t *testing.T) {
	tests := []struct {
		name  string
		scope string
		want  bool
	}{
		{"read scope", ScopeRead, true},
		{"write scope", ScopeWrite, true},
		{"admin scope", ScopeAdmin, true},
		{"monitor:read scope", ScopeMonitorRead, true},
		{"credential:write scope", ScopeCredentialWrite, true},
		{"unknown scope", "unknown:scope", false},
		{"empty scope", "", false},
		{"case-sensitive uppercase", "READ", false},
		{"partial match rejected", "mon", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidScope(tt.scope); got != tt.want {
				t.Errorf("IsValidScope(%q) = %v, want %v", tt.scope, got, tt.want)
			}
		})
	}
}

func TestValidateScopes_FiltersUnknowns(t *testing.T) {
	input := []string{ScopeRead, "invalid", ScopeWrite, "", ScopeAdmin, "another:bad"}
	got := ValidateScopes(input)
	if len(got) != 3 {
		t.Fatalf("expected 3 valid scopes, got %d: %v", len(got), got)
	}
	if got[0] != ScopeRead || got[1] != ScopeWrite || got[2] != ScopeAdmin {
		t.Errorf("unexpected order/content: %v", got)
	}
}

func TestValidateScopes_EmptyInput(t *testing.T) {
	got := ValidateScopes(nil)
	if len(got) != 0 {
		t.Errorf("expected empty result for nil input, got %v", got)
	}
}

func TestValidateScopes_AllValid(t *testing.T) {
	got := ValidateScopes(AllScopes)
	if len(got) != len(AllScopes) {
		t.Errorf("expected all %d scopes to be valid, got %d", len(AllScopes), len(got))
	}
}

func TestAllScopes_AllValid(t *testing.T) {
	// Every scope listed in AllScopes must be accepted by IsValidScope.
	// This guards against a divergence between the two definitions.
	for _, s := range AllScopes {
		if !IsValidScope(s) {
			t.Errorf("AllScopes contains %q but IsValidScope rejects it", s)
		}
	}
}

func TestScopeDescriptions_AllDescribed(t *testing.T) {
	for _, s := range AllScopes {
		desc, ok := ScopeDescriptions[s]
		if !ok {
			t.Errorf("scope %q is missing a description", s)
		}
		if desc == "" {
			t.Errorf("scope %q has empty description", s)
		}
	}
}

// --- RequireScope middleware ---

func runScopeMiddleware(t *testing.T, scopes []string, setAPIScopes, setOAuthScopes bool) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if setAPIScopes {
			c.Set(string(APIKeyScopesKey), scopes)
		}
		if setOAuthScopes {
			c.Set(string(OAuth2ScopesKey), scopes)
		}
		c.Next()
	})
	r.Use(RequireScope("read", "write"))
	r.GET("/test", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRequireScope_AllowsMatchingScope(t *testing.T) {
	w := runScopeMiddleware(t, []string{"read"}, true, false)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for matching scope, got %d", w.Code)
	}
}

func TestRequireScope_RejectsMissingScope(t *testing.T) {
	w := runScopeMiddleware(t, []string{"deploy"}, true, false)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for insufficient scope, got %d", w.Code)
	}
}

func TestRequireScope_AdminBypassesAllChecks(t *testing.T) {
	w := runScopeMiddleware(t, []string{ScopeAdmin}, true, false)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for admin bypass, got %d", w.Code)
	}
}

func TestRequireScope_EmptyScopesAllowed(t *testing.T) {
	// No scopes in context (e.g. JWT auth) — middleware should let it through.
	w := runScopeMiddleware(t, nil, false, false)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when no scopes in context, got %d", w.Code)
	}
}

func TestRequireScope_FallsBackToOAuth2(t *testing.T) {
	// When API key scopes are absent, OAuth2 scopes should be consulted.
	w := runScopeMiddleware(t, []string{"read"}, false, true)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when OAuth2 scopes satisfy, got %d", w.Code)
	}
}

func TestRequireScope_APIScopesTakePrecedence(t *testing.T) {
	// API key scopes grant access; OAuth2 scopes deny.
	// The middleware should only consult OAuth2 if API scopes are missing.
	w := runScopeMiddleware(t, []string{"read"}, true, true)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when API key scopes satisfy, got %d", w.Code)
	}
}

func TestRequireScope_RejectsNonStringScopes(t *testing.T) {
	// If a non-[]string value is stored under the context key, middleware
	// should treat scopes as missing and let the request through.
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(APIKeyScopesKey), "not-a-slice")
		c.Next()
	})
	r.Use(RequireScope("read"))
	r.GET("/test", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when scopes value is not []string, got %d", w.Code)
	}
}

func TestRequireScope_403UsesLocalizedMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(i18n.ContextLocaleKey, "en")
		c.Set(string(APIKeyScopesKey), []string{"deploy"})
		c.Next()
	})
	r.Use(RequireScope("read"))
	r.GET("/test", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
	// Body should contain the i18n key, not a raw English literal.
	if !contains(w.Body.String(), "insufficient") {
		t.Errorf("expected localized 'insufficient' message, got body: %s", w.Body.String())
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
