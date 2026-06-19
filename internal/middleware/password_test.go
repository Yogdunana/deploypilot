package middleware

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/config"
)

// strictPolicy returns a SecurityConfig that requires all four character
// classes and the longest reasonable minimum. Used as the baseline for
// the "happy path" tests.
func strictPolicy() config.SecurityConfig {
	return config.SecurityConfig{
		PasswordMinLen:         12,
		PasswordRequireUpper:   true,
		PasswordRequireLower:   true,
		PasswordRequireDigit:   true,
		PasswordRequireSpecial: true,
	}
}

// TestNewPasswordValidator_AppliesMinLenFloor verifies that the constructor
// enforces a minimum length floor of 8 characters regardless of the
// configured value. This guards against accidentally allowing very short
// passwords through a misconfigured (or malicious) config.
func TestNewPasswordValidator_AppliesMinLenFloor(t *testing.T) {
	cases := []struct {
		name        string
		inputMinLen int
		wantMinLen  int
	}{
		{"below_floor_clamped", 0, 8},
		{"below_floor_negative_clamped", -5, 8},
		{"at_floor", 8, 8},
		{"above_floor", 16, 16},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := NewPasswordValidator(config.SecurityConfig{PasswordMinLen: tc.inputMinLen})
			if v.MinLen != tc.wantMinLen {
				t.Errorf("MinLen = %d, want %d", v.MinLen, tc.wantMinLen)
			}
		})
	}
}

// TestNewPasswordValidator_PropagatesFlags verifies that the boolean
// policy fields are propagated unchanged.
func TestNewPasswordValidator_PropagatesFlags(t *testing.T) {
	cfg := config.SecurityConfig{
		PasswordMinLen:         8,
		PasswordRequireUpper:   true,
		PasswordRequireLower:   false,
		PasswordRequireDigit:   true,
		PasswordRequireSpecial: false,
	}
	v := NewPasswordValidator(cfg)
	if !v.RequireUpper {
		t.Error("RequireUpper should be true")
	}
	if v.RequireLower {
		t.Error("RequireLower should be false")
	}
	if !v.RequireDigit {
		t.Error("RequireDigit should be true")
	}
	if v.RequireSpecial {
		t.Error("RequireSpecial should be false")
	}
}

// TestValidate_AcceptsStrongPassword verifies the happy path.
func TestValidate_AcceptsStrongPassword(t *testing.T) {
	v := NewPasswordValidator(strictPolicy())
	if err := v.Validate("Sup3rStr0ng!Pass"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

// TestValidate_RejectsTooShort verifies the length rule.
func TestValidate_RejectsTooShort(t *testing.T) {
	v := NewPasswordValidator(strictPolicy())
	err := v.Validate("Aa1!bb")
	if err == nil {
		t.Fatal("expected error for short password")
	}
	var pve *PasswordValidationError
	if !errors.As(err, &pve) {
		t.Fatalf("expected *PasswordValidationError, got %T", err)
	}
	if !containsMsg(pve.Errors, "at least") {
		t.Errorf("expected length-related error, got %v", pve.Errors)
	}
}

// TestValidate_RejectsMissingUpper verifies the uppercase rule.
func TestValidate_RejectsMissingUpper(t *testing.T) {
	v := NewPasswordValidator(strictPolicy())
	err := v.Validate("sup3rstr0ng!pass")
	if err == nil {
		t.Fatal("expected error for missing uppercase")
	}
	var pve *PasswordValidationError
	if !errors.As(err, &pve) {
		t.Fatalf("expected *PasswordValidationError, got %T", err)
	}
	if !containsMsg(pve.Errors, "uppercase") {
		t.Errorf("expected uppercase error, got %v", pve.Errors)
	}
}

// TestValidate_RejectsMissingLower verifies the lowercase rule.
func TestValidate_RejectsMissingLower(t *testing.T) {
	v := NewPasswordValidator(strictPolicy())
	err := v.Validate("SUP3RSTR0NG!PASS")
	if err == nil {
		t.Fatal("expected error for missing lowercase")
	}
	var pve *PasswordValidationError
	if !errors.As(err, &pve) {
		t.Fatalf("expected *PasswordValidationError, got %T", err)
	}
	if !containsMsg(pve.Errors, "lowercase") {
		t.Errorf("expected lowercase error, got %v", pve.Errors)
	}
}

// TestValidate_RejectsMissingDigit verifies the digit rule.
func TestValidate_RejectsMissingDigit(t *testing.T) {
	v := NewPasswordValidator(strictPolicy())
	err := v.Validate("SuperStrong!Pass")
	if err == nil {
		t.Fatal("expected error for missing digit")
	}
	var pve *PasswordValidationError
	if !errors.As(err, &pve) {
		t.Fatalf("expected *PasswordValidationError, got %T", err)
	}
	if !containsMsg(pve.Errors, "digit") {
		t.Errorf("expected digit error, got %v", pve.Errors)
	}
}

// TestValidate_RejectsMissingSpecial verifies the special character rule.
func TestValidate_RejectsMissingSpecial(t *testing.T) {
	v := NewPasswordValidator(strictPolicy())
	err := v.Validate("Sup3rStr0ngPass")
	if err == nil {
		t.Fatal("expected error for missing special character")
	}
	var pve *PasswordValidationError
	if !errors.As(err, &pve) {
		t.Fatalf("expected *PasswordValidationError, got %T", err)
	}
	if !containsMsg(pve.Errors, "special") {
		t.Errorf("expected special error, got %v", pve.Errors)
	}
}

// TestValidate_AggregatesAllErrors verifies that all rule violations are
// reported in a single call, not just the first one. This is important
// for UX: a user should see every issue at once.
func TestValidate_AggregatesAllErrors(t *testing.T) {
	v := NewPasswordValidator(strictPolicy())
	err := v.Validate("short")
	if err == nil {
		t.Fatal("expected error for multiple violations")
	}
	var pve *PasswordValidationError
	if !errors.As(err, &pve) {
		t.Fatalf("expected *PasswordValidationError, got %T", err)
	}
	// 1=length, 2=upper, 3=lower (ok, has 's'), 4=digit, 5=special, 6=common
	// "short" is in the common password list, so we should get length+digit+special+common at minimum
	if len(pve.Errors) < 3 {
		t.Errorf("expected at least 3 errors aggregated, got %d: %v", len(pve.Errors), pve.Errors)
	}
}

// TestValidate_RejectsCommonPasswords verifies that the weak-password
// guard works for several known common passwords. The function looks up
// the password in a small built-in list (case-insensitive).
func TestValidate_RejectsCommonPasswords(t *testing.T) {
	v := NewPasswordValidator(config.SecurityConfig{PasswordMinLen: 8})
	// All entries below appear verbatim in commonPasswords in password.go.
	cases := []string{
		"password",
		"Password1", // "password1" is in the list, casing is normalized
		"PASSWORD",
		"qwerty",
		"123456",
		"12345678",
		"admin",
		"root",
		"deploypilot",
		"deploy",
		"pilot",
	}
	for _, p := range cases {
		t.Run(p, func(t *testing.T) {
			err := v.Validate(p)
			if err == nil {
				t.Errorf("expected common-password rejection for %q", p)
			}
		})
	}
}

// TestValidate_AllRulesOptional verifies that when all require-* flags
// are false, only the length rule is enforced.
func TestValidate_AllRulesOptional(t *testing.T) {
	v := NewPasswordValidator(config.SecurityConfig{PasswordMinLen: 8})
	// 8-character alphanumeric password should pass.
	if err := v.Validate("abcdefgh"); err != nil {
		t.Errorf("expected pass with all rules off, got %v", err)
	}
	// 7-character password should fail.
	if err := v.Validate("abcdefg"); err == nil {
		t.Error("expected length failure for short password")
	}
}

// TestPasswordValidationError_ErrorMessage verifies the Error() string
// format (semicolon-joined). This is consumed by API error responses.
func TestPasswordValidationError_ErrorMessage(t *testing.T) {
	e := &PasswordValidationError{Errors: []string{"a", "b", "c"}}
	want := "a; b; c"
	if got := e.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

// TestStrengthScore_HigherForLongerPasswords verifies the monotonic
// length scoring property: longer passwords never score lower than
// shorter ones (when no penalty triggers).
func TestStrengthScore_HigherForLongerPasswords(t *testing.T) {
	v := NewPasswordValidator(strictPolicy())
	short := v.StrengthScore("Aa1!aaaa")
	long := v.StrengthScore("Aa1!aaaaaaaaaaaaaaaaaaaa")
	if long <= short {
		t.Errorf("expected longer to score higher, short=%d long=%d", short, long)
	}
}

// TestStrengthScore_ClampedToZero verifies the floor behavior: even
// heavily penalized passwords cannot go below 0.
func TestStrengthScore_ClampedToZero(t *testing.T) {
	v := NewPasswordValidator(strictPolicy())
	// "aaaaaa" (repeating) -20, sequential (contains "abcdefghijklmnop...")?, common "aaaaaa"
	score := v.StrengthScore("aaaaaa")
	if score < 0 {
		t.Errorf("score should be clamped to 0, got %d", score)
	}
}

// TestStrengthScore_CommonPasswordPenalty verifies that a known common
// password scores significantly lower than a similarly complex non-common
// password.
func TestStrengthScore_CommonPasswordPenalty(t *testing.T) {
	v := NewPasswordValidator(strictPolicy())
	common := v.StrengthScore("Password1!")
	unique := v.StrengthScore("Zb8!kxQ@Vm2p")
	if unique <= common {
		t.Errorf("expected unique to outscore common, common=%d unique=%d", common, unique)
	}
}

// TestStrengthScore_SequentialPenalty verifies that sequential characters
// (like "abcdef") trigger a penalty. Note: the implementation's
// isSequential only flags a password as sequential when the entire
// (lowercased) value is a substring of a known keyboard/alpha sequence,
// so we use a password made up entirely of an alpha sequence.
func TestStrengthScore_SequentialPenalty(t *testing.T) {
	v := NewPasswordValidator(strictPolicy())
	// "Abcdef" lowercased is "abcdef" which is a substring of
	// "abcdefghijklmnopqrstuvwxyz" and triggers isSequential.
	noSeq := v.StrengthScore("Ab1!xkqz")
	withSeq := v.StrengthScore("Abcdef")
	if withSeq >= noSeq {
		t.Errorf("expected sequential password to score lower, withSeq=%d noSeq=%d", withSeq, noSeq)
	}
}

// TestStrengthScore_RepeatingPenalty verifies that repeating characters
// (like "aaaa") trigger a penalty.
func TestStrengthScore_RepeatingPenalty(t *testing.T) {
	v := NewPasswordValidator(strictPolicy())
	noRepeat := v.StrengthScore("Ab1!xkqz")
	repeating := v.StrengthScore("Ab1!aaaa")
	if repeating >= noRepeat {
		t.Errorf("expected repeating password to score lower, repeating=%d noRepeat=%d", repeating, noRepeat)
	}
}

// TestCheckPasswordExpired_DisabledWhenZero verifies that a non-positive
// max age means passwords never expire.
func TestCheckPasswordExpired_DisabledWhenZero(t *testing.T) {
	if err := CheckPasswordExpired("2020-01-01T00:00:00Z", 0); err != nil {
		t.Errorf("expected nil when max age is 0, got %v", err)
	}
	if err := CheckPasswordExpired("2020-01-01T00:00:00Z", -1); err != nil {
		t.Errorf("expected nil when max age is negative, got %v", err)
	}
}

// TestCheckPasswordExpired_EmptyTimestampRejected verifies that missing
// timestamp is treated as expired (fail-closed for safety).
func TestCheckPasswordExpired_EmptyTimestampRejected(t *testing.T) {
	err := CheckPasswordExpired("", 30)
	if !errors.Is(err, ErrPasswordExpired) {
		t.Errorf("expected ErrPasswordExpired, got %v", err)
	}
}

// TestCheckPasswordExpired_UnparseableRejected verifies that an invalid
// timestamp is treated as expired (fail-closed).
func TestCheckPasswordExpired_UnparseableRejected(t *testing.T) {
	err := CheckPasswordExpired("not-a-timestamp", 30)
	if !errors.Is(err, ErrPasswordExpired) {
		t.Errorf("expected ErrPasswordExpired, got %v", err)
	}
}

// TestCheckPasswordExpired_FreshPasswordAccepted verifies a recently
// changed password is accepted.
func TestCheckPasswordExpired_FreshPasswordAccepted(t *testing.T) {
	fresh := time.Now().UTC().Format(time.RFC3339)
	if err := CheckPasswordExpired(fresh, 30); err != nil {
		t.Errorf("expected nil for fresh password, got %v", err)
	}
}

// TestCheckPasswordExpired_ExpiredRejected verifies a password older
// than the max age is rejected.
func TestCheckPasswordExpired_ExpiredRejected(t *testing.T) {
	old := time.Now().UTC().AddDate(0, 0, -60).Format(time.RFC3339)
	if err := CheckPasswordExpired(old, 30); !errors.Is(err, ErrPasswordExpired) {
		t.Errorf("expected ErrPasswordExpired, got %v", err)
	}
}

// containsMsg is a small helper for substring checks over the error list.
func containsMsg(msgs []string, substr string) bool {
	for _, m := range msgs {
		if strings.Contains(m, substr) {
			return true
		}
	}
	return false
}
