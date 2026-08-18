// Package overlay is the terminal UI a user composes a draft prompt in.
package overlay

import (
	"context"
	"errors"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/wazum/herdr-polyglot/internal/promptflow"
)

// Submitter turns the composed draft into a prompt in the agent's input.
type Submitter interface {
	Submit(ctx context.Context, draft string) (string, error)
}

type stage int

const (
	composing stage = iota
	translating
)

type Model struct {
	// ctx spans the whole overlay session; Bubble Tea commands are plain
	// closures, so there is nowhere else to carry it.
	ctx       context.Context
	submitter Submitter
	draft     textarea.Model
	stage     stage
	failure   error
}

func New(ctx context.Context, submitter Submitter) Model {
	draft := textarea.New()
	draft.Placeholder = "Schreib deinen Prompt …"
	draft.ShowLineNumbers = false
	draft.SetHeight(6)
	draft.Focus()

	return Model{ctx: ctx, submitter: submitter, draft: draft}
}

type (
	promptSentMsg   struct{}
	blankDraftMsg   struct{}
	submitFailedMsg struct{ err error }
)

func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.draft.SetWidth(msg.Width - 2)
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case promptSentMsg:
		return m, tea.Quit

	case blankDraftMsg:
		m.stage = composing
		return m, nil

	case submitFailedMsg:
		m.stage = composing
		m.failure = msg.err
		return m, nil
	}

	var cmd tea.Cmd
	m.draft, cmd = m.draft.Update(msg)
	return m, cmd
}

func (m Model) handleKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		return m, tea.Quit

	case tea.KeyEnter:
		if key.Alt {
			break
		}
		if m.stage == translating {
			return m, nil
		}
		return m.startSubmit()
	}

	m.failure = nil
	var cmd tea.Cmd
	m.draft, cmd = m.draft.Update(key)
	return m, cmd
}

func (m Model) startSubmit() (tea.Model, tea.Cmd) {
	draft := m.draft.Value()
	m.stage = translating
	m.failure = nil

	return m, func() tea.Msg {
		switch _, err := m.submitter.Submit(m.ctx, draft); {
		case errors.Is(err, promptflow.ErrBlankDraft):
			return blankDraftMsg{}
		case err != nil:
			return submitFailedMsg{err: err}
		default:
			return promptSentMsg{}
		}
	}
}

var (
	titleStyle   = lipgloss.NewStyle().Bold(true)
	hintStyle    = lipgloss.NewStyle().Faint(true)
	failureStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9"))
)

func (m Model) View() string {
	status := hintStyle.Render("enter senden · alt+enter neue Zeile · esc abbrechen")
	if m.stage == translating {
		status = hintStyle.Render("übersetze …")
	}
	if m.failure != nil {
		status = failureStyle.Render(m.failure.Error())
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render("Prompt (DeepL → English)"),
		m.draft.View(),
		status,
	)
}
