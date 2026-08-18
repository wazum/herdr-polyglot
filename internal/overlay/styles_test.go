package overlay

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Grey on a dark background is what made the popup hard to read. Text is left in
// the terminal's own foreground, which is the one colour guaranteed to contrast
// with its background, whatever the theme.
func TestNoTextIsDrawnInGrey(t *testing.T) {
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI)
	defer lipgloss.SetColorProfile(previous)

	look := newStyles()
	for name, style := range map[string]lipgloss.Style{
		"text":        look.text,
		"badge":       look.badge,
		"hint":        look.hint,
		"key":         look.key,
		"mode":        look.mode,
		"placeholder": look.placeholder,
		"danger":      look.danger,
	} {
		rendered := style.Render("Aa")
		for _, grey := range []string{"30m", "90m", ";30", ";90"} {
			if strings.Contains(rendered, grey) {
				t.Errorf("%s draws text in grey (%s): %q", name, grey, rendered)
			}
		}
	}
}
