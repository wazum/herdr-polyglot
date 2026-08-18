package overlay

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/wazum/herdr-polyglot/internal/translation"
)

const DraftHeight = draftHeight

func DraftRows(m Model) int { return m.draftRows() }

// PulseBeat steps the animation in a test without waiting on a timer.
func PulseBeat(beat int) tea.Msg { return pulseMsg{beat: beat} }

// UsageSeen hands the model an allowance reading without waiting on a service.
func UsageSeen(spent translation.Usage) tea.Msg {
	return usageMsg{spent: spent, reported: true}
}
