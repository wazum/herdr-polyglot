// Package vimarea wraps a text area in the vim bindings that make sense inside
// one: modal editing, motions within the draft, and the usual line edits. There
// are no files, buffers or windows here, so nothing that acts on them exists.
package vimarea

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
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

type Model struct {
	area     textarea.Model
	modal    bool
	mode     Mode
	pending  string
	count    int
	register string
	history  []snapshot
}

func New(options ...Option) Model {
	area := textarea.New()
	area.ShowLineNumbers = false
	area.Prompt = ""
	area.Focus()

	model := Model{area: area, mode: Insert}
	for _, option := range options {
		option(&model)
	}
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
	m.toFirstLine()
	m.area.CursorStart()
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

	if m.mode == Normal {
		return m.normal(key), nil
	}
	if key.Type == tea.KeyEsc {
		m.mode = Normal
		m.pending = ""
		m.count = 0
		return m, nil
	}
	return m.delegate(msg)
}

func (m Model) delegate(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.area, cmd = m.area.Update(msg)
	return m, cmd
}

func (m *Model) send(key tea.KeyMsg) {
	m.area, _ = m.area.Update(key)
}

func (m *Model) repeat(times int, action func()) {
	for range max(times, 1) {
		action()
	}
}

// takeCount consumes a pending count such as the 3 in 3j.
func (m *Model) takeCount() int {
	count := max(m.count, 1)
	m.count = 0
	return count
}

func (m *Model) toFirstLine() {
	for m.area.Line() > 0 {
		m.area.CursorUp()
	}
}

func (m *Model) toLastLine() {
	for m.area.Line() < m.area.LineCount()-1 {
		m.area.CursorDown()
	}
}

func (m Model) lines() []string {
	return strings.Split(m.area.Value(), "\n")
}
