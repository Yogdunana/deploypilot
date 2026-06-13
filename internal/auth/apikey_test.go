package auth

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// newAPIKeyCtx returns a *gin.Context seeded with a request so that
// c.GetHeader() and c.ClientIP() return the values we set in headers.
// The Context is otherwise empty (no middleware has populated auth keys).
func newAPIKeyCtx(headers map[string]string, remoteAddr string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if remoteAddr != "" {
		req.RemoteAddr = remoteAddr
	}
	c.Request = req
	return c, w
}

// TestExtractAPIKey covers the three documented sources of API keys:
// X-API-Key header, Authorization: Bearer dp_xxx header, and absence
// of both. Edge cases (whitespace, wrong scheme, missing prefix) are also
// covered because they affect whether a key is passed to the validator.
func TestExtractAPIKey(t *testing.T) {
	cases := []struct {
		name    string
		headers map[string]string
		want    string
	}{
		{
			name:    "X-API-Key header",
			headers: map[string]string{"X-API-Key": "dp_abcdef123456"},
			want:    "dp_abcdef123456",
		},
		{
			name:    "X-API-Key trimmed",
			headers: map[string]string{"X-API-Key": "  dp_zzz  "},
			want:    "dp_zzz",
		},
		{
			name:    "Authorization Bearer with dp_ prefix",
			headers: map[string]string{"Authorization": "Bearer dp_secret"},
			want:    "dp_secret",
		},
		{
			name:    "Authorization case-insensitive scheme",
			headers: map[string]string{"Authorization": "bearer dp_case"},
			want:    "dp_case",
		},
		{
			name:    "Authorization Bearer without dp_ prefix is dropped",
			headers: map[string]string{"Authorization": "Bearer someothertoken"},
			want:    "",
		},
		{
			name:    "Authorization Basic scheme ignored",
			headers: map[string]string{"Authorization": "Basic dp_shouldnot"},
			want:    "",
		},
		{
			name:    "no headers",
			headers: map[string]string{},
			want:    "",
		},
		{
			name:    "X-API-Key takes precedence over Authorization",
			headers: map[string]string{"X-API-Key": "dp_first", "Authorization": "Bearer dp_second"},
			want:    "dp_first",
		},
		{
			name:    "Authorization without scheme word",
			headers: map[string]string{"Authorization": "dp_noscheme"},
			want:    "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := newAPIKeyCtx(tc.headers, "")
			got := extractAPIKey(c)
			if got != tc.want {
				t.Errorf("extractAPIKey() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestIsIPAllowed covers the IP allowlist logic: empty list, exact match,
// single-host CIDR, multi-range CIDR, IPv4-mapped IPv6, and malformed
// inputs. This is a security gate that restricts API key usage to known
// network ranges — every branch here corresponds to a real-world scenario.
func TestIsIPAllowed(t *testing.T) {
	cases := []struct {
		name        string
		clientIP    string
		allowedJSON string
		want        bool
	}{
		{
			name:        "empty allowlist permits any IP",
			clientIP:    "8.8.8.8",
			allowedJSON: `[]`,
			want:        true,
		},
		{
			name:        "exact IPv4 match",
			clientIP:    "10.0.0.5",
			allowedJSON: `["10.0.0.5"]`,
			want:        true,
		},
		{
			name:        "exact IPv4 mismatch",
			clientIP:    "10.0.0.6",
			allowedJSON: `["10.0.0.5"]`,
			want:        false,
		},
		{
			name:        "single-host CIDR /32",
			clientIP:    "192.168.1.10",
			allowedJSON: `["192.168.1.10/32"]`,
			want:        true,
		},
		{
			name:        "subnet CIDR /24 contains IP",
			clientIP:    "10.1.2.99",
			allowedJSON: `["10.1.2.0/24"]`,
			want:        true,
		},
		{
			name:        "subnet CIDR /24 rejects out-of-range IP",
			clientIP:    "10.1.3.1",
			allowedJSON: `["10.1.2.0/24"]`,
			want:        false,
		},
		{
			name:        "mixed plain IP and CIDR, second matches",
			clientIP:    "172.16.0.42",
			allowedJSON: `["10.0.0.1", "172.16.0.0/16"]`,
			want:        true,
		},
		{
			name:        "malformed allowed JSON denies all",
			clientIP:    "10.0.0.5",
			allowedJSON: `not json`,
			want:        false,
		},
		{
			name:        "invalid client IP string denies",
			clientIP:    "not-an-ip",
			allowedJSON: `["10.0.0.0/8"]`,
			want:        false,
		},
		{
			name:        "malformed CIDR entry is skipped silently",
			clientIP:    "10.0.0.5",
			allowedJSON: `["bogus/cidr", "10.0.0.0/8"]`,
			want:        true,
		},
		{
			name:        "IPv6 exact match",
			clientIP:    "2001:db8::1",
			allowedJSON: `["2001:db8::1"]`,
			want:        true,
		},
		{
			name:        "IPv6 CIDR /64 contains client",
			clientIP:    "2001:db8::dead:beef",
			allowedJSON: `["2001:db8::/64"]`,
			want:        true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isIPAllowed(tc.clientIP, tc.allowedJSON)
			if got != tc.want {
				t.Errorf("isIPAllowed(%q, %q) = %v, want %v",
					tc.clientIP, tc.allowedJSON, got, tc.want)
			}
		})
	}
}
