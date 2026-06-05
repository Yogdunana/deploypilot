package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestIsIPAllowed_EmptyList(t *testing.T) {
	result := isIPAllowed("192.168.1.1", "[]")
	if !result {
		t.Error("empty allowed list should allow all IPs")
	}
}

func TestIsIPAllowed_InvalidJSON(t *testing.T) {
	result := isIPAllowed("192.168.1.1", "not json")
	if result {
		t.Error("invalid JSON should deny access")
	}
}

func TestIsIPAllowed_ExactIPMatch(t *testing.T) {
	allowed := []string{"192.168.1.1", "10.0.0.1"}
	data, _ := json.Marshal(allowed)

	result := isIPAllowed("192.168.1.1", string(data))
	if !result {
		t.Error("exact IP match should be allowed")
	}

	result = isIPAllowed("192.168.1.2", string(data))
	if result {
		t.Error("non-matching IP should be denied")
	}
}

func TestIsIPAllowed_CIDRMatch(t *testing.T) {
	allowed := []string{"192.168.1.0/24", "10.0.0.0/8"}
	data, _ := json.Marshal(allowed)

	// Within 192.168.1.0/24
	result := isIPAllowed("192.168.1.100", string(data))
	if !result {
		t.Error("IP within CIDR should be allowed")
	}

	// Outside 192.168.1.0/24
	result = isIPAllowed("192.168.2.1", string(data))
	if result {
		t.Error("IP outside CIDR should be denied")
	}
}

func TestIsIPAllowed_IPv6(t *testing.T) {
	allowed := []string{"::1", "fe80::/10"}
	data, _ := json.Marshal(allowed)

	result := isIPAllowed("::1", string(data))
	if !result {
		t.Error("IPv6 loopback should be allowed")
	}

	result = isIPAllowed("fe80::1", string(data))
	if !result {
		t.Error("IPv6 link-local should be allowed")
	}
}

func TestIsIPAllowed_InvalidClientIP(t *testing.T) {
	allowed := []string{"192.168.1.0/24"}
	data, _ := json.Marshal(allowed)

	result := isIPAllowed("", string(data))
	if result {
		t.Error("empty client IP should be denied")
	}

	result = isIPAllowed("not-an-ip", string(data))
	if result {
		t.Error("invalid client IP should be denied")
	}
}

func TestIsIPAllowed_InvalidCIDR(t *testing.T) {
	allowed := []string{"192.168.1.0/999"} // invalid CIDR
	data, _ := json.Marshal(allowed)

	result := isIPAllowed("192.168.1.1", string(data))
	if result {
		t.Error("invalid CIDR should be skipped and IP should be denied")
	}
}

func TestExtractAPIKey_FromXAPIKeyHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set("X-API-Key", "dp_test_key_12345")

	result := extractAPIKey(c)
	if result != "dp_test_key_12345" {
		t.Errorf("extractAPIKey() = %q, want %q", result, "dp_test_key_12345")
	}
}

func TestExtractAPIKey_FromXAPIKeyHeaderWithWhitespace(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set("X-API-Key", "  dp_test_key_12345  ")

	result := extractAPIKey(c)
	if result != "dp_test_key_12345" {
		t.Errorf("extractAPIKey() = %q, want trimmed %q", result, "dp_test_key_12345")
	}
}

func TestExtractAPIKey_FromBearerHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set("Authorization", "Bearer dp_test_key_12345")

	result := extractAPIKey(c)
	if result != "dp_test_key_12345" {
		t.Errorf("extractAPIKey() = %q, want %q", result, "dp_test_key_12345")
	}
}

func TestExtractAPIKey_BearerCaseInsensitive(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set("Authorization", "BEARER dp_test_key_12345")

	result := extractAPIKey(c)
	if result != "dp_test_key_12345" {
		t.Errorf("extractAPIKey() = %q, want %q", result, "dp_test_key_12345")
	}
}

func TestExtractAPIKey_NonDPKeyInBearer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set("Authorization", "Bearer other_key_12345")

	result := extractAPIKey(c)
	if result != "" {
		t.Errorf("extractAPIKey() = %q, want empty string for non-dp key", result)
	}
}

func TestExtractAPIKey_NoHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	result := extractAPIKey(c)
	if result != "" {
		t.Errorf("extractAPIKey() = %q, want empty string", result)
	}
}

func TestExtractAPIKey_OnlyXAPIKeyTakesPrecedence(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set("X-API-Key", "dp_from_header")
	c.Request.Header.Set("Authorization", "Bearer dp_from_bearer")

	result := extractAPIKey(c)
	if result != "dp_from_header" {
		t.Errorf("extractAPIKey() = %q, X-API-Key should take precedence", result)
	}
}

func TestExtractAPIKey_BearerWithSpaces(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set("Authorization", "Bearer   dp_key_with_spaces  ")

	result := extractAPIKey(c)
	if result != "dp_key_with_spaces" {
		t.Errorf("extractAPIKey() = %q, want trimmed %q", result, "dp_key_with_spaces")
	}
}

func TestExtractAPIKey_InvalidBearerFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []string{
		"Bearer",              // no value
		"BearerOnly",         // no space
		"dp_test_key",        // wrong scheme, but has dp_ prefix
		"",                   // empty
	}

	for _, auth := range tests {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Request.Header.Set("Authorization", auth)

		result := extractAPIKey(c)
		if result != "" {
			t.Errorf("extractAPIKey() = %q for auth %q, want empty", result, auth)
		}
	}
}
