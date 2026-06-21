package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestIsValidScope locks in the set of accepted scope strings. A
// regression in AllScopes (a typo, an accidental rename, or a duplicate)
// would silently break permission checks.
func TestIsValidScope(t *testing.T) {
	for _, s := range AllScopes {
		if !IsValidScope(s) {
			t.Errorf("IsValidScope(%q) = false, expected true (declared in AllScopes)", s)
		}
	}

	// Spot check some non-scope strings.
	for _, s := range []string{"", "ADMIN", "monitor:write ", "read\n", "nonexistent:scope"} {
		if IsValidScope(s) {
			t.Errorf("IsValidScope(%q) = true, expected false", s)
		}
	}
}

// TestValidateScopes ensures that unknown scopes are dropped and
// known scopes are kept, preserving order. This is the function the
// OAuth2 flow calls to clean up user-supplied scope lists.
func TestValidateScopes(t *testing.T) {
	got := ValidateScopes([]string{"read", "admin", "FAKE", "deploy", "monitor:read"})
	want := []string{"read", "admin", "deploy", "monitor:read"}
	if len(got) != len(want) {
		t.Fatalf("ValidateScopes len = %d, want %d (got=%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ValidateScopes[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestValidateScopes_EmptyInput documents the no-op behavior for an
// empty or nil input: the result is a non-nil empty slice.
func TestValidateScopes_EmptyInput(t *testing.T) {
	if got := ValidateScopes(nil); got == nil {
		t.Error("ValidateScopes(nil) = nil, want empty slice")
	} else if len(got) != 0 {
		t.Errorf("ValidateScopes(nil) len = %d, want 0", len(got))
	}
	if got := ValidateScopes([]string{}); len(got) != 0 {
		t.Errorf("ValidateScopes([]) len = %d, want 0", len(got))
	}
}

// TestScopeDescriptions_AllCovered is a self-consistency check: every
// scope listed in AllScopes must have a human-readable description.
// Otherwise the admin UI would render an empty tooltip.
func TestScopeDescriptions_AllCovered(t *testing.T) {
	for _, s := range AllScopes {
		if _, ok := ScopeDescriptions[s]; !ok {
			t.Errorf("ScopeDescriptions missing entry for %q", s)
		}
	}
}

// runWithScopes issues a GET to /protected with the given scopes
// placed in the gin context under the OAuth2ScopesKey, applies
// RequireScope with the given required scopes, and reports the
// resulting status code.
func runWithScopes(t *testing.T, tokenScopes, required []string) int {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if tokenScopes != nil {
			c.Set(string(OAuth2ScopesKey), tokenScopes)
		}
		c.Next()
	})
	r.Use(RequireScope(required...))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

// TestRequireScope_AdminBypass is the central security guarantee:
// possession of the "admin" scope must bypass every RequireScope
// check, regardless of which specific scopes are required.
func TestRequireScope_AdminBypass(t *testing.T) {
	for _, required := range [][]string{
		{"deploy"},
		{"delete", "ssl:write"},
		{"nonexistent:scope"},
	} {
		code := runWithScopes(t, []string{ScopeAdmin}, required)
		if code != http.StatusOK {
			t.Errorf("admin should bypass RequireScope(%v), got %d", required, code)
		}
	}
}

// TestRequireScope_Satisfied allows the request when the token holds
// at least one of the required scopes.
func TestRequireScope_Satisfied(t *testing.T) {
	cases := []struct {
		token    []string
		required []string
	}{
		{[]string{"read"}, []string{"read"}},
		{[]string{"read", "deploy"}, []string{"deploy", "admin"}},
		{[]string{"monitor:read"}, []string{"monitor:read", "monitor:write"}},
	}
	for _, tc := range cases {
		code := runWithScopes(t, tc.token, tc.required)
		if code != http.StatusOK {
			t.Errorf("RequireScope(%v) with token %v: got %d, want 200", tc.required, tc.token, code)
		}
	}
}

// TestRequireScope_Insufficient returns 403 with a localized error
// body when none of the required scopes are present.
func TestRequireScope_Insufficient(t *testing.T) {
	code := runWithScopes(t, []string{"read"}, []string{"delete"})
	if code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", code)
	}
}

// TestRequireScope_NoScopesIsAllow is a critical edge case: when no
// scopes are attached (e.g., a JWT-authenticated request), the
// middleware must let the request through. JWT-level permissions are
// already enforced by RoleRequired upstream; blocking here would
// cause a regression for every existing JWT user.
func TestRequireScope_NoScopesIsAllow(t *testing.T) {
	code := runWithScopes(t, nil, []string{"deploy"})
	if code != http.StatusOK {
		t.Errorf("expected 200 when no scopes present, got %d", code)
	}
}

// TestRequireScope_RejectionMessageIsLocalized checks that the 403
// response body contains a human-readable error string (via i18n),
// not a developer-facing panic or empty body. The exact wording is
// driven by the embedded locale files.
func TestRequireScope_RejectionMessageIsLocalized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(OAuth2ScopesKey), []string{"read"})
		c.Next()
	})
	r.Use(RequireScope("deploy"))
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "message") {
		t.Errorf("expected 403 body to contain a 'message' field, got: %s", body)
	}
}
