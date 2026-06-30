package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupCSRFTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRF())
	r.GET("/test", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	r.POST("/test", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	r.PUT("/test", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	r.DELETE("/test", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	r.HEAD("/test", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	r.OPTIONS("/test", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	return r
}

func TestCSRF_GetRequest_SetsCookie(t *testing.T) {
	r := setupCSRFTestRouter()

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET request should succeed, got status %d", w.Code)
	}

	cookie := getCSRFCookie(w)
	if cookie == "" {
		t.Error("GET request should set CSRF cookie")
	}
	if len(cookie) != 64 {
		t.Errorf("CSRF token should be 64 hex chars (32 bytes), got length %d", len(cookie))
	}
}

func TestCSRF_GetRequest_ExistingCookie(t *testing.T) {
	r := setupCSRFTestRouter()

	req := httptest.NewRequest("GET", "/test", nil)
	existingToken := "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2"
	req.AddCookie(&http.Cookie{
		Name:  CSRFTokenCookie,
		Value: existingToken,
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET request should succeed, got status %d", w.Code)
	}
}

func TestCSRF_PostRequest_NoCookie(t *testing.T) {
	r := setupCSRFTestRouter()

	req := httptest.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("POST without CSRF cookie should be forbidden, got status %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "CSRF token missing") {
		t.Errorf("response should mention CSRF token missing, got: %s", w.Body.String())
	}
}

func TestCSRF_PostRequest_NoHeader(t *testing.T) {
	r := setupCSRFTestRouter()

	token := "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2"
	req := httptest.NewRequest("POST", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  CSRFTokenCookie,
		Value: token,
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("POST without CSRF header should be forbidden, got status %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "CSRF token mismatch") {
		t.Errorf("response should mention CSRF token mismatch, got: %s", w.Body.String())
	}
}

func TestCSRF_PostRequest_ValidHeader(t *testing.T) {
	r := setupCSRFTestRouter()

	token := "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2"
	req := httptest.NewRequest("POST", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  CSRFTokenCookie,
		Value: token,
	})
	req.Header.Set(CSRFTokenHeader, token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("POST with valid CSRF token should succeed, got status %d", w.Code)
	}
}

func TestCSRF_PostRequest_ValidHeader_CaseInsensitive(t *testing.T) {
	r := setupCSRFTestRouter()

	cookieToken := "A1B2C3D4E5F6A7B8C9D0E1F2A3B4C5D6E7F8A9B0C1D2E3F4A5B6C7D8E9F0A1B2"
	headerToken := "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2"
	req := httptest.NewRequest("POST", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  CSRFTokenCookie,
		Value: cookieToken,
	})
	req.Header.Set(CSRFTokenHeader, headerToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("CSRF token comparison should be case-insensitive, got status %d", w.Code)
	}
}

func TestCSRF_PostRequest_MismatchedToken(t *testing.T) {
	r := setupCSRFTestRouter()

	req := httptest.NewRequest("POST", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  CSRFTokenCookie,
		Value: "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2",
	})
	req.Header.Set(CSRFTokenHeader, "differenttokenvaluehere00000000000000000000000000000000000000")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("POST with mismatched CSRF token should be forbidden, got status %d", w.Code)
	}
}

func TestCSRF_PutRequest_ValidToken(t *testing.T) {
	r := setupCSRFTestRouter()

	token := "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2"
	req := httptest.NewRequest("PUT", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  CSRFTokenCookie,
		Value: token,
	})
	req.Header.Set(CSRFTokenHeader, token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("PUT with valid CSRF token should succeed, got status %d", w.Code)
	}
}

func TestCSRF_DeleteRequest_ValidToken(t *testing.T) {
	r := setupCSRFTestRouter()

	token := "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2"
	req := httptest.NewRequest("DELETE", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  CSRFTokenCookie,
		Value: token,
	})
	req.Header.Set(CSRFTokenHeader, token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("DELETE with valid CSRF token should succeed, got status %d", w.Code)
	}
}

func TestCSRF_HeadRequest_NoTokenNeeded(t *testing.T) {
	r := setupCSRFTestRouter()

	req := httptest.NewRequest("HEAD", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HEAD request should succeed without CSRF token, got status %d", w.Code)
	}
}

func TestCSRF_OptionsRequest_NoTokenNeeded(t *testing.T) {
	r := setupCSRFTestRouter()

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("OPTIONS request should succeed without CSRF token, got status %d", w.Code)
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
	r.POST("/test", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("POST with csrf_skip should succeed, got status %d", w.Code)
	}
}

func TestCSRF_FormFieldToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRF())
	r.POST("/test", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	token := "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2"
	body := strings.NewReader("csrf_token=" + token)
	req := httptest.NewRequest("POST", "/test", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  CSRFTokenCookie,
		Value: token,
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("POST with form field CSRF token should succeed, got status %d", w.Code)
	}
}

func TestGenerateCSRFToken(t *testing.T) {
	token := generateCSRFToken()
	if len(token) != 64 {
		t.Errorf("CSRF token should be 64 hex chars (32 bytes), got length %d", len(token))
	}

	token2 := generateCSRFToken()
	if token == token2 {
		t.Error("two generated CSRF tokens should be different")
	}
}

func getCSRFCookie(w *httptest.ResponseRecorder) string {
	for _, c := range w.Result().Cookies() {
		if c.Name == CSRFTokenCookie {
			return c.Value
		}
	}
	return ""
}
