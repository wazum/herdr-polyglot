package overlay

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// The palette is adaptive so the overlay sits well on light and dark terminals.
var (
	accent = lipgloss.AdaptiveColor{Light: "#6C3EF5", Dark: "#B79BFF"}
	muted  = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#8A8FA3"}
	danger = lipgloss.AdaptiveColor{Light: "#B3261E", Dark: "#FF8A80"}
	frame  = lipgloss.AdaptiveColor{Light: "#C9C4E0", Dark: "#4A4566"}

	accentStyle = lipgloss.NewStyle().Foreground(accent)
	titleStyle  = lipgloss.NewStyle().Foreground(accent).Bold(true)
	badgeStyle  = lipgloss.NewStyle().Foreground(muted)
	hintStyle   = lipgloss.NewStyle().Foreground(muted)
	keyStyle    = lipgloss.NewStyle().Foreground(accent)
	dangerStyle = lipgloss.NewStyle().Foreground(danger)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(frame).
			Padding(0, 1)

	draftBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(frame).
			Padding(0, 1)
)

func (m Model) View() string {
	draft := draftBoxStyle.Width(m.width).Render(m.draft.View())
	line := lipgloss.Width(draft)

	return boxStyle.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		m.header(line),
		draft,
		m.footer(line),
	))
}

func (m Model) header(line int) string {
	destination := "send"
	if m.options.Review {
		destination = "review"
	}
	badge := badgeStyle.Render(strings.Join(
		[]string{m.options.Service, "→", m.options.Language, "·", destination}, " ",
	))

	return spread(titleStyle.Render("✳ polyglot"), badge, line)
}

func (m Model) footer(line int) string {
	switch {
	case m.failure != nil:
		return spread(dangerStyle.Render("✗ "+m.failure.Error()), hintStyle.Render("esc close"), line)
	case m.stage == translating:
		return m.spinner.View() + accentStyle.Render(" translating …")
	default:
		return keyHints()
	}
}

func keyHints() string {
	hints := make([]string, 0, 4)
	for _, hint := range [][2]string{
		{"ctrl+d", "send"},
		{"alt+enter", "send"},
		{"enter", "newline"},
		{"esc", "cancel"},
	} {
		hints = append(hints, keyStyle.Render(hint[0])+hintStyle.Render(" "+hint[1]))
	}
	return strings.Join(hints, hintStyle.Render(" · "))
}

// spread pins left to the start of the line and right to its end.
func spread(left, right string, line int) string {
	gap := line - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return left
	}
	return left + strings.Repeat(" ", gap) + right
}
