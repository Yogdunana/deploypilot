package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func setupGinTest() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	r := setupGinTest()
	token, err := GenerateToken("user-123", "admin")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	handlerCalled := false
	r.Use(AuthMiddleware())
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		userID, exists := c.Get(string(UserIDKey))
		if !exists {
			t.Error("expected userID in context")
		}
		if userID != "user-123" {
			t.Errorf("expected userID=user-123, got %v", userID)
		}
		role, exists := c.Get(string(RoleKey))
		if !exists {
			t.Error("expected role in context")
		}
		if role != "admin" {
			t.Errorf("expected role=admin, got %v", role)
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("expected handler to be called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// getExpiredRegisteredClaims returns RegisteredClaims with an expiry in the past.
func getExpiredRegisteredClaims() jwt.RegisteredClaims {
	return jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
	}
}

// createTokenFromClaims signs and returns a token string from Claims.
func createTokenFromClaims(claims *Claims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString(getJWTSecret())
	if err != nil {
		panic("failed to sign test token: " + err.Error())
	}
	return s
}

func TestAuthMiddleware_NoToken(t *testing.T) {
	r := setupGinTest()

	handlerCalled := false
	r.Use(AuthMiddleware())
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if handlerCalled {
		t.Error("expected handler NOT to be called")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	r := setupGinTest()

	handlerCalled := false
	r.Use(AuthMiddleware())
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer this-is-not-a-valid-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if handlerCalled {
		t.Error("expected handler NOT to be called")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	r := setupGinTest()

	// Create an expired token
	expiredClaims := &Claims{
		UserID: "user-expired",
		Role:   "viewer",
		RegisteredClaims: getExpiredRegisteredClaims(),
	}
	token := createTokenFromClaims(expiredClaims)

	handlerCalled := false
	r.Use(AuthMiddleware())
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if handlerCalled {
		t.Error("expected handler NOT to be called for expired token")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{"no bearer prefix", "Token sometoken"},
		{"only bearer", "Bearer"},
		{"empty bearer", "Bearer "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false
			r2 := setupGinTest()
			r2.Use(AuthMiddleware())
			r2.GET("/test", func(c *gin.Context) {
				handlerCalled = true
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", tt.header)
			w := httptest.NewRecorder()
			r2.ServeHTTP(w, req)

			if handlerCalled {
				t.Error("expected handler NOT to be called")
			}
			if w.Code != http.StatusUnauthorized {
				t.Errorf("expected 401, got %d", w.Code)
			}
		})
	}
}

func TestRoleRequired_OwnerCanAccessAdmin(t *testing.T) {
	r := setupGinTest()

	r.Use(AuthMiddleware())
	r.Use(RoleRequired("admin"))
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	token, _ := GenerateToken("owner-user", "owner")
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for owner accessing admin endpoint, got %d", w.Code)
	}
}

func TestRoleRequired_AdminCannotAccessOwner(t *testing.T) {
	r := setupGinTest()

	r.Use(AuthMiddleware())
	r.Use(RoleRequired("owner"))
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	token, _ := GenerateToken("admin-user", "admin")
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for admin accessing owner endpoint, got %d", w.Code)
	}
}

func TestRoleRequired_ViewerCannotAccessDev(t *testing.T) {
	r := setupGinTest()

	r.Use(AuthMiddleware())
	r.Use(RoleRequired("dev"))
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	token, _ := GenerateToken("viewer-user", "viewer")
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for viewer accessing dev endpoint, got %d", w.Code)
	}
}

func TestRoleRequired_NoAuth(t *testing.T) {
	r := setupGinTest()

	r.Use(RoleRequired("admin"))
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when no auth, got %d", w.Code)
	}
}

func TestRoleRequired_ViewerCanAccessViewer(t *testing.T) {
	r := setupGinTest()

	r.Use(AuthMiddleware())
	r.Use(RoleRequired("viewer"))
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	token, _ := GenerateToken("viewer-user", "viewer")
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for viewer accessing viewer endpoint, got %d", w.Code)
	}
}

func TestOptionalAuth_ValidToken(t *testing.T) {
	r := setupGinTest()

	handlerCalled := false
	r.Use(OptionalAuth())
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		userID, exists := c.Get(string(UserIDKey))
		if !exists {
			t.Error("expected userID in context")
		}
		if userID != "user-opt" {
			t.Errorf("expected userID=user-opt, got %v", userID)
		}
		c.Status(http.StatusOK)
	})

	token, _ := GenerateToken("user-opt", "dev")
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("expected handler to be called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestOptionalAuth_NoToken(t *testing.T) {
	r := setupGinTest()

	handlerCalled := false
	r.Use(OptionalAuth())
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		// Should NOT have userID in context
		_, exists := c.Get(string(UserIDKey))
		if exists {
			t.Error("expected no userID in context when no token")
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("expected handler to be called even without token")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestOptionalAuth_InvalidToken(t *testing.T) {
	r := setupGinTest()

	handlerCalled := false
	r.Use(OptionalAuth())
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		_, exists := c.Get(string(UserIDKey))
		if exists {
			t.Error("expected no userID in context for invalid token")
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("expected handler to be called even with invalid token")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestOptionalAuth_InvalidFormat(t *testing.T) {
	r := setupGinTest()

	handlerCalled := false
	r.Use(OptionalAuth())
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("expected handler to be called even with invalid format")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRoleRequired_InvalidRoleType(t *testing.T) {
	r := setupGinTest()

	r.Use(func(c *gin.Context) {
		c.Set(string(RoleKey), 12345) // non-string role
		c.Next()
	})
	r.Use(RoleRequired("admin"))
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for invalid role type, got %d", w.Code)
	}
}

func TestRoleRequired_UnknownRole(t *testing.T) {
	r := setupGinTest()

	// Create a token with a role that doesn't exist in the hierarchy
	unknownClaims := &Claims{
		UserID: "user-unknown",
		Role:   "superadmin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := createTokenFromClaims(unknownClaims)

	r.Use(AuthMiddleware())
	r.Use(RoleRequired("admin"))
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for unknown role, got %d", w.Code)
	}
}

func TestRoleRequired_EmptyRoles(t *testing.T) {
	r := setupGinTest()

	token, _ := GenerateToken("viewer-user", "viewer")
	r.Use(AuthMiddleware())
	r.Use(RoleRequired()) // no roles required
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// With no roles specified, none should match -> 403
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for empty roles list, got %d", w.Code)
	}
}
