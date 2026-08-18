package overlay

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/wazum/herdr-polyglot/internal/promptflow"
)

const DraftHeight = draftHeight

func DraftRows(m Model) int { return m.draftRows() }

// PulseBeat steps the animation in a test without waiting on a timer.
func PulseBeat(beat int) tea.Msg { return pulseMsg{beat: beat} }

// PromptDelivered is the flow's answer when the agent has the prompt.
func PromptDelivered() tea.Msg { return promptSentMsg{} }

// BlankDraftRefused is the flow's answer when there is nothing to send.
func BlankDraftRefused() tea.Msg { return blankDraftMsg{} }

// UsageSeen hands the model an allowance reading without waiting on a service.
func UsageSeen(spent promptflow.Usage) tea.Msg {
	return usageMsg{spent: spent, reported: true}
}

const MinDraftRows = minDraftRows

// ConfirmationOf puts the model in front of a translation of another draft, the
// state a stale confirmation would leave it in.
func ConfirmationOf(m Model, draft, translated string) Model {
	m.stage = confirming
	m.preview, m.previewOf = translated, draft
	return m
}

func IsConfirming(m Model) bool { return m.stage == confirming }

// ShowsEnglish answers whether the pane has room for the translation, live on or
// off, which is what makes ctrl+l worth pressing.
func ShowsEnglish(m Model, live bool) bool {
	m.options.Live = live
	return m.showsEnglish()
}

// PreviewShown hands the model a finished translation for a draft, so a test can
// lay out a full popup without a service.
func PreviewShown(draft, english string) tea.Msg {
	return previewReadyMsg{of: draft, text: english}
}

const (
	ScrollThumb = scrollThumb
	ScrollTrack = scrollTrack
)
