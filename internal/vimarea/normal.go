package vimarea

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) normal(key tea.KeyMsg) Model {
	if key.Type != tea.KeyRunes || len(key.Runes) != 1 {
		return m.controlKey(key)
	}

	pressed := key.Runes[0]
	if m.digit(pressed) {
		m.addDigit(pressed)
		return m
	}
	if m.pending != "" {
		return m.resolvePending(pressed)
	}
	return m.command(pressed)
}

func (m Model) digit(pressed rune) bool {
	if pressed < '0' || pressed > '9' {
		return false
	}
	// A leading 0 is the motion to the first column, not a count.
	return pressed != '0' || m.countSoFar() > 0
}

func (m Model) countSoFar() int {
	if m.pending != "" {
		return m.pendingCount
	}
	return m.count
}

func (m *Model) addDigit(pressed rune) {
	digit := int(pressed - '0')
	if m.pending != "" {
		m.pendingCount = m.pendingCount*10 + digit
		return
	}
	m.count = m.count*10 + digit
}

func (m Model) command(pressed rune) Model {
	switch pressed {
	case 'd', 'c', 'y', 'g':
		m.pending = string(pressed)
		return m

	case 'h':
		m.repeat(m.takeCount(), func() { m.setCol(max(m.Column()-1, 0)) })
	case 'l':
		m.repeat(m.takeCount(), func() { m.setCol(min(m.Column()+1, m.lastCol())) })
	case 'j':
		m.toLine(m.Row() + m.takeCount())
	case 'k':
		m.toLine(m.Row() - m.takeCount())
	case 'w':
		m.repeat(m.takeCount(), m.wordForward)
	case 'b':
		m.repeat(m.takeCount(), m.wordBackward)
	case 'e':
		m.repeat(m.takeCount(), m.wordEnd)
	case '0':
		m.setCol(0)
	case '^':
		m.setCol(firstNonBlank(m.line()))
	case '$':
		m.toLineEnd(m.takeCount())
	case 'G':
		count := m.count
		m.count = 0
		m.toLineOrLast(count)
	case 'x':
		m.deleteRunes(m.takeCount())
	case 'D':
		m.deleteToLineEnd()
	case 'C':
		m.deleteToLineEnd()
		m.toLineEndForAppend()
		m.enterInsertKeeping()
	case 'i':
		m.enterInsert(m.takeCount())
	case 'a':
		m.stepRightForAppend()
		m.enterInsert(m.takeCount())
	case 'I':
		m.setCol(firstNonBlank(m.line()))
		m.enterInsert(m.takeCount())
	case 'A':
		m.toLineEndForAppend()
		m.enterInsert(m.takeCount())
	case 'o':
		m.openLine(below, m.takeCount())
	case 'O':
		m.openLine(above, m.takeCount())
	case 'p':
		m.paste(below, m.takeCount())
	case 'P':
		m.paste(above, m.takeCount())
	case 'u':
		m.undo()
	}

	m.count = 0
	return m
}

func (m Model) resolvePending(pressed rune) Model {
	operator := m.pending
	m.pending = ""

	count := min(max(m.count, 1)*max(m.pendingCount, 1), maxCount)
	explicit := m.count
	m.count, m.pendingCount = 0, 0

	switch operator + string(pressed) {
	case "gg":
		m.toLineOrFirst(explicit)
	case "dd":
		m.deleteLines(count)
	case "dw":
		m.deleteWord(withTrailingBlanks, count)
	case "db":
		m.deleteWordBack(count)
	case "d$":
		m.deleteToLineEnd()
	case "d0":
		m.deleteToLineStart()
	case "cw":
		m.deleteWord(wordOnly, count)
		m.enterInsertKeeping()
	case "cc":
		m.clearLine()
		m.enterInsertKeeping()
	case "yy":
		m.yankLines(count)
	}
	return m
}

// controlKey handles keys that are not runes: the arrows still move, and escape
// drops whatever command was half typed.
func (m Model) controlKey(key tea.KeyMsg) Model {
	switch key.Type {
	case tea.KeyEsc:
		m.pending, m.count, m.pendingCount = "", 0, 0
	case tea.KeyUp:
		m.toLine(m.Row() - 1)
	case tea.KeyDown:
		m.toLine(m.Row() + 1)
	case tea.KeyLeft:
		m.setCol(max(m.Column()-1, 0))
	case tea.KeyRight:
		m.setCol(min(m.Column()+1, m.lastCol()))
	}
	return m
}

func (m *Model) toLineEnd(count int) {
	if count > 1 {
		m.toRow(m.Row() + count - 1)
	}
	m.area.CursorStart()
	if last := m.lastCol(); last > 0 {
		m.area.SetCursor(last)
	}
	m.desiredCol, m.stickyEnd = m.Column(), true
}

// toLineEndForAppend puts the cursor after the last character, where A types.
func (m *Model) toLineEndForAppend() {
	m.area.CursorEnd()
	m.desiredCol, m.stickyEnd = m.Column(), true
}

// toLineOrLast is G: with a count it goes to that line, without one to the last.
func (m *Model) toLineOrLast(count int) {
	target := m.area.LineCount() - 1
	if count > 0 {
		target = count - 1
	}
	m.toRow(target)
	m.setCol(firstNonBlank(m.line()))
}

func (m *Model) toLineOrFirst(count int) {
	target := 0
	if count > 1 {
		target = count - 1
	}
	m.toRow(target)
	m.setCol(firstNonBlank(m.line()))
}

// stepRightForAppend moves past the character under the cursor so a types after
// it.
func (m *Model) stepRightForAppend() {
	column := m.Column() + 1
	m.area.CursorStart()
	if column > 0 {
		m.area.SetCursor(column)
	}
	m.desiredCol, m.stickyEnd = m.Column(), false
}
