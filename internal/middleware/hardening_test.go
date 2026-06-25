package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestSecurityEntrance_EmptyEntrance(t *testing.T) {
	// Empty entrance should allow all requests
	r := gin.New()
	r.Use(SecurityEntrance(""))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Empty entrance should allow all: status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestSecurityEntrance_SkipHealthAndAPI(t *testing.T) {
	r := gin.New()
	r.Use(SecurityEntrance("/secret-panel"))
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/api", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/swagger", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	tests := []string{"/health", "/api", "/swagger"}
	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", path, nil)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Path %s should be exempt: status = %d, want %d", path, w.Code, http.StatusOK)
			}
		})
	}
}

func TestSecurityEntrance_SkipAPIPaths(t *testing.T) {
	r := gin.New()
	r.Use(SecurityEntrance("/secret-panel"))
	r.GET("/api/v1/apps", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.POST("/api/v1/deploy", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	tests := []string{"/api/v1/apps", "/api/v1/deploy"}
	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", path, nil)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("API path %s should be exempt: status = %d, want %d", path, w.Code, http.StatusOK)
			}
		})
	}
}

func TestSecurityEntrance_SkipWebSocketPaths(t *testing.T) {
	r := gin.New()
	r.Use(SecurityEntrance("/secret-panel"))
	r.GET("/ws/terminal", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/ws/logs", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	tests := []string{"/ws/terminal", "/ws/logs"}
	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", path, nil)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("WebSocket path %s should be exempt: status = %d, want %d", path, w.Code, http.StatusOK)
			}
		})
	}
}

func TestSecurityEntrance_SkipStaticAssets(t *testing.T) {
	r := gin.New()
	r.Use(SecurityEntrance("/secret-panel"))
	r.GET("/assets/main.js", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/icon.svg", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	tests := []string{"/assets/main.js", "/icon.svg"}
	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", path, nil)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Static asset %s should be exempt: status = %d, want %d", path, w.Code, http.StatusOK)
			}
		})
	}
}

func TestSecurityEntrance_RequireEntrancePrefix(t *testing.T) {
	r := gin.New()
	r.Use(SecurityEntrance("/my-secret-panel"))
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"status": "not found"})
	})

	// Request without entrance prefix should get 404
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Missing entrance prefix should return 404: status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestSecurityEntrance_StripsEntrancePrefix(t *testing.T) {
	r := gin.New()
	r.Use(SecurityEntrance("/secret-panel"))
	r.GET("/", func(c *gin.Context) {
		// Check that the path was stripped
		if c.Request.URL.Path != "/" {
			t.Errorf("Path not stripped correctly: got %q, want %q", c.Request.URL.Path, "/")
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/secret-panel", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Valid entrance should pass: status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestSecurityEntrance_NormalizesEntrance(t *testing.T) {
	tests := []struct {
		entrance string
		path     string
		wantOK   bool
	}{
		{"secret-panel", "/secret-panel/dashboard", true},
		{"/secret-panel", "/secret-panel/dashboard", true},
		{"secret-panel/", "/secret-panel/dashboard", true},
		{"/secret-panel/", "/secret-panel/dashboard", true},
		{"secret-panel", "/dashboard", false},
	}

	for _, tt := range tests {
		t.Run(tt.entrance+"_"+tt.path, func(t *testing.T) {
			r := gin.New()
			r.Use(SecurityEntrance(tt.entrance))
			r.NoRoute(func(c *gin.Context) {
				c.Status(http.StatusNotFound)
			})
			r.GET("/", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})
			r.GET("/dashboard", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", tt.path, nil)
			r.ServeHTTP(w, req)

			if tt.wantOK && w.Code != http.StatusOK {
				t.Errorf("Entrance %q path %q should pass: status = %d", tt.entrance, tt.path, w.Code)
			}
			if !tt.wantOK && w.Code != http.StatusNotFound {
				t.Errorf("Entrance %q path %q should fail: status = %d", tt.entrance, tt.path, w.Code)
			}
		})
	}
}

func TestDomainBinding_EmptyDomains(t *testing.T) {
	// Empty domains should allow all requests
	r := gin.New()
	r.Use(DomainBinding(nil))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Host = "evil.com"
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Empty domains should allow all: status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestDomainBinding_AllowedDomain(t *testing.T) {
	r := gin.New()
	r.Use(DomainBinding([]string{"example.com", "localhost"}))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	tests := []struct {
		host      string
		wantOK    bool
	}{
		{"example.com", true},
		{"localhost", true},
		{"EXAMPLE.COM", true}, // case insensitive
		{"LOCALHOST", true},
		{"evil.com", false},
		{"example.com:8080", true}, // port stripped
		{"localhost:3000", true},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/test", nil)
			req.Host = tt.host
			r.ServeHTTP(w, req)

			if tt.wantOK && w.Code != http.StatusOK {
				t.Errorf("Host %q should be allowed: status = %d", tt.host, w.Code)
			}
			if !tt.wantOK && w.Code != http.StatusForbidden {
				t.Errorf("Host %q should be forbidden: status = %d", tt.host, w.Code)
			}
		})
	}
}

func TestDomainBinding_NormalizesDomains(t *testing.T) {
	r := gin.New()
	r.Use(DomainBinding([]string{"  Example.COM  ", " LocalHost "}))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	tests := []struct {
		host   string
		wantOK bool
	}{
		{"example.com", true},
		{"EXAMPLE.COM", true},
		{"localhost", true},
		{"LOCALHOST", true},
		{"evil.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/test", nil)
			req.Host = tt.host
			r.ServeHTTP(w, req)

			if tt.wantOK && w.Code != http.StatusOK {
				t.Errorf("Host %q should be allowed: status = %d", tt.host, w.Code)
			}
			if !tt.wantOK && w.Code != http.StatusForbidden {
				t.Errorf("Host %q should be forbidden: status = %d", tt.host, w.Code)
			}
		})
	}
}

func TestIPWhitelist_EmptyWhitelist(t *testing.T) {
	// Empty whitelist should allow all IPs
	r := gin.New()
	r.Use(IPWhitelist(nil))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Empty whitelist should allow all: status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestIPWhitelist_AllowedIP(t *testing.T) {
	r := gin.New()
	r.Use(IPWhitelist([]string{"192.168.1.1", "10.0.0.1"}))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	tests := []struct {
		name     string
		clientIP string
		wantOK   bool
	}{
		{"exact match", "192.168.1.1", true},
		{"exact match 2", "10.0.0.1", true},
		{"not allowed", "192.168.1.2", false},
		{"not allowed 2", "8.8.8.8", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/test", nil)
			// Gin's ClientIP() uses X-Forwarded-For or X-Real-IP in test mode
			req.Header.Set("X-Real-IP", tt.clientIP)
			r.ServeHTTP(w, req)

			if tt.wantOK && w.Code != http.StatusOK {
				t.Errorf("IP %q should be allowed: status = %d", tt.clientIP, w.Code)
			}
			if !tt.wantOK && w.Code != http.StatusForbidden {
				t.Errorf("IP %q should be forbidden: status = %d", tt.clientIP, w.Code)
			}
		})
	}
}

func TestIPWhitelist_CIDR(t *testing.T) {
	r := gin.New()
	r.Use(IPWhitelist([]string{"192.168.1.0/24", "10.0.0.0/8"}))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	tests := []struct {
		name     string
		clientIP string
		wantOK   bool
	}{
		{"in CIDR range 1", "192.168.1.50", true},
		{"in CIDR range 2", "192.168.1.255", true},
		{"in CIDR range 3", "10.0.5.100", true},
		{"in CIDR range 4", "10.255.255.255", true},
		{"outside CIDR range 1", "192.168.2.1", false},
		{"outside CIDR range 2", "11.0.0.1", false},
		{"public IP", "8.8.8.8", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Real-IP", tt.clientIP)
			r.ServeHTTP(w, req)

			if tt.wantOK && w.Code != http.StatusOK {
				t.Errorf("IP %q should be allowed: status = %d", tt.clientIP, w.Code)
			}
			if !tt.wantOK && w.Code != http.StatusForbidden {
				t.Errorf("IP %q should be forbidden: status = %d", tt.clientIP, w.Code)
			}
		})
	}
}

func TestIsIPAllowed(t *testing.T) {
	tests := []struct {
		ip       string
		allowed  []string
		want     bool
	}{
		{"192.168.1.1", []string{"192.168.1.1"}, true},
		{"192.168.1.2", []string{"192.168.1.1"}, false},
		{"192.168.1.50", []string{"192.168.1.0/24"}, true},
		{"10.0.0.1", []string{"10.0.0.0/8"}, true},
		{"172.16.5.10", []string{"172.16.0.0/12"}, true},
		{"8.8.8.8", []string{"192.168.1.0/24"}, false},
		{"", []string{"192.168.1.1"}, false},
		{"invalid-ip", []string{"192.168.1.1"}, false},
		{"192.168.1.1", []string{"invalid-cidr/99"}, false},
		{"192.168.1.1", []string{"", "192.168.1.1"}, true}, // empty entry skipped
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			if got := isIPAllowed(tt.ip, tt.allowed); got != tt.want {
				t.Errorf("isIPAllowed(%q, %v) = %v, want %v", tt.ip, tt.allowed, got, tt.want)
			}
		})
	}
}

func TestMatchCIDR(t *testing.T) {
	tests := []struct {
		ip      string
		cidr    string
		want    bool
	}{
		{"192.168.1.1", "192.168.1.0/24", true},
		{"192.168.1.255", "192.168.1.0/24", true},
		{"192.168.2.1", "192.168.1.0/24", false},
		{"10.0.0.1", "10.0.0.0/8", true},
		{"10.255.255.255", "10.0.0.0/8", true},
		{"11.0.0.1", "10.0.0.0/8", false},
		{"invalid-ip", "192.168.1.0/24", false},
		{"192.168.1.1", "invalid-cidr", false},
		{"192.168.1.1", "192.168.1.0/99", false}, // invalid CIDR
	}

	for _, tt := range tests {
		t.Run(tt.ip+"_"+tt.cidr, func(t *testing.T) {
			if got := matchCIDR(tt.ip, tt.cidr); got != tt.want {
				t.Errorf("matchCIDR(%q, %q) = %v, want %v", tt.ip, tt.cidr, got, tt.want)
			}
		})
	}
}