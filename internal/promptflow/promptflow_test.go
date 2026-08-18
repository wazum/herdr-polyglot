package promptflow_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/wazum/herdr-polyglot/internal/promptflow"
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

func TestTranslateReturnsTheEnglishWithoutTouchingTheTarget(t *testing.T) {
	translator := &stubTranslator{english: "Please fix the failing test"}
	target := &recordingTarget{}

	translated, err := promptflow.New(translator, target).
		Translate(context.Background(), "Bitte behebe den fehlschlagenden Test")
	if err != nil {
		t.Fatalf("Translate returned unexpected error: %v", err)
	}
	if translated != "Please fix the failing test" {
		t.Errorf("Translate returned %q, want the translation", translated)
	}
	if len(target.inserted) != 0 {
		t.Errorf("target received %v, want nothing inserted", target.inserted)
	}
}

func TestTranslateRejectsABlankDraft(t *testing.T) {
	translator := &stubTranslator{english: "unused"}

	_, err := promptflow.New(translator, &recordingTarget{}).Translate(context.Background(), "  ")

	if !errors.Is(err, promptflow.ErrBlankDraft) {
		t.Errorf("Translate returned %v, want ErrBlankDraft", err)
	}
	if translator.seenDraft != "" {
		t.Errorf("translator was called with %q, want no call", translator.seenDraft)
	}
}

func TestDeliverSendsAlreadyTranslatedTextWithoutTranslatingAgain(t *testing.T) {
	translator := &stubTranslator{english: "unused"}
	target := &recordingTarget{}

	err := promptflow.New(translator, target).
		Deliver(context.Background(), "Please fix the failing test")
	if err != nil {
		t.Fatalf("Deliver returned unexpected error: %v", err)
	}
	if translator.seenDraft != "" {
		t.Errorf("translator was called with %q, want no call at all", translator.seenDraft)
	}
	if len(target.inserted) != 1 || target.inserted[0] != "Please fix the failing test" {
		t.Errorf("target received %v, want the text delivered once", target.inserted)
	}
}

func TestDeliverKeepsControlCharactersOutOfTheAgentsTerminal(t *testing.T) {
	target := &recordingTarget{}

	err := promptflow.New(&stubTranslator{}, target).Deliver(
		context.Background(),
		"fix the bug\x1b]0;title\x07 and \x1b[31mred\x00 now",
	)
	if err != nil {
		t.Fatalf("Deliver returned unexpected error: %v", err)
	}
	if len(target.inserted) != 1 {
		t.Fatalf("target received %v, want one delivery", target.inserted)
	}
	delivered := target.inserted[0]
	for _, forbidden := range []string{"\x1b", "\x07", "\x00"} {
		if strings.Contains(delivered, forbidden) {
			t.Errorf("delivered %q, which still carries %q", delivered, forbidden)
		}
	}
	if !strings.Contains(delivered, "fix the bug") || !strings.Contains(delivered, "red") {
		t.Errorf("delivered %q, want the readable text kept", delivered)
	}
}

func TestDeliverKeepsAMultiLinePromptOnOneLine(t *testing.T) {
	target := &recordingTarget{}

	err := promptflow.New(&stubTranslator{}, target).
		Deliver(context.Background(), "first line\nsecond line\r\nthird")
	if err != nil {
		t.Fatalf("Deliver returned unexpected error: %v", err)
	}
	// A newline typed into an agent's input submits the prompt half written, so
	// the lines have to arrive as one.
	if delivered := target.inserted[0]; strings.ContainsAny(delivered, "\n\r") {
		t.Errorf("delivered %q, want no line breaks", delivered)
	}
	if delivered := target.inserted[0]; delivered != "first line second line third" {
		t.Errorf("delivered %q, want the lines joined by spaces", delivered)
	}
}

func TestTranslateIsNotMangledForReading(t *testing.T) {
	translator := &stubTranslator{english: "line one\nline two"}

	// The preview is read by a person, so its shape is kept; only what goes to
	// the agent is flattened.
	translated, err := promptflow.New(translator, &recordingTarget{}).
		Translate(context.Background(), "Zeile eins\nZeile zwei")
	if err != nil {
		t.Fatalf("Translate returned unexpected error: %v", err)
	}
	if translated != "line one\nline two" {
		t.Errorf("Translate returned %q, want the line break kept for reading", translated)
	}
}

// A translation service does not get to decide what the terminal does, so its
// answer is cleaned where it enters the program: the preview is drawn from it
// and the agent is given it.
func TestTranslateStripsControlCharactersButKeepsLineBreaks(t *testing.T) {
	translator := &stubTranslator{english: "fix it\x1b]0;stolen title\x07 now\nsecond line\x00"}

	translated, err := promptflow.New(translator, &recordingTarget{}).
		Translate(context.Background(), "Behebe es")
	if err != nil {
		t.Fatalf("Translate returned unexpected error: %v", err)
	}
	for _, forbidden := range []string{"\x1b", "\x07", "\x00"} {
		if strings.Contains(translated, forbidden) {
			t.Errorf("Translate returned %q, which still carries %q", translated, forbidden)
		}
	}
	if !strings.Contains(translated, "\n") {
		t.Errorf("Translate returned %q, want the line break kept for reading", translated)
	}
	if !strings.Contains(translated, "fix it") || !strings.Contains(translated, "second line") {
		t.Errorf("Translate returned %q, want the readable text kept", translated)
	}
}
