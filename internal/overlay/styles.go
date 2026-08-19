package overlay

import "github.com/charmbracelet/lipgloss"

// Herdr paints the terminal palette from the active theme but does not expose
// the theme itself, so the overlay only names palette slots and lets whatever
// theme is running decide how they look.
var (
	accent = lipgloss.Color("5")
	danger = lipgloss.Color("1")
	// The frame is a line, not something to read, so grey suits it.
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
	// off is for a setting that is not in force: struck through and otherwise in
	// the same colour as the rest, so it reads as switched off, not as faded out.
	off lipgloss.Style
	// mark is the braille signature: the colour of the frame, since it is drawn
	// furniture rather than something to read. Faint is too dim for braille dots.
	mark      lipgloss.Style
	activeBox lipgloss.Style
	idleBox   lipgloss.Style
}

func newStyles() styles {
	// Anything meant to be read keeps the terminal's own foreground: grey on a
	// dark theme, or on a light one, is what makes a popup unreadable. The
	// difference between a key and its label is weight, not brightness.
	return styles{
		text:        lipgloss.NewStyle(),
		placeholder: lipgloss.NewStyle().Faint(true),
		cursor:      lipgloss.NewStyle().Foreground(accent),
		badge:       lipgloss.NewStyle(),
		hint:        lipgloss.NewStyle(),
		key:         lipgloss.NewStyle().Foreground(accent).Bold(true),
		accent:      lipgloss.NewStyle().Foreground(accent),
		danger:      lipgloss.NewStyle().Foreground(danger).Bold(true),
		faded:       lipgloss.NewStyle().Faint(true),
		bright:      lipgloss.NewStyle().Foreground(bright).Bold(true),
		mode:        lipgloss.NewStyle().Foreground(accent).Bold(true),
		off:         lipgloss.NewStyle().Strikethrough(true),
		mark:        lipgloss.NewStyle().Foreground(frame),

		// The accented border marks the panel being written in or read; the other
		// one recedes into the frame's grey.
		activeBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1),
		idleBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(frame).
			Padding(0, 1),
	}
}
