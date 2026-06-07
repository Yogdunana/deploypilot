package i18n

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// newContextWithRequest builds a minimal gin.Context whose Request carries
// the provided Accept-Language header and lang query string.
func newContextWithRequest(t *testing.T, acceptLang, queryLang string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/", nil)
	if acceptLang != "" {
		req.Header.Set("Accept-Language", acceptLang)
	}
	if queryLang != "" {
		q := req.URL.Query()
		q.Set("lang", queryLang)
		req.URL.RawQuery = q.Encode()
	}
	c.Request = req
	return c, w
}

func runMiddleware(t *testing.T, acceptLang, queryLang string) string {
	t.Helper()
	c, _ := newContextWithRequest(t, acceptLang, queryLang)
	LocaleMiddleware()(c)
	return GetLocaleFromContext(c)
}

func TestLocaleMiddleware_DefaultWhenNoSignal(t *testing.T) {
	got := runMiddleware(t, "", "")
	if got != "en" {
		t.Errorf("default locale = %q, want en", got)
	}
}

func TestLocaleMiddleware_QueryParamWinsOverHeader(t *testing.T) {
	// ?lang=zh must take priority even when Accept-Language is en.
	got := runMiddleware(t, "en", "zh")
	if got != "zh" {
		t.Errorf("locale = %q, want zh (query param must win)", got)
	}
}

func TestLocaleMiddleware_AcceptLanguageEnglish(t *testing.T) {
	got := runMiddleware(t, "en-US,en;q=0.9", "")
	if got != "en" {
		t.Errorf("locale = %q, want en", got)
	}
}

func TestLocaleMiddleware_AcceptLanguageChinese(t *testing.T) {
	got := runMiddleware(t, "zh-CN,zh;q=0.9,en;q=0.8", "")
	if got != "zh" {
		t.Errorf("locale = %q, want zh", got)
	}
}

func TestLocaleMiddleware_AcceptLanguageUnknownFallsBackToEnglish(t *testing.T) {
	got := runMiddleware(t, "fr-FR,fr;q=0.9", "")
	if got != "en" {
		t.Errorf("locale = %q, want en (unsupported locale falls back to en)", got)
	}
}

func TestLocaleMiddleware_StripsQualityFactor(t *testing.T) {
	// Quality factor after ; should be ignored when picking the first language.
	got := runMiddleware(t, "zh;q=0.7", "")
	if got != "zh" {
		t.Errorf("locale = %q, want zh (quality factor should be ignored)", got)
	}
}

func TestLocaleMiddleware_CaseInsensitive(t *testing.T) {
	got := runMiddleware(t, "ZH-cn", "")
	if got != "zh" {
		t.Errorf("locale = %q, want zh (case insensitive)", got)
	}
}

func TestLocaleMiddleware_EmptyHeaderTreatedAsDefault(t *testing.T) {
	// An empty Accept-Language header should not crash and should default to en.
	got := runMiddleware(t, "", "")
	if got != "en" {
		t.Errorf("locale = %q, want en", got)
	}
}

func TestLocaleMiddleware_StoresInGinContext(t *testing.T) {
	c, _ := newContextWithRequest(t, "zh-CN", "")
	LocaleMiddleware()(c)

	val, exists := c.Get(ContextLocaleKey)
	if !exists {
		t.Fatal("locale not stored in gin context")
	}
	if s, ok := val.(string); !ok || s != "zh" {
		t.Errorf("context locale = %v, want \"zh\"", val)
	}
}

func TestGetLocaleFromContext_DefaultWhenMissing(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)

	if got := GetLocaleFromContext(c); got != "en" {
		t.Errorf("GetLocaleFromContext with no value = %q, want en", got)
	}
}

func TestGetLocaleFromContext_DefaultOnWrongType(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Set(ContextLocaleKey, 42) // wrong type

	if got := GetLocaleFromContext(c); got != "en" {
		t.Errorf("GetLocaleFromContext with wrong type = %q, want en", got)
	}
}

func TestParseAcceptLanguage_FirstTagWins(t *testing.T) {
	// The current implementation only respects the first tag, ignoring q-values.
	// This test pins down the documented behavior.
	got := parseAcceptLanguage("zh-CN,en;q=0.9")
	if got != "zh" {
		t.Errorf("parseAcceptLanguage = %q, want zh", got)
	}
}

func TestParseAcceptLanguage_StripsQualityFactorInline(t *testing.T) {
	got := parseAcceptLanguage("en;q=0.5")
	if got != "en" {
		t.Errorf("parseAcceptLanguage = %q, want en", got)
	}
}

func TestParseAcceptLanguage_EmptyFallsBackToEnglish(t *testing.T) {
	got := parseAcceptLanguage("")
	if got != "en" {
		t.Errorf("parseAcceptLanguage(\"\") = %q, want en", got)
	}
}

func TestNormalizeLocale_AllBranches(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"en", "en"},
		{"EN", "en"},
		{"en-US", "en"},
		{"en-GB", "en"},
		{"zh", "zh"},
		{"ZH", "zh"},
		{"zh-CN", "zh"},
		{"zh-TW", "zh"},
		{"fr", "en"},
		{"ja", "en"},
		{"de", "en"},
		{"", "en"},
		{"  en  ", "en"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			if got := normalizeLocale(c.in); got != c.want {
				t.Errorf("normalizeLocale(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestLocaleMiddleware_QueryWithUnknownValueFallsBackToEn(t *testing.T) {
	// An unknown language in ?lang= must not crash and must default to en.
	got := runMiddleware(t, "", "fr")
	if got != "en" {
		t.Errorf("locale = %q, want en (unsupported query value falls back)", got)
	}
}

func TestLocaleMiddleware_QueryWithMixedCase(t *testing.T) {
	got := runMiddleware(t, "", "ZH")
	if got != "zh" {
		t.Errorf("locale = %q, want zh (case insensitive normalization)", got)
	}
}
