package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestIsValidScope(t *testing.T) {
	if !IsValidScope(ScopeRead) {
		t.Error("expected ScopeRead to be valid")
	}
	if !IsValidScope(ScopeWrite) {
		t.Error("expected ScopeWrite to be valid")
	}
	if !IsValidScope(ScopeAdmin) {
		t.Error("expected ScopeAdmin to be valid")
	}
	if !IsValidScope(ScopeDeploy) {
		t.Error("expected ScopeDeploy to be valid")
	}
	if !IsValidScope(ScopeMonitorRead) {
		t.Error("expected ScopeMonitorRead to be valid")
	}
	if !IsValidScope(ScopeCredentialRead) {
		t.Error("expected ScopeCredentialRead to be valid")
	}
	if !IsValidScope(ScopeDNSWrite) {
		t.Error("expected ScopeDNSWrite to be valid")
	}
	if !IsValidScope(ScopeSSLWrite) {
		t.Error("expected ScopeSSLWrite to be valid")
	}
	if !IsValidScope(ScopeBackupRead) {
		t.Error("expected ScopeBackupRead to be valid")
	}
	if !IsValidScope(ScopeBackupWrite) {
		t.Error("expected ScopeBackupWrite to be valid")
	}
	if !IsValidScope(ScopeWebhookManage) {
		t.Error("expected ScopeWebhookManage to be valid")
	}
	if !IsValidScope(ScopeGrafanaManage) {
		t.Error("expected ScopeGrafanaManage to be valid")
	}
	if !IsValidScope(ScopeServerRead) {
		t.Error("expected ScopeServerRead to be valid")
	}
	if !IsValidScope(ScopeServerExec) {
		t.Error("expected ScopeServerExec to be valid")
	}
	if !IsValidScope(ScopeCredentialWrite) {
		t.Error("expected ScopeCredentialWrite to be valid")
	}
	if !IsValidScope(ScopeDelete) {
		t.Error("expected ScopeDelete to be valid")
	}
	if !IsValidScope(ScopeMonitorWrite) {
		t.Error("expected ScopeMonitorWrite to be valid")
	}
}

func TestIsValidScope_Invalid(t *testing.T) {
	if IsValidScope("invalid-scope") {
		t.Error("expected invalid scope to be rejected")
	}
	if IsValidScope("") {
		t.Error("expected empty scope to be rejected")
	}
	if IsValidScope("read/write") {
		t.Error("expected malformed scope to be rejected")
	}
	if IsValidScope("admin:extra") {
		t.Error("expected unknown scope to be rejected")
	}
}

func TestValidateScopes(t *testing.T) {
	scopes := []string{ScopeRead, "invalid", ScopeWrite, "", ScopeAdmin}
	validated := ValidateScopes(scopes)
	if len(validated) != 3 {
		t.Errorf("expected 3 valid scopes, got %d", len(validated))
	}
	for _, s := range validated {
		if !IsValidScope(s) {
			t.Errorf("found invalid scope in result: %s", s)
		}
	}
}

func TestValidateScopes_AllValid(t *testing.T) {
	scopes := []string{ScopeRead, ScopeWrite, ScopeAdmin}
	validated := ValidateScopes(scopes)
	if len(validated) != 3 {
		t.Errorf("expected 3 valid scopes, got %d", len(validated))
	}
}

func TestValidateScopes_AllInvalid(t *testing.T) {
	scopes := []string{"invalid", "another-invalid", ""}
	validated := ValidateScopes(scopes)
	if len(validated) != 0 {
		t.Errorf("expected 0 valid scopes, got %d", len(validated))
	}
}

func TestValidateScopes_Empty(t *testing.T) {
	validated := ValidateScopes([]string{})
	if len(validated) != 0 {
		t.Errorf("expected 0 valid scopes for empty input, got %d", len(validated))
	}
}

func TestValidateScopes_Nil(t *testing.T) {
	validated := ValidateScopes(nil)
	if len(validated) != 0 {
		t.Errorf("expected 0 valid scopes for nil input, got %d", len(validated))
	}
}

func TestRequireScope_AdminBypass(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(APIKeyScopesKey), []string{ScopeAdmin})
		c.Next()
	})
	r.Use(RequireScope(ScopeRead))
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK with admin scope, got %d", w.Code)
	}
}

func TestRequireScope_HasRequiredScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(APIKeyScopesKey), []string{ScopeRead, ScopeWrite})
		c.Next()
	})
	r.Use(RequireScope(ScopeRead))
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK with matching scope, got %d", w.Code)
	}
}

func TestRequireScope_NoRequiredScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(APIKeyScopesKey), []string{ScopeRead})
		c.Next()
	})
	r.Use(RequireScope(ScopeWrite))
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden without required scope, got %d", w.Code)
	}
}

func TestRequireScope_NoScopesSet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequireScope(ScopeRead))
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK when no scopes set (JWT auth path), got %d", w.Code)
	}
}

func TestRequireScope_OAuth2Scopes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(OAuth2ScopesKey), []string{ScopeDeploy})
		c.Next()
	})
	r.Use(RequireScope(ScopeDeploy))
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK with OAuth2 scopes, got %d", w.Code)
	}
}

func TestRequireScope_EmptyScopes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(APIKeyScopesKey), []string{})
		c.Next()
	})
	r.Use(RequireScope(ScopeRead))
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK with empty scopes (JWT auth path), got %d", w.Code)
	}
}

func TestRequireScope_MultipleRequired(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(APIKeyScopesKey), []string{ScopeRead})
		c.Next()
	})
	r.Use(RequireScope(ScopeWrite, ScopeDelete))
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden without any required scope, got %d", w.Code)
	}
}

func TestRequireScope_MultipleRequiredWithMatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(APIKeyScopesKey), []string{ScopeRead, ScopeWrite})
		c.Next()
	})
	r.Use(RequireScope(ScopeWrite, ScopeDelete))
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK with matching scope in multiple required, got %d", w.Code)
	}
}

func TestScopeDescriptions(t *testing.T) {
	for _, scope := range AllScopes {
		if desc, ok := ScopeDescriptions[scope]; !ok || desc == "" {
			t.Errorf("expected description for scope %s", scope)
		}
	}
}

func TestAllScopesSet(t *testing.T) {
	for _, scope := range AllScopes {
		if !allScopesSet[scope] {
			t.Errorf("expected scope %s to be in allScopesSet", scope)
		}
	}
}

func TestAllScopesContainsAdmin(t *testing.T) {
	found := false
	for _, scope := range AllScopes {
		if scope == ScopeAdmin {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected ScopeAdmin to be in AllScopes")
	}
}