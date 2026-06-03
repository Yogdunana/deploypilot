package auth

import (
	"testing"
)

func TestIsValidScope(t *testing.T) {
	tests := []struct {
		name     string
		scope    string
		expected bool
	}{
		{"valid scope read", "read", true},
		{"valid scope write", "write", true},
		{"valid scope delete", "delete", true},
		{"valid scope deploy", "deploy", true},
		{"valid scope admin", "admin", true},
		{"valid scope monitor:read", "monitor:read", true},
		{"valid scope monitor:write", "monitor:write", true},
		{"valid scope server:read", "server:read", true},
		{"valid scope server:exec", "server:exec", true},
		{"valid scope credential:read", "credential:read", true},
		{"valid scope credential:write", "credential:write", true},
		{"invalid scope", "invalid", false},
		{"empty scope", "", false},
		{"scope with space", "read write", false},
		{"scope with slash", "read/write", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidScope(tt.scope)
			if result != tt.expected {
				t.Errorf("IsValidScope(%q) = %v, want %v", tt.scope, result, tt.expected)
			}
		})
	}
}

func TestValidateScopes(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{"all valid", []string{"read", "write", "admin"}, []string{"read", "write", "admin"}},
		{"mixed valid and invalid", []string{"read", "invalid", "write", "bad_scope"}, []string{"read", "write"}},
		{"all invalid", []string{"invalid", "bad_scope"}, []string{}},
		{"empty input", []string{}, []string{}},
		{"nil input", nil, nil},
		{"duplicate scopes", []string{"read", "read", "write"}, []string{"read", "read", "write"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateScopes(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("ValidateScopes(%v) = %v (len=%d), want %v (len=%d)",
					tt.input, result, len(result), tt.expected, len(tt.expected))
				return
			}
			for i, s := range result {
				if s != tt.expected[i] {
					t.Errorf("ValidateScopes(%v)[%d] = %q, want %q", tt.input, i, s, tt.expected[i])
				}
			}
		})
	}
}

func TestAllScopes_ContainsAll(t *testing.T) {
	descriptions := ScopeDescriptions
	for scope := range descriptions {
		found := false
		for _, s := range AllScopes {
			if s == scope {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("AllScopes missing scope %q that exists in ScopeDescriptions", scope)
		}
	}
}

func TestScopeDescriptions_HasAllScopes(t *testing.T) {
	for _, scope := range AllScopes {
		if _, exists := ScopeDescriptions[scope]; !exists {
			t.Errorf("ScopeDescriptions missing description for scope %q", scope)
		}
	}
}

func TestScopeDescriptions_NotEmpty(t *testing.T) {
	for scope, desc := range ScopeDescriptions {
		if desc == "" {
			t.Errorf("Scope %q has empty description", scope)
		}
	}
}