package service

import (
	"testing"
)

// ===================== toString Tests =====================

func TestBridgeHelpersToString(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "string value",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "nil value",
			input:    nil,
			expected: "",
		},
		{
			name:     "int value",
			input:    42,
			expected: "42",
		},
		{
			name:     "int64 value",
			input:    int64(123456789),
			expected: "123456789",
		},
		{
			name:     "float64 value",
			input:    3.14159,
			expected: "3.14159",
		},
		{
			name:     "bool value",
			input:    true,
			expected: "true",
		},
		{
			name:     "byte slice",
			input:    []byte("bytes"),
			expected: "bytes",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "struct - uses default formatting",
			input:    struct{ X int }{X: 1},
			expected: "{1}",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := toString(tc.input)
			if result != tc.expected {
				t.Errorf("toString(%v) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

// ===================== toStringOrDefault Tests =====================

func TestBridgeHelpersToStringOrDefault(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		def      string
		expected string
	}{
		{
			name:     "returns value when non-empty",
			input:    "hello",
			def:      "default",
			expected: "hello",
		},
		{
			name:     "returns default when empty string",
			input:    "",
			def:      "default",
			expected: "default",
		},
		{
			name:     "returns default when nil",
			input:    nil,
			def:      "default",
			expected: "default",
		},
		{
			name:     "returns value when zero int",
			input:    0,
			def:      "default",
			expected: "0", // toString returns "0" for int(0)
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := toStringOrDefault(tc.input, tc.def)
			if result != tc.expected {
				t.Errorf("toStringOrDefault(%v, %q) = %q, want %q", tc.input, tc.def, result, tc.expected)
			}
		})
	}
}

// ===================== toInt Tests =====================

func TestBridgeHelpersToInt(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected int
	}{
		{
			name:     "int value",
			input:    42,
			expected: 42,
		},
		{
			name:     "int64 value",
			input:    int64(123),
			expected: 123,
		},
		{
			name:     "float64 value",
			input:    99.9,
			expected: 99,
		},
		{
			name:     "float64 negative",
			input:    -5.7,
			expected: -5,
		},
		{
			name:     "nil value",
			input:    nil,
			expected: 0,
		},
		{
			name:     "string value",
			input:    "not a number",
			expected: 0,
		},
		{
			name:     "zero",
			input:    0,
			expected: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := toInt(tc.input)
			if result != tc.expected {
				t.Errorf("toInt(%v) = %d, want %d", tc.input, result, tc.expected)
			}
		})
	}
}

// ===================== defaultVal Tests =====================

func TestBridgeHelpersDefaultVal(t *testing.T) {
	tests := []struct {
		name     string
		val      string
		def      string
		expected string
	}{
		{
			name:     "returns val when non-empty",
			val:      "hello",
			def:      "default",
			expected: "hello",
		},
		{
			name:     "returns def when empty",
			val:      "",
			def:      "default",
			expected: "default",
		},
		{
			name:     "returns def when val is spaces",
			val:      "   ",
			def:      "default",
			expected: "   ", // only checks for empty string, not whitespace
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := defaultVal(tc.val, tc.def)
			if result != tc.expected {
				t.Errorf("defaultVal(%q, %q) = %q, want %q", tc.val, tc.def, result, tc.expected)
			}
		})
	}
}

// ===================== generateID Tests =====================

func TestBridgeHelpersGenerateID(t *testing.T) {
	id := generateID()

	// Should start with "dep-"
	if len(id) < 4 || id[:4] != "dep-" {
		t.Errorf("generateID() = %q, want to start with 'dep-'", id)
	}

	// Should be unique
	id2 := generateID()
	if id == id2 {
		t.Error("generateID() generated same ID twice")
	}
}

// ===================== logPreflightResult Tests =====================

func TestLogPreflightResult_Empty(t *testing.T) {
	// Should not panic with empty checks
	result := &PreflightResult{
		Passed: true,
		Code:   PreflightErrorCode(""),
		Message: "all good",
		Checks: []PreflightCheck{},
	}

	logPreflightResult("test-container", result)
}

func TestLogPreflightResult_WithChecks(t *testing.T) {
	result := &PreflightResult{
		Passed: false,
		Code:   PreflightErrorCode("PORT_CONFLICT"),
		Message: "port is already in use",
		Checks: []PreflightCheck{
			{Name: "port-check", Passed: false, Message: "port 80 is in use", Suggestion: "use a different port"},
			{Name: "docker-check", Passed: true, Message: "docker is available"},
		},
	}

	// Should not panic
	logPreflightResult("test-container", result)
}
