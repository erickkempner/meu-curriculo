package validators

import (
	"html"
	"strings"
)

// SanitizeText escapes HTML special characters (<, >, &, ", ')
// to prevent XSS injection when storing user input.
func SanitizeText(s string) string {
	return html.EscapeString(s)
}

// TrimAndSanitize trims leading/trailing whitespace and then
// escapes HTML special characters.
func TrimAndSanitize(s string) string {
	return SanitizeText(strings.TrimSpace(s))
}
