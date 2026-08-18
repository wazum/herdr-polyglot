package vimarea

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) normal(key tea.KeyMsg) Model {
	if key.Type != tea.KeyRunes || len(key.Runes) != 1 {
		return m.arrows(key)
	}

	pressed := key.Runes[0]
	if pressed >= '1' && pressed <= '9' || (pressed == '0' && m.count > 0) {
		m.count = m.count*10 + int(pressed-'0')
		return m
	}

	if m.pending != "" {
		return m.resolvePending(pressed)
	}

	switch pressed {
	case 'd', 'c', 'y', 'g':
		m.pending = string(pressed)
		return m

	case 'h':
		m.repeat(m.takeCount(), func() { m.send(tea.KeyMsg{Type: tea.KeyLeft}) })
	case 'l':
		m.repeat(m.takeCount(), func() { m.send(tea.KeyMsg{Type: tea.KeyRight}) })
	case 'j':
		m.repeat(m.takeCount(), m.area.CursorDown)
	case 'k':
		m.repeat(m.takeCount(), m.area.CursorUp)
	case 'w':
		m.repeat(m.takeCount(), m.wordForward)
	case 'b':
		m.repeat(m.takeCount(), m.wordBackward)
	case 'e':
		m.repeat(m.takeCount(), m.wordEnd)
	case '0':
		m.area.CursorStart()
	case '^':
		m.area.CursorStart()
	case '$':
		m.toLineEnd()
	case 'G':
		m.toLastLine()
		m.area.CursorStart()

	case 'i':
		m.mode = Insert
	case 'a':
		m.send(tea.KeyMsg{Type: tea.KeyRight})
		m.mode = Insert
	case 'I':
		m.area.CursorStart()
		m.mode = Insert
	case 'A':
		m.area.CursorEnd()
		m.mode = Insert
	case 'o':
		m.openLine(below)
	case 'O':
		m.openLine(above)

	case 'x':
		m.repeat(m.takeCount(), func() { m.send(tea.KeyMsg{Type: tea.KeyDelete}) })
	case 'D':
		m.send(tea.KeyMsg{Type: tea.KeyCtrlK})
	case 'C':
		m.send(tea.KeyMsg{Type: tea.KeyCtrlK})
		m.mode = Insert
	case 'p':
		m.paste(below)
	case 'P':
		m.paste(above)
	case 'u':
		m.undo()
	}

	m.count = 0
	return m
}

func (m Model) resolvePending(pressed rune) Model {
	operator := m.pending
	m.pending = ""
	count := m.takeCount()

	switch operator + string(pressed) {
	case "gg":
		m.toFirstLine()
		m.area.CursorStart()
	case "dd":
		m.repeat(count, m.deleteLine)
	case "dw":
		m.repeat(count, func() { m.deleteWord(withTrailingBlanks) })
	case "db":
		m.repeat(count, func() { m.send(tea.KeyMsg{Type: tea.KeyBackspace, Alt: true}) })
	case "d$":
		m.send(tea.KeyMsg{Type: tea.KeyCtrlK})
	case "d0":
		m.send(tea.KeyMsg{Type: tea.KeyCtrlU})
	case "cw":
		m.repeat(count, func() { m.deleteWord(wordOnly) })
		m.mode = Insert
	case "cc":
		m.clearLine()
		m.mode = Insert
	case "yy":
		m.yankLine(count)
	}
	return m
}

// arrows keeps the cursor keys working in normal mode, as vim does.
func (m Model) arrows(key tea.KeyMsg) Model {
	switch key.Type {
	case tea.KeyUp:
		m.area.CursorUp()
	case tea.KeyDown:
		m.area.CursorDown()
	case tea.KeyLeft, tea.KeyRight:
		m.send(key)
	}
	return m
}

func (m *Model) toLineEnd() {
	m.area.CursorEnd()
	if length := len([]rune(m.lines()[m.area.Line()])); length > 0 {
		m.area.SetCursor(length - 1)
	}
}
