package i18n

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestParseAcceptLanguage_EmptyReturnsDefault confirms that a missing
// or empty header falls back to the default locale. A bug here could
// cause a 100% English fallback on every request even when the client
// has a clear preference.
func TestParseAcceptLanguage_EmptyReturnsDefault(t *testing.T) {
	if got := parseAcceptLanguage(""); got != "en" {
		t.Errorf("parseAcceptLanguage(\"\") = %q, want en", got)
	}
}

// TestParseAcceptLanguage_TakesFirstTag confirms the simple "first
// wins" parsing strategy. A real Accept-Language header is
// "tag;q=0.x,tag;q=0.x,..."; we ignore q-values for the first tag
// (the first tag is what the user most prefers).
func TestParseAcceptLanguage_TakesFirstTag(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"zh-CN", "zh"},
		{"zh-CN,zh;q=0.9,en;q=0.8", "zh"},
		{"en-US,en;q=0.9", "en"},
		{"ZH", "zh"},        // case insensitive
		{"  en-US  ", "en"}, // trimmed
		{"de", "en"},        // unsupported -> default
		{"fr-CA", "en"},     // unsupported -> default
	}
	for _, tc := range cases {
		if got := parseAcceptLanguage(tc.in); got != tc.want {
			t.Errorf("parseAcceptLanguage(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestNormalizeLocale_MapsCommonTags confirms that the well-known
// language tags all map to a supported locale. The function is the
// only thing standing between an arbitrary client header and the
// translation table; missing a tag here causes silent fallback to
// English.
func TestNormalizeLocale_MapsCommonTags(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"en", "en"},
		{"EN", "en"},
		{"en-US", "en"},
		{"en-GB", "en"},
		{"zh", "zh"},
		{"zh-CN", "zh"},
		{"zh-TW", "zh"},
		{"zh-Hans", "zh"},
		{"de", "en"}, // unsupported
		{"", "en"},
		{"  en  ", "en"},
		{"fr", "en"},
	}
	for _, tc := range cases {
		if got := normalizeLocale(tc.in); got != tc.want {
			t.Errorf("normalizeLocale(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestLocaleMiddleware_QueryParamWins confirms that the "?lang=zh"
// query parameter takes precedence over the Accept-Language header.
// This is important for users who want to override their browser
// preference (e.g. for QA or for users in multi-locale deployments).
func TestLocaleMiddleware_QueryParamWins(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("GET", "/?lang=zh", nil)
	c.Request.Header.Set("Accept-Language", "en-US,en;q=0.9")

	LocaleMiddleware()(c)

	if got := GetLocaleFromContext(c); got != "zh" {
		t.Errorf("query lang should win, got %q", got)
	}
}

// TestLocaleMiddleware_AcceptLanguageFallback confirms that without
// a query parameter, the Accept-Language header is used.
func TestLocaleMiddleware_AcceptLanguageFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")

	LocaleMiddleware()(c)

	if got := GetLocaleFromContext(c); got != "zh" {
		t.Errorf("Accept-Language should set locale to zh, got %q", got)
	}
}

// TestLocaleMiddleware_DefaultsToEnglish confirms that without any
// locale signal at all, we default to English.
func TestLocaleMiddleware_DefaultsToEnglish(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("GET", "/", nil)

	LocaleMiddleware()(c)

	if got := GetLocaleFromContext(c); got != "en" {
		t.Errorf("default locale should be en, got %q", got)
	}
}

// TestLocaleMiddleware_UnsupportedLocaleFallsBackToEnglish confirms that
// a header like "de-DE" (unsupported language) is treated as English
// rather than causing an error or empty string.
func TestLocaleMiddleware_UnsupportedLocaleFallsBackToEnglish(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Accept-Language", "de-DE,de;q=0.9")

	LocaleMiddleware()(c)

	if got := GetLocaleFromContext(c); got != "en" {
		t.Errorf("unsupported locale should fall back to en, got %q", got)
	}
}

// TestGetLocaleFromContext_NoValueInContext confirms the safe-default
// behaviour: if the middleware never ran (e.g. in a unit test), we
// still get a usable locale.
func TestGetLocaleFromContext_NoValueInContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	if got := GetLocaleFromContext(c); got != "en" {
		t.Errorf("missing context value should default to en, got %q", got)
	}
}
