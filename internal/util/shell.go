package util

import (
	"strings"
	"unicode"
)

// ShellQuote safely escapes a string for use in a shell command.
func ShellQuote(s string) string {
	// Reject control characters (especially newlines) that can break
	// out of single-quoted shell arguments.
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
