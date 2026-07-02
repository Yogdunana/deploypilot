package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestIsValidScope(t *testing.T) {
	tests := []struct {
		scope string
		want  bool
	}{
		{"read", true},
		{"write", true},
		{"delete", true},
		{"deploy", true},
		{"admin", true},
		{"monitor:read", true},
		{"monitor:write", true},
		{"server:read", true},
		{"server:exec", true},
		{"credential:read", true},
		{"credential:write", true},
		{"dns:write", true},
		{"ssl:write", true},
		{"backup:read", true},
		{"backup:write", true},
		{"webhook:manage", true},
		{"grafana:manage", true},
		{"nonexistent", false},
		{"", false},
		{"ADMIN", false}, // case-sensitive
		{"read write", false},
	}

	for _, tt := range tests {
		t.Run(tt.scope, func(t *testing.T) {
			if got := IsValidScope(tt.scope); got != tt.want {
				t.Errorf("IsValidScope(%q) = %v, want %v", tt.scope, got, tt.want)
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
			name:   "all valid",
			scopes: []string{"read", "write"},
			want:   []string{"read", "write"},
		},
		{
			name:   "mixed valid and invalid",
			scopes: []string{"read", "invalid", "write"},
			want:   []string{"read", "write"},
		},
		{
			name:   "all invalid",
			scopes: []string{"foo", "bar"},
			want:   []string{},
		},
		{
			name:   "empty input",
			scopes: []string{},
			want:   []string{},
		},
		{
			name:   "admin scope",
			scopes: []string{"admin"},
			want:   []string{"admin"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateScopes(tt.scopes)
			if len(got) != len(tt.want) {
				t.Errorf("ValidateScopes() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ValidateScopes()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestRequireScope_NoScopesSet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handlerCalled := false
	r.Use(RequireScope("admin"))
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("expected handler to be called when no scopes set (JWT auth)")
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
		c.Set(string(APIKeyScopesKey), []string{"read", "admin"})
		c.Next()
	})
	r.Use(RequireScope("deploy"))
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("admin scope should bypass scope check")
	}
}

func TestRequireScope_SufficientScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handlerCalled := false
	r.Use(func(c *gin.Context) {
		c.Set(string(APIKeyScopesKey), []string{"read", "deploy"})
		c.Next()
	})
	r.Use(RequireScope("deploy"))
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("handler should be called with matching scope")
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
		c.Set(string(APIKeyScopesKey), []string{"read"})
		c.Next()
	})
	r.Use(RequireScope("deploy"))
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if handlerCalled {
		t.Error("handler should NOT be called with insufficient scope")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestRequireScope_OAuth2Scopes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handlerCalled := false
	r.Use(func(c *gin.Context) {
		c.Set(string(OAuth2ScopesKey), []string{"monitor:read"})
		c.Next()
	})
	r.Use(RequireScope("monitor:read"))
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("handler should be called with OAuth2 scope")
	}
}

func TestRequireScope_APIKeyScopesTakePrecedence(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handlerCalled := false
	r.Use(func(c *gin.Context) {
		c.Set(string(APIKeyScopesKey), []string{"read"})
		c.Set(string(OAuth2ScopesKey), []string{"deploy"})
		c.Next()
	})
	r.Use(RequireScope("deploy"))
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// API key scopes take precedence, and they don't have "deploy"
	if handlerCalled {
		t.Error("API key scopes should take precedence over OAuth2 scopes")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestAllScopesConsistency(t *testing.T) {
	// Verify AllScopes matches ScopeDescriptions
	for _, scope := range AllScopes {
		if _, ok := ScopeDescriptions[scope]; !ok {
			t.Errorf("scope %q in AllScopes but missing from ScopeDescriptions", scope)
		}
	}
	for scope := range ScopeDescriptions {
		if !IsValidScope(scope) {
			t.Errorf("scope %q in ScopeDescriptions but not in AllScopes", scope)
		}
	}
}
