package auth

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestExtractAPIKey_FromXAPIKeyHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "dp_test_xyz")
	c.Request = req

	got := extractAPIKey(c)
	if got != "dp_test_xyz" {
		t.Errorf("extractAPIKey() = %q, want %q", got, "dp_test_xyz")
	}
}

func TestExtractAPIKey_XAPIKeyTrimmed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "  dp_test_xyz  ")
	c.Request = req

	got := extractAPIKey(c)
	if got != "dp_test_xyz" {
		t.Errorf("extractAPIKey() should trim whitespace, got %q", got)
	}
}

func TestExtractAPIKey_FromAuthorizationBearer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer dp_test_xyz")
	c.Request = req

	got := extractAPIKey(c)
	if got != "dp_test_xyz" {
		t.Errorf("extractAPIKey() = %q, want %q", got, "dp_test_xyz")
	}
}

func TestExtractAPIKey_BearerCaseInsensitive(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "bearer dp_test_xyz")
	c.Request = req

	got := extractAPIKey(c)
	if got != "dp_test_xyz" {
		t.Errorf("extractAPIKey() should accept any case for 'bearer', got %q", got)
	}
}

func TestExtractAPIKey_BearerWithoutPrefixIgnored(t *testing.T) {
	// Authorization: Bearer <key> only returns the key if it has the dp_ prefix.
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer someotherkey")
	c.Request = req

	got := extractAPIKey(c)
	if got != "" {
		t.Errorf("extractAPIKey() = %q, want empty (non-dp_ key should be ignored)", got)
	}
}

func TestExtractAPIKey_NoHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest("GET", "/", nil)
	c.Request = req

	got := extractAPIKey(c)
	if got != "" {
		t.Errorf("extractAPIKey() = %q, want empty for missing headers", got)
	}
}

func TestExtractAPIKey_XAPIKeyTakesPrecedence(t *testing.T) {
	// When both headers are present, X-API-Key wins.
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "dp_from_xheader")
	req.Header.Set("Authorization", "Bearer dp_from_bearer")
	c.Request = req

	got := extractAPIKey(c)
	if got != "dp_from_xheader" {
		t.Errorf("extractAPIKey() = %q, want %q (X-API-Key should win)", got, "dp_from_xheader")
	}
}

func TestIsIPAllowed_EmptyList(t *testing.T) {
	// Empty allowlist is interpreted as "no restriction" and should allow.
	if !isIPAllowed("10.0.0.1", "[]") {
		t.Error("isIPAllowed() with empty allowlist should return true")
	}
}

func TestIsIPAllowed_ExactMatch(t *testing.T) {
	allowed := `["192.168.1.10"]`
	if !isIPAllowed("192.168.1.10", allowed) {
		t.Error("exact-match IP should be allowed")
	}
	if isIPAllowed("192.168.1.11", allowed) {
		t.Error("non-matching IP should be denied")
	}
}

func TestIsIPAllowed_CIDR(t *testing.T) {
	allowed := `["10.0.0.0/24"]`
	tests := []struct {
		ip     string
		expect bool
	}{
		{"10.0.0.1", true},
		{"10.0.0.100", true},
		{"10.0.0.255", true},
		{"10.0.1.0", false},
		{"192.168.1.1", false},
	}
	for _, tc := range tests {
		got := isIPAllowed(tc.ip, allowed)
		if got != tc.expect {
			t.Errorf("isIPAllowed(%q) = %v, want %v", tc.ip, got, tc.expect)
		}
	}
}

func TestIsIPAllowed_MixedCIDRAndPlain(t *testing.T) {
	allowed := `["192.168.1.10", "10.0.0.0/8", "172.16.5.5"]`
	tests := []struct {
		ip     string
		expect bool
	}{
		{"192.168.1.10", true},
		{"10.1.2.3", true},
		{"172.16.5.5", true},
		{"172.16.5.6", false},
		{"8.8.8.8", false},
	}
	for _, tc := range tests {
		got := isIPAllowed(tc.ip, allowed)
		if got != tc.expect {
			t.Errorf("isIPAllowed(%q) = %v, want %v", tc.ip, got, tc.expect)
		}
	}
}

func TestIsIPAllowed_InvalidJSON(t *testing.T) {
	// Malformed JSON is treated as deny-by-default to fail closed.
	if isIPAllowed("10.0.0.1", "not-json") {
		t.Error("isIPAllowed() with invalid JSON should return false (fail-closed)")
	}
}

func TestIsIPAllowed_InvalidClientIP(t *testing.T) {
	allowed := `["10.0.0.0/24"]`
	if isIPAllowed("not-an-ip", allowed) {
		t.Error("isIPAllowed() with invalid client IP should return false")
	}
}

func TestIsIPAllowed_InvalidCIDRIsSkipped(t *testing.T) {
	// An invalid CIDR in the list should be ignored, not crash the call.
	allowed := `["not-a-cidr", "10.0.0.0/24"]`
	if !isIPAllowed("10.0.0.1", allowed) {
		t.Error("valid CIDR match should still allow the request")
	}
	if isIPAllowed("192.168.1.1", allowed) {
		t.Error("non-matching IP should still be denied when only invalid entries match")
	}
}
