package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestIsIPAllowed(t *testing.T) {
	testCases := []struct {
		name       string
		clientIP   string
		allowedIPs string
		expected   bool
	}{
		{
			name:       "empty allowed IPs",
			clientIP:   "192.168.1.1",
			allowedIPs: "[]",
			expected:   true,
		},
		{
			name:       "allowed single IP",
			clientIP:   "192.168.1.100",
			allowedIPs: `["192.168.1.100"]`,
			expected:   true,
		},
		{
			name:       "not allowed IP",
			clientIP:   "192.168.1.200",
			allowedIPs: `["192.168.1.100"]`,
			expected:   false,
		},
		{
			name:       "allowed multiple IPs",
			clientIP:   "10.0.0.5",
			allowedIPs: `["192.168.1.100", "10.0.0.5", "172.16.0.1"]`,
			expected:   true,
		},
		{
			name:       "allowed CIDR",
			clientIP:   "192.168.1.150",
			allowedIPs: `["192.168.1.0/24"]`,
			expected:   true,
		},
		{
			name:       "not allowed CIDR",
			clientIP:   "192.168.2.150",
			allowedIPs: `["192.168.1.0/24"]`,
			expected:   false,
		},
		{
			name:       "allowed CIDR and IP",
			clientIP:   "10.0.0.1",
			allowedIPs: `["192.168.1.0/24", "10.0.0.1"]`,
			expected:   true,
		},
		{
			name:       "invalid JSON allowed IPs",
			clientIP:   "192.168.1.1",
			allowedIPs: `invalid json`,
			expected:   false,
		},
		{
			name:       "invalid client IP",
			clientIP:   "invalid-ip",
			allowedIPs: `["192.168.1.0/24"]`,
			expected:   false,
		},
		{
			name:       "loopback allowed",
			clientIP:   "127.0.0.1",
			allowedIPs: `["127.0.0.1"]`,
			expected:   true,
		},
		{
			name:       "IPv6 allowed",
			clientIP:   "::1",
			allowedIPs: `["::1"]`,
			expected:   true,
		},
		{
			name:       "IPv6 CIDR allowed",
			clientIP:   "2001:db8:85a3::8a2e:370:7334",
			allowedIPs: `["2001:db8:85a3::/64"]`,
			expected:   true,
		},
		{
			name:       "invalid CIDR",
			clientIP:   "192.168.1.1",
			allowedIPs: `["invalid-cidr"]`,
			expected:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isIPAllowed(tc.clientIP, tc.allowedIPs)
			if result != tc.expected {
				t.Errorf("isIPAllowed(%q, %q) = %v, want %v", tc.clientIP, tc.allowedIPs, result, tc.expected)
			}
		})
	}
}

func TestExtractAPIKey(t *testing.T) {
	testCases := []struct {
		name          string
		xAPIKeyHeader string
		authHeader    string
		expected      string
	}{
		{
			name:          "X-API-Key header",
			xAPIKeyHeader: "dp_abc123",
			authHeader:    "",
			expected:      "dp_abc123",
		},
		{
			name:          "Authorization Bearer header",
			xAPIKeyHeader: "",
			authHeader:    "Bearer dp_xyz789",
			expected:      "dp_xyz789",
		},
		{
			name:          "X-API-Key takes precedence",
			xAPIKeyHeader: "dp_apikey1",
			authHeader:    "Bearer dp_bearer1",
			expected:      "dp_apikey1",
		},
		{
			name:          "no headers",
			xAPIKeyHeader: "",
			authHeader:    "",
			expected:      "",
		},
		{
			name:          "Bearer with extra spaces",
			xAPIKeyHeader: "",
			authHeader:    "  Bearer   dp_abc123  ",
			expected:      "dp_abc123",
		},
		{
			name:          "Bearer case insensitive",
			xAPIKeyHeader: "",
			authHeader:    "bearer dp_abc123",
			expected:      "dp_abc123",
		},
		{
			name:          "invalid Authorization format",
			xAPIKeyHeader: "",
			authHeader:    "Bearer",
			expected:      "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			ctx.Request = &http.Request{Header: make(http.Header)}
			if tc.xAPIKeyHeader != "" {
				ctx.Request.Header.Set("X-API-Key", tc.xAPIKeyHeader)
			}
			if tc.authHeader != "" {
				ctx.Request.Header.Set("Authorization", tc.authHeader)
			}
			result := extractAPIKey(ctx)
			if result != tc.expected {
				t.Errorf("extractAPIKey() = %q, want %q", result, tc.expected)
			}
		})
	}
}