package overlay

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// A column for the bar and one to keep it off the text, kept whether or not there
// is anything to scroll, so text never rewraps when a bar appears.
const scrollColumn = 2

// A dashed hairline for the track, so it does not read as a second border, and a
// heavier line in the accent colour for the thumb.
const (
	scrollTrack = "╎"
	scrollThumb = "┃"
)

func (m Model) scrolled(body string, first, visible, total int) string {
	lines := strings.Split(body, "\n")
	if total <= visible || len(lines) == 0 {
		return body
	}

	// The thumb is as long a part of the bar as the box is of the text, and sits
	// where the view sits — at the bottom once the last row is on screen.
	height := max(len(lines)*visible/total, 1)
	room := len(lines) - height
	scrollable := total - visible

	top := 0
	if room > 0 && scrollable > 0 {
		top = min((first*room+scrollable/2)/scrollable, room)
	}

	for index, line := range lines {
		// The thumb is read, the track only tells it where it can go.
		mark := m.styles.mark.Render(scrollTrack)
		if index >= top && index < top+height {
			mark = m.styles.accent.Render(scrollThumb)
		}
		// Padded first: a bar is a column, not something that follows the text.
		lines[index] = line + strings.Repeat(" ",
			max(m.contentWidth()-lipgloss.Width(line), 0)) + " " + mark
	}
	return strings.Join(lines, "\n")
}

// rowsOf counts the rows text takes at that width, breaking a word too long to
// fit the way the text area does.
func rowsOf(text string, width int) int {
	if text == "" {
		return 1
	}
	if width < 1 {
		return len(strings.Split(text, "\n"))
	}
	return len(strings.Split(wrapped(text, width), "\n"))
}

func wrapped(text string, width int) string {
	return ansi.Wrap(text, width, "")
}

// draftScroll works out where the view sits from the cursor, which the text area
// always keeps on screen.
func (m Model) draftScroll() (first, visible, total int) {
	width := m.contentWidth()
	visible = m.draftRows()
	total = rowsOf(m.draft.Value(), width)

	lines := strings.Split(m.draft.Value(), "\n")
	cursor := m.draft.RowOffset()
	for index, line := range lines {
		if index >= m.draft.Row() {
			break
		}
		cursor += rowsOf(line, width)
	}

	first = min(max(cursor-visible+1, 0), max(total-visible, 0))
	return first, visible, total
}

// contentWidth is what a box leaves for text beside its padding and the bar.
func (m Model) contentWidth() int {
	return max(m.width-boxPadding-scrollColumn, 1)
}
