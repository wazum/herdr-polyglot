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
	Submit(ctx context.Context, draft string, how promptflow.Delivery) (string, error)
	Translate(ctx context.Context, draft string) (string, error)
	Deliver(ctx context.Context, text string, how promptflow.Delivery) error
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
	// Pulse fills and empties a circle beside "live" while a translation runs.
	Pulse bool
	// NoticeLinger is how long a message stays before it goes by itself.
	NoticeLinger time.Duration
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
	// A prompt is rarely one line, so the draft keeps this many rows and scrolls.
	minDraftRows = 4

	defaultDebounce     = 600 * time.Millisecond
	defaultMaxDraft     = 2_000
	defaultNoticeLinger = 5 * time.Second

	// Herdr keeps some of a popup for its own frame: measured against 0.8.0, a
	// pane comes back three columns and two rows smaller than the size asked for.
	popupChromeColumns = 3
	popupChromeRows    = 2

	// PopupBorder is what a caller must add to a wanted height.
	PopupBorder = popupChromeRows

	dialogRows  = 10 // heading, draft box and footer
	englishRows = 5  // the pane holding the translation
	// draftFrame is the border around the draft. Its padding is not counted:
	// lipgloss measures a width as including padding but not the border.
	draftFrame = 2
	headerRows = 1
	footerRows = 1

	// PopupWidth keeps the dialog to the width of a comfortable prompt; the
	// dialog then fills the popup exactly, leaving no unused space.
	PopupWidth = 90
)

// A popup larger than this hides the agent's output for nothing.
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
	styles   styles
	draft    vimarea.Model
	spinner  spinner.Model
	stage    stage
	// failure is a broken way out, so it stays on screen.
	failure error
	// notice is something that did not work. It goes by itself; escape is quicker.
	notice  error
	notices int
	width   int

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
	// delivery can be changed while writing: sending or only typing is easier to
	// choose once the English is there to read.
	delivery promptflow.Delivery
	// resumed says the draft came from an earlier session.
	resumed bool
	// delivered says the agent has the prompt, so there is nothing left to keep.
	delivered       bool
	spent           translation.Usage
	spentKnown      bool
	beat            int
	pulsing         bool
	translationDone bool
	pane            tea.WindowSizeMsg
}

func New(ctx context.Context, prompter Prompter, options Options) Model {
	look := newStyles()

	draft := vimarea.New(
		vimarea.WithVim(options.Vim),
		vimarea.WithPlaceholder("Write your prompt in your own language …"),
		vimarea.WithStyles(look.text, look.placeholder, look.cursor),
	)
	draft.SetHeight(draftHeight)

	working := spinner.New()
	working.Spinner = spinner.Dot
	working.Style = look.accent

	if options.Debounce <= 0 {
		options.Debounce = defaultDebounce
	}
	if options.MaxDraft <= 0 {
		options.MaxDraft = defaultMaxDraft
	}
	if options.NoticeLinger <= 0 {
		options.NoticeLinger = defaultNoticeLinger
	}

	model := Model{
		ctx:      ctx,
		prompter: prompter,
		options:  options,
		styles:   look,
		draft:    draft,
		spinner:  working,
		delivery: deliveryFor(options.Review),
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
	// shown names the notice this timer belongs to, so a newer one stays up.
	noticeExpiredMsg struct{ shown int }

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
		m.pane = msg
		m.resize(msg.Width - draftFrame)
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case usageMsg:
		m.spent, m.spentKnown = msg.spent, msg.reported && msg.spent.Limit > 0
		return m, nil

	case pulseMsg:
		if !m.pulsing || msg.beat <= m.beat {
			return m, nil
		}
		beating := m.keepBreathing(msg.beat)
		return m, beating

	case promptSentMsg:
		m.forgetDraft()
		m.delivered = true
		return m, tea.Quit

	case blankDraftMsg:
		return m.raiseNotice(promptflow.ErrBlankDraft)

	case submitFailedMsg:
		return m.raiseNotice(msg.err)

	case noticeExpiredMsg:
		if msg.shown == m.notices {
			m.notice = nil
		}
		return m, nil

	case previewDueMsg:
		// Starting a translation while sending would spend a second call and
		// leave the finished preview describing the wrong draft.
		if msg.revision != m.revision || m.stage == translating {
			return m, nil
		}
		started, cmd := m.startPreview()
		breathing := started.(Model)
		return breathing, tea.Batch(cmd, breathing.beginPulse())

	case previewReadyMsg:
		if msg.request != m.requested || errors.Is(msg.err, context.Canceled) {
			return m, nil
		}
		m.previewOf, m.preview, m.previewError = msg.of, msg.text, msg.err
		m.endPulse()

		// A translation asked for in order to send it waits for the go-ahead.
		if m.stage == translating && m.options.Confirm {
			if msg.err != nil {
				return m.raiseNotice(msg.err)
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

// raiseNotice puts a message up and starts the clock that takes it down again.
func (m Model) raiseNotice(why error) (tea.Model, tea.Cmd) {
	m.stage = composing
	m.notice = why
	m.notices++

	shown := m.notices
	return m, tea.Tick(m.options.NoticeLinger, func(time.Time) tea.Msg {
		return noticeExpiredMsg{shown: shown}
	})
}

func (m *Model) resize(contentWidth int) {
	m.width = min(max(contentWidth, minContentWidth), maxContentWidth)
	m.draft.SetWidth(m.width)
	m.draft.SetHeight(m.draftRows())
}

func deliveryFor(review bool) promptflow.Delivery {
	if review {
		return promptflow.Typing
	}
	return promptflow.Sending
}

// switchDelivery is ctrl+r: whether the prompt is sent or only typed is a
// decision worth making once the English can be read, not when the popup opens.
func (m Model) switchDelivery() Model {
	if m.delivery == promptflow.Sending {
		m.delivery = promptflow.Typing
	} else {
		m.delivery = promptflow.Sending
	}
	return m
}

// switchLive is ctrl+l: translating while writing costs characters, so it can be
// turned on for a sentence that needs watching and off again.
func (m Model) switchLive() (Model, tea.Cmd) {
	m.options.Live = !m.options.Live
	m.resize(m.pane.Width - draftFrame)

	if !m.options.Live {
		m.stopPreview()
		m.pulsing = false
		return m, nil
	}

	m.revision++
	return m, m.schedulePreview()
}

func (m Model) draftRows() int {
	if m.pane.Height <= 0 {
		return draftHeight
	}

	rows := m.pane.Height - headerRows - footerRows - draftFrame
	if m.showsEnglish() {
		rows -= englishRows
	}
	return max(rows, 1)
}

// A pane too short for both boxes drops the translation rather than overflow.
func (m Model) showsEnglish() bool {
	if !m.options.Live && m.stage != confirming {
		return false
	}
	if m.pane.Height <= 0 {
		return true
	}
	return m.pane.Height-headerRows-footerRows-draftFrame-englishRows >= 1
}

func (m Model) handleKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Type == tea.KeyCtrlC:
		return m, tea.Quit

	// Escape takes the message away. It must not also close the popup.
	case key.Type == tea.KeyEsc && m.notice != nil:
		m.notice = nil
		return m, nil

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

	case key.Type == tea.KeyCtrlR:
		return m.switchDelivery(), nil

	case key.Type == tea.KeyCtrlL:
		return m.switchLive()

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

	m.failure, m.notice = nil, nil
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
		if err := m.prompter.Deliver(m.ctx, prompt, m.delivery); err != nil {
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
	m.notice = nil
	m.stopPreview()

	// Nothing to confirm about an empty draft.
	if m.options.Confirm && strings.TrimSpace(draft) == "" {
		return m.raiseNotice(promptflow.ErrBlankDraft)
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
			if err := m.prompter.Deliver(m.ctx, preview, m.delivery); err != nil {
				return submitFailedMsg{err: err}
			}
			return promptSentMsg{}
		})
	}

	return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
		switch _, err := m.prompter.Submit(m.ctx, draft, m.delivery); {
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
