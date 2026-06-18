package auth

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestIsIPAllowed_ParsesInvalidJSON confirms that a malformed
// AllowedIPs JSON string is treated as "deny" rather than "allow" or
// "panic". A bug here could open up access to any caller in the event
// of a corrupt API key record.
func TestIsIPAllowed_ParsesInvalidJSON(t *testing.T) {
	if isIPAllowed("1.2.3.4", "not-json") {
		t.Error("invalid AllowedIPs JSON should be treated as deny")
	}
}

// TestIsIPAllowed_EmptyAllowListIsAllowAll confirms the documented
// "empty list = unrestricted" behaviour. This is important for users
// who want to permit the API key from anywhere.
func TestIsIPAllowed_EmptyAllowListIsAllowAll(t *testing.T) {
	if !isIPAllowed("8.8.8.8", "[]") {
		t.Error("empty AllowedIPs list should allow all callers")
	}
	if !isIPAllowed("::1", "[]") {
		t.Error("empty AllowedIPs list should allow IPv6 callers")
	}
}

// TestIsIPAllowed_PlainIPMatch confirms a single plain IP is matched
// exactly.
func TestIsIPAllowed_PlainIPMatch(t *testing.T) {
	allowList := `["192.0.2.10"]`
	if !isIPAllowed("192.0.2.10", allowList) {
		t.Error("matching plain IPv4 should be allowed")
	}
	if isIPAllowed("192.0.2.11", allowList) {
		t.Error("non-matching IPv4 should be denied")
	}
}

// TestIsIPAllowed_CIDRMatch confirms CIDR ranges are respected. The
// check uses net.ParseCIDR which requires that the IP is within the
// network.
func TestIsIPAllowed_CIDRMatch(t *testing.T) {
	allowList := `["10.0.0.0/8"]`
	cases := []struct {
		ip    string
		allow bool
	}{
		{"10.0.0.1", true},
		{"10.255.255.254", true},
		{"11.0.0.1", false},
		{"192.168.0.1", false},
	}
	for _, tc := range cases {
		if got := isIPAllowed(tc.ip, allowList); got != tc.allow {
			t.Errorf("isIPAllowed(%q, %q) = %v, want %v", tc.ip, allowList, got, tc.allow)
		}
	}
}

// TestIsIPAllowed_MixedList confirms that the first match (whether
// plain IP or CIDR) is enough to allow access. Mixed lists are a
// common configuration in production.
func TestIsIPAllowed_MixedList(t *testing.T) {
	allowList := `["203.0.113.5", "10.0.0.0/8", "198.51.100.0/24"]`
	cases := []struct {
		ip    string
		allow bool
	}{
		{"203.0.113.5", true},   // plain IP
		{"10.42.42.42", true},   // CIDR
		{"198.51.100.7", true},  // CIDR
		{"198.51.101.7", false}, // not in any range
		{"8.8.8.8", false},      // not allowed
	}
	for _, tc := range cases {
		if got := isIPAllowed(tc.ip, allowList); got != tc.allow {
			t.Errorf("isIPAllowed(%q) = %v, want %v", tc.ip, got, tc.allow)
		}
	}
}

// TestIsIPAllowed_IPv6 confirms IPv6 addresses are handled by net.ParseIP
// and that CIDR matching works for them too. The previous test set
// focused on IPv4.
func TestIsIPAllowed_IPv6(t *testing.T) {
	allowList := `["2001:db8::/32"]`
	if !isIPAllowed("2001:db8::1", allowList) {
		t.Error("IPv6 in CIDR should be allowed")
	}
	if isIPAllowed("2001:db9::1", allowList) {
		t.Error("IPv6 outside CIDR should be denied")
	}
}

// TestIsIPAllowed_InvalidClientIPIsDenied confirms that an unparseable
// client IP string is denied rather than allowed.
func TestIsIPAllowed_InvalidClientIPIsDenied(t *testing.T) {
	allowList := `["10.0.0.0/8"]`
	if isIPAllowed("not-an-ip", allowList) {
		t.Error("unparseable client IP should be denied")
	}
}

// TestIsIPAllowed_InvalidCIDRIgnored confirms that malformed CIDR
// entries in the list are skipped (not fatal). A typo in one entry
// must not lock out every caller.
func TestIsIPAllowed_InvalidCIDRIgnored(t *testing.T) {
	// First entry is invalid, second is a valid plain IP.
	allowList := `["10.0.0.0/not-a-cidr", "203.0.113.5"]`
	if !isIPAllowed("203.0.113.5", allowList) {
		t.Error("valid entry after invalid entry should still be matched")
	}
}

// TestExtractAPIKey_XAPIKeyHeader confirms that an X-API-Key header is
// the highest-priority source for the key. This is the standard header
// used by the OpenAI-style API.
func TestExtractAPIKey_XAPIKeyHeader(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("X-API-Key", "dp_test_key_1")

	if got := extractAPIKey(c); got != "dp_test_key_1" {
		t.Errorf("extractAPIKey = %q, want %q", got, "dp_test_key_1")
	}
}

// TestExtractAPIKey_BearerHeader confirms the secondary path: a
// "Authorization: Bearer dp_xxx" header. Note: keys without the "dp_"
// prefix are intentionally rejected at this layer (the middleware
// passes them through to JWT auth instead).
func TestExtractAPIKey_BearerHeader(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Bearer dp_bearer_key_123")

	if got := extractAPIKey(c); got != "dp_bearer_key_123" {
		t.Errorf("extractAPIKey = %q, want %q", got, "dp_bearer_key_123")
	}
}

// TestExtractAPIKey_BearerIsCaseInsensitive confirms that the bearer
// scheme name is matched case-insensitively (per RFC 7235).
func TestExtractAPIKey_BearerIsCaseInsensitive(t *testing.T) {
	for _, scheme := range []string{"bearer", "BEARER", "Bearer", "BeArEr"} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/", nil)
		c.Request.Header.Set("Authorization", scheme+" dp_case_key")

		if got := extractAPIKey(c); got != "dp_case_key" {
			t.Errorf("scheme %q: extractAPIKey = %q, want %q", scheme, got, "dp_case_key")
		}
	}
}

// TestExtractAPIKey_BearerWithoutPrefixIsIgnored confirms that bearer
// tokens without the "dp_" prefix are not treated as API keys (this
// prevents a JWT in the Authorization header from being misinterpreted
// as an API key).
func TestExtractAPIKey_BearerWithoutPrefixIsIgnored(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiJ9.payload.sig")

	if got := extractAPIKey(c); got != "" {
		t.Errorf("non-dp_ bearer should be ignored, got %q", got)
	}
}

// TestExtractAPIKey_XAPIKeyTakesPriorityOverBearer confirms the
// precedence: if both headers are present, the X-API-Key header wins.
func TestExtractAPIKey_XAPIKeyTakesPriorityOverBearer(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("X-API-Key", "dp_xkey_winner")
	c.Request.Header.Set("Authorization", "Bearer dp_bearer_loser")

	if got := extractAPIKey(c); got != "dp_xkey_winner" {
		t.Errorf("X-API-Key should win, got %q", got)
	}
}

// TestExtractAPIKey_NoHeadersReturnsEmpty confirms the function does
// not panic or invent a key when neither header is present.
func TestExtractAPIKey_NoHeadersReturnsEmpty(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)

	if got := extractAPIKey(c); got != "" {
		t.Errorf("extractAPIKey with no headers = %q, want empty", got)
	}
}

// TestExtractAPIKey_MalformedAuthorizationIgnored confirms a malformed
// Authorization header (e.g. no scheme) does not crash the extractor.
func TestExtractAPIKey_MalformedAuthorizationIgnored(t *testing.T) {
	cases := []string{
		"dp_bare_token",         // no scheme
		"Basic dp_basic",        // wrong scheme
		"Bearer",                // scheme only
		"Bearer ",               // empty token
		strings.Repeat("a", 64), // garbage
	}
	for _, h := range cases {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/", nil)
		c.Request.Header.Set("Authorization", h)

		if got := extractAPIKey(c); got != "" {
			t.Errorf("malformed Authorization %q: extractAPIKey = %q, want empty", h, got)
		}
	}
}

// TestExtractAPIKey_TrimsWhitespace confirms that stray whitespace in
// the header is trimmed (e.g. from badly-configured proxies).
func TestExtractAPIKey_TrimsWhitespace(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("X-API-Key", "  dp_padded_key  ")

	if got := extractAPIKey(c); got != "dp_padded_key" {
		t.Errorf("extractAPIKey = %q, want %q (whitespace trimmed)", got, "dp_padded_key")
	}
}
