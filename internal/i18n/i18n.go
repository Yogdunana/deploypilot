// Package i18n provides internationalization support for DeployPilot.
// It uses a simple map-based approach with embedded JSON locale files.
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"sync"
)

//go:embed locales/*.json
var localeFS embed.FS

var (
	translations map[string]map[string]string // locale -> key -> value
	mu           sync.RWMutex
	defaultLocale = "en"
)

func init() {
	translations = make(map[string]map[string]string)
	loadLocale("en")
	loadLocale("zh")
}

// loadLocale reads a locale JSON file from the embedded filesystem.
func loadLocale(locale string) {
	data, err := localeFS.ReadFile("locales/" + locale + ".json")
	if err != nil {
		return
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return
	}
	mu.Lock()
	translations[locale] = m
	mu.Unlock()
}

// SetDefaultLocale sets the default locale (defaults to "en").
func SetDefaultLocale(locale string) {
	mu.Lock()
	defaultLocale = locale
	mu.Unlock()
}

// GetLocale returns the translations map for a given locale.
// Falls back to the default locale if the requested locale is not found.
func GetLocale(locale string) map[string]string {
	mu.RLock()
	defer mu.RUnlock()
	if m, ok := translations[locale]; ok {
		return m
	}
	if m, ok := translations[defaultLocale]; ok {
		return m
	}
	return nil
}

// T translates a key using the given locale.
// If the key is not found, it returns the key itself.
func T(locale, key string) string {
	m := GetLocale(locale)
	if m == nil {
		return key
	}
	if v, ok := m[key]; ok {
		return v
	}
	// Fallback to default locale
	m = GetLocale(defaultLocale)
	if m == nil {
		return key
	}
	if v, ok := m[key]; ok {
		return v
	}
	return key
}

// Tf translates a key with format arguments using the given locale.
// It uses fmt.Sprintf internally. If the key is not found, it returns
// the key formatted with the provided arguments.
func Tf(locale, key string, args ...interface{}) string {
	template := T(locale, key)
	return fmt.Sprintf(template, args...)
}
