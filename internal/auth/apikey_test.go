package auth

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// --- extractAPIKey ---

// TestExtractAPIKey_FromXAPIKeyHeader verifies the primary header path.
func TestExtractAPIKey_FromXAPIKeyHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("X-API-Key", "  dp_abc123  ")

	if got := extractAPIKey(c); got != "dp_abc123" {
		t.Errorf("expected trimmed dp_abc123, got %q", got)
	}
}

// TestExtractAPIKey_FromBearerHeader verifies the secondary Authorization
// header path. Only the Bearer scheme is accepted, and only if the token
// starts with the "dp_" prefix used by DeployPilot API keys.
func TestExtractAPIKey_FromBearerHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Bearer dp_secret-token")

	if got := extractAPIKey(c); got != "dp_secret-token" {
		t.Errorf("expected dp_secret-token from Bearer, got %q", got)
	}
}

// TestExtractAPIKey_BearerCaseInsensitive confirms the scheme match is
// case-insensitive (Bearer, bearer, BEARER all work).
func TestExtractAPIKey_BearerCaseInsensitive(t *testing.T) {
	for _, scheme := range []string{"bearer", "Bearer", "BEARER", "BeArEr"} {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/", nil)
		c.Request.Header.Set("Authorization", scheme+" dp_xyz")

		if got := extractAPIKey(c); got != "dp_xyz" {
			t.Errorf("scheme=%q: expected dp_xyz, got %q", scheme, got)
		}
	}
}

// TestExtractAPIKey_BearerWithoutPrefixIgnored ensures that a Bearer token
// that does NOT start with the dp_ prefix is ignored. This prevents the
// middleware from misinterpreting a JWT access token (which uses the
// same Authorization header) as an API key.
func TestExtractAPIKey_BearerWithoutPrefixIgnored(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Bearer eyJhbGciOi...not-an-api-key")

	if got := extractAPIKey(c); got != "" {
		t.Errorf("expected empty (non-dp_ token ignored), got %q", got)
	}
}

// TestExtractAPIKey_NonBearerAuthSchemeIgnored ensures non-Bearer auth
// schemes (Basic, Digest, etc.) are not treated as API keys.
func TestExtractAPIKey_NonBearerAuthSchemeIgnored(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Basic dXNlcjpwYXNz")

	if got := extractAPIKey(c); got != "" {
		t.Errorf("expected empty (Basic auth ignored), got %q", got)
	}
}

// TestExtractAPIKey_HeaderPriority checks that the X-API-Key header wins
// when both it and the Authorization header are present. This avoids
// ambiguity in shared-auth environments.
func TestExtractAPIKey_HeaderPriority(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("X-API-Key", "dp_from_header")
	c.Request.Header.Set("Authorization", "Bearer dp_from_bearer")

	if got := extractAPIKey(c); got != "dp_from_header" {
		t.Errorf("expected X-API-Key to take priority, got %q", got)
	}
}

// TestExtractAPIKey_NoHeaders verifies the function returns empty when no
// authentication header is present, so the caller (the middleware) can
// safely fall through to other auth methods.
func TestExtractAPIKey_NoHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)

	if got := extractAPIKey(c); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// --- isIPAllowed ---

// TestIsIPAllowed_EmptyAllowedListAllows verifies the default-allow
// behavior when an API key has no IP restrictions configured.
func TestIsIPAllowed_EmptyAllowedListAllows(t *testing.T) {
	if !isIPAllowed("203.0.113.1", "[]") {
		t.Error("empty allowed list should permit any IP")
	}
	if !isIPAllowed("198.51.100.7", "[]") {
		t.Error("empty allowed list should permit any IP")
	}
}

// TestIsIPAllowed_PlainIPMatch verifies exact-IP matching in the whitelist.
func TestIsIPAllowed_PlainIPMatch(t *testing.T) {
	allowed := `["203.0.113.5", "198.51.100.10"]`

	if !isIPAllowed("203.0.113.5", allowed) {
		t.Error("exact match for 203.0.113.5 should be allowed")
	}
	if !isIPAllowed("198.51.100.10", allowed) {
		t.Error("exact match for 198.51.100.10 should be allowed")
	}
	if isIPAllowed("203.0.113.6", allowed) {
		t.Error("non-listed IP should be rejected")
	}
}

// TestIsIPAllowed_CIDRMatch verifies CIDR-range matching.
func TestIsIPAllowed_CIDRMatch(t *testing.T) {
	allowed := `["10.0.0.0/8", "192.168.1.0/24"]`

	allowedCases := []string{"10.0.0.1", "10.255.255.254", "192.168.1.50", "192.168.1.1"}
	for _, ip := range allowedCases {
		if !isIPAllowed(ip, allowed) {
			t.Errorf("expected %s to be allowed under CIDR", ip)
		}
	}

	deniedCases := []string{"11.0.0.1", "192.168.2.1", "172.16.0.1", "203.0.113.1"}
	for _, ip := range deniedCases {
		if isIPAllowed(ip, allowed) {
			t.Errorf("expected %s to be denied", ip)
		}
	}
}

// TestIsIPAllowed_InvalidClientIP ensures that a malformed client IP
// is rejected rather than silently allowed.
func TestIsIPAllowed_InvalidClientIP(t *testing.T) {
	allowed := `["203.0.113.5"]`

	for _, ip := range []string{"not-an-ip", "", "999.999.999.999", "abc.def.ghi.jkl"} {
		if isIPAllowed(ip, allowed) {
			t.Errorf("malformed client IP %q should be denied", ip)
		}
	}
}

// TestIsIPAllowed_InvalidJSONAllowedList ensures that a corrupted allowed-IPs
// JSON denies access (fail-closed), preventing bypass by tampering with
// stored data.
func TestIsIPAllowed_InvalidJSONAllowedList(t *testing.T) {
	if isIPAllowed("203.0.113.5", "{not-json") {
		t.Error("corrupted allowed-IPs JSON must fail-closed (deny access)")
	}
	if isIPAllowed("203.0.113.5", "") {
		t.Error("empty allowed-IPs JSON must fail-closed (deny access)")
	}
}

// TestIsIPAllowed_InvalidCIDREntryIsIgnored verifies that an entry that
// is neither a valid IP nor a valid CIDR is silently skipped (the matcher
// continues to the next entry), so a single bad entry does not lock
// out all callers.
func TestIsIPAllowed_InvalidCIDREntryIsIgnored(t *testing.T) {
	allowed := `["not-an-ip", "not-cidr/8", "10.0.0.5"]`

	if !isIPAllowed("10.0.0.5", allowed) {
		t.Error("valid IP after invalid entries should still be allowed")
	}
	if isIPAllowed("10.0.0.6", allowed) {
		t.Error("IP not in valid entries should still be rejected")
	}
}

// TestIsIPAllowed_IPv6Loopback verifies the matcher handles IPv6 addresses.
func TestIsIPAllowed_IPv6Loopback(t *testing.T) {
	allowed := `["::1"]`

	if !isIPAllowed("::1", allowed) {
		t.Error("IPv6 loopback should be allowed when listed")
	}
	if isIPAllowed("::2", allowed) {
		t.Error("unlisted IPv6 should be denied")
	}
}

// --- ParseScopes / HasScope (service package) ---

// TestParseScopes_Empty ensures a nil/empty input produces a non-nil empty slice.
func TestParseScopes_Empty(t *testing.T) {
	if got := service.ParseScopes(""); got != nil {
		t.Errorf("expected nil for empty input, got %v", got)
	}
	if got := service.ParseScopes("[]"); len(got) != 0 {
		t.Errorf("expected empty slice for [], got %v", got)
	}
}

// TestParseScopes_ValidJSON verifies normal happy-path JSON parsing.
func TestParseScopes_ValidJSON(t *testing.T) {
	scopes := []string{"read", "deploy", "monitor:read"}
	data, _ := json.Marshal(scopes)

	got := service.ParseScopes(string(data))
	if len(got) != len(scopes) {
		t.Fatalf("expected %d scopes, got %d", len(scopes), len(got))
	}
	for i, s := range scopes {
		if got[i] != s {
			t.Errorf("scope[%d]=%q, want %q", i, got[i], s)
		}
	}
}

// TestParseScopes_InvalidJSONReturnsNil ensures the helper does not panic
// or return garbage when the stored JSON is malformed.
func TestParseScopes_InvalidJSONReturnsNil(t *testing.T) {
	if got := service.ParseScopes("{not-json"); got != nil {
		t.Errorf("expected nil for invalid JSON, got %v", got)
	}
}

// TestHasScope_AdminBypass verifies the documented admin bypass in HasScope.
func TestHasScope_AdminBypass(t *testing.T) {
	if !service.HasScope([]string{"admin"}, "anything") {
		t.Error("admin scope should bypass any required scope")
	}
	if !service.HasScope([]string{"admin", "read"}, "deploy") {
		t.Error("admin scope in a multi-scope token should bypass")
	}
	if !service.HasScope([]string{"read"}, "read") {
		t.Error("matching scope should satisfy")
	}
	// Case-insensitive: ADMIN (uppercase) also counts as the admin bypass
	// because the function checks `s == "admin"` literally — the
	// documented bypass is case-sensitive.
	if service.HasScope([]string{"ADMIN"}, "deploy") {
		t.Error("admin bypass is case-sensitive: ADMIN alone should not satisfy")
	}
	if service.HasScope([]string{"read"}, "deploy") {
		t.Error("read scope should not satisfy deploy")
	}
	if service.HasScope(nil, "read") {
		t.Error("nil scopes should not satisfy any requirement")
	}
}
