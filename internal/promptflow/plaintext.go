package promptflow

import (
	"strings"
	"unicode"
)

// readable is what may be shown: the text without anything the terminal would
// act on. An escape sequence in a translation would be interpreted by whichever
// emulator draws it, so it never travels further than this.
func readable(text string) string {
	var kept strings.Builder
	kept.Grow(len(text))

	for _, r := range text {
		switch {
		case r == '\n':
			kept.WriteRune(r)
		case r == '\t':
			kept.WriteRune(' ')
		case unicode.IsControl(r):
			// Dropped: the terminal would act on it.
		default:
			kept.WriteRune(r)
		}
	}
	return strings.TrimSpace(kept.String())
}

// plainText is what may be typed into another program's terminal: readable, and
// on one line, because a line break would submit a half-written prompt.
func plainText(text string) string {
	var plain strings.Builder
	plain.Grow(len(text))

	lastWasSpace := false
	for _, r := range text {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			if !lastWasSpace {
				plain.WriteRune(' ')
				lastWasSpace = true
			}
		case unicode.IsControl(r):
			// Dropped: the agent's terminal would act on it.
		default:
			plain.WriteRune(r)
			lastWasSpace = r == ' '
		}
	}
	return strings.TrimSpace(plain.String())
}
