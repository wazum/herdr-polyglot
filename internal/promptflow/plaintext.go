package promptflow

import (
	"strings"
	"unicode"
)

const tabWidth = "    "

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
		case r == '\r':
			// A lone carriage return would move the cursor to the start of the line;
			// paired with a newline it is redundant.
		case r == '\t':
			// Code is often indented with tabs, but a tab keystroke in an agent's
			// input can be a completion rather than text, so it arrives as the
			// spaces it stands for.
			kept.WriteString(tabWidth)
		case unicode.IsControl(r):
			// Dropped: the terminal would act on it.
		default:
			kept.WriteRune(r)
		}
	}
	return strings.TrimSpace(kept.String())
}
