package overlay

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tone int

const (
	fadedTone tone = iota
	accentTone
	brightTone
)

// Held at each end so it reads as breathing rather than spinning, and always a
// circle: a dot would look like another of the separators beside it.
var breath = []struct {
	glyph string
	tone  tone
}{
	{"○", fadedTone},
	{"○", fadedTone},
	{"◔", accentTone},
	{"◑", accentTone},
	{"◕", brightTone},
	{"●", brightTone},
	{"●", brightTone},
	{"◕", brightTone},
	{"◑", accentTone},
	{"◔", accentTone},
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
	return m.toneStyle(frame.tone).Render(frame.glyph)
}

func (m Model) toneStyle(which tone) lipgloss.Style {
	switch which {
	case brightTone:
		return m.styles.bright
	case accentTone:
		return m.styles.accent
	default:
		return m.styles.faded
	}
}
