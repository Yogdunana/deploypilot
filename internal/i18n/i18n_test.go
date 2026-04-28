package i18n

import "testing"

func TestTReturnsKnownKey(t *testing.T) {
	result := T("en", "error.auth.authorization_header_required")
	if result == "" {
		t.Error("T() should return a non-empty translation for a known key")
	}
	if result == "error.auth.authorization_header_required" {
		t.Error("T() should return the translated value, not the key itself")
	}
}

func TestTReturnsKeyForUnknownKey(t *testing.T) {
	key := "nonexistent.key.that.does.not.exist"
	result := T("en", key)
	if result != key {
		t.Errorf("T() should return the key itself for unknown keys, got %q", result)
	}
}

func TestTFWithFormatArgs(t *testing.T) {
	result := Tf("en", "error.app.invalid_request", "test error")
	if result == "" {
		t.Error("Tf() should return a non-empty formatted translation")
	}
	// The template is "invalid request: %s", so it should contain the arg
	if result == "error.app.invalid_request" {
		t.Error("Tf() should return the formatted translation, not the key")
	}
}

func TestTChineseLocale(t *testing.T) {
	result := T("zh", "error.auth.authorization_header_required")
	if result == "" {
		t.Error("T() should return a non-empty translation for zh locale")
	}
}

func TestTFallbackToDefaultLocale(t *testing.T) {
	// Request a nonexistent locale; should fall back to "en"
	result := T("fr", "error.auth.authorization_header_required")
	if result == "" {
		t.Error("T() should fall back to default locale when requested locale is not found")
	}
}

func TestGetLocaleReturnsMap(t *testing.T) {
	m := GetLocale("en")
	if m == nil {
		t.Error("GetLocale('en') should return a non-nil map")
	}
	if len(m) == 0 {
		t.Error("GetLocale('en') should return a non-empty map")
	}
}

func TestSetDefaultLocale(t *testing.T) {
	original := "en"
	SetDefaultLocale("zh")
	defer SetDefaultLocale(original)

	// After changing default, unknown locale should fall back to zh
	result := T("nonexistent", "error.auth.authorization_header_required")
	if result == "" {
		t.Error("T() should fall back to zh locale after SetDefaultLocale('zh')")
	}
}
