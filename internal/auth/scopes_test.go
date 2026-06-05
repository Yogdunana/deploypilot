package auth

import (
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
		// Valid scopes
		{ScopeRead, true},
		{ScopeWrite, true},
		{ScopeDelete, true},
		{ScopeDeploy, true},
		{ScopeAdmin, true},
		{ScopeMonitorRead, true},
		{ScopeMonitorWrite, true},
		{ScopeServerRead, true},
		{ScopeServerExec, true},
		{ScopeCredentialRead, true},
		{ScopeCredentialWrite, true},
		{ScopeDNSWrite, true},
		{ScopeSSLWrite, true},
		{ScopeBackupRead, true},
		{ScopeBackupWrite, true},
		{ScopeWebhookManage, true},
		{ScopeGrafanaManage, true},
		// Invalid scopes
		{"invalid_scope", false},
		{"", false},
		{"READ", false},   // case-sensitive
		{"Write", false},  // case-sensitive
		{"admin ", false}, // no trailing space
		{" admin", false}, // no leading space
	}

	for _, tt := range tests {
		t.Run(tt.scope, func(t *testing.T) {
			got := IsValidScope(tt.scope)
			if got != tt.expect {
				t.Errorf("IsValidScope(%q) = %v, want %v", tt.scope, got, tt.expect)
			}
		})
	}
}

func TestValidateScopes(t *testing.T) {
	tests := []struct {
		name   string
		scopes []string
		want   []string
	}{
		{
			name:   "all valid scopes",
			scopes: []string{ScopeRead, ScopeWrite, ScopeDeploy},
			want:   []string{ScopeRead, ScopeWrite, ScopeDeploy},
		},
		{
			name:   "mixed valid and invalid",
			scopes: []string{ScopeRead, "invalid", ScopeWrite, "bogus"},
			want:   []string{ScopeRead, ScopeWrite},
		},
		{
			name:   "all invalid scopes",
			scopes: []string{"invalid", "bogus"},
			want:   nil,
		},
		{
			name:   "empty input",
			scopes: []string{},
			want:   nil,
		},
		{
			name:   "nil input",
			scopes: nil,
			want:   nil,
		},
		{
			name:   "duplicate valid scopes",
			scopes: []string{ScopeRead, ScopeRead, ScopeWrite},
			want:   []string{ScopeRead, ScopeRead, ScopeWrite}, // ValidateScopes doesn't deduplicate
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateScopes(tt.scopes)
			if len(got) != len(tt.want) {
				t.Errorf("ValidateScopes() returned %d items, want %d", len(got), len(tt.want))
				return
			}
			for i, s := range got {
				if s != tt.want[i] {
					t.Errorf("ValidateScopes()[%d] = %q, want %q", i, s, tt.want[i])
				}
			}
		})
	}
}

func TestRequireScope_NoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// When no scopes are found in context, RequireScope should call c.Next()
	// (this allows JWT auth without scopes to pass through)
	handler := RequireScope(ScopeRead)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if c.IsAborted() {
		t.Error("context should not be aborted when no scopes found")
	}
}

func TestRequireScope_AdminBypassesAllChecks(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := RequireScope(ScopeAdmin)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set(string(APIKeyScopesKey), []string{ScopeAdmin})

	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if c.IsAborted() {
		t.Error("admin scope should bypass all checks")
	}
}

func TestRequireScope_AdminBypassesOAuth2(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := RequireScope(ScopeRead)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set(string(OAuth2ScopesKey), []string{ScopeAdmin})

	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if c.IsAborted() {
		t.Error("admin scope via OAuth2 should bypass all checks")
	}
}

func TestRequireScope_ApiKeyHasRequiredScope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := RequireScope(ScopeRead)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set(string(APIKeyScopesKey), []string{ScopeRead, ScopeWrite})

	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if c.IsAborted() {
		t.Error("context should not be aborted when API key has required scope")
	}
}

func TestRequireScope_ApiKeyMissingRequiredScope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := RequireScope(ScopeAdmin) // requires admin, but API key only has read

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set(string(APIKeyScopesKey), []string{ScopeRead, ScopeWrite})

	handler(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
	if !c.IsAborted() {
		t.Error("context should be aborted when scope is insufficient")
	}
}

func TestRequireScope_OAuth2HasRequiredScope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := RequireScope(ScopeWrite)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set(string(OAuth2ScopesKey), []string{ScopeWrite, ScopeRead})

	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if c.IsAborted() {
		t.Error("context should not be aborted when OAuth2 has required scope")
	}
}

func TestRequireScope_OAuth2MissingRequiredScope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := RequireScope(ScopeDeploy)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set(string(OAuth2ScopesKey), []string{ScopeRead}) // missing deploy scope

	handler(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
	if !c.IsAborted() {
		t.Error("context should be aborted when OAuth2 scope is insufficient")
	}
}

func TestRequireScope_MultipleRequiredScopes_OneMatches(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Requires either ScopeRead OR ScopeWrite
	handler := RequireScope(ScopeRead, ScopeWrite)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set(string(APIKeyScopesKey), []string{ScopeRead})

	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestRequireScope_ApiKeyTakesPrecedence(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := RequireScope(ScopeAdmin)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	// API key scopes take precedence over OAuth2
	c.Set(string(APIKeyScopesKey), []string{ScopeAdmin})
	c.Set(string(OAuth2ScopesKey), []string{ScopeRead}) // would fail if OAuth2 was used

	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestRequireScope_FallsBackToOAuth2(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := RequireScope(ScopeWrite)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	// No API key scopes, should fall back to OAuth2
	c.Set(string(OAuth2ScopesKey), []string{ScopeWrite, ScopeRead})

	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestRequireScope_ResponseContainsErrorMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := RequireScope(ScopeAdmin)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set(string(APIKeyScopesKey), []string{ScopeRead}) // insufficient

	handler(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}

	// Verify error response format
	body := w.Body.String()
	if body == "" {
		t.Error("expected non-empty response body")
	}
}

func TestAllScopes_ContainsAllConstants(t *testing.T) {
	expectedScopes := []string{
		ScopeRead, ScopeWrite, ScopeDelete, ScopeDeploy, ScopeAdmin,
		ScopeMonitorRead, ScopeMonitorWrite,
		ScopeServerRead, ScopeServerExec,
		ScopeCredentialRead, ScopeCredentialWrite,
		ScopeDNSWrite, ScopeSSLWrite,
		ScopeBackupRead, ScopeBackupWrite,
		ScopeWebhookManage, ScopeGrafanaManage,
	}

	if len(AllScopes) != len(expectedScopes) {
		t.Errorf("AllScopes has %d items, want %d", len(AllScopes), len(expectedScopes))
	}

	scopeSet := make(map[string]bool)
	for _, s := range AllScopes {
		scopeSet[s] = true
	}

	for _, s := range expectedScopes {
		if !scopeSet[s] {
			t.Errorf("AllScopes missing expected scope %q", s)
		}
	}
}

func TestScopeDescriptions_HasAllScopes(t *testing.T) {
	for _, scope := range AllScopes {
		desc, ok := ScopeDescriptions[scope]
		if !ok {
			t.Errorf("ScopeDescriptions missing description for scope %q", scope)
		}
		if desc == "" {
			t.Errorf("ScopeDescriptions[%q] is empty", scope)
		}
	}
}
