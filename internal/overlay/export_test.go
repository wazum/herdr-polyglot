package overlay

import tea "github.com/charmbracelet/bubbletea"

const DraftHeight = draftHeight

func DraftRows(m Model) int { return m.draftRows() }

// PulseBeat steps the animation in a test without waiting on a timer.
func PulseBeat(beat int) tea.Msg { return pulseMsg{beat: beat} }
