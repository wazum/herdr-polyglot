package vimarea

// Word motions work on runes, so a draft full of umlauts counts columns the
// same way the text area draws them.

func (m *Model) moveTo(row, col int) {
	m.toFirstLine()
	for range min(max(row, 0), m.area.LineCount()-1) {
		m.area.CursorDown()
	}
	m.area.CursorStart()
	m.area.SetCursor(max(col, 0))
}

func (m *Model) wordForward() {
	lines := m.lines()
	row, col := m.Row(), m.Column()
	line := []rune(lines[row])

	index := col
	for index < len(line) && !isBlank(line[index]) {
		index++
	}
	for index < len(line) && isBlank(line[index]) {
		index++
	}
	if index < len(line) {
		m.moveTo(row, index)
		return
	}

	if row+1 < len(lines) {
		next := []rune(lines[row+1])
		start := 0
		for start < len(next) && isBlank(next[start]) {
			start++
		}
		m.moveTo(row+1, start)
		return
	}
	m.moveTo(row, max(len(line)-1, 0))
}

func (m *Model) wordBackward() {
	lines := m.lines()
	row, col := m.Row(), m.Column()
	line := []rune(lines[row])

	index := col - 1
	for index >= 0 && isBlank(index2rune(line, index)) {
		index--
	}
	if index < 0 {
		if row == 0 {
			m.moveTo(row, 0)
			return
		}
		previous := []rune(lines[row-1])
		m.moveTo(row-1, max(len(previous)-1, 0))
		return
	}

	for index > 0 && !isBlank(line[index-1]) {
		index--
	}
	m.moveTo(row, index)
}

func (m *Model) wordEnd() {
	lines := m.lines()
	row, col := m.Row(), m.Column()
	line := []rune(lines[row])

	index := col + 1
	for index < len(line) && isBlank(line[index]) {
		index++
	}
	for index+1 < len(line) && !isBlank(line[index+1]) {
		index++
	}
	m.moveTo(row, min(index, max(len(line)-1, 0)))
}

func index2rune(line []rune, index int) rune {
	if index < 0 || index >= len(line) {
		return ' '
	}
	return line[index]
}

func isBlank(r rune) bool {
	return r == ' ' || r == '\t'
}

type wordSpan int

const (
	wordOnly wordSpan = iota
	withTrailingBlanks
)

func (m *Model) deleteWord(span wordSpan) {
	m.remember()

	lines := m.lines()
	row, col := m.Row(), m.Column()
	line := []rune(lines[row])

	end := col
	for end < len(line) && !isBlank(line[end]) {
		end++
	}
	if span == withTrailingBlanks {
		for end < len(line) && isBlank(line[end]) {
			end++
		}
	}

	lines[row] = string(line[:col]) + string(line[end:])
	m.replace(lines, row, col)
}
