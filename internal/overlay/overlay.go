// Package overlay is the terminal UI a user composes a draft prompt in.
package overlay

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/wazum/herdr-polyglot/internal/promptflow"
	"github.com/wazum/herdr-polyglot/internal/translation"
	"github.com/wazum/herdr-polyglot/internal/vimarea"
)

// Prompter is the draft's way out: translated for reading, delivered for the
// agent, or both at once when nothing has been previewed.
type Prompter interface {
	Submit(ctx context.Context, draft string) (string, error)
	Translate(ctx context.Context, draft string) (string, error)
	Deliver(ctx context.Context, text string) error
	Usage(ctx context.Context) (translation.Usage, bool, error)
}

type Options struct {
	Service  string
	Language string
	// Review says the prompt is only typed into the agent's input, not sent.
	Review bool
	Vim    bool
	// Live translates the draft while it is written instead of only on send.
	Live bool
	// Confirm shows the English and waits for a second key before delivering.
	Confirm  bool
	Debounce time.Duration
	// MaxDraft is how long a prompt may get before the box says something. This
	// is a place for prompts, not for pasted files.
	MaxDraft int
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
	// confirming is the finished translation waiting to be let go.
	confirming
)

const (
	minContentWidth = 32
	maxContentWidth = 96
	draftHeight     = 6

	defaultDebounce = 600 * time.Millisecond
	defaultMaxDraft = 2_000

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
	resumed    bool
	spent      translation.Usage
	spentKnown bool
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
	if options.MaxDraft <= 0 {
		options.MaxDraft = defaultMaxDraft
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

	usageMsg struct {
		spent    translation.Usage
		reported bool
	}
)

func (m Model) Init() tea.Cmd {
	// The allowance is asked for after the box is already on screen, so nothing
	// waits for the network.
	return tea.Batch(textarea.Blink, m.askUsage())
}

func (m Model) askUsage() tea.Cmd {
	return func() tea.Msg {
		spent, reported, err := m.prompter.Usage(m.ctx)
		if err != nil {
			// A service that will not say is simply not shown.
			return usageMsg{}
		}
		return usageMsg{spent: spent, reported: reported}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg.Width - 6)
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case usageMsg:
		m.spent, m.spentKnown = msg.spent, msg.reported && msg.spent.Limit > 0
		return m, nil

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

		// A translation asked for in order to send it waits for the go-ahead.
		if m.stage == translating && m.options.Confirm {
			if msg.err != nil {
				m.stage, m.failure = composing, msg.err
				return m, nil
			}
			m.stage = confirming
		}
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

	case key.Type == tea.KeyEsc && m.stage == confirming:
		m.stage = composing
		return m, nil

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

	// Sending is deliberate; a bare enter belongs to the draft. Only ctrl+d
	// sends: an alt chord reaches the program as an escape followed by the key,
	// so escape and then enter, typed quickly, would send a draft by accident —
	// and on macOS an alt chord often does not arrive at all.
	case key.Type == tea.KeyCtrlD:
		switch m.stage {
		case translating:
			return m, nil
		case confirming:
			return m.deliverPreview()
		default:
			return m.startSubmit()
		}
	}

	m.failure = nil
	before := m.draft.Value()

	var cmd tea.Cmd
	m.draft, cmd = m.draft.Update(key)

	if m.draft.Value() != before {
		// Once it has been edited it is this session's draft, not an old one.
		m.resumed = false
	}

	// Translating a pasted wall of text again after every pause would spend the
	// allowance on something this box is not for.
	if after := m.draft.Value(); m.options.Live && after != before && !m.draftIsTooLong() {
		m.revision++
		m.stopPreview()
		m.previewError = nil
		return m, tea.Batch(cmd, m.schedulePreview())
	}
	return m, cmd
}

func (m Model) draftIsTooLong() bool {
	return len([]rune(m.draft.Value())) > m.options.MaxDraft
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

// translateForConfirmation reuses the preview machinery, so the finished text
// arrives as a reply and the confirmation shows it.
func (m Model) translateForConfirmation() (tea.Model, tea.Cmd) {
	model, cmd := m.startPreview()
	confirming := model.(Model)
	confirming.stage = translating
	return confirming, tea.Batch(m.spinner.Tick, cmd)
}

func (m Model) deliverPreview() (tea.Model, tea.Cmd) {
	prompt := m.preview
	m.stage = translating

	return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
		if err := m.prompter.Deliver(m.ctx, prompt); err != nil {
			return submitFailedMsg{err: err}
		}
		return promptSentMsg{}
	})
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

	// Nothing to confirm about an empty draft.
	if m.options.Confirm && strings.TrimSpace(draft) == "" {
		m.stage, m.failure = composing, promptflow.ErrBlankDraft
		return m, nil
	}

	// Read once, sent as it stands: a translation already on screen is what the
	// author means, whether it was asked for or arrived while writing.
	if m.options.Confirm && m.previewIsCurrent() {
		m.stage = confirming
		return m, nil
	}
	if m.options.Confirm {
		return m.translateForConfirmation()
	}

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
	if m.options.Drafts == nil {
		return
	}
	if err := m.options.Drafts.Clear(); err != nil {
		// The prompt is already delivered, so this cannot stop anything; herdr
		// keeps what a plugin writes here in its log.
		fmt.Fprintln(os.Stderr, "polyglot: the sent draft could not be forgotten:", err)
	}
}

func (m Model) draftIsBlank() bool {
	return strings.TrimSpace(m.draft.Value()) == ""
}

func (m Model) previewIsCurrent() bool {
	return m.preview != "" && m.previewError == nil && m.previewOf == m.draft.Value()
}
