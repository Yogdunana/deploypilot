package middleware

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/config"
)

// strongPassword returns a password that satisfies a fully-loaded policy.
func strongPassword() string {
	return "Str0ng!Passw0rdXYZ"
}

func defaultValidatorCfg() config.SecurityConfig {
	return config.SecurityConfig{
		PasswordMinLen:        8,
		PasswordRequireUpper:  true,
		PasswordRequireLower:  true,
		PasswordRequireDigit:  true,
		PasswordRequireSpecial: true,
	}
}

func TestNewPasswordValidator_MinLengthFloor(t *testing.T) {
	v := NewPasswordValidator(config.SecurityConfig{PasswordMinLen: 3})
	if v.MinLen != 8 {
		t.Errorf("MinLen = %d, want 8 (floor enforced)", v.MinLen)
	}
}

func TestNewPasswordValidator_PassesThrough(t *testing.T) {
	v := NewPasswordValidator(defaultValidatorCfg())
	if v.MinLen != 8 {
		t.Errorf("MinLen = %d, want 8", v.MinLen)
	}
	if !v.RequireUpper || !v.RequireLower || !v.RequireDigit || !v.RequireSpecial {
		t.Errorf("expected all character classes required, got %+v", v)
	}
}

func TestPasswordValidator_HappyPath(t *testing.T) {
	v := NewPasswordValidator(defaultValidatorCfg())
	if err := v.Validate(strongPassword()); err != nil {
		t.Errorf("strong password rejected: %v", err)
	}
}

func TestPasswordValidator_TooShort(t *testing.T) {
	v := NewPasswordValidator(defaultValidatorCfg())
	err := v.Validate("Aa1!")
	if err == nil {
		t.Fatal("expected error for short password")
	}
	var pve *PasswordValidationError
	if !errors.As(err, &pve) {
		t.Fatalf("expected *PasswordValidationError, got %T", err)
	}
	found := false
	for _, e := range pve.Errors {
		if strings.Contains(e, "at least 8") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected length error, got %v", pve.Errors)
	}
}

func TestPasswordValidator_MissingClasses(t *testing.T) {
	v := NewPasswordValidator(defaultValidatorCfg())

	tests := []struct {
		name     string
		password string
		wantSubs []string
	}{
		{"no upper", "abcdefg1!", []string{"uppercase"}},
		{"no lower", "ABCDEFG1!", []string{"lowercase"}},
		{"no digit", "Abcdefgh!", []string{"digit"}},
		{"no special", "Abcdefg1", []string{"special"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := v.Validate(tc.password)
			if err == nil {
				t.Fatalf("expected error for %q", tc.password)
			}
			var pve *PasswordValidationError
			if !errors.As(err, &pve) {
				t.Fatalf("expected *PasswordValidationError, got %T", err)
			}
			for _, sub := range tc.wantSubs {
				found := false
				for _, e := range pve.Errors {
					if strings.Contains(e, sub) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error containing %q, got %v", sub, pve.Errors)
				}
			}
		})
	}
}

func TestPasswordValidator_AllMissingClassesReported(t *testing.T) {
	v := NewPasswordValidator(defaultValidatorCfg())
	// Single lowercase letter: short, no upper, no digit, no special
	err := v.Validate("a")
	if err == nil {
		t.Fatal("expected error")
	}
	var pve *PasswordValidationError
	if !errors.As(err, &pve) {
		t.Fatalf("expected *PasswordValidationError, got %T", err)
	}
	// Should have: length, upper, digit, special errors (4)
	if len(pve.Errors) < 4 {
		t.Errorf("expected at least 4 errors, got %d: %v", len(pve.Errors), pve.Errors)
	}
}

func TestPasswordValidator_CommonPasswordsRejected(t *testing.T) {
	v := NewPasswordValidator(config.SecurityConfig{PasswordMinLen: 8})
	for _, pw := range []string{"password", "Password", "PASSWORD", "12345678", "qwerty", "abc123", "admin", "deploypilot"} {
		t.Run(pw, func(t *testing.T) {
			err := v.Validate(pw)
			if err == nil {
				t.Fatalf("common password %q accepted", pw)
			}
			var pve *PasswordValidationError
			if !errors.As(err, &pve) {
				t.Fatalf("expected *PasswordValidationError, got %T", err)
			}
			found := false
			for _, e := range pve.Errors {
				if strings.Contains(e, "too common") {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected 'too common' error, got %v", pve.Errors)
			}
		})
	}
}

func TestPasswordValidator_LenientPolicy(t *testing.T) {
	// NewPasswordValidator enforces a MinLen floor of 8 regardless of config,
	// so we have to send a string of at least 8 chars to exercise the policy
	// branches. "insecure" is not a common password and is 8 chars long.
	v := NewPasswordValidator(config.SecurityConfig{
		PasswordMinLen:        4, // will be floored to 8
		PasswordRequireUpper:  false,
		PasswordRequireLower:  false,
		PasswordRequireDigit:  false,
		PasswordRequireSpecial: false,
	})
	if err := v.Validate("insecure"); err != nil {
		t.Errorf("lenient policy rejected simple password: %v", err)
	}
	// But common passwords are still flagged
	if err := v.Validate("password"); err == nil {
		t.Error("common password should be rejected even with lenient policy")
	}
}

func TestPasswordValidator_DisabledClassChecks(t *testing.T) {
	v := NewPasswordValidator(config.SecurityConfig{
		PasswordMinLen:        4, // floored to 8
		PasswordRequireUpper:  false,
		PasswordRequireLower:  false,
		PasswordRequireDigit:  false,
		PasswordRequireSpecial: false,
	})
	// 8-char non-common string should pass with all class checks disabled
	if err := v.Validate("insecure"); err != nil {
		t.Errorf("expected lenient policy to accept %q, got %v", "insecure", err)
	}
}

func TestPasswordStrengthScore_RangesAndPenalties(t *testing.T) {
	v := NewPasswordValidator(defaultValidatorCfg())

	// Score breakdown:
	//   length   = +10 per tier (8/12/16/20) -> max 40
	//   classes  = +8 each (lower/upper/digit/special/unicode) -> max 40
	//   patterns = -10 sequential, -10 repeating, -20 common
	// Clamped to [0, 100].
	tests := []struct {
		name     string
		password string
		minScore int
		maxScore int
	}{
		// No content, no length, no character classes
		{"empty", "", 0, 0},
		// 3 chars, 1 class -> 0 + 8 = 8 (then clamped)
		{"short lowercase", "abc", 0, 20},
		// 8 chars, lower+upper+digit -> 10 + 8*3 = 34
		{"decent mixed", "Abc12345", 30, 40},
		// 25 chars, all classes -> 40 + 8*4 = 72
		{"long strong", "Th!sIsAVeryL0ngStr0ngPwd!", 70, 80},
		// Common 8-char, no upper/lower/digit/special: 10 + 0 - 20 = clamped 0
		{"common", "password", 0, 20},
		// 10 chars, lower only: 10 (length>=8) + 8 (lower) - 10 (sequential) = 8
		{"sequential alpha", "abcdefghij", 0, 50},
		// 10 digits: 10 + 8 (digit) - 10 (sequential) = 8
		{"sequential digit", "1234567890", 0, 50},
		// 10 a's: 10 + 8 (lower) - 10 (repeating) = 8
		{"repeating", "aaaaaaaaaa", 0, 50},
		// 8 chars, 5 classes: 10 + 8*5 = 50
		{"unicode", "Abcdefg1中", 40, 60},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			score := v.StrengthScore(tc.password)
			if score < tc.minScore || score > tc.maxScore {
				t.Errorf("score %d not in [%d, %d] for %q", score, tc.minScore, tc.maxScore, tc.password)
			}
		})
	}
}

func TestPasswordStrengthScore_ClampedToZero(t *testing.T) {
	v := NewPasswordValidator(defaultValidatorCfg())
	// A common, short, repeating, sequential password should be clamped at 0
	score := v.StrengthScore("aaa")
	if score != 0 {
		t.Errorf("score = %d, want clamped to 0", score)
	}
}

func TestPasswordStrengthScore_UnicodeBonus(t *testing.T) {
	v := NewPasswordValidator(config.SecurityConfig{PasswordMinLen: 8})
	with := v.StrengthScore("Abcdefg1中")
	without := v.StrengthScore("Abcdefg1a")
	if with <= without {
		t.Errorf("unicode password score (%d) should be > ascii-only (%d)", with, without)
	}
}

func TestCheckPasswordExpired_DisabledByZero(t *testing.T) {
	if err := CheckPasswordExpired("", 0); err != nil {
		t.Errorf("maxAgeDays=0 should never expire, got %v", err)
	}
	if err := CheckPasswordExpired("", -1); err != nil {
		t.Errorf("negative maxAgeDays should never expire, got %v", err)
	}
}

func TestCheckPasswordExpired_EmptyTimestamp(t *testing.T) {
	err := CheckPasswordExpired("", 30)
	if !errors.Is(err, ErrPasswordExpired) {
		t.Errorf("empty timestamp should be expired, got %v", err)
	}
}

func TestCheckPasswordExpired_InvalidTimestamp(t *testing.T) {
	err := CheckPasswordExpired("not-a-real-date", 30)
	if !errors.Is(err, ErrPasswordExpired) {
		t.Errorf("unparseable timestamp should be treated as expired, got %v", err)
	}
}

func TestCheckPasswordExpired_FreshPassword(t *testing.T) {
	fresh := time.Now().UTC().Format(time.RFC3339)
	if err := CheckPasswordExpired(fresh, 30); err != nil {
		t.Errorf("fresh password should not be expired, got %v", err)
	}
}

func TestCheckPasswordExpired_OldPassword(t *testing.T) {
	old := time.Now().UTC().Add(-31 * 24 * time.Hour).Format(time.RFC3339)
	err := CheckPasswordExpired(old, 30)
	if !errors.Is(err, ErrPasswordExpired) {
		t.Errorf("31-day-old password should be expired under 30-day policy, got %v", err)
	}
}

func TestCheckPasswordExpired_Boundary(t *testing.T) {
	// 29 days, 23 hours ago — should not be expired under 30-day policy
	borderline := time.Now().UTC().Add(-29*24*time.Hour - 23*time.Hour).Format(time.RFC3339)
	if err := CheckPasswordExpired(borderline, 30); err != nil {
		t.Errorf("borderline password should not be expired, got %v", err)
	}
}

func TestPasswordValidationError_ErrorMessage(t *testing.T) {
	pve := &PasswordValidationError{Errors: []string{"a", "b"}}
	if got := pve.Error(); got != "a; b" {
		t.Errorf("Error() = %q, want %q", got, "a; b")
	}
	empty := &PasswordValidationError{}
	if got := empty.Error(); got != "" {
		t.Errorf("empty Errors should produce empty string, got %q", got)
	}
}

func TestIsCommonPassword_CaseInsensitive(t *testing.T) {
	if !isCommonPassword("PASSWORD") {
		t.Error("expected PASSWORD to be recognized as common (case insensitive)")
	}
	if !isCommonPassword("Admin") {
		t.Error("expected Admin to be recognized as common")
	}
	if isCommonPassword("UncommonSecret") {
		t.Error("expected UncommonSecret to NOT be common")
	}
}
