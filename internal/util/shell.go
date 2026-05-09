package util

import "strings"

// ShellQuote safely escapes a string for use in a shell command.
func ShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
