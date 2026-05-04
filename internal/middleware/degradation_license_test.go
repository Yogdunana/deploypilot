package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// mockDegradationChecker implements DegradationChecker for testing.
type mockDegradationChecker struct {
	readonly bool
}

func (m *mockDegradationChecker) CheckReadOnly(ctx interface{}) error {
	if m.readonly {
		return http.ErrAbortHandler // simulate read-only error
	}
	return nil
}

func TestReadOnlyMiddleware_NotReadOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ReadOnlyMiddleware(&mockDegradationChecker{readonly: false}))
	r.POST("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("POST", "/api/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestReadOnlyMiddleware_IsReadOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ReadOnlyMiddleware(&mockDegradationChecker{readonly: true}))
	r.POST("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("POST", "/api/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestReadOnlyMiddleware_GETAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ReadOnlyMiddleware(&mockDegradationChecker{readonly: true}))
	r.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET should be allowed in read-only mode, got %d", w.Code)
	}
}

func TestReadOnlyMiddleware_PUTBlocked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ReadOnlyMiddleware(&mockDegradationChecker{readonly: true}))
	r.PUT("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("PUT", "/api/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("PUT should be blocked in read-only mode, got %d", w.Code)
	}
}

func TestReadOnlyMiddleware_DELETEBlocked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ReadOnlyMiddleware(&mockDegradationChecker{readonly: true}))
	r.DELETE("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("DELETE", "/api/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("DELETE should be blocked in read-only mode, got %d", w.Code)
	}
}

// mockLicenseValidator implements the license validation interface.
type mockLicenseValidator struct {
	valid bool
}

func (m *mockLicenseValidator) Validate() error {
	if m.valid {
		return nil
	}
	return context.DeadlineExceeded // any error
}

func TestLicenseCheckMiddleware_Valid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(LicenseCheckMiddleware(&mockLicenseValidator{valid: true}))
	r.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestLicenseCheckMiddleware_Invalid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(LicenseCheckMiddleware(&mockLicenseValidator{valid: false}))
	r.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}
