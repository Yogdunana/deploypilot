package i18n

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const ContextLocaleKey = "locale"

// LocaleMiddleware extracts the locale from the Accept-Language header
// or the "lang" query parameter and stores it in the gin.Context.
// Falls back to "en" if not recognized.
func LocaleMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		locale := "en"

		// Query parameter takes highest priority
		if lang := c.Query("lang"); lang != "" {
			locale = normalizeLocale(lang)
		} else if accept := c.GetHeader("Accept-Language"); accept != "" {
			locale = parseAcceptLanguage(accept)
		}

		c.Set(ContextLocaleKey, locale)
		c.Next()
	}
}

// parseAcceptLanguage parses the Accept-Language header and returns
// the best matching supported locale.
func parseAcceptLanguage(accept string) string {
	// Simple parsing: take the first language tag
	// e.g., "zh-CN,zh;q=0.9,en;q=0.8" -> "zh-CN"
	parts := strings.Split(accept, ",")
	if len(parts) == 0 {
		return "en"
	}
	lang := strings.TrimSpace(strings.Split(parts[0], ";")[0])
	return normalizeLocale(lang)
}

// normalizeLocale maps language tags to supported locales.
func normalizeLocale(lang string) string {
	lang = strings.TrimSpace(strings.ToLower(lang))
	switch {
	case strings.HasPrefix(lang, "zh"):
		return "zh"
	case strings.HasPrefix(lang, "en"):
		return "en"
	default:
		return "en"
	}
}

// GetLocaleFromContext returns the locale stored in the gin.Context.
func GetLocaleFromContext(c *gin.Context) string {
	if val, exists := c.Get(ContextLocaleKey); exists {
		if locale, ok := val.(string); ok {
			return locale
		}
	}
	return "en"
}
