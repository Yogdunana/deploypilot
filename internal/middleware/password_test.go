package middleware

import (
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/config"
)

func TestNewPasswordValidator(t *testing.T) {
	tests := []struct {
		name   string
		cfg    config.SecurityConfig
		want   PasswordValidator
	}{
		{
			name: "default minimum length",
			cfg:  config.SecurityConfig{PasswordMinLen: 0},
			want: PasswordValidator{MinLen: 8},
		},
		{
			name: "configured minimum length",
			cfg:  config.SecurityConfig{PasswordMinLen: 12},
			want: PasswordValidator{MinLen: 12},
		},
		{
			name: "too small minimum length",
			cfg:  config.SecurityConfig{PasswordMinLen: 4},
			want: PasswordValidator{MinLen: 8},
		},
		{
			name: "all requirements enabled",
			cfg: config.SecurityConfig{
				PasswordMinLen:         10,
				PasswordRequireUpper:   true,
				PasswordRequireLower:   true,
				PasswordRequireDigit:   true,
				PasswordRequireSpecial: true,
			},
			want: PasswordValidator{
				MinLen:         10,
				RequireUpper:   true,
				RequireLower:   true,
				RequireDigit:   true,
				RequireSpecial: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewPasswordValidator(tt.cfg)
			if got.MinLen != tt.want.MinLen {
				t.Errorf("MinLen = %d, want %d", got.MinLen, tt.want.MinLen)
			}
			if got.RequireUpper != tt.want.RequireUpper {
				t.Errorf("RequireUpper = %v, want %v", got.RequireUpper, tt.want.RequireUpper)
			}
			if got.RequireLower != tt.want.RequireLower {
				t.Errorf("RequireLower = %v, want %v", got.RequireLower, tt.want.RequireLower)
			}
			if got.RequireDigit != tt.want.RequireDigit {
				t.Errorf("RequireDigit = %v, want %v", got.RequireDigit, tt.want.RequireDigit)
			}
			if got.RequireSpecial != tt.want.RequireSpecial {
				t.Errorf("RequireSpecial = %v, want %v", got.RequireSpecial, tt.want.RequireSpecial)
			}
		})
	}
}

func TestPasswordValidator_Validate(t *testing.T) {
	v := NewPasswordValidator(config.SecurityConfig{
		PasswordMinLen:         8,
		PasswordRequireUpper:   true,
		PasswordRequireLower:   true,
		PasswordRequireDigit:   true,
		PasswordRequireSpecial: true,
	})

	tests := []struct {
		name     string
		password string
		wantErr  bool
		errCount int
	}{
		{
			name:     "valid password",
			password: "StrongPass123!",
			wantErr:  false,
		},
		{
			name:     "too short",
			password: "Short1!",
			wantErr:  true,
			errCount: 1,
		},
		{
			name:     "no uppercase",
			password: "lowercase123!",
			wantErr:  true,
			errCount: 1,
		},
		{
			name:     "no lowercase",
			password: "UPPERCASE123!",
			wantErr:  true,
			errCount: 1,
		},
		{
			name:     "no digit",
			password: "NoDigits!",
			wantErr:  true,
			errCount: 1,
		},
		{
			name:     "no special",
			password: "NoSpecial123",
			wantErr:  true,
			errCount: 1,
		},
		{
			name:     "common password",
			password: "password",
			wantErr:  true,
			errCount: 5, // short + no upper + no digit + no special + common
		},
		{
			name:     "common password with requirements",
			password: "Password123!",
			wantErr:  false, // "Password123!" is not in commonPasswords list
		},
		{
			name:     "multiple errors",
			password: "short",
			wantErr:  true,
			errCount: 4, // short + no upper + no digit + no special
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  true,
			errCount: 5, // short + no upper + no lower + no digit + no special
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.password)
			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.wantErr {
				if pve, ok := err.(*PasswordValidationError); ok {
					if len(pve.Errors) != tt.errCount {
						t.Errorf("Error count = %d, want %d, errors: %v", len(pve.Errors), tt.errCount, pve.Errors)
					}
				} else {
					t.Errorf("Expected PasswordValidationError, got %T", err)
				}
			}
		})
	}
}

func TestPasswordValidator_ValidateNoRequirements(t *testing.T) {
	// Validator with only minimum length requirement
	v := NewPasswordValidator(config.SecurityConfig{
		PasswordMinLen: 8,
	})

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"valid length", "abcdefgh", false},
		{"too short", "abc", true},
		{"valid with digits", "12345678", true}, // "12345678" is in commonPasswords
		{"valid with special", "!@#$%^&*", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.password)
			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestPasswordValidationError_Error(t *testing.T) {
	tests := []struct {
		errors []string
		want   string
	}{
		{[]string{"error1"}, "error1"},
		{[]string{"error1", "error2"}, "error1; error2"},
		{[]string{"a", "b", "c"}, "a; b; c"},
		{[]string{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			e := &PasswordValidationError{Errors: tt.errors}
			if got := e.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPasswordValidator_StrengthScore(t *testing.T) {
	v := NewPasswordValidator(config.SecurityConfig{})

	tests := []struct {
		name     string
		password string
		wantMin  int
		wantMax  int
	}{
		{"very weak", "a", 0, 20},
		{"weak", "abc", 0, 30},
		{"medium", "password12", 10, 50},
		{"good", "GoodPass123", 30, 60},
		{"strong", "StrongPass123!", 50, 80},
		{"very strong", "VeryStrongPass123!@#", 60, 100},
		{"sequential penalty", "abc123456", 0, 50},
		{"repeating penalty", "aaa111bbb", 0, 50},
		{"common password penalty", "password", 0, 30},
		{"unicode bonus", "中文密码Pass123!", 60, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := v.StrengthScore(tt.password)
			if score < tt.wantMin {
				t.Errorf("Score = %d, want at least %d", score, tt.wantMin)
			}
			if score > tt.wantMax {
				t.Errorf("Score = %d, want at most %d", score, tt.wantMax)
			}
		})
	}
}

func TestHasUpper(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"ABC", true},
		{"abc", false},
		{"Abc", true},
		{"aBc", true},
		{"", false},
		{"123", false},
		{"!@#", false},
		{"中文", false},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := hasUpper(tt.s); got != tt.want {
				t.Errorf("hasUpper(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestHasLower(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"abc", true},
		{"ABC", false},
		{"Abc", true},
		{"ABc", true},
		{"", false},
		{"123", false},
		{"!@#", false},
		{"中文", false},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := hasLower(tt.s); got != tt.want {
				t.Errorf("hasLower(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestHasDigit(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"123", true},
		{"abc", false},
		{"abc123", true},
		{"", false},
		{"!@#", false},
		{"中文123", true},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := hasDigit(tt.s); got != tt.want {
				t.Errorf("hasDigit(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestHasSpecial(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"!@#", true},
		{"abc", false},
		{"abc!", true},
		{"", false},
		{"123", false},
		{"中文", false},
		{"abc$", true},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := hasSpecial(tt.s); got != tt.want {
				t.Errorf("hasSpecial(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestHasUnicode(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"中文", true},
		{"abc", false},
		{"abc中文", true},
		{"", false},
		{"123", false},
		{"!@#", false},
		{"日本語", true},
		{"emoji🎉", true},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := hasUnicode(tt.s); got != tt.want {
				t.Errorf("hasUnicode(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestIsSequential(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"abc", true},
		{"xyz", true},
		{"123", true},
		{"qwerty", true},
		{"asdfgh", true},
		{"zxcvbn", true},
		{"cba", true}, // reverse
		{"zyx", true}, // reverse
		{"random", false},
		{"", false},
		{"a", false},
		{"ab", false},
		{"ABC", true}, // case insensitive
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := isSequential(tt.s); got != tt.want {
				t.Errorf("isSequential(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestIsRepeating(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"aaa", true},
		{"111", true},
		{"abc", false},
		{"aab", false},
		{"", false},
		{"a", false},
		{"aa", false},
		{"aaaa", true},
		{"aaabbb", true},
		// Note: Unicode repeating check may not work for multi-byte chars
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := isRepeating(tt.s); got != tt.want {
				t.Errorf("isRepeating(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestIsCommonPassword(t *testing.T) {
	common := []string{
		"password", "123456", "12345678", "qwerty",
		"abc123", "monkey", "master", "dragon",
		"login", "princess", "football", "shadow",
		"sunshine", "trustno1", "iloveyou", "batman",
		"access", "hello", "charlie", "donald",
		"123456789", "1234567890", "password1", "qwerty123",
		"admin", "root", "toor", "pass", "test",
		"deploypilot", "deploy", "pilot",
	}

	for _, pw := range common {
		t.Run(pw, func(t *testing.T) {
			if !isCommonPassword(pw) {
				t.Errorf("isCommonPassword(%q) = false, want true", pw)
			}
			// Case insensitive
			if !isCommonPassword("PASSWORD") {
				t.Errorf("isCommonPassword should be case insensitive")
			}
		})
	}

	// Non-common passwords
	nonCommon := []string{"uniquePassword123!", "randomPass", ""}
	for _, pw := range nonCommon {
		t.Run("non-common_"+pw, func(t *testing.T) {
			if isCommonPassword(pw) {
				t.Errorf("isCommonPassword(%q) = true, want false", pw)
			}
		})
	}
}

func TestReverseString(t *testing.T) {
	tests := []struct {
		s    string
		want string
	}{
		{"abc", "cba"},
		{"", ""},
		{"a", "a"},
		{"12345", "54321"},
		{"中文", "文中"},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := reverseString(tt.s); got != tt.want {
				t.Errorf("reverseString(%q) = %q, want %q", tt.s, got, tt.want)
			}
		})
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{123, "123"},
		{999, "999"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := itoa(tt.n); got != tt.want {
				t.Errorf("itoa(%d) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}

func TestCheckPasswordExpired(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	tenDaysAgo := time.Now().Add(-10 * 24 * time.Hour).UTC().Format(time.RFC3339)
	thirtyDaysAgo := time.Now().Add(-30 * 24 * time.Hour).UTC().Format(time.RFC3339)

	tests := []struct {
		name           string
		changedAt      string
		maxAgeDays     int
		wantErr        bool
	}{
		{"no expiry", now, 0, false},
		{"negative max age", now, -1, false},
		{"recent change", tenDaysAgo, 30, false},
		{"expired", thirtyDaysAgo, 7, true},
		{"empty timestamp", "", 30, true},
		{"invalid timestamp", "invalid", 30, true},
		{"just expired", tenDaysAgo, 10, true},
		{"almost expired", tenDaysAgo, 11, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckPasswordExpired(tt.changedAt, tt.maxAgeDays)
			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.wantErr && err != ErrPasswordExpired {
				t.Errorf("Expected ErrPasswordExpired, got %v", err)
			}
		})
	}
}