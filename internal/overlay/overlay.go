// Package overlay is the terminal UI a user composes a draft prompt in.
package overlay

import (
	"context"
	"errors"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/wazum/herdr-polyglot/internal/promptflow"
	"github.com/wazum/herdr-polyglot/internal/vimarea"
)

type Submitter interface {
	Submit(ctx context.Context, draft string) (string, error)
}

type Options struct {
	Service  string
	Language string
	// Review says the prompt is only typed into the agent's input, not sent.
	Review bool
	Vim    bool
}

type stage int

const (
	composing stage = iota
	translating
)

const (
	minContentWidth = 32
	maxContentWidth = 96
	draftHeight     = 6
)

type Model struct {
	// ctx spans the whole overlay session; Bubble Tea commands are plain
	// closures, so there is nowhere else to carry it.
	ctx       context.Context
	submitter Submitter
	options   Options
	draft     vimarea.Model
	spinner   spinner.Model
	stage     stage
	failure   error
	width     int
}

func New(ctx context.Context, submitter Submitter, options Options) Model {
	draft := vimarea.New(vimarea.WithVim(options.Vim))
	draft.SetHeight(draftHeight)

	working := spinner.New()
	working.Spinner = spinner.Dot
	working.Style = accentStyle

	model := Model{
		ctx:       ctx,
		submitter: submitter,
		options:   options,
		draft:     draft,
		spinner:   working,
	}
	model.resize(maxContentWidth)
	return model
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
		m.resize(msg.Width - 6)
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

	case spinner.TickMsg:
		if m.stage != translating {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.draft, cmd = m.draft.Update(msg)
	return m, cmd
}

func (m *Model) resize(contentWidth int) {
	m.width = min(max(contentWidth, minContentWidth), maxContentWidth)
	m.draft.SetWidth(m.width)
}

func (m Model) handleKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Type == tea.KeyCtrlC:
		return m, tea.Quit

	case key.Type == tea.KeyEsc && !m.draft.Modal():
		return m, tea.Quit

	// q closes only from normal mode, where it cannot be part of a draft.
	case key.String() == "q" && m.draft.Mode() == vimarea.Normal:
		return m, tea.Quit

	// Sending is deliberate; a bare enter belongs to the draft.
	case key.Type == tea.KeyCtrlD, key.Type == tea.KeyEnter && key.Alt:
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

	return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
		switch _, err := m.submitter.Submit(m.ctx, draft); {
		case errors.Is(err, promptflow.ErrBlankDraft):
			return blankDraftMsg{}
		case err != nil:
			return submitFailedMsg{err: err}
		default:
			return promptSentMsg{}
		}
	})
}
