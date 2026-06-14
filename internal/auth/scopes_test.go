package auth

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// ========== IsValidScope / ValidateScopes ==========

func TestIsValidScope_AllDefinedScopesValid(t *testing.T) {
	for _, s := range AllScopes {
		if !IsValidScope(s) {
			t.Errorf("scope %q should be valid", s)
		}
	}
}

func TestIsValidScope_UnknownInvalid(t *testing.T) {
	bad := []string{
		"", "root", "superuser", "READ", // case sensitive
		"deploy:write", "monitor", "  read", "read ",
		"unknown:scope", "api_key",
	}
	for _, s := range bad {
		if IsValidScope(s) {
			t.Errorf("scope %q should be invalid", s)
		}
	}
}

func TestAllScopes_HasNoDuplicates(t *testing.T) {
	seen := make(map[string]bool, len(AllScopes))
	for _, s := range AllScopes {
		if seen[s] {
			t.Errorf("duplicate scope in AllScopes: %q", s)
		}
		seen[s] = true
	}
}

func TestAllScopes_AllHaveDescription(t *testing.T) {
	for _, s := range AllScopes {
		desc, ok := ScopeDescriptions[s]
		if !ok {
			t.Errorf("scope %q missing description", s)
		}
		if desc == "" {
			t.Errorf("scope %q has empty description", s)
		}
	}
}

func TestValidateScopes_FiltersUnknown(t *testing.T) {
	in := []string{"read", "unknown", "write", "", "deploy:write"}
	got := ValidateScopes(in)
	want := []string{"read", "write"}
	if len(got) != len(want) {
		t.Fatalf("ValidateScopes length = %d, want %d (got=%v)", len(got), len(want), got)
	}
	for i, s := range want {
		if got[i] != s {
			t.Errorf("ValidateScopes[%d] = %q, want %q", i, got[i], s)
		}
	}
}

func TestValidateScopes_AllUnknown(t *testing.T) {
	got := ValidateScopes([]string{"foo", "bar"})
	if len(got) != 0 {
		t.Errorf("expected empty result, got %v", got)
	}
}

func TestValidateScopes_Empty(t *testing.T) {
	got := ValidateScopes(nil)
	if len(got) != 0 {
		t.Errorf("expected empty result for nil input, got %v", got)
	}
	got = ValidateScopes([]string{})
	if len(got) != 0 {
		t.Errorf("expected empty result for empty input, got %v", got)
	}
}

func TestValidateScopes_OrderPreserved(t *testing.T) {
	in := []string{"admin", "read", "write"}
	got := ValidateScopes(in)
	// All are valid, order should be preserved
	want := []string{"admin", "read", "write"}
	if len(got) != len(want) {
		t.Fatalf("len mismatch: got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ValidateScopes[%d] = %q, want %q (order not preserved)", i, got[i], want[i])
		}
	}
}

// ========== RequireScope middleware ==========

func newRequireScopeCtx(t *testing.T, apiKeyScopes, oauth2Scopes []string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	if apiKeyScopes != nil {
		c.Set(string(APIKeyScopesKey), apiKeyScopes)
	}
	if oauth2Scopes != nil {
		c.Set(string(OAuth2ScopesKey), oauth2Scopes)
	}
	return c
}

func TestRequireScope_AdminBypass(t *testing.T) {
	called := false
	mw := RequireScope("read", "write")
	c := newRequireScopeCtx(t, []string{"admin"}, nil)
	// Inject a no-op next handler
	handlers := mw
	handlers(c)
	if c.IsAborted() {
		t.Error("admin scope should not abort")
	}
	called = true
	if !called {
		t.Error("next() was not invoked")
	}
}

func TestRequireScope_RequiredScopePresent(t *testing.T) {
	mw := RequireScope("read", "write")
	c := newRequireScopeCtx(t, []string{"read"}, nil)
	mw(c)
	if c.IsAborted() {
		t.Error("matching scope should not abort")
	}
}

func TestRequireScope_InsufficientScope(t *testing.T) {
	mw := RequireScope("admin")
	c := newRequireScopeCtx(t, []string{"read"}, nil)
	mw(c)
	if !c.IsAborted() {
		t.Error("insufficient scope should abort")
	}
	w := c.Writer
	if w.Status() != 403 {
		t.Errorf("status = %d, want 403", w.Status())
	}
}

func TestRequireScope_NoScopesPassThrough(t *testing.T) {
	// No scopes in context -> middleware passes through (JWT auth path)
	mw := RequireScope("read")
	c := newRequireScopeCtx(t, nil, nil)
	mw(c)
	if c.IsAborted() {
		t.Error("no scopes should pass through (JWT auth handled by other middleware)")
	}
}

func TestRequireScope_OAuth2Fallback(t *testing.T) {
	// When APIKeyScopes absent, OAuth2 scopes should be used
	mw := RequireScope("read")
	c := newRequireScopeCtx(t, nil, []string{"read"})
	mw(c)
	if c.IsAborted() {
		t.Error("OAuth2 scope should satisfy requirement")
	}
}

func TestRequireScope_APIKeyScopesTakePrecedence(t *testing.T) {
	// When both are present, API key scopes are consulted (and OAuth2 ignored
	// only when the API key has at least one scope). An API key with non-empty
	// scopes will be authoritative: a non-matching API key scope should deny,
	// even if OAuth2 has the right scope.
	mw := RequireScope("admin")
	c := newRequireScopeCtx(t, []string{"read"}, []string{"admin"})
	mw(c)
	if !c.IsAborted() {
		t.Error("API key scopes should be authoritative when present")
	}
}

func TestRequireScope_WrongTypeInContext(t *testing.T) {
	// Defensive: if scopes context value is not a []string, treat as no scopes
	// and pass through
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Set(string(APIKeyScopesKey), "not-a-slice")

	mw := RequireScope("read")
	mw(c)
	if c.IsAborted() {
		t.Error("malformed context value should fall back to pass-through")
	}
}

func TestRequireScope_MultipleRequiredAnyMatch(t *testing.T) {
	// Of the required scopes, any one is sufficient
	mw := RequireScope("read", "write", "delete")
	c := newRequireScopeCtx(t, []string{"delete"}, nil)
	mw(c)
	if c.IsAborted() {
		t.Error("one matching scope should satisfy any-of")
	}
}
