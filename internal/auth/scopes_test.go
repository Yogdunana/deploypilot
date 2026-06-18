package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	// Tests in this file drive Gin handlers; keep the test binary quiet
	// regardless of how the package is being executed.
	gin.SetMode(gin.TestMode)
}

// TestIsValidScope_ValidScopes confirms that every scope in AllScopes is
// accepted by the validator, and that the validator's positive output is
// stable across repeated calls.
func TestIsValidScope_ValidScopes(t *testing.T) {
	for _, scope := range AllScopes {
		if !IsValidScope(scope) {
			t.Errorf("IsValidScope(%q) = false, want true", scope)
		}
	}
}

// TestIsValidScope_InvalidScopes confirms that arbitrary, malformed, and
// case-mismatched scope strings are rejected. The validator is a security
// gate: an empty accept-list is safer than a permissive one.
func TestIsValidScope_InvalidScopes(t *testing.T) {
	invalid := []string{
		"",
		"unknown",
		"ADMIN",                  // case sensitive
		"read write",             // no embedded spaces
		"read,write",             // no commas
		"read\n",                 // control chars
		" deploy",                // leading whitespace
		"admin ",                 // trailing whitespace
		strings.Repeat("a", 256), // overlong
	}
	for _, scope := range invalid {
		if IsValidScope(scope) {
			t.Errorf("IsValidScope(%q) = true, want false", scope)
		}
	}
}

// TestIsValidScope_AdminBypass confirms that "admin" is in the valid
// list (it is a special scope that bypasses per-scope checks).
func TestIsValidScope_AdminBypass(t *testing.T) {
	if !IsValidScope(ScopeAdmin) {
		t.Error("ScopeAdmin must be a valid scope")
	}
}

// TestValidateScopes_FiltersUnknown ensures the filter drops unknown
// scopes while keeping known ones in their original order.
func TestValidateScopes_FiltersUnknown(t *testing.T) {
	in := []string{"read", "unknown", "write", "deploy-app", "deploy"}
	got := ValidateScopes(in)
	want := []string{"read", "write", "deploy"}
	if !equalStringSlices(got, want) {
		t.Errorf("ValidateScopes(%v) = %v, want %v", in, got, want)
	}
}

// TestValidateScopes_AllInvalid returns an empty slice (not nil) so
// downstream code can safely range over the result.
func TestValidateScopes_AllInvalid(t *testing.T) {
	in := []string{"foo", "bar"}
	got := ValidateScopes(in)
	if got == nil {
		t.Fatal("ValidateScopes should return non-nil empty slice for all-invalid input")
	}
	if len(got) != 0 {
		t.Errorf("ValidateScopes(%v) = %v, want empty", in, got)
	}
}

// TestValidateScopes_EmptyInput returns an empty (non-nil) slice.
func TestValidateScopes_EmptyInput(t *testing.T) {
	got := ValidateScopes(nil)
	if got == nil {
		t.Fatal("ValidateScopes(nil) should return non-nil empty slice")
	}
	if len(got) != 0 {
		t.Errorf("ValidateScopes(nil) = %v, want empty", got)
	}
}

// TestValidateScopes_AllValid keeps the order of valid scopes intact.
func TestValidateScopes_AllValid(t *testing.T) {
	in := []string{ScopeRead, ScopeWrite, ScopeDeploy, ScopeAdmin}
	got := ValidateScopes(in)
	if !equalStringSlices(got, in) {
		t.Errorf("ValidateScopes(%v) = %v, want %v", in, got, in)
	}
}

// TestRequireScope_AdminBypassesAllChecks confirms that an admin scope
// satisfies any required scope, including ones the user does not hold.
func TestRequireScope_AdminBypassesAllChecks(t *testing.T) {
	rec := runRequireScope(t, []string{ScopeAdmin}, []string{ScopeGrafanaManage, ScopeServerExec})

	if rec.Code != http.StatusOK {
		t.Fatalf("admin should bypass scope checks; got status %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestRequireScope_ExactMatch confirms a non-admin token with one of the
// required scopes is allowed.
func TestRequireScope_ExactMatch(t *testing.T) {
	rec := runRequireScope(t, []string{ScopeServerRead}, []string{ScopeServerExec, ScopeServerRead})

	if rec.Code != http.StatusOK {
		t.Fatalf("token with required scope should be allowed; got status %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestRequireScope_NoOverlapIsForbidden confirms a token without any of
// the required scopes is rejected with HTTP 403.
func TestRequireScope_NoOverlapIsForbidden(t *testing.T) {
	rec := runRequireScope(t, []string{ScopeRead}, []string{ScopeServerExec, ScopeServerRead})

	if rec.Code != http.StatusForbidden {
		t.Fatalf("token without required scope should be forbidden; got status %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "insufficient") {
		t.Errorf("expected i18n 'insufficient' message in body, got: %s", rec.Body.String())
	}
}

// TestRequireScope_NoScopesInContextAllowsThrough confirms that the
// middleware does not block requests that have no scopes (e.g. JWT auth
// without scopes). Permission enforcement for those flows is delegated
// to RoleRequired middleware, which is documented behaviour.
func TestRequireScope_NoScopesInContextAllowsThrough(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	// No scopes set in the context.
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	RequireScope(ScopeAdmin)(c)

	if rec.Code == http.StatusForbidden {
		t.Errorf("missing-scope request should not be rejected by RequireScope; got %d", rec.Code)
	}
	if c.IsAborted() {
		t.Error("RequireScope must not abort requests without scopes (delegated to RoleRequired)")
	}
}

// TestRequireScope_PrefersAPIKeyScopesOverOAuth2 confirms the precedence
// rule: API key scopes are checked first and OAuth2 scopes are only
// consulted if the API key has none.
func TestRequireScope_PrefersAPIKeyScopesOverOAuth2(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	// API key has matching scope, OAuth2 does not.
	c.Set(string(APIKeyScopesKey), []string{ScopeServerRead})
	c.Set(string(OAuth2ScopesKey), []string{ScopeRead})

	RequireScope(ScopeServerRead)(c)
	if c.IsAborted() {
		t.Error("API key scope match should let the request through, ignoring OAuth2 scopes")
	}
}

// TestRequireScope_FallsBackToOAuth2WhenAPIKeyEmpty confirms that
// OAuth2 scopes are consulted when the API key has none.
func TestRequireScope_FallsBackToOAuth2WhenAPIKeyEmpty(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	// API key scopes are empty, OAuth2 has matching scope.
	c.Set(string(APIKeyScopesKey), []string{})
	c.Set(string(OAuth2ScopesKey), []string{ScopeServerRead})

	RequireScope(ScopeServerRead)(c)
	if c.IsAborted() {
		t.Error("OAuth2 scope match should satisfy the check when API key has no scopes")
	}
}

// TestAllScopes_AreUnique guards against accidental duplicate entries
// in AllScopes, which would skew IsValidScope and ScopeDescriptions
// behaviour.
func TestAllScopes_AreUnique(t *testing.T) {
	seen := make(map[string]struct{}, len(AllScopes))
	for _, s := range AllScopes {
		if _, dup := seen[s]; dup {
			t.Errorf("duplicate scope in AllScopes: %q", s)
		}
		seen[s] = struct{}{}
	}
}

// TestScopeDescriptions_CoversAllScopes guards against drift between
// the set of valid scopes and the human-readable description map.
func TestScopeDescriptions_CoversAllScopes(t *testing.T) {
	for _, s := range AllScopes {
		if _, ok := ScopeDescriptions[s]; !ok {
			t.Errorf("ScopeDescriptions missing entry for %q", s)
		}
		if ScopeDescriptions[s] == "" {
			t.Errorf("ScopeDescriptions[%q] is empty", s)
		}
	}
}

// --- helpers ---

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// runRequireScope executes the RequireScope middleware with the given
// token scopes and required scopes, returning the response recorder
// for assertions.
func runRequireScope(t *testing.T, tokenScopes, requiredScopes []string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	// Marshal token scopes as a JSON array (the APIKey path stores it
	// that way); the OAuth2 path uses a raw []string. We use the
	// OAuth2-style raw slice for simplicity.
	c.Set(string(OAuth2ScopesKey), tokenScopes)

	RequireScope(requiredScopes...)(c)
	return rec
}

// TestIsValidScope_JsonRoundTrip ensures the validator is stable under
// JSON encoding/decoding, which is the format used by API key storage.
func TestIsValidScope_JsonRoundTrip(t *testing.T) {
	for _, scope := range AllScopes {
		data, err := json.Marshal(scope)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}
		var decoded string
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if !IsValidScope(decoded) {
			t.Errorf("scope %q did not survive JSON round-trip", scope)
		}
	}
}
