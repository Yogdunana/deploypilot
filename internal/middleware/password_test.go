package middleware

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/config"
)

// ===================== NewPasswordValidator =====================

func TestNewPasswordValidator_ClampsMinLengthBelow8(t *testing.T) {
	v := NewPasswordValidator(config.SecurityConfig{PasswordMinLen: 3})
	if v.MinLen != 8 {
		t.Errorf("expected MinLen clamped to 8, got %d", v.MinLen)
	}
}

func TestNewPasswordValidator_PassesConfiguredValues(t *testing.T) {
	v := NewPasswordValidator(config.SecurityConfig{
		PasswordMinLen:         12,
		PasswordRequireUpper:   true,
		PasswordRequireLower:   true,
		PasswordRequireDigit:   true,
		PasswordRequireSpecial: true,
	})
	if v.MinLen != 12 {
		t.Errorf("expected MinLen=12, got %d", v.MinLen)
	}
	if !v.RequireUpper || !v.RequireLower || !v.RequireDigit || !v.RequireSpecial {
		t.Error("expected all character-class requirements to be enabled")
	}
}

// ===================== Validate =====================

func strictValidator() *PasswordValidator {
	return NewPasswordValidator(config.SecurityConfig{
		PasswordMinLen:         8,
		PasswordRequireUpper:   true,
		PasswordRequireLower:   true,
		PasswordRequireDigit:   true,
		PasswordRequireSpecial: true,
	})
}

func TestValidate_AcceptsCompliantPassword(t *testing.T) {
	if err := strictValidator().Validate("Good#Pass1"); err != nil {
		t.Errorf("expected nil for compliant password, got %v", err)
	}
}

func TestValidate_RejectsBelowMinLength(t *testing.T) {
	err := strictValidator().Validate("Aa1!")
	if err == nil {
		t.Fatal("expected error for short password")
	}
	var verr *PasswordValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected *PasswordValidationError, got %T", err)
	}
	if !contains(verr.Errors, "at least 8 characters") {
		t.Errorf("expected length error in %v", verr.Errors)
	}
}

func TestValidate_RejectsMissingUppercase(t *testing.T) {
	err := strictValidator().Validate("lower#1pass")
	var verr *PasswordValidationError
	if !errors.As(err, &verr) || !contains(verr.Errors, "uppercase") {
		t.Errorf("expected uppercase error, got %v", err)
	}
}

func TestValidate_RejectsMissingLowercase(t *testing.T) {
	err := strictValidator().Validate("UPPER#1PASS")
	var verr *PasswordValidationError
	if !errors.As(err, &verr) || !contains(verr.Errors, "lowercase") {
		t.Errorf("expected lowercase error, got %v", err)
	}
}

func TestValidate_RejectsMissingDigit(t *testing.T) {
	err := strictValidator().Validate("Upper#pass")
	var verr *PasswordValidationError
	if !errors.As(err, &verr) || !contains(verr.Errors, "digit") {
		t.Errorf("expected digit error, got %v", err)
	}
}

func TestValidate_RejectsMissingSpecial(t *testing.T) {
	err := strictValidator().Validate("Upperpass1")
	var verr *PasswordValidationError
	if !errors.As(err, &verr) || !contains(verr.Errors, "special") {
		t.Errorf("expected special character error, got %v", err)
	}
}

func TestValidate_RejectsCommonPasswords(t *testing.T) {
	// isCommonPassword checks whether the *full* lowered password matches an
	// entry in the weak list, so we use a minimal validator (no min-length,
	// no character-class requirements) to test the common-password rule in
	// isolation.
	v := NewPasswordValidator(config.SecurityConfig{
		PasswordMinLen: 1,
	})
	common := []string{
		"password", "123456", "qwerty", "abc123", "admin", "root",
		"deploypilot", "qwerty123", "password1", "1234567890",
		"monkey", "dragon", "iloveyou", "shadow", "sunshine",
		"trustno1", "PASSWORD", // case-insensitive
	}
	for _, p := range common {
		t.Run(p, func(t *testing.T) {
			if err := v.Validate(p); err == nil {
				t.Errorf("expected common-password rejection for %q", p)
			}
		})
	}
}

func TestValidate_AggregatesMultipleErrors(t *testing.T) {
	// "abc" fails on length, missing upper, missing digit, missing special, and
	// is not in the common-password list — the function should report each
	// failing rule in one shot.
	err := strictValidator().Validate("abc")
	var verr *PasswordValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected *PasswordValidationError, got %T", err)
	}
	if len(verr.Errors) < 3 {
		t.Errorf("expected multiple aggregated errors, got %v", verr.Errors)
	}
}

func TestValidate_SelectiveRequirements(t *testing.T) {
	// A validator that only requires a minimum length of 10 should accept
	// a long lowercase-only password.
	v := NewPasswordValidator(config.SecurityConfig{
		PasswordMinLen: 10,
	})
	if err := v.Validate("longpassword"); err != nil {
		t.Errorf("expected acceptance when only length is required, got %v", err)
	}
	// But a too-short password should still be rejected.
	if err := v.Validate("short"); err == nil {
		t.Error("expected rejection for too-short password")
	}
}

func TestPasswordValidationError_Message(t *testing.T) {
	e := &PasswordValidationError{Errors: []string{"a", "b", "c"}}
	if got := e.Error(); got != "a; b; c" {
		t.Errorf("Error() = %q, want %q", got, "a; b; c")
	}
}

// ===================== StrengthScore =====================

func TestStrengthScore_EmptyPassword(t *testing.T) {
	if score := strictValidator().StrengthScore(""); score != 0 {
		t.Errorf("expected score 0 for empty password, got %d", score)
	}
}

func TestStrengthScore_IncreasesWithLength(t *testing.T) {
	v := strictValidator()
	s8 := v.StrengthScore("Good#1")   // exactly 8 chars
	s12 := v.StrengthScore("Good#1aaaaaaa") // ~15 chars (>= 12 and >= 16 ranges)
	if s12 <= s8 {
		t.Errorf("expected longer password to score higher (8-char=%d, 15-char=%d)", s8, s12)
	}
}

func TestStrengthScore_PenalisesCommonPassword(t *testing.T) {
	v := strictValidator()
	good := v.StrengthScore("Good#Pass1!extra")
	bad := v.StrengthScore("password")
	if bad >= good {
		t.Errorf("common password should score lower than a strong one (good=%d, bad=%d)", good, bad)
	}
}

func TestStrengthScore_PenalisesRepeating(t *testing.T) {
	v := strictValidator()
	a := v.StrengthScore("Strong#9aaaaaaa") // contains "aaa"
	b := v.StrengthScore("Strong#9xyzwqjk")
	if a >= b {
		t.Errorf("repeating-character password should score lower (a=%d, b=%d)", a, b)
	}
}

func TestStrengthScore_PenalisesSequential(t *testing.T) {
	v := strictValidator()
	// isSequential uses strings.Contains(seq, lower), so the *entire* lowercased
	// password must be a substring of a known sequence. Plain alpha passwords
	// exercise the check.
	sequential := v.StrengthScore("abcdef")
	random := v.StrengthScore("zqxwjk")
	if sequential >= random {
		t.Errorf("sequential password should score lower (seq=%d, rand=%d)", sequential, random)
	}
}

func TestStrengthScore_NeverNegative(t *testing.T) {
	v := strictValidator()
	// Common+sequential+repeating all stack to < 0 if the math allowed it.
	if score := v.StrengthScore("password"); score < 0 {
		t.Errorf("score should be clamped to >= 0, got %d", score)
	}
}

// ===================== CheckPasswordExpired =====================

func TestCheckPasswordExpired_DisabledWhenMaxAgeIsZero(t *testing.T) {
	if err := CheckPasswordExpired("not-a-date", 0); err != nil {
		t.Errorf("expected nil for maxAge=0, got %v", err)
	}
	if err := CheckPasswordExpired("not-a-date", -5); err != nil {
		t.Errorf("expected nil for negative maxAge, got %v", err)
	}
}

func TestCheckPasswordExpired_EmptyTimestampTriggersExpiry(t *testing.T) {
	if err := CheckPasswordExpired("", 30); !errors.Is(err, ErrPasswordExpired) {
		t.Errorf("expected ErrPasswordExpired for empty timestamp, got %v", err)
	}
}

func TestCheckPasswordExpired_UnparseableTimestampTriggersExpiry(t *testing.T) {
	// An unparseable timestamp should be treated as expired for safety.
	if err := CheckPasswordExpired("not-a-real-date", 30); !errors.Is(err, ErrPasswordExpired) {
		t.Errorf("expected ErrPasswordExpired for unparseable timestamp, got %v", err)
	}
}

func TestCheckPasswordExpired_FreshPasswordNotExpired(t *testing.T) {
	ts := time.Now().Add(-2 * 24 * time.Hour).Format(time.RFC3339)
	if err := CheckPasswordExpired(ts, 30); err != nil {
		t.Errorf("expected nil for fresh password (2 days old), got %v", err)
	}
}

func TestCheckPasswordExpired_OldPasswordIsExpired(t *testing.T) {
	ts := time.Now().Add(-60 * 24 * time.Hour).Format(time.RFC3339)
	if err := CheckPasswordExpired(ts, 30); !errors.Is(err, ErrPasswordExpired) {
		t.Errorf("expected ErrPasswordExpired for 60-day-old password, got %v", err)
	}
}

func TestCheckPasswordExpired_BoundaryIsExclusive(t *testing.T) {
	// A password exactly at the age limit (30 days) should be considered
	// expired, because the check uses time.Since > maxAge.
	ts := time.Now().Add(-31 * 24 * time.Hour).Format(time.RFC3339)
	if err := CheckPasswordExpired(ts, 30); !errors.Is(err, ErrPasswordExpired) {
		t.Errorf("expected ErrPasswordExpired at the boundary, got %v", err)
	}
}

// ===================== helpers =====================

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}
