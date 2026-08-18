// Package vimarea wraps a text area in the vim bindings that make sense inside
// one: modal editing, motions within the draft, and the usual line edits. There
// are no files, buffers or windows here, so nothing that acts on them exists.
package vimarea

import (
	"math"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Mode int

const (
	Insert Mode = iota
	Normal
)

func (m Mode) String() string {
	if m == Insert {
		return "INSERT"
	}
	return "NORMAL"
}

type Option func(*Model)

// WithVim switches the modal bindings on or off; plain editing is the default
// because modal editing is a matter of taste.
func WithVim(enabled bool) Option {
	return func(m *Model) { m.modal = enabled }
}

func WithPlaceholder(text string) Option {
	return func(m *Model) { m.area.Placeholder = text }
}

func WithStyles(text, placeholder, cursor lipgloss.Style) Option {
	return func(m *Model) {
		for _, styles := range []*textarea.Style{&m.area.FocusedStyle, &m.area.BlurredStyle} {
			styles.Base = lipgloss.NewStyle()
			// The default highlights the cursor's line, which reads as a dark
			// bar inside a bordered box.
			styles.CursorLine = text
			styles.EndOfBuffer = lipgloss.NewStyle()
			styles.Text = text
			styles.Placeholder = placeholder
		}
		m.area.Cursor.Style = cursor
	}
}

type Model struct {
	area    textarea.Model
	modal   bool
	mode    Mode
	pending string
	// count applies to the whole command; pendingCount is the one typed
	// between an operator and its motion, as in the 3 of d3w.
	count        int
	pendingCount int
	// desiredCol is the column j and k aim for, which vim remembers across
	// short lines. stickyEnd is $ holding on to the end of every line.
	desiredCol int
	stickyEnd  bool
	register   register
	history    []snapshot
	// insertRemembered says this insert session is already on the undo stack,
	// so a whole typed sentence undoes in one step.
	insertRemembered bool
	// A count before an insert command repeats what was typed once the session
	// ends, the way vim replays 3iab or 3o. typed collects it; a key that is
	// not plain text abandons the replay rather than guess at it.
	insertRepeat   int
	insertTyped    []rune
	insertNewLines bool
}

func New(options ...Option) Model {
	area := textarea.New()
	area.ShowLineNumbers = false
	area.Prompt = ""

	model := Model{area: area, mode: Insert}
	for _, option := range options {
		option(&model)
	}
	// Focus last: it stores a pointer into the struct it is called on, so any
	// style set afterwards, or any copy of the struct, would be ignored.
	model.area.Focus()
	return model
}

func (m Model) Mode() Mode { return m.mode }

func (m Model) Modal() bool { return m.modal }

func (m Model) Value() string { return m.area.Value() }

func (m Model) Row() int { return m.area.Line() }

func (m Model) Column() int {
	info := m.area.LineInfo()
	return info.StartColumn + info.ColumnOffset
}

// SetValue seeds the draft and leaves the cursor at the beginning, where
// reading and editing start.
func (m *Model) SetValue(text string) {
	m.area.SetValue(text)
	m.toRow(0)
	m.setCol(0)
}

func (m *Model) SetWidth(width int)   { m.area.SetWidth(width) }
func (m *Model) SetHeight(height int) { m.area.SetHeight(height) }
func (m *Model) Focus() tea.Cmd       { return m.area.Focus() }

func (m Model) View() string { return m.area.View() }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	key, isKey := msg.(tea.KeyMsg)
	if !isKey || !m.modal {
		return m.delegate(msg)
	}

	// Bracketed paste is text, never commands — the same call nvim makes.
	if key.Paste {
		m.rememberBefore(m.area.Value(), m.Row(), m.Column())
		m.area.InsertString(string(key.Runes))
		return m, nil
	}

	if m.mode == Normal {
		return m.normal(key), nil
	}
	return m.insert(key, msg)
}

func (m Model) insert(key tea.KeyMsg, msg tea.Msg) (Model, tea.Cmd) {
	if key.Type == tea.KeyEsc {
		m.replayInsert()
		m.mode = Normal
		m.pending, m.count, m.pendingCount = "", 0, 0
		m.insertRemembered = false
		// Vim steps back on leaving insert mode: in insert the cursor sits
		// after the character just typed.
		m.setCol(max(m.Column()-1, 0))
		m.clampToLine()
		return m, nil
	}

	before, row, col := m.area.Value(), m.Row(), m.Column()

	var cmd tea.Cmd
	m.area, cmd = m.area.Update(msg)

	// The whole typed passage becomes one undo step, as in vim.
	if !m.insertRemembered && m.area.Value() != before {
		m.rememberBefore(before, row, col)
		m.insertRemembered = true
	}
	m.recordTyped(key)
	m.desiredCol = m.Column()
	return m, cmd
}

func (m *Model) recordTyped(key tea.KeyMsg) {
	if m.insertRepeat <= 1 {
		return
	}
	if key.Type != tea.KeyRunes {
		m.insertRepeat = 1
		return
	}
	m.insertTyped = append(m.insertTyped, key.Runes...)
}

func (m *Model) replayInsert() {
	if m.insertRepeat <= 1 || len(m.insertTyped) == 0 {
		return
	}

	typed := string(m.insertTyped)
	if m.insertNewLines {
		typed = "\n" + typed
	}
	for range m.insertRepeat - 1 {
		m.area.InsertString(typed)
	}
	m.insertRepeat, m.insertTyped = 1, nil
}

func (m Model) delegate(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.area, cmd = m.area.Update(msg)
	return m, cmd
}

func (m *Model) enterInsert(times int) {
	m.mode = Insert
	m.insertRemembered = false
	m.insertRepeat, m.insertTyped, m.insertNewLines = max(times, 1), nil, false
}

// enterInsertLines is o and O, where each repetition starts a new line.
func (m *Model) enterInsertLines(times int) {
	m.enterInsert(times)
	m.insertNewLines = true
}

// enterInsertKeeping starts an insert session whose undo step is already on the
// stack, so a change command and the typing that follows undo together.
func (m *Model) enterInsertKeeping() {
	m.mode = Insert
	m.insertRemembered = true
	m.insertRepeat, m.insertTyped, m.insertNewLines = 1, nil, false
}

func (m *Model) repeat(times int, action func()) {
	for range max(times, 1) {
		action()
	}
}

// maxCount bounds a repetition. A draft is a prompt, not a file, so anything
// past this is a typo, and honouring it literally would hang the popup or run
// the machine out of memory.
const maxCount = 1_000

// takeCount consumes the count typed before a command, such as the 3 in 3j.
func (m *Model) takeCount() int {
	count := min(max(m.count, 1), maxCount)
	m.count = 0
	return count
}

func (m Model) lines() []string {
	return strings.Split(m.area.Value(), "\n")
}

func (m Model) line() []rune {
	lines := m.lines()
	row := min(max(m.Row(), 0), len(lines)-1)
	return []rune(lines[row])
}

// lastCol is where the cursor may rest in normal mode: on the last character,
// never past it.
func (m Model) lastCol() int {
	return max(len(m.line())-1, 0)
}

func (m *Model) setCol(col int) {
	m.area.CursorStart()
	if col > 0 {
		m.area.SetCursor(col)
	}
	m.desiredCol = m.Column()
	m.stickyEnd = false
}

// toRow moves between logical lines. A soft-wrapped line spans several screen
// rows, which is what the text area's own cursor movement counts, so this walks
// screen rows until the logical line changes. A line exactly as wide as the box
// wraps to a row the cursor cannot enter, so give up as soon as a step stops
// moving rather than walk on the spot for ever.
func (m *Model) toRow(target int) {
	target = min(max(target, 0), m.area.LineCount()-1)

	for m.area.Line() > target {
		if !m.step(m.area.CursorUp) {
			return
		}
	}
	for m.area.Line() < target {
		if !m.step(m.area.CursorDown) {
			return
		}
	}
}

// step reports whether a cursor movement changed anything at all.
func (m *Model) step(move func()) bool {
	line, info := m.area.Line(), m.area.LineInfo()
	move()
	after := m.area.LineInfo()
	return m.area.Line() != line || after.RowOffset != info.RowOffset
}

// toLine goes to a row and lands on the column vim would remember.
func (m *Model) toLine(target int) {
	m.toRow(target)

	wanted := m.desiredCol
	if m.stickyEnd {
		wanted = math.MaxInt
	}
	m.area.CursorStart()
	if column := min(wanted, m.lastCol()); column > 0 {
		m.area.SetCursor(column)
	}
}

// clampToLine pulls the cursor back onto the line, which is where normal mode
// keeps it.
func (m *Model) clampToLine() {
	if column := m.Column(); column > m.lastCol() {
		m.setCol(m.lastCol())
	}
}

func firstNonBlank(line []rune) int {
	for index, r := range line {
		if !isBlank(r) {
			return index
		}
	}
	return max(len(line)-1, 0)
}
