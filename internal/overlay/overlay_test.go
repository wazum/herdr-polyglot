package overlay_test

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/wazum/herdr-deepl-prompt/internal/overlay"
	"github.com/wazum/herdr-deepl-prompt/internal/promptflow"
)

type stubTranslator struct{ english string }

func (s stubTranslator) Translate(context.Context, string) (string, error) {
	return s.english, nil
}

type recordingTarget struct{ inserted string }

func (r *recordingTarget) Insert(_ context.Context, text string) error {
	r.inserted = text
	return nil
}

func TestSubmittingADraftInsertsTheEnglishTranslationIntoTheTarget(t *testing.T) {
	const english = "Please fix the failing test"
	target := &recordingTarget{}
	flow := promptflow.New(stubTranslator{english: english}, target)

	overlayUnderTest := teatest.NewTestModel(
		t,
		overlay.New(flow),
		teatest.WithInitialTermSize(80, 20),
	)
	overlayUnderTest.Type("Bitte behebe den fehlschlagenden Test")
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyEnter})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	if target.inserted != english {
		t.Fatalf("target received %q, want %q", target.inserted, english)
	}
}
