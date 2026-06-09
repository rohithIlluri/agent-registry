package cmd

import "strings"

// truncate shortens s to at most max display runes, appending … when cut.
// Operating on runes (not bytes) keeps multibyte characters intact.
func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}
