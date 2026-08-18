package overlay_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/wazum/herdr-polyglot/internal/overlay"
	"github.com/wazum/herdr-polyglot/internal/promptflow"
)

// markingTranslator answers with English that says which draft it came from, so a
// test can tell one translation from another.
type markingTranslator struct{}

func (markingTranslator) Translate(_ context.Context, draft string) (string, error) {
	return "EN(" + draft + ")", nil
}

// Reading the English before it goes is only worth anything if what goes is what
// was read. Writing after the confirmation therefore takes the question back.
func TestWritingAfterAConfirmationNeverDeliversTheOlderEnglish(t *testing.T) {
	t.Parallel()
	target := &recordingTarget{}

	overlayUnderTest := confirmingOverlay(t, markingTranslator{}, target)
	overlayUnderTest.Type("erste Fassung")
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlD})

	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("EN(erste Fassung)"))
	}, teatest.WithDuration(2*time.Second))

	// The draft moves on while the English of the older one is on screen.
	overlayUnderTest.Type(" und mehr")
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlD})

	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("EN(erste Fassung und mehr)"))
	}, teatest.WithDuration(2*time.Second))

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlD})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	if len(target.inserted) != 1 || target.inserted[0] != "EN(erste Fassung und mehr)" {
		t.Errorf("target received %v, want only the English of the draft as it stood", target.inserted)
	}
}

// Throwing the draft away while its English waits leaves nothing to send, and
// saying so beats handing the agent an empty prompt.
func TestDiscardingADraftDuringConfirmationSendsNothing(t *testing.T) {
	t.Parallel()
	target := &recordingTarget{}

	overlayUnderTest := confirmingOverlay(t, markingTranslator{}, target)
	overlayUnderTest.Type("wieder weg")
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlD})

	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("EN(wieder weg)"))
	}, teatest.WithDuration(2*time.Second))

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlU})
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlD})

	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("the draft is empty"))
	}, teatest.WithDuration(2*time.Second))

	if len(target.inserted) != 0 {
		t.Errorf("target received %v, want nothing after the draft was thrown away", target.inserted)
	}

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

// A translation is only ever delivered for the draft it was made from, whatever
// order the model is driven in.
func TestAStaleConfirmationIsTranslatedAgainRatherThanDelivered(t *testing.T) {
	t.Parallel()
	target := &recordingTarget{}
	flow := promptflow.New(markingTranslator{}, target, target)

	var model tea.Model = overlay.New(context.Background(), flow, overlay.Options{
		Service: "deepl", Language: "EN-US", Confirm: true,
	})
	model, _ = model.Update(tea.WindowSizeMsg{Width: 87, Height: 17})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("eine Fassung")})

	// Confirmation reached with a translation that belongs to an older draft.
	var next tea.Model = overlay.ConfirmationOf(
		model.(overlay.Model), "eine andere Fassung", "EN(eine andere Fassung)")

	next, cmd := next.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	drive(next, cmd)

	for _, delivered := range target.inserted {
		if delivered == "EN(eine andere Fassung)" {
			t.Errorf("target received %q, the English of a draft that is not there", delivered)
		}
	}
}

// drive runs commands the way the runtime would, feeding every message back into
// the model until nothing is left to do.
func drive(model tea.Model, cmd tea.Cmd) tea.Model {
	pending := []tea.Cmd{cmd}
	for rounds := 0; len(pending) > 0 && rounds < 32; rounds++ {
		next := pending[0]
		pending = pending[1:]
		if next == nil {
			continue
		}

		switch msg := next().(type) {
		case nil:
		case tea.BatchMsg:
			pending = append(pending, msg...)
		default:
			var following tea.Cmd
			model, following = model.Update(msg)
			pending = append(pending, following)
		}
	}
	return model
}
