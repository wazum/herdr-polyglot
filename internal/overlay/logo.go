package overlay

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// The plugin signs the draft box while there is nothing in it, in the corner
// furthest from where the writing starts. Braille cells are one column each.
var logo = []string{
	"⠀⠀⠀⢀⣀⣠⣤⣤⣤⡀",
	"⠀⠀⣰⣿⣿⣿⣿⣿⣿⡿",
	"⢀⣼⣿⣿⣿⡟⠛⢻⡿⣧",
	"⠘⣿⠏⣸⠏⢷⢀⡾⢁⣿",
}

// sign puts the mark in the bottom right of the box, on lines that show nothing.
// A line of the text area is styled even when it is empty, so what is on it has to
// be judged after the escape sequences are taken off.
func (m Model) sign(body string) string {
	lines := strings.Split(body, "\n")
	// The first line belongs to the cursor and the placeholder.
	if len(lines) <= len(logo) {
		return body
	}

	first := len(lines) - len(logo)
	for _, line := range lines[first:] {
		if strings.TrimSpace(ansi.Strip(line)) != "" {
			return body
		}
	}

	for index, mark := range logo {
		// The box adds padding of its own, so the mark stops short of the line's
		// full width or the line would wrap and take the box with it.
		room := m.width - draftFrame - lipgloss.Width(mark)
		if room < 0 {
			return body
		}
		lines[first+index] = strings.Repeat(" ", room) + m.styles.mark.Render(mark)
	}
	return strings.Join(lines, "\n")
}
