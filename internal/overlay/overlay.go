// Package overlay is the terminal UI a user composes a draft prompt in.
package overlay

import (
	"context"
	"errors"
	"strings"
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
	// Drafts keeps an unfinished prompt between sessions. Without one the draft
	// simply goes when the popup closes.
	Drafts Drafts
}

// Drafts is the pane's own unfinished prompt.
type Drafts interface {
	Load() string
	Save(text string) error
	Clear() error
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

	// PopupBorder is the frame herdr draws around a popup pane.
	PopupBorder = 2

	dialogRows  = 12 // outer frame, heading, draft box and footer
	englishRows = 5  // the pane holding the translation

	// PopupWidth keeps the dialog to the width of a comfortable prompt; the
	// dialog then fills the popup exactly, leaving no unused space.
	PopupWidth = 90
)

// PopupHeight is how tall the popup must be for the dialog to fit without the
// agent's output disappearing behind a pane larger than it needs to be.
func PopupHeight(live bool) int {
	height := dialogRows + PopupBorder
	if live {
		height += englishRows
	}
	return height
}

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

	// preview holds the last translation and the draft it belongs to, so a
	// prompt the author has read is delivered as it stands.
	preview      string
	previewOf    string
	previewError error
	// revision counts draft changes; a reply for an older one is discarded.
	revision  int
	requested int
	// cancelPreview stops the translation started for an older draft. It is a
	// closure, so every copy of the model cancels the same request.
	cancelPreview context.CancelFunc
	// resumed says the draft on screen was written in an earlier session, which
	// is worth saying until the author takes it over.
	resumed bool
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
	if options.Drafts != nil {
		if kept := options.Drafts.Load(); kept != "" {
			model.draft.Resume(kept)
			model.resumed = true
		}
	}
	return model
}

type (
	promptSentMsg   struct{}
	blankDraftMsg   struct{}
	submitFailedMsg struct{ err error }

	previewDueMsg   struct{ revision int }
	previewReadyMsg struct {
		request int
		// of is the draft this translation was made from. Without it the
		// translation could be paired with a draft it never belonged to.
		of   string
		text string
		err  error
	}
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
		m.forgetDraft()
		return m, tea.Quit

	case blankDraftMsg:
		m.stage = composing
		m.failure = promptflow.ErrBlankDraft
		return m, nil

	case submitFailedMsg:
		m.stage = composing
		m.failure = msg.err
		return m, nil

	case previewDueMsg:
		// Starting a translation while sending would spend a second call and
		// leave the finished preview describing the wrong draft.
		if msg.revision != m.revision || m.stage == translating {
			return m, nil
		}
		return m.startPreview()

	case previewReadyMsg:
		if msg.request != m.requested || errors.Is(msg.err, context.Canceled) {
			return m, nil
		}
		m.previewOf, m.preview, m.previewError = msg.of, msg.text, msg.err
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
		return m.close()

	// In normal mode there is nothing left for escape to do, so an empty draft
	// closes. A written one stays: escape must not throw away work.
	case key.Type == tea.KeyEsc && m.draft.Mode() == vimarea.Normal && m.draftIsBlank():
		return m.close()

	// q closes only from normal mode, where it cannot be part of a draft.
	case key.String() == "q" && m.draft.Mode() == vimarea.Normal:
		return m.close()

	// ctrl+u clears the whole draft, as it clears a line in a shell. The text
	// area would otherwise use it to delete back to the line start.
	case key.Type == tea.KeyCtrlU:
		m.draft.Clear()
		m.forgetDraft()
		m.resumed = false
		m.preview, m.previewOf, m.previewError = "", "", nil
		return m, nil

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

	if m.draft.Value() != before {
		// Once it has been edited it is this session's draft, not an old one.
		m.resumed = false
	}

	if after := m.draft.Value(); m.options.Live && after != before {
		m.revision++
		m.stopPreview()
		m.previewError = nil
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

	m.stopPreview()
	previewCtx, cancel := context.WithCancel(m.ctx)
	m.cancelPreview = cancel

	return m, func() tea.Msg {
		translated, err := m.prompter.Translate(previewCtx, draft)
		if errors.Is(err, promptflow.ErrBlankDraft) {
			return previewReadyMsg{request: request, of: draft}
		}
		return previewReadyMsg{request: request, of: draft, text: translated, err: err}
	}
}

func (m *Model) stopPreview() {
	if m.cancelPreview != nil {
		m.cancelPreview()
		m.cancelPreview = nil
	}
}

func (m Model) startSubmit() (tea.Model, tea.Cmd) {
	draft := m.draft.Value()
	m.stage = translating
	m.failure = nil
	m.stopPreview()

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

// close keeps the draft for next time. Failing to keep it is worth saying,
// because closing anyway would throw the writing away; ctrl+c is the way out.
func (m Model) close() (tea.Model, tea.Cmd) {
	if m.options.Drafts == nil {
		return m, tea.Quit
	}
	if err := m.options.Drafts.Save(m.draft.Value()); err != nil {
		m.failure = err
		return m, nil
	}
	return m, tea.Quit
}

func (m Model) forgetDraft() {
	if m.options.Drafts != nil {
		_ = m.options.Drafts.Clear()
	}
}

func (m Model) draftIsBlank() bool {
	return strings.TrimSpace(m.draft.Value()) == ""
}

func (m Model) previewIsCurrent() bool {
	return m.preview != "" && m.previewError == nil && m.previewOf == m.draft.Value()
}
