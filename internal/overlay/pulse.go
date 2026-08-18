package overlay

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// The circle grows and fades and grows again, held for a moment at each end, so
// it reads as breathing rather than spinning. Brightness follows the size, which
// is as close to a fade as the terminal palette gets.
var breath = []struct {
	glyph string
	tone  lipgloss.Style
}{
	{"·", fadedStyle},
	{"·", fadedStyle},
	{"○", fadedStyle},
	{"◔", accentStyle},
	{"◑", accentStyle},
	{"◕", brightStyle},
	{"●", brightStyle},
	{"●", brightStyle},
	{"◕", brightStyle},
	{"◑", accentStyle},
	{"◔", accentStyle},
	{"○", fadedStyle},
}

const breathStep = 120 * time.Millisecond

type pulseMsg struct{ beat int }

func (m *Model) beginPulse() tea.Cmd {
	if !m.options.Live || !m.options.Pulse {
		return nil
	}

	m.translationDone = false
	if m.pulsing {
		return nil
	}
	m.pulsing, m.beat = true, 0
	return m.nextBeat(0)
}

// A translation can answer in a blink, and a single frame of animation is not
// something anybody sees, so the breath is allowed to finish.
func (m *Model) endPulse() {
	m.translationDone = true
}

func (m *Model) keepBreathing(beat int) tea.Cmd {
	m.beat = beat
	if m.translationDone && beat%len(breath) == 0 {
		m.pulsing = false
		return nil
	}
	return m.nextBeat(beat)
}

func (m Model) nextBeat(beat int) tea.Cmd {
	return tea.Tick(breathStep, func(time.Time) tea.Msg {
		return pulseMsg{beat: beat + 1}
	})
}

func (m Model) pulseGlyph() string {
	frame := breath[0]
	if m.pulsing {
		frame = breath[m.beat%len(breath)]
	}
	return frame.tone.Render(frame.glyph)
}
