package overlay

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func (m Model) box(active bool) lipgloss.Style {
	if active {
		return m.styles.activeBox.Width(m.width)
	}
	return m.styles.idleBox.Width(m.width)
}

// labelled writes a word into the bottom border, near the right corner, where a
// box has room for it and nothing else is drawn. The border is one colour, so the
// line is rebuilt from its characters rather than picked apart.
func (m Model) labelled(box, label string, active bool) string {
	if label == "" {
		return box
	}

	lines := strings.Split(box, "\n")
	bottom := ansi.Strip(lines[len(lines)-1])
	runes := []rune(bottom)

	const margin = 2
	room := len(runes) - margin*2
	if lipgloss.Width(label) > room {
		return box
	}

	border := m.styles.mark
	if active {
		border = m.styles.accent
	}
	at := len(runes) - margin - lipgloss.Width(label)
	lines[len(lines)-1] = border.Render(string(runes[:at])) +
		m.styles.badge.Render(label) +
		border.Render(string(runes[at+lipgloss.Width(label):]))
	return strings.Join(lines, "\n")
}

// howFarThrough is the share of the text that has been seen, for the border of
// whichever panel is being read.
func howFarThrough(first, visible, total int) string {
	if total <= visible {
		return ""
	}
	return strconv.Itoa(min((first+visible)*100/total, 100)) + "%"
}
