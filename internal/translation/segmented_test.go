package translation_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/wazum/herdr-polyglot/internal/translation"
)

type spyTranslator struct {
	mu       sync.Mutex
	requests []string
	contexts []string
}

func (s *spyTranslator) Translate(_ context.Context, text string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, text)
	return "<" + strings.TrimSpace(text) + ">", nil
}

func (s *spyTranslator) TranslateWithContext(_ context.Context, text, surrounding string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, text)
	s.contexts = append(s.contexts, surrounding)
	return "<" + strings.TrimSpace(text) + ">", nil
}

func (s *spyTranslator) sent() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.requests...)
}

func TestSegmentedTranslatesEachSentenceAndKeepsThemInOrder(t *testing.T) {
	t.Parallel()
	spy := &spyTranslator{}

	translated, err := translation.Segmented(spy).
		Translate(context.Background(), "Erster Satz. Zweiter Satz!")
	if err != nil {
		t.Fatalf("Translate returned unexpected error: %v", err)
	}
	if translated != "<Erster Satz.> <Zweiter Satz!>" {
		t.Errorf("Translate returned %q, want the sentences joined in order", translated)
	}
	if len(spy.sent()) != 2 {
		t.Errorf("translator saw %v, want one request per sentence", spy.sent())
	}
}

func TestSegmentedOnlyPaysForTheSentenceThatChanged(t *testing.T) {
	t.Parallel()
	spy := &spyTranslator{}
	segmented := translation.Segmented(spy)

	if _, err := segmented.Translate(context.Background(), "Erster Satz. Zweiter Sa"); err != nil {
		t.Fatalf("first Translate returned unexpected error: %v", err)
	}
	before := len(spy.sent())

	if _, err := segmented.Translate(context.Background(), "Erster Satz. Zweiter Satz!"); err != nil {
		t.Fatalf("second Translate returned unexpected error: %v", err)
	}

	added := spy.sent()[before:]
	if len(added) != 1 {
		t.Fatalf("second run sent %v, want only the changed sentence", added)
	}
	if strings.TrimSpace(added[0]) != "Zweiter Satz!" {
		t.Errorf("second run sent %q, want the sentence that was still being written", added[0])
	}
}

func TestSegmentedRepeatsNothingWhenTheDraftIsUnchanged(t *testing.T) {
	t.Parallel()
	spy := &spyTranslator{}
	segmented := translation.Segmented(spy)
	const draft = "Erster Satz. Zweiter Satz!"

	first, err := segmented.Translate(context.Background(), draft)
	if err != nil {
		t.Fatalf("first Translate returned unexpected error: %v", err)
	}
	before := len(spy.sent())

	second, err := segmented.Translate(context.Background(), draft)
	if err != nil {
		t.Fatalf("second Translate returned unexpected error: %v", err)
	}

	if second != first {
		t.Errorf("second run returned %q, want the same as the first %q", second, first)
	}
	if added := spy.sent()[before:]; len(added) != 0 {
		t.Errorf("second run sent %v, want nothing", added)
	}
}

func TestSegmentedGivesTheRestOfTheDraftAsUnbilledContext(t *testing.T) {
	t.Parallel()
	spy := &spyTranslator{}

	if _, err := translation.Segmented(spy).
		Translate(context.Background(), "Erster Satz. Zweiter Satz!"); err != nil {
		t.Fatalf("Translate returned unexpected error: %v", err)
	}

	spy.mu.Lock()
	contexts := append([]string(nil), spy.contexts...)
	spy.mu.Unlock()

	if len(contexts) != 2 {
		t.Fatalf("translator received %d contexts, want one per sentence", len(contexts))
	}
	if !strings.Contains(contexts[0], "Zweiter Satz!") {
		t.Errorf("first sentence had context %q, want the rest of the draft", contexts[0])
	}
	if !strings.Contains(contexts[1], "Erster Satz.") {
		t.Errorf("second sentence had context %q, want the rest of the draft", contexts[1])
	}
}

func TestSegmentedKeepsParagraphsApart(t *testing.T) {
	t.Parallel()
	spy := &spyTranslator{}

	translated, err := translation.Segmented(spy).
		Translate(context.Background(), "Erste Zeile\nZweite Zeile")
	if err != nil {
		t.Fatalf("Translate returned unexpected error: %v", err)
	}
	if translated != "<Erste Zeile>\n<Zweite Zeile>" {
		t.Errorf("Translate returned %q, want the line break preserved", translated)
	}
}

func TestSegmentedWorksWithATranslatorThatHasNoContextSupport(t *testing.T) {
	t.Parallel()
	plain := &stubProviderTranslator{}

	translated, err := translation.Segmented(plain).
		Translate(context.Background(), "Erster Satz. Zweiter Satz!")
	if err != nil {
		t.Fatalf("Translate returned unexpected error: %v", err)
	}
	if translated != "[Erster Satz.] [Zweiter Satz!]" {
		t.Errorf("Translate returned %q, want both sentences translated", translated)
	}
}

type stubProviderTranslator struct{}

func (stubProviderTranslator) Translate(_ context.Context, text string) (string, error) {
	return "[" + strings.TrimSpace(text) + "]", nil
}
