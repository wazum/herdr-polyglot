package vimarea

import "strings"

type placement int

const (
	above placement = iota
	below
)

// register holds the last delete or yank. Vim keeps whole lines apart from
// pieces of a line, because that decides where a paste lands.
type register struct {
	text     string
	linewise bool
	filled   bool
}

// snapshot is what u restores: the draft plus where the cursor stood.
type snapshot struct {
	text string
	row  int
	col  int
}

func (m *Model) rememberBefore(text string, row, col int) {
	m.history = append(m.history, snapshot{text: text, row: row, col: col})
}

func (m *Model) remember() {
	m.rememberBefore(m.area.Value(), m.Row(), m.Column())
}

func (m *Model) undo() {
	if len(m.history) == 0 {
		return
	}
	last := m.history[len(m.history)-1]
	m.history = m.history[:len(m.history)-1]
	m.replace(strings.Split(last.text, "\n"), last.row, last.col)
}

// replace swaps the whole draft and puts the cursor back, since setting a value
// leaves it at the end of the text.
func (m *Model) replace(lines []string, row, col int) {
	m.area.SetValue(strings.Join(lines, "\n"))
	m.toRow(row)
	m.setCol(max(col, 0))
}

func (m *Model) openLine(where placement, count int) {
	m.remember()

	lines := m.lines()
	row := m.Row()
	if where == below {
		row++
	}

	lines = append(lines[:row], append([]string{""}, lines[row:]...)...)
	m.replace(lines, row, 0)

	// The undo step is already on the stack, but a count still repeats the
	// typing, so start a session that knows both.
	m.enterInsertLines(count)
	m.insertRemembered = true
}

func (m *Model) deleteLines(count int) {
	m.remember()

	lines := m.lines()
	row := m.Row()
	end := min(row+max(count, 1), len(lines))
	m.register = register{text: strings.Join(lines[row:end], "\n"), linewise: true, filled: true}

	lines = append(lines[:row], lines[end:]...)
	if len(lines) == 0 {
		lines = []string{""}
	}

	row = min(row, len(lines)-1)
	m.replace(lines, row, 0)
	m.setCol(firstNonBlank(m.line()))
}

func (m *Model) clearLine() {
	m.remember()

	lines := m.lines()
	row := m.Row()
	lines[row] = ""
	m.replace(lines, row, 0)
}

func (m *Model) yankLines(count int) {
	lines := m.lines()
	row := m.Row()
	end := min(row+max(count, 1), len(lines))
	m.register = register{text: strings.Join(lines[row:end], "\n"), linewise: true, filled: true}
}

func (m *Model) paste(where placement, count int) {
	if !m.register.filled {
		return
	}
	m.remember()

	if m.register.linewise {
		m.pasteLines(where, count)
		return
	}
	m.pastePiece(where, count)
}

func (m *Model) pasteLines(where placement, count int) {
	lines := m.lines()
	row := m.Row()
	if where == below {
		row++
	}

	var pasted []string
	for range max(count, 1) {
		pasted = append(pasted, strings.Split(m.register.text, "\n")...)
	}

	lines = append(lines[:row], append(pasted, lines[row:]...)...)
	m.replace(lines, row, 0)
	m.setCol(firstNonBlank(m.line()))
}

// pastePiece puts text back inside a line: p after the cursor, P at it.
func (m *Model) pastePiece(where placement, count int) {
	lines := m.lines()
	row := m.Row()
	line := m.line()

	at := m.Column()
	if where == below && len(line) > 0 {
		at++
	}
	at = min(at, len(line))

	pasted := strings.Repeat(m.register.text, max(count, 1))
	lines[row] = string(line[:at]) + pasted + string(line[at:])
	m.replace(lines, row, at+len([]rune(pasted))-1)
}

func (m *Model) deleteRunes(count int) {
	line := m.line()
	col := m.Column()
	if col >= len(line) {
		return
	}
	m.remember()

	end := min(col+max(count, 1), len(line))
	lines := m.lines()
	row := m.Row()

	m.register = register{text: string(line[col:end]), filled: true}
	lines[row] = string(line[:col]) + string(line[end:])
	m.replace(lines, row, min(col, max(len([]rune(lines[row]))-1, 0)))
}

func (m *Model) deleteToLineEnd() {
	line := m.line()
	col := m.Column()
	if col >= len(line) {
		return
	}
	m.remember()

	lines := m.lines()
	row := m.Row()
	m.register = register{text: string(line[col:]), filled: true}
	lines[row] = string(line[:col])
	m.replace(lines, row, max(col-1, 0))
}

func (m *Model) deleteToLineStart() {
	line := m.line()
	col := m.Column()
	if col == 0 {
		return
	}
	m.remember()

	lines := m.lines()
	row := m.Row()
	m.register = register{text: string(line[:col]), filled: true}
	lines[row] = string(line[col:])
	m.replace(lines, row, 0)
}
