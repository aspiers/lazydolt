package components

import (
	"strings"
	"unicode"
)

// SanitizeForDisplay replaces non-printable characters (except newline
// and tab) with the middle-dot character (·). This prevents binary or
// control characters from corrupting the terminal display.
func SanitizeForDisplay(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\n' || r == '\t':
			b.WriteRune(r)
		case unicode.IsControl(r):
			b.WriteRune('·')
		case !unicode.IsPrint(r) && !unicode.IsSpace(r):
			b.WriteRune('·')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
