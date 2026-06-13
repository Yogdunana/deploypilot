package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestIsValidScope verifies that the scope validator returns the correct
// result for known, unknown, and edge-case inputs. This is a security-critical
// check used by the OAuth2 / API key scope gate.
func TestIsValidScope(t *testing.T) {
	cases := []struct {
		name  string
		scope string
		want  bool
	}{
		{"empty", "", false},
		{"read", ScopeRead, true},
		{"admin", ScopeAdmin, true},
		{"monitor:read", ScopeMonitorRead, true},
		{"unknown", "totally.made.up", false},
		{"case-sensitive", "READ", false},
		{"whitespace", " read", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsValidScope(tc.scope); got != tc.want {
				t.Errorf("IsValidScope(%q) = %v, want %v", tc.scope, got, tc.want)
			}
		})
	}
}

// TestValidateScopes filters out unknown entries but preserves valid ones
// and the original order — this matters because the resulting slice is
// returned to clients via the OAuth2 /token endpoint.
func TestValidateScopes(t *testing.T) {
	in := []string{ScopeRead, "bogus", ScopeDeploy, "", ScopeAdmin, "another-bad"}
	got := ValidateScopes(in)
	want := []string{ScopeRead, ScopeDeploy, ScopeAdmin}
	if len(got) != len(want) {
		t.Fatalf("ValidateScopes() length = %d, want %d (got=%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ValidateScopes()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestValidateScopes_AllValid is the happy path: every scope is recognized.
func TestValidateScopes_AllValid(t *testing.T) {
	in := AllScopes
	got := ValidateScopes(in)
	if len(got) != len(in) {
		t.Fatalf("expected all %d scopes valid, got %d", len(in), len(got))
	}
}

// TestValidateScopes_Empty ensures an empty input produces a non-nil empty slice
// so callers can safely range over the result without a nil check.
func TestValidateScopes_Empty(t *testing.T) {
	got := ValidateScopes(nil)
	if got == nil {
		t.Fatal("expected non-nil empty slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
	}
}

// requireScopeEngine builds a gin engine whose only middleware is RequireScope,
// plus an outer middleware that injects the supplied scopes into the context.
// A sentinel handler records whether it was called.
func requireScopeEngine(t *testing.T, scopes ...string) (*gin.Engine, *bool) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	called := false
	r.Use(RequireScope(scopes...))
	r.GET("/test", func(c *gin.Context) {
		called = true
		c.Status(http.StatusOK)
	})
	return r, &called
}

// withAPIKeyScopes is a helper that wraps a router with middleware injecting
// the supplied API-key scopes into the request context.
func withAPIKeyScopes(scopes []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(string(APIKeyScopesKey), scopes)
		c.Next()
	}
}

func withOAuth2Scopes(scopes []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(string(OAuth2ScopesKey), scopes)
		c.Next()
	}
}

func doGET(r *gin.Engine) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestRequireScope_AdminBypass verifies the documented behavior that the
// admin scope is treated as a superset of every other scope.
func TestRequireScope_AdminBypass(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	called := false
	// Order matters: scope-injector must run BEFORE RequireScope.
	r.Use(withAPIKeyScopes([]string{ScopeAdmin}))
	r.Use(RequireScope(ScopeMonitorWrite, ScopeServerExec))
	r.GET("/test", func(c *gin.Context) {
		called = true
		c.Status(http.StatusOK)
	})

	w := doGET(r)
	if w.Code != http.StatusOK {
		t.Errorf("admin scope should bypass checks, got %d", w.Code)
	}
	if !called {
		t.Error("expected handler to be invoked for admin scope")
	}
}

// TestRequireScope_RequiredMatch is the happy path: token has the
// required scope and the handler is reached.
func TestRequireScope_RequiredMatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	called := false
	r.Use(withAPIKeyScopes([]string{ScopeRead, ScopeMonitorRead}))
	r.Use(RequireScope(ScopeMonitorRead))
	r.GET("/test", func(c *gin.Context) {
		called = true
		c.Status(http.StatusOK)
	})

	w := doGET(r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !called {
		t.Error("expected handler to be called")
	}
}

// TestRequireScope_Insufficient exercises the failure path: token has scopes
// but none of them satisfies the requirement, and we must return 403.
func TestRequireScope_Insufficient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	called := false
	r.Use(withAPIKeyScopes([]string{ScopeRead}))
	r.Use(RequireScope(ScopeDeploy))
	r.GET("/test", func(c *gin.Context) {
		called = true
		c.Status(http.StatusOK)
	})

	w := doGET(r)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
	if called {
		t.Error("handler should NOT be called when scopes are insufficient")
	}
}

// TestRequireScope_NoScopes covers the documented behavior that JWT auth
// (which sets no APIKeyScopesKey / OAuth2ScopesKey) is allowed through, with
// permissions handled by RoleRequired middleware instead.
func TestRequireScope_NoScopes(t *testing.T) {
	r, called := requireScopeEngine(t, ScopeWrite)
	w := doGET(r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 (no scopes present), got %d", w.Code)
	}
	if !*called {
		t.Error("expected handler to be called when no scopes are present")
	}
}

// TestRequireScope_OAuth2Fallback ensures that when the APIKey context entry
// is absent, the middleware falls back to OAuth2ScopesKey.
func TestRequireScope_OAuth2Fallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	called := false
	r.Use(withOAuth2Scopes([]string{ScopeCredentialRead}))
	r.Use(RequireScope(ScopeCredentialRead))
	r.GET("/test", func(c *gin.Context) {
		called = true
		c.Status(http.StatusOK)
	})

	w := doGET(r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !called {
		t.Error("expected handler to be called via OAuth2 scope fallback")
	}
}

// TestAllScopes_Unique guards against accidental scope-constant duplication,
// which would inflate the public permission catalog and confuse clients.
func TestAllScopes_Unique(t *testing.T) {
	seen := make(map[string]struct{}, len(AllScopes))
	for _, s := range AllScopes {
		if _, dup := seen[s]; dup {
			t.Errorf("duplicate scope in AllScopes: %q", s)
		}
		seen[s] = struct{}{}
	}
	if len(seen) != len(AllScopes) {
		t.Errorf("AllScopes length %d, unique count %d", len(AllScopes), len(seen))
	}
}
