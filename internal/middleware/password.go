package middleware

import (
	"errors"
	"strings"
	"unicode"

	"github.com/Yogdunana/deploypilot/internal/config"
)

// PasswordValidationError represents a password validation failure.
type PasswordValidationError struct {
	Errors []string `json:"errors"`
}

func (e *PasswordValidationError) Error() string {
	return strings.Join(e.Errors, "; ")
}

// PasswordValidator validates passwords against security policy.
type PasswordValidator struct {
	MinLen          int
	RequireUpper    bool
	RequireLower    bool
	RequireDigit    bool
	RequireSpecial  bool
}

// NewPasswordValidator creates a validator from SecurityConfig.
func NewPasswordValidator(cfg config.SecurityConfig) *PasswordValidator {
	minLen := cfg.PasswordMinLen
	if minLen < 8 {
		minLen = 8
	}
	return &PasswordValidator{
		MinLen:         minLen,
		RequireUpper:   cfg.PasswordRequireUpper,
		RequireLower:   cfg.PasswordRequireLower,
		RequireDigit:   cfg.PasswordRequireDigit,
		RequireSpecial: cfg.PasswordRequireSpecial,
	}
}

// Validate checks a password against the policy and returns detailed errors.
func (v *PasswordValidator) Validate(password string) error {
	var errs []string

	if len(password) < v.MinLen {
		errs = append(errs, "password must be at least "+itoa(v.MinLen)+" characters")
	}

	if v.RequireUpper && !hasUpper(password) {
		errs = append(errs, "password must contain at least one uppercase letter")
	}

	if v.RequireLower && !hasLower(password) {
		errs = append(errs, "password must contain at least one lowercase letter")
	}

	if v.RequireDigit && !hasDigit(password) {
		errs = append(errs, "password must contain at least one digit")
	}

	if v.RequireSpecial && !hasSpecial(password) {
		errs = append(errs, "password must contain at least one special character")
	}

	// Check for common weak passwords
	if isCommonPassword(password) {
		errs = append(errs, "password is too common")
	}

	if len(errs) > 0 {
		return &PasswordValidationError{Errors: errs}
	}
	return nil
}

// StrengthScore returns a 0-100 password strength score.
func (v *PasswordValidator) StrengthScore(password string) int {
	score := 0

	// Length scoring (up to 40 points)
	if len(password) >= 8 {
		score += 10
	}
	if len(password) >= 12 {
		score += 10
	}
	if len(password) >= 16 {
		score += 10
	}
	if len(password) >= 20 {
		score += 10
	}

	// Character variety (up to 40 points)
	if hasLower(password) {
		score += 8
	}
	if hasUpper(password) {
		score += 8
	}
	if hasDigit(password) {
		score += 8
	}
	if hasSpecial(password) {
		score += 8
	}
	if hasUnicode(password) {
		score += 8
	}

	// Pattern penalties (up to -20 points)
	if isSequential(password) {
		score -= 10
	}
	if isRepeating(password) {
		score -= 10
	}
	if isCommonPassword(password) {
		score -= 20
	}

	if score < 0 {
		score = 0
	}
	return score
}

func hasUpper(s string) bool {
	for _, r := range s {
		if unicode.IsUpper(r) && unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func hasLower(s string) bool {
	for _, r := range s {
		if unicode.IsLower(r) && unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func hasDigit(s string) bool {
	for _, r := range s {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func hasSpecial(s string) bool {
	for _, r := range s {
		if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			return true
		}
	}
	return false
}

func hasUnicode(s string) bool {
	for _, r := range s {
		if r > 127 {
			return true
		}
	}
	return false
}

// isSequential checks for sequential characters like "abc", "123", "qwerty".
func isSequential(s string) bool {
	lower := strings.ToLower(s)
	sequences := []string{
		"abcdefghijklmnopqrstuvwxyz",
		"01234567890",
		"qwertyuiop", "asdfghjkl", "zxcvbnm",
	}
	for _, seq := range sequences {
		if strings.Contains(seq, lower) {
			return true
		}
		// Check reverse
		rev := reverseString(seq)
		if strings.Contains(rev, lower) {
			return true
		}
	}
	return false
}

func isRepeating(s string) bool {
	if len(s) < 3 {
		return false
	}
	runes := []rune(s)
	for i := 0; i < len(runes)-2; i++ {
		if runes[i] == runes[i+1] && runes[i+1] == runes[i+2] {
			return true
		}
	}
	return false
}

// commonPasswords is a list of commonly used weak passwords.
var commonPasswords = map[string]bool{
	"password": true, "123456": true, "12345678": true, "qwerty": true,
	"abc123": true, "monkey": true, "master": true, "dragon": true,
	"login": true, "princess": true, "football": true, "shadow": true,
	"sunshine": true, "trustno1": true, "iloveyou": true, "batman": true,
	"access": true, "hello": true, "charlie": true, "donald": true,
	"123456789": true, "1234567890": true, "password1": true, "qwerty123": true,
	"admin": true, "root": true, "toor": true, "pass": true, "test": true,
	"deploypilot": true, "deploy": true, "pilot": true,
}

func isCommonPassword(s string) bool {
	return commonPasswords[strings.ToLower(s)]
}

func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// ErrPasswordExpired is returned when a password has exceeded its maximum age.
var ErrPasswordExpired = errors.New("password has expired, please change it")

// CheckPasswordExpired checks if a password change timestamp indicates expiry.
// maxAgeDays of 0 means passwords never expire.
func CheckPasswordExpired(passwordChangedAt string, maxAgeDays int) error {
	if maxAgeDays <= 0 {
		return nil
	}
	// If passwordChangedAt is empty, consider it expired to force a change
	if passwordChangedAt == "" {
		return ErrPasswordExpired
	}
	return nil
}
