package vimarea

import "strings"

type placement int

const (
	above placement = iota
	below
)

// snapshot is what u restores: the draft plus where the cursor stood.
type snapshot struct {
	text string
	row  int
	col  int
}

func (m *Model) remember() {
	m.history = append(m.history, snapshot{text: m.area.Value(), row: m.Row(), col: m.Column()})
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

	m.toFirstLine()
	for range min(max(row, 0), m.area.LineCount()-1) {
		m.area.CursorDown()
	}
	m.area.CursorStart()
	m.area.SetCursor(max(col, 0))
}

func (m *Model) openLine(where placement) {
	m.remember()

	lines := m.lines()
	row := m.Row()
	if where == below {
		row++
	}

	lines = append(lines[:row], append([]string{""}, lines[row:]...)...)
	m.replace(lines, row, 0)
	m.mode = Insert
}

func (m *Model) deleteLine() {
	m.remember()

	lines := m.lines()
	row := m.Row()
	if len(lines) == 1 {
		m.replace([]string{""}, 0, 0)
		return
	}

	m.register = lines[row]
	lines = append(lines[:row], lines[row+1:]...)
	m.replace(lines, min(row, len(lines)-1), 0)
}

func (m *Model) clearLine() {
	m.remember()

	lines := m.lines()
	row := m.Row()
	lines[row] = ""
	m.replace(lines, row, 0)
}

func (m *Model) yankLine(count int) {
	lines := m.lines()
	row := m.Row()
	end := min(row+max(count, 1), len(lines))
	m.register = strings.Join(lines[row:end], "\n")
}

func (m *Model) paste(where placement) {
	if m.register == "" {
		return
	}
	m.remember()

	lines := m.lines()
	row := m.Row()
	if where == below {
		row++
	}

	pasted := strings.Split(m.register, "\n")
	lines = append(lines[:row], append(pasted, lines[row:]...)...)
	m.replace(lines, row, 0)
}
