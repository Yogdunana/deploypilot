package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupIPWhitelistMiddleware(t *testing.T) (*gin.Engine, *service.IPWhitelistService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&model.IPWhitelist{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	svc := service.NewIPWhitelistService(db)

	r := gin.New()
	r.Use(UserIPWhitelistMiddleware(svc))
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	return r, svc
}

func setUser(c *gin.Context, userID string) {
	c.Set(string(auth.UserIDKey), userID)
}

func TestUserIPWhitelist_NoUser_AllowsThrough(t *testing.T) {
	r, _ := setupIPWhitelistMiddleware(t)

	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for unauthenticated request, got %d", w.Code)
	}
	if w.Body.String() != "pong" {
		t.Errorf("expected body=pong, got %q", w.Body.String())
	}
}

func TestUserIPWhitelist_NoEntries_AllowsThrough(t *testing.T) {
	// Pre-middleware sets user context
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err := db.AutoMigrate(&model.IPWhitelist{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	svc := service.NewIPWhitelistService(db)
	r2 := gin.New()
	r2.Use(func(c *gin.Context) {
		setUser(c, "user-empty")
		c.Next()
	})
	r2.Use(UserIPWhitelistMiddleware(svc))
	r2.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	req := httptest.NewRequest("GET", "/ping", nil)
	req.RemoteAddr = "203.0.113.1:1234"
	w := httptest.NewRecorder()
	r2.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for user with no entries, got %d", w.Code)
	}
}

func TestUserIPWhitelist_AllowedIP(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err := db.AutoMigrate(&model.IPWhitelist{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	svc := service.NewIPWhitelistService(db)
	if _, err := svc.Create("user-1", "office", "10.0.0.0/24", "tenant-1", "admin"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		setUser(c, "user-1")
		c.Next()
	})
	r.Use(UserIPWhitelistMiddleware(svc))
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	req := httptest.NewRequest("GET", "/ping", nil)
	req.RemoteAddr = "10.0.0.5:54321"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for whitelisted IP, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestUserIPWhitelist_BlockedIP(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err := db.AutoMigrate(&model.IPWhitelist{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	svc := service.NewIPWhitelistService(db)
	if _, err := svc.Create("user-1", "office", "10.0.0.0/24", "tenant-1", "admin"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		setUser(c, "user-1")
		c.Next()
	})
	r.Use(UserIPWhitelistMiddleware(svc))
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	req := httptest.NewRequest("GET", "/ping", nil)
	req.RemoteAddr = "203.0.113.99:54321"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for non-whitelisted IP, got %d", w.Code)
	}
	if w.Body.String() == "" {
		t.Error("expected non-empty error body for blocked request")
	}
}

func TestUserIPWhitelist_EmptyUserID_AllowsThrough(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err := db.AutoMigrate(&model.IPWhitelist{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	svc := service.NewIPWhitelistService(db)
	if _, err := svc.Create("user-1", "office", "10.0.0.0/24", "tenant-1", "admin"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	r2 := gin.New()
	r2.Use(func(c *gin.Context) {
		setUser(c, "")
		c.Next()
	})
	r2.Use(UserIPWhitelistMiddleware(svc))
	r2.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	req := httptest.NewRequest("GET", "/ping", nil)
	req.RemoteAddr = "203.0.113.99:54321"
	w := httptest.NewRecorder()
	r2.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for empty user_id, got %d", w.Code)
	}
}

func TestUserIPWhitelist_DifferentUser_SameIP(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err := db.AutoMigrate(&model.IPWhitelist{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	svc := service.NewIPWhitelistService(db)
	// user-A has whitelist, user-B does not
	if _, err := svc.Create("user-A", "office", "10.0.0.0/24", "tenant-1", "admin"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		// simulate different user per request via header trick
		if uid := c.GetHeader("X-User"); uid != "" {
			setUser(c, uid)
		}
		c.Next()
	})
	r.Use(UserIPWhitelistMiddleware(svc))
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	// user-B (no entries) should be allowed even though the IP is "wrong" for user-A
	req := httptest.NewRequest("GET", "/ping", nil)
	req.RemoteAddr = "203.0.113.99:54321"
	req.Header.Set("X-User", "user-B")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for user-B with no whitelist, got %d", w.Code)
	}
}
