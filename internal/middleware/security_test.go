package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeaders_Default(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("expected X-Content-Type-Options: nosniff, got %s", w.Header().Get("X-Content-Type-Options"))
	}
	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Errorf("expected X-Frame-Options: DENY, got %s", w.Header().Get("X-Frame-Options"))
	}
	if w.Header().Get("X-XSS-Protection") != "1; mode=block" {
		t.Errorf("expected X-XSS-Protection: 1; mode=block, got %s", w.Header().Get("X-XSS-Protection"))
	}
	if w.Header().Get("Referrer-Policy") != "strict-origin-when-cross-origin" {
		t.Errorf("expected Referrer-Policy: strict-origin-when-cross-origin, got %s", w.Header().Get("Referrer-Policy"))
	}
	if w.Header().Get("Content-Security-Policy") == "" {
		t.Error("expected Content-Security-Policy header")
	}
	if w.Header().Get("Strict-Transport-Security") == "" {
		t.Error("expected Strict-Transport-Security header")
	}
}

func TestSecurityHeaders_CSPDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	csp := w.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Error("expected Content-Security-Policy header")
	}
	if len(csp) == 0 {
		t.Error("expected non-empty Content-Security-Policy")
	}
}

func TestSecurityHeaders_CSPOverride(t *testing.T) {
	gin.SetMode(gin.TestMode)

	original := os.Getenv("DEPLOYPILOT_CSP_POLICY")
	os.Setenv("DEPLOYPILOT_CSP_POLICY", "default-src 'self'; script-src 'self' 'unsafe-inline'")
	defer os.Setenv("DEPLOYPILOT_CSP_POLICY", original)

	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	csp := w.Header().Get("Content-Security-Policy")
	expected := "default-src 'self'; script-src 'self' 'unsafe-inline'"
	if csp != expected {
		t.Errorf("expected Content-Security-Policy: %s, got %s", expected, csp)
	}
}

func TestSecurityHeaders_HSTSDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)

	original := os.Getenv("DEPLOYPILOT_HSTS_MAX_AGE")
	os.Setenv("DEPLOYPILOT_HSTS_MAX_AGE", "")
	defer os.Setenv("DEPLOYPILOT_HSTS_MAX_AGE", original)

	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	hsts := w.Header().Get("Strict-Transport-Security")
	if hsts == "" {
		t.Error("expected Strict-Transport-Security header")
	}
	if len(hsts) == 0 {
		t.Error("expected non-empty Strict-Transport-Security")
	}
}

func TestSecurityHeaders_HSTSOverride(t *testing.T) {
	gin.SetMode(gin.TestMode)

	original := os.Getenv("DEPLOYPILOT_HSTS_MAX_AGE")
	os.Setenv("DEPLOYPILOT_HSTS_MAX_AGE", "86400")
	defer os.Setenv("DEPLOYPILOT_HSTS_MAX_AGE", original)

	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	hsts := w.Header().Get("Strict-Transport-Security")
	expected := "max-age=86400; includeSubDomains"
	if hsts != expected {
		t.Errorf("expected Strict-Transport-Security: %s, got %s", expected, hsts)
	}
}

func TestSecurityHeaders_HSTSDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	original := os.Getenv("DEPLOYPILOT_HSTS_MAX_AGE")
	os.Setenv("DEPLOYPILOT_HSTS_MAX_AGE", "0")
	defer os.Setenv("DEPLOYPILOT_HSTS_MAX_AGE", original)

	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	hsts := w.Header().Get("Strict-Transport-Security")
	if hsts != "" {
		t.Errorf("expected no Strict-Transport-Security header when HSTS is disabled, got %s", hsts)
	}
}

func TestSecurityHeaders_HSTSInvalidValue(t *testing.T) {
	gin.SetMode(gin.TestMode)

	original := os.Getenv("DEPLOYPILOT_HSTS_MAX_AGE")
	os.Setenv("DEPLOYPILOT_HSTS_MAX_AGE", "invalid")
	defer os.Setenv("DEPLOYPILOT_HSTS_MAX_AGE", original)

	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	hsts := w.Header().Get("Strict-Transport-Security")
	if hsts != "" {
		t.Errorf("expected no Strict-Transport-Security header for invalid value, got %s", hsts)
	}
}

func TestSecurityHeaders_HSTSNegativeValue(t *testing.T) {
	gin.SetMode(gin.TestMode)

	original := os.Getenv("DEPLOYPILOT_HSTS_MAX_AGE")
	os.Setenv("DEPLOYPILOT_HSTS_MAX_AGE", "-1")
	defer os.Setenv("DEPLOYPILOT_HSTS_MAX_AGE", original)

	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	hsts := w.Header().Get("Strict-Transport-Security")
	if hsts != "" {
		t.Errorf("expected no Strict-Transport-Security header for negative value, got %s", hsts)
	}
}

func TestSecurityHeaders_AllMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	r.POST("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	r.PUT("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	r.DELETE("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		req := httptest.NewRequest(method, "/", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("expected X-Content-Type-Options for %s, got %s", method, w.Header().Get("X-Content-Type-Options"))
		}
	}
}

func TestSecurityHeaders_Preflight(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityHeaders())
	r.OPTIONS("/", func(c *gin.Context) {
		c.String(http.StatusNoContent, "")
	})

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("expected X-Content-Type-Options for OPTIONS, got %s", w.Header().Get("X-Content-Type-Options"))
	}
}

func TestGetCSPPolicy(t *testing.T) {
	original := os.Getenv("DEPLOYPILOT_CSP_POLICY")
	os.Setenv("DEPLOYPILOT_CSP_POLICY", "")
	defer os.Setenv("DEPLOYPILOT_CSP_POLICY", original)

	policy := getCSPPolicy()
	if policy == "" {
		t.Error("expected non-empty CSP policy")
	}
}

func TestGetHSTSMaxAge(t *testing.T) {
	original := os.Getenv("DEPLOYPILOT_HSTS_MAX_AGE")
	os.Setenv("DEPLOYPILOT_HSTS_MAX_AGE", "")
	defer os.Setenv("DEPLOYPILOT_HSTS_MAX_AGE", original)

	maxAge := getHSTSMaxAge()
	if maxAge != 31536000 {
		t.Errorf("expected default HSTS max-age 31536000, got %d", maxAge)
	}
}