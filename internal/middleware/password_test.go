package middleware

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/config"
)

func TestNewPasswordValidator_DefaultMinLen(t *testing.T) {
	cfg := config.SecurityConfig{
		PasswordMinLen: 0,
	}
	v := NewPasswordValidator(cfg)
	if v.MinLen != 8 {
		t.Errorf("expected default MinLen 8, got %d", v.MinLen)
	}
}

func TestNewPasswordValidator_CustomMinLen(t *testing.T) {
	cfg := config.SecurityConfig{
		PasswordMinLen: 12,
	}
	v := NewPasswordValidator(cfg)
	if v.MinLen != 12 {
		t.Errorf("expected MinLen 12, got %d", v.MinLen)
	}
}

func TestPasswordValidator_Validate_AllRequirementsPass(t *testing.T) {
	v := &PasswordValidator{
		MinLen:         8,
		RequireUpper:   true,
		RequireLower:   true,
		RequireDigit:   true,
		RequireSpecial: true,
	}
	err := v.Validate("Str0ng!Pass")
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestPasswordValidator_Validate_TooShort(t *testing.T) {
	v := &PasswordValidator{MinLen: 8}
	err := v.Validate("short")
	if err == nil {
		t.Error("expected error for short password")
	}
	var pve *PasswordValidationError
	if !errors.As(err, &pve) {
		t.Error("expected PasswordValidationError type")
	}
	if !containsError(pve.Errors, "at least 8 characters") {
		t.Errorf("expected length error, got: %v", pve.Errors)
	}
}

func TestPasswordValidator_Validate_MissingUpper(t *testing.T) {
	v := &PasswordValidator{
		MinLen:       8,
		RequireUpper: true,
	}
	err := v.Validate("lowercase1!")
	if err == nil {
		t.Error("expected error for missing uppercase")
	}
	var pve *PasswordValidationError
	if !errors.As(err, &pve) {
		t.Error("expected PasswordValidationError type")
	}
	if !containsError(pve.Errors, "uppercase letter") {
		t.Errorf("expected uppercase error, got: %v", pve.Errors)
	}
}

func TestPasswordValidator_Validate_MissingLower(t *testing.T) {
	v := &PasswordValidator{
		MinLen:       8,
		RequireLower: true,
	}
	err := v.Validate("UPPERCASE1!")
	if err == nil {
		t.Error("expected error for missing lowercase")
	}
	var pve *PasswordValidationError
	if !errors.As(err, &pve) {
		t.Error("expected PasswordValidationError type")
	}
	if !containsError(pve.Errors, "lowercase letter") {
		t.Errorf("expected lowercase error, got: %v", pve.Errors)
	}
}

func TestPasswordValidator_Validate_MissingDigit(t *testing.T) {
	v := &PasswordValidator{
		MinLen:       8,
		RequireDigit: true,
	}
	err := v.Validate("UpperLower!")
	if err == nil {
		t.Error("expected error for missing digit")
	}
	var pve *PasswordValidationError
	if !errors.As(err, &pve) {
		t.Error("expected PasswordValidationError type")
	}
	if !containsError(pve.Errors, "digit") {
		t.Errorf("expected digit error, got: %v", pve.Errors)
	}
}

func TestPasswordValidator_Validate_MissingSpecial(t *testing.T) {
	v := &PasswordValidator{
		MinLen:         8,
		RequireSpecial: true,
	}
	err := v.Validate("UpperLower1")
	if err == nil {
		t.Error("expected error for missing special char")
	}
	var pve *PasswordValidationError
	if !errors.As(err, &pve) {
		t.Error("expected PasswordValidationError type")
	}
	if !containsError(pve.Errors, "special character") {
		t.Errorf("expected special char error, got: %v", pve.Errors)
	}
}

func TestPasswordValidator_Validate_CommonPassword(t *testing.T) {
	v := &PasswordValidator{MinLen: 6}
	err := v.Validate("password")
	if err == nil {
		t.Error("expected error for common password")
	}
	var pve *PasswordValidationError
	if !errors.As(err, &pve) {
		t.Error("expected PasswordValidationError type")
	}
	if !containsError(pve.Errors, "too common") {
		t.Errorf("expected common password error, got: %v", pve.Errors)
	}
}

func TestPasswordValidator_Validate_CommonPasswordCaseInsensitive(t *testing.T) {
	v := &PasswordValidator{MinLen: 6}
	err := v.Validate("Password1")
	if err == nil {
		t.Error("expected error for common password (case insensitive)")
	}
	var pve *PasswordValidationError
	if !errors.As(err, &pve) {
		t.Error("expected PasswordValidationError type")
	}
	if !containsError(pve.Errors, "too common") {
		t.Errorf("expected common password error, got: %v", pve.Errors)
	}
}

func TestPasswordValidationError_Error(t *testing.T) {
	e := &PasswordValidationError{
		Errors: []string{"err1", "err2", "err3"},
	}
	msg := e.Error()
	if !strings.Contains(msg, "err1") || !strings.Contains(msg, "err2") || !strings.Contains(msg, "err3") {
		t.Errorf("Error() should contain all errors, got: %s", msg)
	}
}

func TestPasswordValidator_StrengthScore_Length(t *testing.T) {
	v := &PasswordValidator{}
	tests := []struct {
		pw       string
		minScore int
		maxScore int
	}{
		{"abc", 0, 0},
		{"abcdefgh", 0, 20},
		{"abcdefghijkl", 10, 30},
		{"abcdefghijklmnop", 20, 40},
		{"abcdefghijklmnopqrst", 30, 50},
	}
	for _, tt := range tests {
		score := v.StrengthScore(tt.pw)
		if score < tt.minScore || score > tt.maxScore {
			t.Errorf("password %q: expected score between %d-%d, got %d", tt.pw, tt.minScore, tt.maxScore, score)
		}
	}
}

func TestPasswordValidator_StrengthScore_CharacterVariety(t *testing.T) {
	v := &PasswordValidator{}
	tests := []struct {
		pw       string
		minScore int
	}{
		{"kxmqprst", 8},
		{"KXMQPRST", 8},
		{"kxmqprst", 8},
		{"kxmQprsT", 16},
		{"kxmQprs1", 16},
		{"kxmQprs!", 16},
		{"kxmQpr1!", 24},
	}
	for _, tt := range tests {
		score := v.StrengthScore(tt.pw)
		if score < tt.minScore {
			t.Errorf("password %q: expected min score %d, got %d", tt.pw, tt.minScore, score)
		}
	}
}

func TestPasswordValidator_StrengthScore_PatternPenalties(t *testing.T) {
	v := &PasswordValidator{}
	seqScore := v.StrengthScore("abcdefgh")
	normalScore := v.StrengthScore("axkqjzpw")
	if seqScore >= normalScore {
		t.Errorf("sequential password should score lower than random: seq=%d, normal=%d", seqScore, normalScore)
	}

	repeatScore := v.StrengthScore("aaaabbbb")
	if repeatScore >= normalScore {
		t.Errorf("repeating password should score lower than random: repeat=%d, normal=%d", repeatScore, normalScore)
	}

	commonScore := v.StrengthScore("password")
	if commonScore >= normalScore {
		t.Errorf("common password should score lower than random: common=%d, normal=%d", commonScore, normalScore)
	}
}

func TestPasswordValidator_StrengthScore_NeverNegative(t *testing.T) {
	v := &PasswordValidator{}
	score := v.StrengthScore("aaaa")
	if score < 0 {
		t.Errorf("score should never be negative, got %d", score)
	}
}

func TestIsSequential(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"abc", true},
		{"cba", true},
		{"123", true},
		{"321", true},
		{"qwe", true},
		{"zxc", true},
		{"axk", false},
		{"p@ss", false},
	}
	for _, tt := range tests {
		got := isSequential(tt.s)
		if got != tt.want {
			t.Errorf("isSequential(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestIsRepeating(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"aaa", true},
		{"111", true},
		{"aaab", true},
		{"abbb", true},
		{"aa", false},
		{"aba", false},
		{"abc", false},
		{"", false},
	}
	for _, tt := range tests {
		got := isRepeating(tt.s)
		if got != tt.want {
			t.Errorf("isRepeating(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestIsCommonPassword(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"password", true},
		{"PASSWORD", true},
		{"Password", true},
		{"123456", true},
		{"qwerty", true},
		{"admin", true},
		{"deploypilot", true},
		{"MyStr0ngP@ss!", false},
		{"", false},
	}
	for _, tt := range tests {
		got := isCommonPassword(tt.s)
		if got != tt.want {
			t.Errorf("isCommonPassword(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestHasUpper(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"Hello", true},
		{"HELLO", true},
		{"hello", false},
		{"123", false},
		{"", false},
		{"Hello123!", true},
	}
	for _, tt := range tests {
		got := hasUpper(tt.s)
		if got != tt.want {
			t.Errorf("hasUpper(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestHasLower(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"Hello", true},
		{"hello", true},
		{"HELLO", false},
		{"123", false},
		{"", false},
		{"Hello123!", true},
	}
	for _, tt := range tests {
		got := hasLower(tt.s)
		if got != tt.want {
			t.Errorf("hasLower(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestHasDigit(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"hello1", true},
		{"123", true},
		{"hello", false},
		{"", false},
		{"Hello!2", true},
	}
	for _, tt := range tests {
		got := hasDigit(tt.s)
		if got != tt.want {
			t.Errorf("hasDigit(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestHasSpecial(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"hello!", true},
		{"hello@world", true},
		{"hello#world", true},
		{"hello", false},
		{"123", false},
		{"", false},
		{"Hello1!", true},
	}
	for _, tt := range tests {
		got := hasSpecial(tt.s)
		if got != tt.want {
			t.Errorf("hasSpecial(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestCheckPasswordExpired_Disabled(t *testing.T) {
	err := CheckPasswordExpired("2020-01-01T00:00:00Z", 0)
	if err != nil {
		t.Errorf("expected no error when maxAgeDays=0, got: %v", err)
	}
}

func TestCheckPasswordExpired_EmptyTimestamp(t *testing.T) {
	err := CheckPasswordExpired("", 90)
	if !errors.Is(err, ErrPasswordExpired) {
		t.Errorf("expected ErrPasswordExpired for empty timestamp, got: %v", err)
	}
}

func TestCheckPasswordExpired_InvalidTimestamp(t *testing.T) {
	err := CheckPasswordExpired("invalid-date", 90)
	if !errors.Is(err, ErrPasswordExpired) {
		t.Errorf("expected ErrPasswordExpired for invalid timestamp, got: %v", err)
	}
}

func TestCheckPasswordExpired_NotExpired(t *testing.T) {
	recent := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	err := CheckPasswordExpired(recent, 90)
	if err != nil {
		t.Errorf("expected no error for recent password, got: %v", err)
	}
}

func TestCheckPasswordExpired_Expired(t *testing.T) {
	old := time.Now().Add(-100 * 24 * time.Hour).Format(time.RFC3339)
	err := CheckPasswordExpired(old, 90)
	if !errors.Is(err, ErrPasswordExpired) {
		t.Errorf("expected ErrPasswordExpired for old password, got: %v", err)
	}
}

func TestReverseString(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"abc", "cba"},
		{"", ""},
		{"a", "a"},
		{"hello", "olleh"},
		{"12345", "54321"},
	}
	for _, tt := range tests {
		got := reverseString(tt.in)
		if got != tt.want {
			t.Errorf("reverseString(%q) = %q, want %q", tt.in, got, tt.want)
		}
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
		{1000, "1000"},
	}
	for _, tt := range tests {
		got := itoa(tt.n)
		if got != tt.want {
			t.Errorf("itoa(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func containsError(errs []string, substr string) bool {
	for _, e := range errs {
		if strings.Contains(e, substr) {
			return true
		}
	}
	return false
}
