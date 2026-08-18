package overlay

import tea "github.com/charmbracelet/bubbletea"

// flipReading swaps the split for the whole popup and back. Reading starts at the
// top of the translation; writing shows its end, where the writing is.
func (m Model) flipReading() Model {
	m.reading = !m.reading
	m.readingFrom = 0
	return m
}

// readKey is what the keys do while the translation has the popup. Nothing here
// moves a cursor, so the arrows are free to move the text.
func (m Model) readKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	rows := m.readingRows()

	switch {
	case key.Type == tea.KeyEsc:
		return m.flipReading(), nil

	case key.Type == tea.KeyUp, key.String() == "k":
		return m.readFrom(m.readingFrom - 1), nil
	case key.Type == tea.KeyDown, key.String() == "j":
		return m.readFrom(m.readingFrom + 1), nil
	case key.Type == tea.KeyPgUp:
		return m.readFrom(m.readingFrom - rows), nil
	case key.Type == tea.KeyPgDown, key.Type == tea.KeySpace:
		return m.readFrom(m.readingFrom + rows), nil
	case key.Type == tea.KeyHome, key.String() == "g":
		return m.readFrom(0), nil
	case key.Type == tea.KeyEnd, key.String() == "G":
		return m.readFrom(m.readingTotal()), nil

	// Sending from here is the same key as anywhere else: what is on screen is what
	// the author has just read.
	case key.Type == tea.KeyCtrlD, key.Type == tea.KeyEnter && key.Alt:
		return m.flipReading().startSubmit()
	}

	// Anything else is writing, so the draft takes it back.
	writing := m.flipReading()
	return writing.handleKey(key)
}

func (m Model) readFrom(row int) Model {
	m.readingFrom = min(max(row, 0), max(m.readingTotal()-m.readingRows(), 0))
	return m
}

// readingRows is what the popup leaves for the translation once the heading, the
// footer and the box's own border have their rows.
func (m Model) readingRows() int {
	if m.pane.Height <= 0 {
		return englishRows - 2
	}
	return max(m.pane.Height-headerRows-footerRows-draftFrame, 1)
}

func (m Model) readingTotal() int {
	return rowsOf(m.preview, m.contentWidth())
}
