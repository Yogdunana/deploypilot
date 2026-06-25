package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// Note: contextKey and APIKeyScopesKey are defined in middleware.go and apikey.go

func TestIsValidScope(t *testing.T) {
	tests := []struct {
		scope   string
		want    bool
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
		{"read:extra", false},
		{"READ", false}, // case sensitive
		{"admin:extra", false},
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
		name    string
		scopes  []string
		want    []string
	}{
		{
			name:   "all valid scopes",
			scopes: []string{ScopeRead, ScopeWrite, ScopeDelete},
			want:   []string{ScopeRead, ScopeWrite, ScopeDelete},
		},
		{
			name:   "mixed valid and invalid",
			scopes: []string{ScopeRead, "invalid", ScopeWrite, "another_invalid"},
			want:   []string{ScopeRead, ScopeWrite},
		},
		{
			name:   "all invalid scopes",
			scopes: []string{"invalid1", "invalid2", "invalid3"},
			want:   []string{},
		},
		{
			name:   "empty input",
			scopes: []string{},
			want:   []string{},
		},
		{
			name:   "nil input",
			scopes: nil,
			want:   []string{},
		},
		{
			name:   "duplicate scopes",
			scopes: []string{ScopeRead, ScopeRead, ScopeWrite},
			want:   []string{ScopeRead, ScopeRead, ScopeWrite},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateScopes(tt.scopes)
			if len(got) != len(tt.want) {
				t.Errorf("ValidateScopes() length = %d, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ValidateScopes()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestAllScopesContainsAllConstants(t *testing.T) {
	// Ensure AllScopes list contains all defined scope constants
	scopeConstants := []string{
		ScopeRead, ScopeWrite, ScopeDelete, ScopeDeploy, ScopeAdmin,
		ScopeMonitorRead, ScopeMonitorWrite, ScopeServerRead, ScopeServerExec,
		ScopeCredentialRead, ScopeCredentialWrite, ScopeDNSWrite, ScopeSSLWrite,
		ScopeBackupRead, ScopeBackupWrite, ScopeWebhookManage, ScopeGrafanaManage,
	}

	for _, constant := range scopeConstants {
		found := false
		for _, scope := range AllScopes {
			if scope == constant {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Scope constant %q not found in AllScopes list", constant)
		}
	}
}

func TestScopeDescriptionsComplete(t *testing.T) {
	// Ensure all scopes have descriptions
	for _, scope := range AllScopes {
		if desc, ok := ScopeDescriptions[scope]; !ok || desc == "" {
			t.Errorf("Scope %q missing description", scope)
		}
	}
}

func TestRequireScope_AdminBypass(t *testing.T) {
	// Admin scope should bypass all scope checks
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(OAuth2ScopesKey), []string{ScopeAdmin})
		c.Next()
	})
	r.Use(RequireScope(ScopeRead, ScopeWrite, ScopeDelete))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Admin bypass failed: status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequireScope_HasRequiredScope(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(APIKeyScopesKey), []string{ScopeRead, ScopeWrite})
		c.Next()
	})
	r.Use(RequireScope(ScopeWrite))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Has required scope failed: status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequireScope_MissingRequiredScope(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(OAuth2ScopesKey), []string{ScopeRead})
		c.Next()
	})
	r.Use(RequireScope(ScopeWrite, ScopeDelete))
	r.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Missing scope should return 403: status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestRequireScope_NoScopesInContext(t *testing.T) {
	// When no scopes are set (JWT auth), middleware should allow through
	// JWT auth permissions are handled by RoleRequired middleware
	r := gin.New()
	r.Use(RequireScope(ScopeRead))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("No scopes should allow through: status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequireScope_PrefersAPIKeyScopes(t *testing.T) {
	// API key scopes should be checked before OAuth2 scopes
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(APIKeyScopesKey), []string{ScopeRead})
		c.Set(string(OAuth2ScopesKey), []string{ScopeWrite}) // Different scope
		c.Next()
	})
	r.Use(RequireScope(ScopeRead))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Should use API key scopes first: status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequireScope_FallbackToOAuth2Scopes(t *testing.T) {
	// When API key scopes not set, should use OAuth2 scopes
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(OAuth2ScopesKey), []string{ScopeDeploy})
		c.Next()
	})
	r.Use(RequireScope(ScopeDeploy))
	r.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Should fallback to OAuth2 scopes: status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequireScope_MultipleRequiredScopes(t *testing.T) {
	// Any one of the required scopes should allow access
	tests := []struct {
		name        string
		userScopes  []string
		required    []string
		wantStatus  int
	}{
		{
			name:       "has first required",
			userScopes: []string{ScopeRead},
			required:   []string{ScopeRead, ScopeWrite},
			wantStatus: http.StatusOK,
		},
		{
			name:       "has second required",
			userScopes: []string{ScopeWrite},
			required:   []string{ScopeRead, ScopeWrite},
			wantStatus: http.StatusOK,
		},
		{
			name:       "has none required",
			userScopes: []string{ScopeDelete},
			required:   []string{ScopeRead, ScopeWrite},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "admin bypasses all",
			userScopes: []string{ScopeAdmin},
			required:   []string{ScopeRead, ScopeWrite, ScopeDelete, ScopeDeploy},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(func(c *gin.Context) {
				c.Set(string(OAuth2ScopesKey), tt.userScopes)
				c.Next()
			})
			r.Use(RequireScope(tt.required...))
			r.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/test", nil)
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}