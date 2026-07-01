package auth

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestExtractAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		expected string
	}{
		{"X-API-Key header", map[string]string{"X-API-Key": "dp_abc123"}, "dp_abc123"},
		{"Authorization Bearer", map[string]string{"Authorization": "Bearer dp_abc123"}, "dp_abc123"},
		{"X-API-Key with whitespace", map[string]string{"X-API-Key": "  dp_abc123  "}, "dp_abc123"},
		{"Authorization Bearer with whitespace", map[string]string{"Authorization": "Bearer  dp_abc123  "}, "dp_abc123"},
		{"Empty X-API-Key", map[string]string{"X-API-Key": ""}, ""},
		{"Empty Authorization", map[string]string{"Authorization": ""}, ""},
		{"No Bearer prefix", map[string]string{"Authorization": "Token dp_abc123"}, ""},
		{"Bearer without key", map[string]string{"Authorization": "Bearer"}, ""},
		{"Bearer with non-dp prefix", map[string]string{"Authorization": "Bearer sk_abc123"}, ""},
		{"No headers", map[string]string{}, ""},
		{"Other header", map[string]string{"Content-Type": "application/json"}, ""},
		{"Authorization with extra content", map[string]string{"Authorization": "Bearer dp_abc123 extra"}, "dp_abc123 extra"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test", nil)
			for k, v := range tt.headers {
				c.Request.Header.Set(k, v)
			}
			result := extractAPIKey(c)
			if result != tt.expected {
				t.Errorf("extractAPIKey() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestIsIPAllowed(t *testing.T) {
	tests := []struct {
		name       string
		clientIP   string
		allowedIPs string
		expected   bool
	}{
		{"empty allowed IPs", "192.168.1.100", "[]", true},
		{"exact match", "192.168.1.100", `["192.168.1.100"]`, true},
		{"no match", "192.168.1.100", `["192.168.1.99"]`, false},
		{"multiple IPs - match", "192.168.1.100", `["192.168.1.99", "192.168.1.100", "192.168.1.101"]`, true},
		{"multiple IPs - no match", "192.168.1.50", `["192.168.1.99", "192.168.1.100"]`, false},
		{"CIDR match", "192.168.1.100", `["192.168.1.0/24"]`, true},
		{"CIDR no match", "192.168.2.100", `["192.168.1.0/24"]`, false},
		{"mixed IP and CIDR - IP match", "192.168.1.100", `["10.0.0.0/8", "192.168.1.100"]`, true},
		{"mixed IP and CIDR - CIDR match", "10.5.5.5", `["10.0.0.0/8", "192.168.1.100"]`, true},
		{"invalid JSON", "192.168.1.100", `invalid`, false},
		{"invalid client IP", "not-an-ip", `["192.168.1.0/24"]`, false},
		{"invalid CIDR - rejected", "192.168.1.0", `["invalid-cidr"]`, false},
		{"IPv6 exact match", "::1", `["::1"]`, true},
		{"IPv6 CIDR match", "2001:db8::1", `["2001:db8::/32"]`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isIPAllowed(tt.clientIP, tt.allowedIPs)
			if result != tt.expected {
				t.Errorf("isIPAllowed(%q, %q) = %v, want %v", tt.clientIP, tt.allowedIPs, result, tt.expected)
			}
		})
	}
}