package mermaid

import "strings"

// sanitizeLabel makes a string safe to use as a Mermaid node label inside
// square brackets. Mermaid uses square-bracket labels: `n0["label here"]`.
func sanitizeLabel(s string) string {
	// Escape double quotes and strip characters that could break Mermaid parsing.
	s = strings.ReplaceAll(s, `"`, `'`)
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}
