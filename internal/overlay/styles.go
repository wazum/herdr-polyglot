package overlay

import "github.com/charmbracelet/lipgloss"

// Herdr paints the terminal palette from the active theme but does not expose
// the theme itself, so the overlay only names palette slots and lets whatever
// theme is running decide how they look.
var (
	accent = lipgloss.Color("5")
	muted  = lipgloss.Color("8")
	danger = lipgloss.Color("1")
	frame  = lipgloss.Color("8")
	bright = lipgloss.Color("13")
)

// styles carries the same background into every piece the overlay draws. Herdr
// paints the popup in its own colour, and a cell written without it reads as a
// patch of the wrong shade.
type styles struct {
	background  lipgloss.TerminalColor
	known       bool
	text        lipgloss.Style
	placeholder lipgloss.Style
	cursor      lipgloss.Style
	badge       lipgloss.Style
	hint        lipgloss.Style
	key         lipgloss.Style
	accent      lipgloss.Style
	danger      lipgloss.Style
	faded       lipgloss.Style
	bright      lipgloss.Style
	mode        lipgloss.Style
	pane        lipgloss.Style
	draftBox    lipgloss.Style
	englishBox  lipgloss.Style
}

func newStyles(background string) styles {
	set := styles{known: background != ""}
	on := func(style lipgloss.Style) lipgloss.Style {
		if !set.known {
			return style
		}
		return style.Background(lipgloss.Color(background))
	}

	if set.known {
		set.background = lipgloss.Color(background)
	}
	set.text = on(lipgloss.NewStyle())
	set.placeholder = on(lipgloss.NewStyle().Foreground(muted))
	set.cursor = on(lipgloss.NewStyle().Foreground(accent))
	set.badge = on(lipgloss.NewStyle().Foreground(muted))
	set.hint = on(lipgloss.NewStyle().Foreground(muted))
	set.key = on(lipgloss.NewStyle().Foreground(accent))
	set.accent = on(lipgloss.NewStyle().Foreground(accent))
	set.danger = on(lipgloss.NewStyle().Foreground(danger))
	set.faded = on(lipgloss.NewStyle().Foreground(muted).Faint(true))
	set.bright = on(lipgloss.NewStyle().Foreground(bright).Bold(true))
	set.mode = on(lipgloss.NewStyle().Foreground(accent).Bold(true))
	set.pane = on(lipgloss.NewStyle())

	set.draftBox = on(lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(frame).
		Padding(0, 1))
	set.englishBox = on(lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Padding(0, 1).
		Height(englishRows - 2))

	if set.known {
		set.draftBox = set.draftBox.BorderBackground(set.background)
		set.englishBox = set.englishBox.BorderBackground(set.background)
	}
	return set
}
