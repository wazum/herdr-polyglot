package promptflow_test

import (
	"context"
	"errors"
	"testing"

	"github.com/wazum/herdr-deepl-prompt/internal/promptflow"
)

type stubTranslator struct {
	english   string
	err       error
	seenDraft string
}

func (s *stubTranslator) Translate(_ context.Context, draft string) (string, error) {
	s.seenDraft = draft
	return s.english, s.err
}

type recordingTarget struct{ inserted []string }

func (r *recordingTarget) Insert(_ context.Context, text string) error {
	r.inserted = append(r.inserted, text)
	return nil
}

func TestSubmitTranslatesTheDraftAndInsertsTheResultIntoTheTarget(t *testing.T) {
	translator := &stubTranslator{english: "Please fix the failing test"}
	target := &recordingTarget{}

	translated, err := promptflow.New(translator, target).
		Submit(context.Background(), "Bitte behebe den fehlschlagenden Test")

	if err != nil {
		t.Fatalf("Submit returned unexpected error: %v", err)
	}
	if translator.seenDraft != "Bitte behebe den fehlschlagenden Test" {
		t.Errorf("translator received %q, want the original draft", translator.seenDraft)
	}
	if translated != "Please fix the failing test" {
		t.Errorf("Submit returned %q, want %q", translated, "Please fix the failing test")
	}
	if len(target.inserted) != 1 || target.inserted[0] != "Please fix the failing test" {
		t.Errorf("target received %v, want one insert of the translation", target.inserted)
	}
}

func TestSubmitLeavesTheTargetUntouchedWhenTranslationFails(t *testing.T) {
	translationFailure := errors.New("deepl unreachable")
	target := &recordingTarget{}

	_, err := promptflow.New(&stubTranslator{err: translationFailure}, target).
		Submit(context.Background(), "Bitte behebe den fehlschlagenden Test")

	if !errors.Is(err, translationFailure) {
		t.Errorf("Submit returned error %v, want the translation failure", err)
	}
	if len(target.inserted) != 0 {
		t.Errorf("target received %v, want nothing inserted", target.inserted)
	}
}

func TestSubmitRejectsABlankDraftWithoutCallingTheTranslator(t *testing.T) {
	translator := &stubTranslator{english: "unused"}
	target := &recordingTarget{}

	_, err := promptflow.New(translator, target).Submit(context.Background(), "   \n\t ")

	if !errors.Is(err, promptflow.ErrBlankDraft) {
		t.Errorf("Submit returned error %v, want ErrBlankDraft", err)
	}
	if translator.seenDraft != "" {
		t.Errorf("translator was called with %q, want no call at all", translator.seenDraft)
	}
	if len(target.inserted) != 0 {
		t.Errorf("target received %v, want nothing inserted", target.inserted)
	}
}
