package vimarea

type wordSpan int

const (
	// wordOnly is what cw deletes: vim treats cw as ce, so the blanks after
	// the word stay put.
	wordOnly wordSpan = iota
	withTrailingBlanks
)

func (m *Model) deleteWord(span wordSpan, count int) {
	line := m.line()
	col := m.Column()
	if col >= len(line) {
		return
	}

	words := max(count, 1)
	end := col
	for word := range words {
		end = skipClass(line, end)
		if span == withTrailingBlanks || word < words-1 {
			end = skipBlanks(line, end)
		}
	}
	m.cutInLine(col, end)
}

func (m *Model) deleteWordBack(count int) {
	line := m.line()
	end := m.Column()

	start := end
	for range max(count, 1) {
		previous := backwardStart(line, start)
		if previous < 0 {
			start = 0
			break
		}
		start = previous
	}
	m.cutInLine(start, end)
}

func (m *Model) cutInLine(from, to int) {
	if from >= to {
		return
	}
	m.remember()

	line := m.line()
	lines := m.lines()
	row := m.Row()

	from, to = max(from, 0), min(to, len(line))
	m.register = register{text: string(line[from:to]), filled: true}
	lines[row] = string(line[:from]) + string(line[to:])
	m.replace(lines, row, from)
}
