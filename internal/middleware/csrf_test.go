package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCSRF_SafeMethodsGetCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRF())
	r.GET("/test", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for GET, got %d", w.Code)
	}
	// Should set a CSRF cookie
	cookie := ""
	for _, c := range w.Result().Cookies() {
		if c.Name == CSRFTokenCookie {
			cookie = c.Value
		}
	}
	if cookie == "" {
		t.Error("expected CSRF cookie to be set for GET request")
	}
}

func TestCSRF_SafeMethodsHead(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRF())
	r.HEAD("/test", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("HEAD", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for HEAD, got %d", w.Code)
	}
}

func TestCSRF_SafeMethodsOptions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRF())
	r.OPTIONS("/test", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for OPTIONS, got %d", w.Code)
	}
}

func TestCSRF_PostWithoutToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRF())
	r.POST("/test", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for POST without CSRF cookie, got %d", w.Code)
	}
}

func TestCSRF_PostWithValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRF())
	r.POST("/submit", func(c *gin.Context) { c.String(200, "ok") })

	// Use a known token for both cookie and header
	token := "test-csrf-token-for-validation"
	req := httptest.NewRequest("POST", "/submit", nil)
	req.Header.Set(CSRFTokenHeader, token)
	req.AddCookie(&http.Cookie{Name: CSRFTokenCookie, Value: token})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for POST with valid CSRF token, got %d", w.Code)
	}
}

func TestCSRF_PostWithMismatchedToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRF())
	r.POST("/test", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set(CSRFTokenHeader, "wrong-token")
	req.AddCookie(&http.Cookie{Name: CSRFTokenCookie, Value: "correct-token"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for mismatched CSRF token, got %d", w.Code)
	}
}

func TestCSRF_PostWithMissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRF())
	r.POST("/test", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("POST", "/test", nil)
	req.AddCookie(&http.Cookie{Name: CSRFTokenCookie, Value: "some-token"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for POST without CSRF header, got %d", w.Code)
	}
}

func TestCSRF_SkipFlag(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("csrf_skip", true)
		c.Next()
	})
	r.Use(CSRF())
	r.POST("/test", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with csrf_skip flag, got %d", w.Code)
	}
}

func TestCSRF_PutWithValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRF())
	r.PUT("/test", func(c *gin.Context) { c.String(200, "ok") })

	token := "test-csrf-token-12345"
	req := httptest.NewRequest("PUT", "/test", nil)
	req.Header.Set(CSRFTokenHeader, token)
	req.AddCookie(&http.Cookie{Name: CSRFTokenCookie, Value: token})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for PUT with valid token, got %d", w.Code)
	}
}

func TestCSRF_DeleteWithValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRF())
	r.DELETE("/test", func(c *gin.Context) { c.String(200, "ok") })

	token := "test-csrf-token-12345"
	req := httptest.NewRequest("DELETE", "/test", nil)
	req.Header.Set(CSRFTokenHeader, token)
	req.AddCookie(&http.Cookie{Name: CSRFTokenCookie, Value: token})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for DELETE with valid token, got %d", w.Code)
	}
}

func TestCSRF_CookieSameSiteStrict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRF())
	r.GET("/test", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	for _, c := range w.Result().Cookies() {
		if c.Name == CSRFTokenCookie {
			if c.SameSite != http.SameSiteStrictMode {
				t.Errorf("expected SameSite=Strict, got %v", c.SameSite)
			}
			// HttpOnly is false by design (JS needs to read the token for the header)
			if c.HttpOnly {
				t.Error("CSRF token cookie should NOT be HttpOnly since JS needs to read it")
			}
			if !c.Secure {
				t.Error("expected Secure cookie")
			}
		}
	}
}
