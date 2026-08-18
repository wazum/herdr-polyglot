// Package overlay is the terminal UI a user composes a draft prompt in.
package overlay

import (
	"context"
	"errors"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/wazum/herdr-polyglot/internal/promptflow"
	"github.com/wazum/herdr-polyglot/internal/vimarea"
)

// Prompter is the draft's way out: translated for reading, delivered for the
// agent, or both at once when nothing has been previewed.
type Prompter interface {
	Submit(ctx context.Context, draft string) (string, error)
	Translate(ctx context.Context, draft string) (string, error)
	Deliver(ctx context.Context, text string) error
}

type Options struct {
	Service  string
	Language string
	// Review says the prompt is only typed into the agent's input, not sent.
	Review bool
	Vim    bool
	// Live translates the draft while it is written instead of only on send.
	Live     bool
	Debounce time.Duration
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

	defaultDebounce = 600 * time.Millisecond
)

type Model struct {
	// ctx spans the whole overlay session; Bubble Tea commands are plain
	// closures, so there is nowhere else to carry it.
	ctx      context.Context
	prompter Prompter
	options  Options
	draft    vimarea.Model
	spinner  spinner.Model
	stage    stage
	failure  error
	width    int
	height   int

	// preview holds the last translation and the draft it belongs to, so a
	// prompt the author has read is delivered as it stands.
	preview      string
	previewOf    string
	previewError error
	// revision counts draft changes; a reply for an older one is discarded.
	revision  int
	requested int
}

func New(ctx context.Context, prompter Prompter, options Options) Model {
	draft := vimarea.New(
		vimarea.WithVim(options.Vim),
		vimarea.WithPlaceholder("Write your prompt in your own language …"),
		vimarea.WithStyles(textStyle, placeholderStyle, cursorStyle),
	)
	draft.SetHeight(draftHeight)

	working := spinner.New()
	working.Spinner = spinner.Dot
	working.Style = accentStyle

	if options.Debounce <= 0 {
		options.Debounce = defaultDebounce
	}

	model := Model{
		ctx:      ctx,
		prompter: prompter,
		options:  options,
		draft:    draft,
		spinner:  working,
	}
	model.resize(maxContentWidth)
	return model
}

type (
	promptSentMsg   struct{}
	blankDraftMsg   struct{}
	submitFailedMsg struct{ err error }

	previewDueMsg   struct{ revision int }
	previewReadyMsg struct {
		request int
		text    string
		err     error
	}
)

func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
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

	case previewDueMsg:
		if msg.revision != m.revision {
			return m, nil
		}
		return m.startPreview()

	case previewReadyMsg:
		if msg.request != m.requested {
			return m, nil
		}
		m.preview, m.previewError = msg.text, msg.err
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
	before := m.draft.Value()

	var cmd tea.Cmd
	m.draft, cmd = m.draft.Update(key)

	if after := m.draft.Value(); m.options.Live && after != before {
		m.revision++
		return m, tea.Batch(cmd, m.schedulePreview())
	}
	return m, cmd
}

func (m Model) schedulePreview() tea.Cmd {
	revision := m.revision
	return tea.Tick(m.options.Debounce, func(time.Time) tea.Msg {
		return previewDueMsg{revision: revision}
	})
}

func (m Model) startPreview() (tea.Model, tea.Cmd) {
	draft := m.draft.Value()
	m.requested++
	request := m.requested
	m.previewOf = draft

	return m, func() tea.Msg {
		translated, err := m.prompter.Translate(m.ctx, draft)
		if errors.Is(err, promptflow.ErrBlankDraft) {
			return previewReadyMsg{request: request}
		}
		return previewReadyMsg{request: request, text: translated, err: err}
	}
}

func (m Model) startSubmit() (tea.Model, tea.Cmd) {
	draft := m.draft.Value()
	m.stage = translating
	m.failure = nil

	// A preview the author has just read needs no second translation.
	if m.previewIsCurrent() {
		preview := m.preview
		return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
			if err := m.prompter.Deliver(m.ctx, preview); err != nil {
				return submitFailedMsg{err: err}
			}
			return promptSentMsg{}
		})
	}

	return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
		switch _, err := m.prompter.Submit(m.ctx, draft); {
		case errors.Is(err, promptflow.ErrBlankDraft):
			return blankDraftMsg{}
		case err != nil:
			return submitFailedMsg{err: err}
		default:
			return promptSentMsg{}
		}
	})
}

func (m Model) previewIsCurrent() bool {
	return m.preview != "" && m.previewError == nil && m.previewOf == m.draft.Value()
}
