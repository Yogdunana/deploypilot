package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newCSRFServer(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRF())
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	r.POST("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	return r
}

// extractCSRFToken pulls the CSRF cookie value out of the response so we
// can echo it back in a state-changing request.
func extractCSRFToken(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == CSRFTokenCookie {
			return cookie.Value
		}
	}
	t.Fatalf("no CSRF cookie set on response")
	return ""
}

func TestCSRF_GetRequestSetsCookie(t *testing.T) {
	r := newCSRFServer(t)
	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	token := extractCSRFToken(t, w)
	if len(token) < 32 {
		t.Errorf("expected CSRF cookie of at least 32 chars, got %d", len(token))
	}
}

func TestCSRF_GetRequestReusesExistingCookie(t *testing.T) {
	r := newCSRFServer(t)

	// First request sets the cookie.
	req1 := httptest.NewRequest("GET", "/ping", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	token1 := extractCSRFToken(t, w1)

	// Second request carries that cookie — server should accept it and
	// pass through without rejecting the request.
	req2 := httptest.NewRequest("GET", "/ping", nil)
	req2.AddCookie(&http.Cookie{Name: CSRFTokenCookie, Value: token1})
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("expected 200 for safe method with existing cookie, got %d", w2.Code)
	}

	// The original token can still be used in a state-changing request
	// to prove it was accepted (and therefore the cookie was honored).
	postReq := httptest.NewRequest("POST", "/ping", nil)
	postReq.AddCookie(&http.Cookie{Name: CSRFTokenCookie, Value: token1})
	postReq.Header.Set(CSRFTokenHeader, token1)
	postW := httptest.NewRecorder()
	r.ServeHTTP(postW, postReq)

	if postW.Code != http.StatusOK {
		t.Errorf("expected 200 for POST with original CSRF token, got %d", postW.Code)
	}
}

func TestCSRF_PostWithoutCookie_Rejected(t *testing.T) {
	r := newCSRFServer(t)
	req := httptest.NewRequest("POST", "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for POST without CSRF cookie, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "CSRF token missing") {
		t.Errorf("expected 'CSRF token missing' message, got %s", w.Body.String())
	}
}

func TestCSRF_PostWithMismatchedHeader_Rejected(t *testing.T) {
	r := newCSRFServer(t)
	// Establish a cookie.
	preReq := httptest.NewRequest("GET", "/ping", nil)
	preW := httptest.NewRecorder()
	r.ServeHTTP(preW, preReq)
	cookie := extractCSRFToken(t, preW)

	// POST with a different token in the header.
	postReq := httptest.NewRequest("POST", "/ping", nil)
	postReq.AddCookie(&http.Cookie{Name: CSRFTokenCookie, Value: cookie})
	postReq.Header.Set(CSRFTokenHeader, "not-the-same-token")
	postW := httptest.NewRecorder()
	r.ServeHTTP(postW, postReq)

	if postW.Code != http.StatusForbidden {
		t.Errorf("expected 403 for mismatched CSRF token, got %d", postW.Code)
	}
	if !strings.Contains(postW.Body.String(), "mismatch") {
		t.Errorf("expected 'mismatch' message, got %s", postW.Body.String())
	}
}

func TestCSRF_PostWithMatchingHeader_Accepted(t *testing.T) {
	r := newCSRFServer(t)
	preReq := httptest.NewRequest("GET", "/ping", nil)
	preW := httptest.NewRecorder()
	r.ServeHTTP(preW, preReq)
	cookie := extractCSRFToken(t, preW)

	postReq := httptest.NewRequest("POST", "/ping", nil)
	postReq.AddCookie(&http.Cookie{Name: CSRFTokenCookie, Value: cookie})
	postReq.Header.Set(CSRFTokenHeader, cookie)
	postW := httptest.NewRecorder()
	r.ServeHTTP(postW, postReq)

	if postW.Code != http.StatusOK {
		t.Errorf("expected 200 for matching CSRF token, got %d", postW.Code)
	}
}

func TestCSRF_PostWithMatchingFormField_Accepted(t *testing.T) {
	r := newCSRFServer(t)
	preReq := httptest.NewRequest("GET", "/ping", nil)
	preW := httptest.NewRecorder()
	r.ServeHTTP(preW, preReq)
	cookie := extractCSRFToken(t, preW)

	// HTML form submission: token comes in csrf_token form field.
	form := strings.NewReader("csrf_token=" + cookie)
	postReq := httptest.NewRequest("POST", "/ping", form)
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.AddCookie(&http.Cookie{Name: CSRFTokenCookie, Value: cookie})
	postW := httptest.NewRecorder()
	r.ServeHTTP(postW, postReq)

	if postW.Code != http.StatusOK {
		t.Errorf("expected 200 for form-based CSRF token, got %d", postW.Code)
	}
}

func TestCSRF_HeaderTakesPrecedenceOverFormField(t *testing.T) {
	// If both header and form are present but the form matches and the
	// header doesn't, the request should be rejected — the header is checked
	// first, so a non-empty header is always consulted.
	r := newCSRFServer(t)
	preReq := httptest.NewRequest("GET", "/ping", nil)
	preW := httptest.NewRecorder()
	r.ServeHTTP(preW, preReq)
	cookie := extractCSRFToken(t, preW)

	form := strings.NewReader("csrf_token=" + cookie)
	postReq := httptest.NewRequest("POST", "/ping", form)
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.AddCookie(&http.Cookie{Name: CSRFTokenCookie, Value: cookie})
	postReq.Header.Set(CSRFTokenHeader, "wrong-token")
	postW := httptest.NewRecorder()
	r.ServeHTTP(postW, postReq)

	if postW.Code != http.StatusForbidden {
		t.Errorf("expected 403 when header token is wrong, got %d", postW.Code)
	}
}

func TestCSRF_OptionsIsSafe(t *testing.T) {
	r := newCSRFServer(t)
	// OPTIONS is a safe method — no CSRF enforcement.
	req := httptest.NewRequest("OPTIONS", "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code == http.StatusForbidden {
		t.Error("OPTIONS should not be CSRF-enforced")
	}
}

func TestCSRF_HeadIsSafe(t *testing.T) {
	r := newCSRFServer(t)
	req := httptest.NewRequest("HEAD", "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code == http.StatusForbidden {
		t.Error("HEAD should not be CSRF-enforced")
	}
}

func TestCSRF_SkipFlag_BypassesCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("csrf_skip", true)
		c.Next()
	})
	r.Use(CSRF())
	r.POST("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	req := httptest.NewRequest("POST", "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when csrf_skip is set, got %d", w.Code)
	}
}

func TestCSRF_CaseInsensitiveHeaderMatch(t *testing.T) {
	r := newCSRFServer(t)
	preReq := httptest.NewRequest("GET", "/ping", nil)
	preW := httptest.NewRecorder()
	r.ServeHTTP(preW, preReq)
	cookie := extractCSRFToken(t, preW)

	// strings.EqualFold is used in the middleware, so casing differences
	// between the cookie and the header value should still match.
	postReq := httptest.NewRequest("POST", "/ping", nil)
	postReq.AddCookie(&http.Cookie{Name: CSRFTokenCookie, Value: cookie})
	postReq.Header.Set(CSRFTokenHeader, strings.ToUpper(cookie))
	postW := httptest.NewRecorder()
	r.ServeHTTP(postW, postReq)

	if postW.Code != http.StatusOK {
		t.Errorf("expected 200 for case-insensitive matching token, got %d", postW.Code)
	}
}

func TestCSRF_EmptyHeaderAndNoFormField_Rejected(t *testing.T) {
	r := newCSRFServer(t)
	preReq := httptest.NewRequest("GET", "/ping", nil)
	preW := httptest.NewRecorder()
	r.ServeHTTP(preW, preReq)
	cookie := extractCSRFToken(t, preW)

	postReq := httptest.NewRequest("POST", "/ping", nil)
	postReq.AddCookie(&http.Cookie{Name: CSRFTokenCookie, Value: cookie})
	// No header, no form field with csrf_token.
	postW := httptest.NewRecorder()
	r.ServeHTTP(postW, postReq)

	if postW.Code != http.StatusForbidden {
		t.Errorf("expected 403 when no token is provided, got %d", postW.Code)
	}
}
