package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestIsValidScope covers the static scope catalog.
func TestIsValidScope(t *testing.T) {
	for _, s := range AllScopes {
		if !IsValidScope(s) {
			t.Errorf("IsValidScope(%q) = false, expected true", s)
		}
	}
	invalid := []string{
		"",
		"unknown",
		"READ",       // case-sensitive
		"superadmin", // not in catalog
		" monitor:read",
	}
	for _, s := range invalid {
		if IsValidScope(s) {
			t.Errorf("IsValidScope(%q) = true, expected false", s)
		}
	}
}

// TestValidateScopes_KeepsOnlyKnown verifies that unknown entries are dropped
// from the input slice while known entries are preserved in order.
func TestValidateScopes_KeepsOnlyKnown(t *testing.T) {
	in := []string{"read", "unknown", "deploy", "", "monitor:read", "gibberish"}
	got := ValidateScopes(in)
	want := []string{"read", "deploy", "monitor:read"}
	if len(got) != len(want) {
		t.Fatalf("len(got)=%d, want %d (got=%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d]=%q, want %q", i, got[i], want[i])
		}
	}
}

// TestValidateScopes_AllUnknownReturnsEmpty ensures that a fully invalid input
// returns a (non-nil) empty slice, which the middleware treats as "no scopes".
func TestValidateScopes_AllUnknownReturnsEmpty(t *testing.T) {
	got := ValidateScopes([]string{"foo", "bar"})
	if got == nil {
		t.Fatal("ValidateScopes should never return nil; expected empty slice")
	}
	if len(got) != 0 {
		t.Errorf("expected empty result, got %v", got)
	}
}

// runRequireScope invokes RequireScope(scopes) against an in-memory Gin
// engine, after first invoking setup to inject context values.
func runRequireScope(t *testing.T, setup func(c *gin.Context), scopes ...string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if setup != nil {
		r.Use(func(c *gin.Context) {
			setup(c)
			c.Next()
		})
	}
	r.GET("/protected", RequireScope(scopes...), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestRequireScope_NoScopes_AllowsThrough verifies that when no scope context
// is present (e.g. JWT auth without scopes), the middleware is permissive
// and lets the request through, since role-based checks are handled
// elsewhere by RoleRequired.
func TestRequireScope_NoScopes_AllowsThrough(t *testing.T) {
	w := runRequireScope(t, nil, ScopeRead)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
}

// TestRequireScope_MatchingScope_Allows ensures that a token with one of the
// required scopes is allowed to proceed.
func TestRequireScope_MatchingScope_Allows(t *testing.T) {
	w := runRequireScope(t, func(c *gin.Context) {
		c.Set(string(APIKeyScopesKey), []string{ScopeRead, ScopeWrite})
	}, ScopeDeploy, ScopeWrite)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
}

// TestRequireScope_NonMatchingScope_Rejects verifies that a token whose
// scopes do not include any of the required scopes is rejected with 403.
func TestRequireScope_NonMatchingScope_Rejects(t *testing.T) {
	w := runRequireScope(t, func(c *gin.Context) {
		c.Set(string(APIKeyScopesKey), []string{ScopeRead})
	}, ScopeDeploy, ScopeDelete)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d (%s)", w.Code, w.Body.String())
	}
}

// TestRequireScope_AdminBypass verifies that an "admin" scope on a token
// always satisfies any required-scope check, regardless of which scopes
// are required.
func TestRequireScope_AdminBypass(t *testing.T) {
	w := runRequireScope(t, func(c *gin.Context) {
		c.Set(string(APIKeyScopesKey), []string{ScopeRead, ScopeAdmin})
	}, ScopeDeploy, ScopeDelete, ScopeSSLWrite)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 (admin bypass), got %d (%s)", w.Code, w.Body.String())
	}
}

// TestRequireScope_OAuth2ScopesUsedWhenAPIKeyMissing ensures that the
// middleware falls back to OAuth2 scopes when API key scopes are not set.
func TestRequireScope_OAuth2ScopesUsedWhenAPIKeyMissing(t *testing.T) {
	w := runRequireScope(t, func(c *gin.Context) {
		c.Set(string(OAuth2ScopesKey), []string{ScopeServerRead})
	}, ScopeServerRead)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 (OAuth2 scope match), got %d (%s)", w.Code, w.Body.String())
	}
}

// TestRequireScope_APIKeyTakesPriority ensures that when both API key and
// OAuth2 scopes are present, the API key scopes win. This is the documented
// behavior of RequireScope.
func TestRequireScope_APIKeyTakesPriority(t *testing.T) {
	w := runRequireScope(t, func(c *gin.Context) {
		c.Set(string(APIKeyScopesKey), []string{ScopeRead}) // satisfies required
		c.Set(string(OAuth2ScopesKey), []string{ScopeDeploy}) // would not satisfy
	}, ScopeRead)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 (APIKey wins), got %d (%s)", w.Code, w.Body.String())
	}
}

// TestRequireScope_WrongTypeInContext verifies that if a non-[]string is
// stored under the APIKeyScopesKey (e.g. someone passed in wrong data),
// the middleware falls through to OAuth2 scopes and behaves correctly.
func TestRequireScope_WrongTypeInContext(t *testing.T) {
	w := runRequireScope(t, func(c *gin.Context) {
		c.Set(string(APIKeyScopesKey), "not-a-slice") // wrong type
		c.Set(string(OAuth2ScopesKey), []string{ScopeRead})
	}, ScopeRead)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 (fall through to OAuth2), got %d (%s)", w.Code, w.Body.String())
	}
}

// TestRequireScope_EmptyRequiredScopesWithTokenScopes checks that supplying
// no required scopes for a request still demands an admin (or matching)
// scope — i.e. the request is gated by RequireScope and a token that has
// only unrelated scopes cannot pass.
func TestRequireScope_EmptyRequiredScopesWithTokenScopes(t *testing.T) {
	w := runRequireScope(t, func(c *gin.Context) {
		c.Set(string(APIKeyScopesKey), []string{ScopeRead})
	}) // no required scopes
	// With no required scopes, the inner loop matches nothing, so the
	// request should be rejected unless the admin scope is present.
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 (no required scopes satisfied), got %d (%s)", w.Code, w.Body.String())
	}
}
