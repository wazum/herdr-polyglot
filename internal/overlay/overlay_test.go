package overlay_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/wazum/herdr-polyglot/internal/overlay"
	"github.com/wazum/herdr-polyglot/internal/promptflow"
)

const (
	draft   = "Bitte behebe den fehlschlagenden Test"
	english = "Please fix the failing test"
)

type stubTranslator struct {
	english string
	err     error
}

func (s stubTranslator) Translate(context.Context, string) (string, error) {
	return s.english, s.err
}

type recordingTarget struct{ inserted []string }

func (r *recordingTarget) Insert(_ context.Context, text string) error {
	r.inserted = append(r.inserted, text)
	return nil
}

func newOverlay(t *testing.T, translator promptflow.Translator, target promptflow.Target) *teatest.TestModel {
	t.Helper()
	return teatest.NewTestModel(
		t,
		overlay.New(context.Background(), promptflow.New(translator, target), overlay.Options{
			Service:  "deepl",
			Language: "EN-US",
		}),
		teatest.WithInitialTermSize(80, 20),
	)
}

func TestTheOverlayShowsWhichServiceAndLanguageItWillUse(t *testing.T) {
	t.Parallel()

	overlayUnderTest := newOverlay(t, stubTranslator{english: english}, &recordingTarget{})
	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("deepl")) && bytes.Contains(out, []byte("EN-US"))
	}, teatest.WithDuration(2*time.Second))

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyEsc})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestSubmittingADraftInsertsTheEnglishTranslationIntoTheTarget(t *testing.T) {
	t.Parallel()
	target := &recordingTarget{}

	overlayUnderTest := newOverlay(t, stubTranslator{english: english}, target)
	overlayUnderTest.Type(draft)
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyEnter})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	if len(target.inserted) != 1 || target.inserted[0] != english {
		t.Errorf("target received %v, want one insert of %q", target.inserted, english)
	}
}

func TestCancellingLeavesTheTargetUntouched(t *testing.T) {
	t.Parallel()
	target := &recordingTarget{}

	overlayUnderTest := newOverlay(t, stubTranslator{english: english}, target)
	overlayUnderTest.Type(draft)
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyEsc})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	if len(target.inserted) != 0 {
		t.Errorf("target received %v, want nothing inserted", target.inserted)
	}
}

func TestAFailedTranslationKeepsTheOverlayOpenAndReportsWhy(t *testing.T) {
	t.Parallel()
	target := &recordingTarget{}

	overlayUnderTest := newOverlay(t, stubTranslator{err: errors.New("deepl unreachable")}, target)
	overlayUnderTest.Type(draft)
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyEnter})

	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("deepl unreachable"))
	}, teatest.WithDuration(2*time.Second))

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyEsc})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestABlankDraftIsNotSentAnywhere(t *testing.T) {
	t.Parallel()
	translatorThatMustNotRun := stubTranslator{err: errors.New("translator was called")}
	target := &recordingTarget{}

	overlayUnderTest := newOverlay(t, translatorThatMustNotRun, target)
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyEnter})
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyEsc})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	if len(target.inserted) != 0 {
		t.Errorf("target received %v, want nothing inserted", target.inserted)
	}
}
