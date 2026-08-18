package overlay_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/muesli/termenv"

	"github.com/wazum/herdr-polyglot/internal/overlay"
	"github.com/wazum/herdr-polyglot/internal/promptflow"
)

// plainModel renders without colour, so a test can read the footer as text.
func plainModel(t *testing.T, options overlay.Options) tea.Model {
	t.Helper()

	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(previous) })

	target := &recordingTarget{}
	flow := promptflow.New(stubTranslator{english: english}, target, target)
	var model tea.Model = overlay.New(context.Background(), flow, options)
	model, _ = model.Update(tea.WindowSizeMsg{Width: 87, Height: 17})
	return model
}

// Something that did not work is worth saying, but it is not a dead end: the way
// out of the message is escape, not closing the popup and losing the draft.
func TestANoticeOffersEscapeRatherThanClosing(t *testing.T) {
	model := plainModel(t, overlay.Options{Service: "deepl", Language: "EN-US", Vim: true})
	model, _ = model.Update(overlay.BlankDraftRefused())

	footer := lastLine(model.View())
	if !strings.Contains(footer, "nothing to translate — the draft is empty") {
		t.Fatalf("the footer does not say what went wrong: %q", footer)
	}
	if !strings.Contains(footer, "esc dismiss") {
		t.Errorf("the footer does not offer escape to take the message away: %q", footer)
	}
	if strings.Contains(footer, "ctrl+c") {
		t.Errorf("the footer tells the author to close the popup over a blank draft: %q", footer)
	}
}

// Nobody reads at the same speed, so the message stays a while, then goes.
func TestANoticeGoesByItself(t *testing.T) {
	model := plainModel(t, overlay.Options{
		Service: "deepl", Language: "EN-US", Vim: true, NoticeLinger: 20 * time.Millisecond,
	})

	model, linger := model.Update(overlay.BlankDraftRefused())
	if linger == nil {
		t.Fatal("the notice was put up with nothing to take it down again")
	}

	model, _ = model.Update(linger())
	if footer := lastLine(model.View()); strings.Contains(footer, "nothing to translate — the draft is empty") {
		t.Errorf("the notice is still there after its time: %q", footer)
	}
}

func TestEscapeTakesTheNoticeAwayWithoutClosingThePopup(t *testing.T) {
	t.Parallel()

	overlayUnderTest := newOverlay(t, stubTranslator{english: english}, &recordingTarget{})
	overlayUnderTest.Send(overlay.BlankDraftRefused())

	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("nothing to translate — the draft is empty"))
	}, teatest.WithDuration(2*time.Second))

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyEsc})
	overlayUnderTest.Type("weiter geht es")

	// A closed popup would never show this.
	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("weiter geht es"))
	}, teatest.WithDuration(2*time.Second))

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

// A kept draft that cannot be read must be said out loud: writing here and
// closing would write over whatever is in that file.
func TestADraftThatCannotBeReadIsReported(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{loadError: errors.New("permission denied")}
	overlayUnderTest := newOverlayWith(t, stubTranslator{english: english}, &recordingTarget{},
		overlay.Options{Service: "deepl", Language: "EN-US", Drafts: drafts})

	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("permission denied"))
	}, teatest.WithDuration(2*time.Second))

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}
