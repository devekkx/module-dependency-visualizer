package dot

import "strings"

// escapeID wraps a DOT node identifier in double quotes, escaping any
// internal double quotes.
func escapeID(s string) string {
	escaped := strings.ReplaceAll(s, `"`, `\"`)
	return `"` + escaped + `"`
}

// escapeLabel escapes a string for use inside a DOT label attribute value,
// which is already inside double quotes.
func escapeLabel(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
