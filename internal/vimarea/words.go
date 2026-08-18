package vimarea

import "unicode"

// Vim treats a run of letters, a run of punctuation and a run of blanks as
// three different things, and word motions stop wherever that changes. Working
// on runes keeps a draft full of umlauts counting the way it reads.
type runeClass int

const (
	blankClass runeClass = iota
	wordClass
	punctClass
)

func classOf(r rune) runeClass {
	switch {
	case isBlank(r):
		return blankClass
	case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_':
		return wordClass
	default:
		return punctClass
	}
}

func isBlank(r rune) bool {
	return r == ' ' || r == '\t'
}

func (m *Model) wordForward() {
	lines := m.lines()
	row, col := m.Row(), m.Column()
	line := []rune(lines[row])

	index := skipClass(line, col)
	index = skipBlanks(line, index)
	if index < len(line) {
		m.moveTo(row, index)
		return
	}

	if row+1 < len(lines) {
		next := []rune(lines[row+1])
		m.moveTo(row+1, skipBlanks(next, 0))
		return
	}
	m.moveTo(row, max(len(line)-1, 0))
}

func (m *Model) wordBackward() {
	lines := m.lines()
	row, col := m.Row(), m.Column()

	if start := backwardStart([]rune(lines[row]), col); start >= 0 {
		m.moveTo(row, start)
		return
	}
	if row == 0 {
		m.moveTo(row, 0)
		return
	}

	previous := []rune(lines[row-1])
	if start := backwardStart(previous, len(previous)); start >= 0 {
		m.moveTo(row-1, start)
		return
	}
	m.moveTo(row-1, 0)
}

// backwardStart is the beginning of the run before col, or -1 when col already
// sits at the start of the line.
func backwardStart(line []rune, col int) int {
	index := min(col, len(line)) - 1
	for index >= 0 && isBlank(line[index]) {
		index--
	}
	if index < 0 {
		return -1
	}

	class := classOf(line[index])
	for index > 0 && classOf(line[index-1]) == class {
		index--
	}
	return index
}

func (m *Model) wordEnd() {
	lines := m.lines()
	row, col := m.Row(), m.Column()
	line := []rune(lines[row])

	index := skipBlanks(line, col+1)
	if index >= len(line) {
		m.moveTo(row, max(len(line)-1, 0))
		return
	}

	class := classOf(line[index])
	for index+1 < len(line) && classOf(line[index+1]) == class {
		index++
	}
	m.moveTo(row, index)
}

// skipClass walks past the run the cursor is standing in.
func skipClass(line []rune, col int) int {
	if col >= len(line) {
		return len(line)
	}

	class := classOf(line[col])
	index := col
	for index < len(line) && classOf(line[index]) == class && class != blankClass {
		index++
	}
	return index
}

func skipBlanks(line []rune, from int) int {
	index := max(from, 0)
	for index < len(line) && isBlank(line[index]) {
		index++
	}
	return index
}

func (m *Model) moveTo(row, col int) {
	m.toRow(row)
	m.setCol(col)
}
