package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCSRF_SafeMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRF())
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for GET request, got %d", w.Code)
	}

	cookie := w.Header().Get("Set-Cookie")
	if cookie == "" {
		t.Error("expected Set-Cookie header with CSRF token")
	}
}

func TestCSRF_HEAD(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRF())
	r.HEAD("/", func(c *gin.Context) {
		c.String(http.StatusOK, "")
	})

	req := httptest.NewRequest(http.MethodHead, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for HEAD request, got %d", w.Code)
	}
}

func TestCSRF_OPTIONS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRF())
	r.OPTIONS("/", func(c *gin.Context) {
		c.String(http.StatusOK, "")
	})

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for OPTIONS request, got %d", w.Code)
	}
}

func TestCSRF_PostWithoutToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRF())
	r.POST("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for POST without CSRF token, got %d", w.Code)
	}
}

func TestCSRF_PostWithToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(CSRF())
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	r.POST("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	cookie := w.Header().Get("Set-Cookie")
	if cookie == "" {
		t.Fatal("expected Set-Cookie header")
	}

	token := extractTokenFromCookie(cookie)
	if token == "" {
		t.Fatal("could not extract CSRF token from cookie")
	}

	req = httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Cookie", cookie)
	req.Header.Set(CSRFTokenHeader, token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for POST with valid CSRF token, got %d", w.Code)
	}
}

func TestCSRF_PostWithFormToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(CSRF())
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	r.POST("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	cookie := w.Header().Get("Set-Cookie")
	if cookie == "" {
		t.Fatal("expected Set-Cookie header")
	}

	token := extractTokenFromCookie(cookie)
	if token == "" {
		t.Fatal("could not extract CSRF token from cookie")
	}

	formData := url.Values{}
	formData.Set("csrf_token", token)
	req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(formData.Encode()))
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.PostForm = map[string][]string{"csrf_token": {token}}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for POST with form CSRF token, got %d", w.Code)
	}
}

func TestCSRF_TokenMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(CSRF())
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	r.POST("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	cookie := w.Header().Get("Set-Cookie")
	if cookie == "" {
		t.Fatal("expected Set-Cookie header")
	}

	req = httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Cookie", cookie)
	req.Header.Set(CSRFTokenHeader, "wrong-token-12345")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for POST with wrong CSRF token, got %d", w.Code)
	}
}

func TestCSRF_Skip(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("csrf_skip", true)
		c.Next()
	})
	r.Use(CSRF())
	r.POST("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK when CSRF is skipped, got %d", w.Code)
	}
}

func TestCSRF_PutWithoutToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRF())
	r.PUT("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPut, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for PUT without CSRF token, got %d", w.Code)
	}
}

func TestCSRF_DeleteWithoutToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRF())
	r.DELETE("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for DELETE without CSRF token, got %d", w.Code)
	}
}

func TestCSRF_GenerateToken(t *testing.T) {
	token := generateCSRFToken()
	if len(token) != csrfTokenLength*2 {
		t.Errorf("expected token length %d, got %d", csrfTokenLength*2, len(token))
	}
}

func TestCSRF_GenerateToken_Unique(t *testing.T) {
	token1 := generateCSRFToken()
	token2 := generateCSRFToken()
	if token1 == token2 {
		t.Error("expected unique CSRF tokens")
	}
}

func extractTokenFromCookie(cookie string) string {
	prefix := CSRFTokenCookie + "="
	start := len(prefix)
	for i, ch := range cookie {
		if ch == ';' {
			return cookie[start:i]
		}
	}
	return cookie[start:]
}