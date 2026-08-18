// Package overlay is the terminal UI a user composes a draft prompt in.
package overlay

import (
	"context"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// Submitter turns the composed draft into a prompt in the agent's input.
type Submitter interface {
	Submit(ctx context.Context, draft string) (string, error)
}

type Model struct {
	submitter Submitter
	draft     textarea.Model
}

func New(submitter Submitter) Model {
	draft := textarea.New()
	draft.Placeholder = "Schreib deinen Prompt …"
	draft.ShowLineNumbers = false
	draft.Focus()

	return Model{submitter: submitter, draft: draft}
}

type submittedMsg struct{ err error }

func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			return m, m.submit()
		}
	case submittedMsg:
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.draft, cmd = m.draft.Update(msg)
	return m, cmd
}

func (m Model) submit() tea.Cmd {
	draft := m.draft.Value()
	return func() tea.Msg {
		_, err := m.submitter.Submit(context.Background(), draft)
		return submittedMsg{err: err}
	}
}

func (m Model) View() string {
	return m.draft.View()
}
