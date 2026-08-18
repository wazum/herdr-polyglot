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
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		m.header(),
		draftBoxStyle.Width(m.width+2).Render(m.draft.View()),
		m.footer(),
	)
	return boxStyle.Render(content)
}

func (m Model) header() string {
	title := titleStyle.Render("✳ polyglot")

	destination := "send"
	if m.options.Review {
		destination = "review"
	}
	badge := badgeStyle.Render(strings.Join([]string{
		m.options.Service,
		"→",
		m.options.Language,
		"·",
		destination,
	}, " "))

	return m.spread(title, badge)
}

func (m Model) footer() string {
	if m.failure != nil {
		return m.spread(dangerStyle.Render("✗ "+m.failure.Error()), hintStyle.Render("esc close"))
	}
	if m.stage == translating {
		return m.spread(m.spinner.View()+accentStyle.Render(" translating …"), "")
	}
	return m.spread(m.keyHints(), "")
}

func (m Model) keyHints() string {
	hints := []string{}
	for _, hint := range [][2]string{
		{"enter", "send"},
		{"alt+enter", "newline"},
		{"esc", "cancel"},
	} {
		hints = append(hints, keyStyle.Render(hint[0])+hintStyle.Render(" "+hint[1]))
	}
	return strings.Join(hints, hintStyle.Render(" · "))
}

func (m Model) spread(left, right string) string {
	total := m.width + 2
	gap := total - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return lipgloss.NewStyle().Width(total).Render(left)
	}
	return left + strings.Repeat(" ", gap) + right
}
