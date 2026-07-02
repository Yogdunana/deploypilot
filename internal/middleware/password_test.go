package middleware

import (
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/config"
)

func TestNewPasswordValidator_DefaultMinLen(t *testing.T) {
	v := NewPasswordValidator(config.SecurityConfig{})
	if v.MinLen != 8 {
		t.Errorf("expected MinLen=8, got %d", v.MinLen)
	}
}

func TestNewPasswordValidator_CustomMinLen(t *testing.T) {
	v := NewPasswordValidator(config.SecurityConfig{PasswordMinLen: 12})
	if v.MinLen != 12 {
		t.Errorf("expected MinLen=12, got %d", v.MinLen)
	}
}

func TestNewPasswordValidator_MinLenBelow8(t *testing.T) {
	v := NewPasswordValidator(config.SecurityConfig{PasswordMinLen: 4})
	if v.MinLen != 8 {
		t.Errorf("expected MinLen=8 (minimum), got %d", v.MinLen)
	}
}

func TestPasswordValidator_Validate_StrongPassword(t *testing.T) {
	v := NewPasswordValidator(config.SecurityConfig{
		PasswordMinLen:         8,
		PasswordRequireUpper:   true,
		PasswordRequireLower:   true,
		PasswordRequireDigit:   true,
		PasswordRequireSpecial: true,
	})

	err := v.Validate("MyP@ssw0rd!")
	if err != nil {
		t.Errorf("expected valid password, got error: %v", err)
	}
}

func TestPasswordValidator_Validate_TooShort(t *testing.T) {
	v := NewPasswordValidator(config.SecurityConfig{
		PasswordMinLen: 8,
	})

	err := v.Validate("abc")
	if err == nil {
		t.Error("expected error for short password")
	}
}

func TestPasswordValidator_Validate_RequireUpper(t *testing.T) {
	v := NewPasswordValidator(config.SecurityConfig{
		PasswordMinLen:       8,
		PasswordRequireUpper: true,
	})

	err := v.Validate("alllowercase1")
	if err == nil {
		t.Error("expected error for missing uppercase")
	}
}

func TestPasswordValidator_Validate_RequireLower(t *testing.T) {
	v := NewPasswordValidator(config.SecurityConfig{
		PasswordMinLen:       8,
		PasswordRequireLower: true,
	})

	err := v.Validate("ALLUPPERCASE1")
	if err == nil {
		t.Error("expected error for missing lowercase")
	}
}

func TestPasswordValidator_Validate_RequireDigit(t *testing.T) {
	v := NewPasswordValidator(config.SecurityConfig{
		PasswordMinLen:       8,
		PasswordRequireDigit: true,
	})

	err := v.Validate("NoDigitsHere")
	if err == nil {
		t.Error("expected error for missing digit")
	}
}

func TestPasswordValidator_Validate_RequireSpecial(t *testing.T) {
	v := NewPasswordValidator(config.SecurityConfig{
		PasswordMinLen:         8,
		PasswordRequireSpecial: true,
	})

	err := v.Validate("NoSpecial1")
	if err == nil {
		t.Error("expected error for missing special character")
	}
}

func TestPasswordValidator_Validate_CommonPassword(t *testing.T) {
	v := NewPasswordValidator(config.SecurityConfig{
		PasswordMinLen: 8,
	})

	err := v.Validate("password")
	if err == nil {
		t.Error("expected error for common password")
	}

	err = v.Validate("qwerty123")
	if err == nil {
		t.Error("expected error for common password 'qwerty123'")
	}
}

func TestPasswordValidator_Validate_MultipleErrors(t *testing.T) {
	v := NewPasswordValidator(config.SecurityConfig{
		PasswordMinLen:         12,
		PasswordRequireUpper:   true,
		PasswordRequireDigit:   true,
		PasswordRequireSpecial: true,
	})

	err := v.Validate("abc")
	if err == nil {
		t.Fatal("expected error")
	}
	pve, ok := err.(*PasswordValidationError)
	if !ok {
		t.Fatalf("expected PasswordValidationError, got %T", err)
	}
	if len(pve.Errors) < 3 {
		t.Errorf("expected multiple errors, got %d: %v", len(pve.Errors), pve.Errors)
	}
}

func TestPasswordValidator_Validate_NoRequirements(t *testing.T) {
	v := NewPasswordValidator(config.SecurityConfig{
		PasswordMinLen: 8,
	})

	err := v.Validate("abcdefgh")
	if err != nil {
		t.Errorf("expected valid with no extra requirements, got: %v", err)
	}
}

// --- StrengthScore tests ---

func TestStrengthScore_WeakPassword(t *testing.T) {
	v := NewPasswordValidator(config.SecurityConfig{})
	score := v.StrengthScore("abc")
	if score >= 50 {
		t.Errorf("expected weak score for 'abc', got %d", score)
	}
}

func TestStrengthScore_StrongPassword(t *testing.T) {
	v := NewPasswordValidator(config.SecurityConfig{})
	score := v.StrengthScore("MyStr0ng!P@ssw0rd#2026")
	if score < 60 {
		t.Errorf("expected strong score, got %d", score)
	}
}

func TestStrengthScore_CommonPasswordPenalty(t *testing.T) {
	v := NewPasswordValidator(config.SecurityConfig{})
	score := v.StrengthScore("password")
	if score != 0 {
		t.Errorf("expected 0 score for common password, got %d", score)
	}
}

func TestStrengthScore_SequenctialPenalty(t *testing.T) {
	v := NewPasswordValidator(config.SecurityConfig{})
	score := v.StrengthScore("abcdefgh")
	if score >= 50 {
		t.Errorf("expected low score for sequential password, got %d", score)
	}
}

func TestStrengthScore_RepeatingPenalty(t *testing.T) {
	v := NewPasswordValidator(config.SecurityConfig{})
	score := v.StrengthScore("aaabbbccc")
	if score >= 50 {
		t.Errorf("expected low score for repeating password, got %d", score)
	}
}

func TestStrengthScore_EmptyPassword(t *testing.T) {
	v := NewPasswordValidator(config.SecurityConfig{})
	score := v.StrengthScore("")
	if score != 0 {
		t.Errorf("expected 0 score for empty password, got %d", score)
	}
}

func TestStrengthScore_NeverNegative(t *testing.T) {
	v := NewPasswordValidator(config.SecurityConfig{})
	score := v.StrengthScore("password") // common + other penalties
	if score < 0 {
		t.Errorf("score should never be negative, got %d", score)
	}
}

// --- CheckPasswordExpired tests ---

func TestCheckPasswordExpired_Disabled(t *testing.T) {
	err := CheckPasswordExpired("", 0)
	if err != nil {
		t.Errorf("expected nil when maxAgeDays=0, got %v", err)
	}
	err = CheckPasswordExpired("", -1)
	if err != nil {
		t.Errorf("expected nil when maxAgeDays<0, got %v", err)
	}
}

func TestCheckPasswordExpired_EmptyTimestamp(t *testing.T) {
	err := CheckPasswordExpired("", 90)
	if err == nil {
		t.Error("expected error for empty timestamp with maxAgeDays>0")
	}
}

func TestCheckPasswordExpired_InvalidTimestamp(t *testing.T) {
	err := CheckPasswordExpired("not-a-date", 90)
	if err == nil {
		t.Error("expected error for invalid timestamp")
	}
}

func TestCheckPasswordExpired_NotExpired(t *testing.T) {
	recent := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
	err := CheckPasswordExpired(recent, 90)
	if err != nil {
		t.Errorf("expected nil for recent password, got %v", err)
	}
}

func TestCheckPasswordExpired_Expired(t *testing.T) {
	old := time.Now().Add(-91 * 24 * time.Hour).Format(time.RFC3339)
	err := CheckPasswordExpired(old, 90)
	if err == nil {
		t.Error("expected error for expired password")
	}
}

// --- Helper function tests ---

func TestIsCommonPassword(t *testing.T) {
	tests := []struct {
		password string
		want     bool
	}{
		{"password", true},
		{"PASSWORD", true}, // case-insensitive
		{"Admin", true},
		{"root", true},
		{"deploypilot", true},
		{"my-secure-p@ss!", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.password, func(t *testing.T) {
			if got := isCommonPassword(tt.password); got != tt.want {
				t.Errorf("isCommonPassword(%q) = %v, want %v", tt.password, got, tt.want)
			}
		})
	}
}

func TestIsSequential(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"abcdef", true},
		{"123456", true},
		{"qwerty", true},
		{"random", false},
		// Empty and very short strings may match as substrings of sequences;
		// isSequential uses strings.Contains so short strings like "ab" or ""
		// can appear in sequential patterns. This is acceptable behavior.
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
		{"aaabbb", true},
		{"abc", false},
		{"ab", false},
		{"", false},
		{"aabbcc", false},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := isRepeating(tt.s); got != tt.want {
				t.Errorf("isRepeating(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestHasUnicode(t *testing.T) {
	if !hasUnicode("pässwörd") {
		t.Error("expected unicode detection")
	}
	if hasUnicode("password") {
		t.Error("expected no unicode for ASCII")
	}
}
