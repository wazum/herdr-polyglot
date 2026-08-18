package overlay

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Filling and emptying again reads as breathing rather than spinning.
var pulseFrames = []string{"○", "◔", "◑", "◕", "●", "◕", "◑", "◔"}

const pulseInterval = 110 * time.Millisecond

type pulseMsg struct{ beat int }

func (m Model) startPulse() tea.Cmd {
	if !m.options.Pulse || m.pulsing {
		return nil
	}
	return m.nextBeat(m.beat)
}

func (m Model) nextBeat(beat int) tea.Cmd {
	return tea.Tick(pulseInterval, func(time.Time) tea.Msg {
		return pulseMsg{beat: beat + 1}
	})
}

func (m Model) pulseGlyph() string {
	if !m.pulsing {
		return pulseFrames[0]
	}
	return pulseFrames[m.beat%len(pulseFrames)]
}
