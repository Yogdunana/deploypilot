package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	return db
}

func TestRequireResourceAccess_ValidAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	db := setupTestDB(t)

	handlerCalled := false
	r.Use(func(c *gin.Context) {
		c.Set(string(UserIDKey), "user-123")
		c.Set(string(RoleKey), "owner")
		c.Next()
	})
	r.GET("/test/:id", RequireResourceAccess(db, "app", "id"), func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test/app-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("expected handler to be called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRequireResourceAccess_MissingResourceID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	db := setupTestDB(t)

	handlerCalled := false
	r.GET("/test", RequireResourceAccess(db, "app", "id"), func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if handlerCalled {
		t.Error("expected handler NOT to be called")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRequireResourceAccess_NoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	db := setupTestDB(t)

	handlerCalled := false
	r.GET("/test/:id", RequireResourceAccess(db, "app", "id"), func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test/app-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if handlerCalled {
		t.Error("expected handler NOT to be called")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestRequireResourceAccess_InvalidUserIDType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	db := setupTestDB(t)

	handlerCalled := false
	r.Use(func(c *gin.Context) {
		c.Set(string(UserIDKey), 12345)
		c.Set(string(RoleKey), "viewer")
		c.Next()
	})
	r.GET("/test/:id", RequireResourceAccess(db, "app", "id"), func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test/app-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if handlerCalled {
		t.Error("expected handler NOT to be called")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestRequireResourceAccess_InvalidRoleType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	db := setupTestDB(t)

	handlerCalled := false
	r.Use(func(c *gin.Context) {
		c.Set(string(UserIDKey), "user-123")
		c.Set(string(RoleKey), 12345)
		c.Next()
	})
	r.GET("/test/:id", RequireResourceAccess(db, "app", "id"), func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test/app-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if handlerCalled {
		t.Error("expected handler NOT to be called")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}