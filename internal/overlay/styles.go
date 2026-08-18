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

// Foregrounds only: a cell left alone keeps the background herdr painted.
type styles struct {
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
	draftBox    lipgloss.Style
	englishBox  lipgloss.Style
}

func newStyles() styles {
	return styles{
		text:        lipgloss.NewStyle(),
		placeholder: lipgloss.NewStyle().Foreground(muted),
		cursor:      lipgloss.NewStyle().Foreground(accent),
		badge:       lipgloss.NewStyle().Foreground(muted),
		hint:        lipgloss.NewStyle().Foreground(muted),
		key:         lipgloss.NewStyle().Foreground(accent),
		accent:      lipgloss.NewStyle().Foreground(accent),
		danger:      lipgloss.NewStyle().Foreground(danger),
		faded:       lipgloss.NewStyle().Foreground(muted).Faint(true),
		bright:      lipgloss.NewStyle().Foreground(bright).Bold(true),
		mode:        lipgloss.NewStyle().Foreground(accent).Bold(true),

		draftBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(frame).
			Padding(0, 1),
		englishBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1).
			Height(englishRows - 2),
	}
}
