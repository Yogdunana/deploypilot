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
		{"valid scope monitor:read", ScopeMonitorRead, true},
		{"valid scope monitor:write", ScopeMonitorWrite, true},
		{"valid scope server:read", ScopeServerRead, true},
		{"valid scope server:exec", ScopeServerExec, true},
		{"valid scope credential:read", ScopeCredentialRead, true},
		{"valid scope credential:write", ScopeCredentialWrite, true},
		{"valid scope dns:write", ScopeDNSWrite, true},
		{"valid scope ssl:write", ScopeSSLWrite, true},
		{"valid scope backup:read", ScopeBackupRead, true},
		{"valid scope backup:write", ScopeBackupWrite, true},
		{"valid scope webhook:manage", ScopeWebhookManage, true},
		{"valid scope grafana:manage", ScopeGrafanaManage, true},
		{"invalid scope", "invalid", false},
		{"empty scope", "", false},
		{"scope with typo", "readd", false},
		{"scope with wrong prefix", "monitor:delete", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidScope(tt.scope)
			if result != tt.expected {
				t.Errorf("IsValidScope(%q) = %v, want %v", tt.scope, result, tt.expected)
			}
		})
	}
}

func TestValidateScopes(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{"all valid", []string{ScopeRead, ScopeWrite}, []string{ScopeRead, ScopeWrite}},
		{"mixed valid and invalid", []string{ScopeRead, "invalid", ScopeAdmin}, []string{ScopeRead, ScopeAdmin}},
		{"all invalid", []string{"invalid1", "invalid2"}, []string{}},
		{"empty input", []string{}, []string{}},
		{"nil input", nil, []string{}},
		{"duplicate scopes", []string{ScopeRead, ScopeRead, ScopeRead}, []string{ScopeRead, ScopeRead, ScopeRead}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateScopes(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("ValidateScopes(%v) returned %d items, want %d", tt.input, len(result), len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("ValidateScopes(%v)[%d] = %q, want %q", tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestRequireScope_WithAPIKeyScopes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	handlerCalled := false
	r.Use(func(c *gin.Context) {
		c.Set(string(APIKeyScopesKey), []string{ScopeRead, ScopeWrite})
		c.Next()
	})
	r.Use(RequireScope(ScopeRead))
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("expected handler to be called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRequireScope_WithOAuth2Scopes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	handlerCalled := false
	r.Use(func(c *gin.Context) {
		c.Set(string(OAuth2ScopesKey), []string{ScopeDeploy})
		c.Next()
	})
	r.Use(RequireScope(ScopeDeploy))
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("expected handler to be called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRequireScope_AdminBypass(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	handlerCalled := false
	r.Use(func(c *gin.Context) {
		c.Set(string(APIKeyScopesKey), []string{ScopeAdmin})
		c.Next()
	})
	r.Use(RequireScope(ScopeDelete, ScopeDeploy))
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("expected handler to be called with admin scope")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRequireScope_InsufficientScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	handlerCalled := false
	r.Use(func(c *gin.Context) {
		c.Set(string(APIKeyScopesKey), []string{ScopeRead})
		c.Next()
	})
	r.Use(RequireScope(ScopeDelete))
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if handlerCalled {
		t.Error("expected handler NOT to be called")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestRequireScope_NoScopes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	handlerCalled := false
	r.Use(RequireScope(ScopeRead))
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("expected handler to be called when no scopes in context")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRequireScope_MultipleRequiredScopes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name          string
		tokenScopes   []string
		requiredScopes []string
		expectedCode  int
	}{
		{"one matches", []string{ScopeRead, ScopeWrite}, []string{ScopeWrite, ScopeDelete}, http.StatusOK},
		{"none match", []string{ScopeRead}, []string{ScopeWrite, ScopeDelete}, http.StatusForbidden},
		{"all match", []string{ScopeRead, ScopeWrite}, []string{ScopeRead, ScopeWrite}, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			handlerCalled := false

			r.Use(func(c *gin.Context) {
				c.Set(string(APIKeyScopesKey), tt.tokenScopes)
				c.Next()
			})
			r.Use(RequireScope(tt.requiredScopes...))
			r.GET("/test", func(c *gin.Context) {
				handlerCalled = true
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if tt.expectedCode == http.StatusOK && !handlerCalled {
				t.Error("expected handler to be called")
			}
			if tt.expectedCode == http.StatusForbidden && handlerCalled {
				t.Error("expected handler NOT to be called")
			}
			if w.Code != tt.expectedCode {
				t.Errorf("expected %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}