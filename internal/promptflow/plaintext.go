package promptflow

import (
	"strings"
	"unicode"
)

// plainText is what may be typed into another program's terminal: readable
// characters and nothing that the terminal would act on. Escape sequences would
// be interpreted by the agent's emulator, and a line break would submit a
// half-written prompt, so both are removed rather than passed along.
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
