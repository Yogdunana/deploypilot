package i18n

import (
	"testing"
)

func TestT_English(t *testing.T) {
	result := T("en", "error.auth.insufficient_permissions")
	if result == "error.auth.insufficient_permissions" {
		t.Error("expected translated message, got key")
	}
}

func TestT_Chinese(t *testing.T) {
	result := T("zh", "error.auth.insufficient_permissions")
	if result == "error.auth.insufficient_permissions" {
		t.Error("expected translated message, got key")
	}
}

func TestT_FallbackToDefault(t *testing.T) {
	result := T("nonexistent", "error.auth.insufficient_permissions")
	if result == "error.auth.insufficient_permissions" {
		t.Error("expected fallback to default locale, got key")
	}
}

func TestT_UnknownKey(t *testing.T) {
	result := T("en", "unknown.key.that.does.not.exist")
	if result != "unknown.key.that.does.not.exist" {
		t.Errorf("expected key as fallback, got %q", result)
	}
}

func TestT_EmptyKey(t *testing.T) {
	result := T("en", "")
	if result != "" {
		t.Errorf("expected empty string for empty key, got %q", result)
	}
}

func TestT_EmptyLocale(t *testing.T) {
	result := T("", "error.auth.insufficient_permissions")
	if result == "error.auth.insufficient_permissions" {
		t.Error("expected fallback to default locale for empty locale")
	}
}

func TestTf_Formatting(t *testing.T) {
	result := Tf("en", "%s %d", "hello", 42)
	if result != "hello 42" {
		t.Errorf("expected formatted string, got %q", result)
	}
}

func TestTf_UnknownKeyWithFormat(t *testing.T) {
	result := Tf("en", "unknown.key %s", "test")
	expected := "unknown.key test"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestGetLocale(t *testing.T) {
	locale := GetLocale("en")
	if locale == nil {
		t.Error("expected non-nil locale map for 'en'")
	}
	if len(locale) == 0 {
		t.Error("expected non-empty locale map")
	}
}

func TestGetLocale_Nonexistent(t *testing.T) {
	locale := GetLocale("nonexistent")
	if locale == nil {
		t.Error("expected fallback to default locale for nonexistent locale")
	}
}

func TestGetLocale_Empty(t *testing.T) {
	locale := GetLocale("")
	if locale == nil {
		t.Error("expected default locale for empty locale")
	}
}

func TestSetDefaultLocale(t *testing.T) {
	original := defaultLocale
	defer func() {
		SetDefaultLocale(original)
	}()

	SetDefaultLocale("zh")
	if defaultLocale != "zh" {
		t.Errorf("expected default locale 'zh', got %q", defaultLocale)
	}
}

func TestSetDefaultLocale_Invalid(t *testing.T) {
	original := defaultLocale
	defer func() {
		SetDefaultLocale(original)
	}()

	SetDefaultLocale("nonexistent")
	result := T("en", "error.auth.insufficient_permissions")
	if result == "error.auth.insufficient_permissions" {
		t.Error("expected translation for valid locale even when default is invalid")
	}
}