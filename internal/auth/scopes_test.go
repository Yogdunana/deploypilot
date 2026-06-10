package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestIsValidScope(t *testing.T) {
	tests := []struct {
		name     string
		scope    string
		expected bool
	}{
		{"valid scope read", ScopeRead, true},
		{"valid scope write", ScopeWrite, true},
		{"valid scope delete", ScopeDelete, true},
		{"valid scope deploy", ScopeDeploy, true},
		{"valid scope admin", ScopeAdmin, true},
		{"valid scope monitor read", ScopeMonitorRead, true},
		{"valid scope monitor write", ScopeMonitorWrite, true},
		{"valid scope server read", ScopeServerRead, true},
		{"valid scope server exec", ScopeServerExec, true},
		{"valid scope credential read", ScopeCredentialRead, true},
		{"valid scope credential write", ScopeCredentialWrite, true},
		{"valid scope dns write", ScopeDNSWrite, true},
		{"valid scope ssl write", ScopeSSLWrite, true},
		{"valid scope backup read", ScopeBackupRead, true},
		{"valid scope backup write", ScopeBackupWrite, true},
		{"valid scope webhook manage", ScopeWebhookManage, true},
		{"valid scope grafana manage", ScopeGrafanaManage, true},
		{"invalid scope", "invalid_scope", false},
		{"empty scope", "", false},
		{"scope with typo", "readd", false},
		{"scope with space", "read write", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidScope(tt.scope); got != tt.expected {
				t.Errorf("IsValidScope(%q) = %v, want %v", tt.scope, got, tt.expected)
			}
		})
	}
}

func TestValidateScopes(t *testing.T) {
	tests := []struct {
		name     string
		scopes   []string
		expected []string
	}{
		{"all valid scopes", []string{"read", "write", "deploy"}, []string{"read", "write", "deploy"}},
		{"mixed valid and invalid", []string{"read", "invalid", "write", "unknown"}, []string{"read", "write"}},
		{"all invalid", []string{"invalid1", "invalid2"}, []string{}},
		{"empty input", []string{}, []string{}},
		{"nil input", nil, nil},
		{"admin scope", []string{"admin", "read"}, []string{"admin", "read"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateScopes(tt.scopes)
			if len(got) != len(tt.expected) {
				t.Errorf("ValidateScopes(%v) = %v, want %v", tt.scopes, got, tt.expected)
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("ValidateScopes(%v)[%d] = %q, want %q", tt.scopes, i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestScopeDescriptions_Completeness(t *testing.T) {
	for _, scope := range AllScopes {
		if _, exists := ScopeDescriptions[scope]; !exists {
			t.Errorf("ScopeDescriptions missing description for scope %q", scope)
		}
	}
}

func TestRequireScope_AdminBypass(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/protected", nil)
	ctx.Set(string(APIKeyScopesKey), []string{ScopeAdmin})

	middleware := RequireScope("write")
	middleware(ctx)

	if w.Code != http.StatusOK {
		t.Errorf("expected status OK for admin scope bypass, got %d", w.Code)
	}
}

func TestRequireScope_HasRequiredScope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/protected", nil)
	ctx.Set(string(OAuth2ScopesKey), []string{"read"})

	middleware := RequireScope("read", "write")
	middleware(ctx)

	if w.Code != http.StatusOK {
		t.Errorf("expected status OK when has required scope, got %d", w.Code)
	}
}

func TestRequireScope_NoScopes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/protected", nil)

	middleware := RequireScope("read")
	middleware(ctx)

	if w.Code != http.StatusOK {
		t.Errorf("expected status OK when no scopes present (JWT fallback), got %d", w.Code)
	}
}

func TestRequireScope_InsufficientScope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/protected", nil)
	ctx.Set(string(APIKeyScopesKey), []string{"read"})

	middleware := RequireScope("admin")
	middleware(ctx)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status Forbidden for insufficient scope, got %d", w.Code)
	}
}

func TestRequireScope_MultipleRequiredScopes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		scopes   []string
		wantCode int
	}{
		{"has write", []string{"write"}, http.StatusOK},
		{"has deploy", []string{"deploy"}, http.StatusOK},
		{"has both", []string{"write", "deploy"}, http.StatusOK},
		{"has neither", []string{"read"}, http.StatusForbidden},
		{"empty scopes", []string{}, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/protected", nil)
			ctx.Set(string(OAuth2ScopesKey), tt.scopes)

			middleware := RequireScope("write", "deploy")
			middleware(ctx)

			if w.Code != tt.wantCode {
				t.Errorf("got status %d, want %d", w.Code, tt.wantCode)
			}
		})
	}
}