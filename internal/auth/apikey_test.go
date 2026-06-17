package auth

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// ===================== extractAPIKey =====================

func newExtractAPIKeyTest(t *testing.T, headerName, headerValue string) string {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest("GET", "/", nil)
	if headerName != "" {
		req.Header.Set(headerName, headerValue)
	}
	c.Request = req
	return extractAPIKey(c)
}

func TestExtractAPIKey_FromXAPIKeyHeader(t *testing.T) {
	got := newExtractAPIKeyTest(t, "X-API-Key", "dp_abcdef0123456789")
	if got != "dp_abcdef0123456789" {
		t.Errorf("expected dp_abcdef0123456789, got %q", got)
	}
}

func TestExtractAPIKey_TrimsWhitespaceFromXAPIKey(t *testing.T) {
	got := newExtractAPIKeyTest(t, "X-API-Key", "   dp_with_padding   ")
	if got != "dp_with_padding" {
		t.Errorf("expected trimmed key, got %q", got)
	}
}

func TestExtractAPIKey_EmptyXAPIKeyHeader_FallsBackToAuthorization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "")
	req.Header.Set("Authorization", "Bearer dp_bearerkey")
	c.Request = req
	if got := extractAPIKey(c); got != "dp_bearerkey" {
		t.Errorf("expected fallback to Authorization, got %q", got)
	}
}

func TestExtractAPIKey_FromAuthorizationBearer(t *testing.T) {
	got := newExtractAPIKeyTest(t, "Authorization", "Bearer dp_xyz")
	if got != "dp_xyz" {
		t.Errorf("expected dp_xyz, got %q", got)
	}
}

func TestExtractAPIKey_BearerCaseInsensitive(t *testing.T) {
	for _, scheme := range []string{"bearer", "BEARER", "Bearer", "BeArEr"} {
		t.Run(scheme, func(t *testing.T) {
			got := newExtractAPIKeyTest(t, "Authorization", scheme+" dp_key")
			if got != "dp_key" {
				t.Errorf("expected scheme %q to be accepted, got %q", scheme, got)
			}
		})
	}
}

func TestExtractAPIKey_NonBearerSchemeIgnored(t *testing.T) {
	got := newExtractAPIKeyTest(t, "Authorization", "Basic dXNlcjpwYXNz")
	if got != "" {
		t.Errorf("expected empty result for non-bearer scheme, got %q", got)
	}
}

func TestExtractAPIKey_BearerWithoutPrefixIgnored(t *testing.T) {
	// The middleware only returns the bearer token if it starts with "dp_".
	// Other tokens (e.g. JWTs) should be left for the JWT middleware.
	got := newExtractAPIKeyTest(t, "Authorization", "Bearer eyJhbGciOiJIUzI1NiJ9.foo.bar")
	if got != "" {
		t.Errorf("expected empty result for non-dp_ token, got %q", got)
	}
}

func TestExtractAPIKey_AuthorizationWithoutBearer(t *testing.T) {
	got := newExtractAPIKeyTest(t, "Authorization", "dp_nakedkey")
	if got != "" {
		t.Errorf("expected empty result when Authorization lacks Bearer prefix, got %q", got)
	}
}

func TestExtractAPIKey_NoHeaders(t *testing.T) {
	got := newExtractAPIKeyTest(t, "", "")
	if got != "" {
		t.Errorf("expected empty result, got %q", got)
	}
}

func TestExtractAPIKey_TrimsBearerWhitespace(t *testing.T) {
	got := newExtractAPIKeyTest(t, "Authorization", "Bearer   dp_padded   ")
	if got != "dp_padded" {
		t.Errorf("expected trimmed dp_ token from bearer header, got %q", got)
	}
}

// ===================== isIPAllowed =====================

func TestIsIPAllowed_EmptyJSONArray_AllowsAll(t *testing.T) {
	// The JSON []string{""} unmarshals to an empty slice, which is treated
	// as "no IP restriction".
	if !isIPAllowed("203.0.113.5", "[]") {
		t.Error("expected empty list to allow all IPs")
	}
}

func TestIsIPAllowed_ValidJSON_ExactMatch(t *testing.T) {
	json := `["10.0.0.1", "192.168.1.10"]`
	if !isIPAllowed("10.0.0.1", json) {
		t.Error("expected 10.0.0.1 to be allowed by exact match")
	}
	if isIPAllowed("10.0.0.2", json) {
		t.Error("expected 10.0.0.2 to be rejected")
	}
}

func TestIsIPAllowed_ValidJSON_CIDRMatch(t *testing.T) {
	json := `["192.168.1.0/24"]`
	if !isIPAllowed("192.168.1.42", json) {
		t.Error("expected 192.168.1.42 to be allowed by CIDR")
	}
	if isIPAllowed("192.168.2.1", json) {
		t.Error("expected 192.168.2.1 to be rejected")
	}
}

func TestIsIPAllowed_MalformedJSON_Rejects(t *testing.T) {
	if isIPAllowed("10.0.0.1", "not-json") {
		t.Error("expected malformed JSON to result in rejection")
	}
}

func TestIsIPAllowed_InvalidIP_Rejects(t *testing.T) {
	json := `["10.0.0.1"]`
	if isIPAllowed("not-an-ip", json) {
		t.Error("expected invalid client IP to be rejected")
	}
}

func TestIsIPAllowed_InvalidCIDRSkipped_NoAllowListMatch(t *testing.T) {
	// Both entries are invalid → no match → reject.
	json := `["not-a-cidr/24", "also-bad/8"]`
	if isIPAllowed("10.0.0.1", json) {
		t.Error("expected rejection when no entries parse as CIDR/IP")
	}
}

func TestIsIPAllowed_MixedValidAndInvalid(t *testing.T) {
	// Malformed entry is skipped, the valid one should still match.
	json := `["not-a-cidr", "10.0.0.0/8"]`
	if !isIPAllowed("10.5.6.7", json) {
		t.Error("expected match from valid CIDR despite malformed entry")
	}
}
