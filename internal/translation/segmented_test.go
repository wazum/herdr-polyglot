package translation_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

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

func TestSegmentedGivesEachSentenceWhatCameBeforeItAsUnbilledContext(t *testing.T) {
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
	if contexts[0] != "" {
		t.Errorf("the first sentence had context %q, want none", contexts[0])
	}
	if contexts[1] != "Erster Satz." {
		t.Errorf("the second sentence had context %q, want the sentence before it", contexts[1])
	}
}

// What came before a sentence is part of its meaning, so changing it has to
// retranslate what follows.
func TestSegmentedRetranslatesLaterSentencesWhenAnEarlierOneChanges(t *testing.T) {
	t.Parallel()
	spy := &spyTranslator{}
	segmented := translation.Segmented(spy)

	if _, err := segmented.Translate(context.Background(), "Sie ist hier. Die Bank ist zu."); err != nil {
		t.Fatalf("first Translate returned unexpected error: %v", err)
	}
	before := len(spy.sent())

	if _, err := segmented.Translate(context.Background(), "Sie sass am Fluss. Die Bank ist zu."); err != nil {
		t.Fatalf("second Translate returned unexpected error: %v", err)
	}

	added := spy.sent()[before:]
	if len(added) != 2 {
		t.Errorf("second run sent %v, want both the changed sentence and the one after it", added)
	}
}

// Typing at the end must not throw away what is already translated.
func TestSegmentedKeepsEarlierSentencesWhileTheLastOneIsWritten(t *testing.T) {
	t.Parallel()
	spy := &spyTranslator{}
	segmented := translation.Segmented(spy)

	if _, err := segmented.Translate(context.Background(), "Erster Satz. Zwei"); err != nil {
		t.Fatalf("first Translate returned unexpected error: %v", err)
	}
	before := len(spy.sent())

	if _, err := segmented.Translate(context.Background(), "Erster Satz. Zweiter Satz!"); err != nil {
		t.Fatalf("second Translate returned unexpected error: %v", err)
	}

	added := spy.sent()[before:]
	if len(added) != 1 || strings.TrimSpace(added[0]) != "Zweiter Satz!" {
		t.Errorf("second run sent %v, want only the sentence being written", added)
	}
}

// Two previews can overlap while a draft is written; the same sentence must not
// be paid for twice.
func TestSegmentedSendsASentenceOnceEvenWhenAskedTwiceAtOnce(t *testing.T) {
	t.Parallel()
	spy := &blockingTranslator{started: make(chan struct{}, 4), release: make(chan struct{})}
	segmented := translation.Segmented(spy)

	const draft = "Der Test schlaegt fehl. Bitte behebe ihn."
	results := make(chan string, 2)
	for range 2 {
		go func() {
			translated, err := segmented.Translate(context.Background(), draft)
			if err != nil {
				t.Errorf("Translate returned unexpected error: %v", err)
			}
			results <- translated
		}()
	}

	<-spy.started
	close(spy.release)
	first, second := <-results, <-results

	if first != second {
		t.Errorf("the two callers got %q and %q, want the same translation", first, second)
	}
	if calls := spy.count(); calls > 2 {
		t.Errorf("the translator was called %d times for 2 sentences, want each sent once", calls)
	}
}

type blockingTranslator struct {
	mu      sync.Mutex
	calls   int
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (b *blockingTranslator) Translate(_ context.Context, text string) (string, error) {
	b.mu.Lock()
	b.calls++
	b.mu.Unlock()

	b.once.Do(func() { b.started <- struct{}{} })
	<-b.release
	return "<" + strings.TrimSpace(text) + ">", nil
}

func (b *blockingTranslator) count() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.calls
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

func draftOf(sentences int) string {
	parts := make([]string, sentences)
	for index := range parts {
		parts[index] = fmt.Sprintf("Satz Nummer %d ist hier.", index)
	}
	return strings.Join(parts, " ")
}

// Editing early in a long draft must not resend everything after it.
func TestSegmentedRetranslatesOnlyTheNeighbourhoodOfAnEdit(t *testing.T) {
	t.Parallel()
	spy := &spyTranslator{}
	segmented := translation.Segmented(spy)
	draft := draftOf(50)

	if _, err := segmented.Translate(context.Background(), draft); err != nil {
		t.Fatalf("first Translate returned unexpected error: %v", err)
	}
	before := len(spy.sent())

	edited := strings.Replace(draft, "Satz Nummer 0", "Satz Nummer null", 1)
	if _, err := segmented.Translate(context.Background(), edited); err != nil {
		t.Fatalf("second Translate returned unexpected error: %v", err)
	}

	if added := len(spy.sent()) - before; added > 3 {
		t.Errorf("editing one sentence resent %d of them, want only its neighbourhood", added)
	}
}

// Context is unbilled but not free: sending the whole draft with every sentence
// grows with the square of its length.
func TestSegmentedKeepsTheContextItSendsSmall(t *testing.T) {
	t.Parallel()
	spy := &spyTranslator{}
	draft := draftOf(40)

	if _, err := translation.Segmented(spy).Translate(context.Background(), draft); err != nil {
		t.Fatalf("Translate returned unexpected error: %v", err)
	}

	spy.mu.Lock()
	contexts := append([]string(nil), spy.contexts...)
	spy.mu.Unlock()

	total := 0
	for _, surrounding := range contexts {
		total += len(surrounding)
	}
	if total > 3*len(draft) {
		t.Errorf("context sent totalled %d characters for a draft of %d, want it bounded",
			total, len(draft))
	}
}

func TestSegmentedForgetsOldSentencesInsteadOfGrowingForEver(t *testing.T) {
	t.Parallel()
	spy := &spyTranslator{}
	segmented := translation.Segmented(spy)

	// Every draft is different, so nothing may be reused; the point is that the
	// store stays bounded rather than keeping one entry per sentence ever seen.
	for round := range 400 {
		draft := fmt.Sprintf("Einmalig %d. Und noch etwas %d.", round, round)
		if _, err := segmented.Translate(context.Background(), draft); err != nil {
			t.Fatalf("Translate returned unexpected error: %v", err)
		}
	}

	if remembered := translation.Remembered(segmented); remembered > 512 {
		t.Errorf("the store holds %d sentences, want it capped", remembered)
	}
}

// Sending must not inherit the cancellation of a preview it happened to share.
func TestSegmentedDoesNotHandOnACancelledResult(t *testing.T) {
	t.Parallel()
	spy := &cancelAwareTranslator{entered: make(chan struct{}), release: make(chan struct{})}
	segmented := translation.Segmented(spy)

	const draft = "Der Test schlaegt fehl."
	abandoned, cancel := context.WithCancel(context.Background())

	preview := make(chan error, 1)
	go func() {
		_, err := segmented.Translate(abandoned, draft)
		preview <- err
	}()
	<-spy.entered

	// The draft moves on: the preview is abandoned while the author sends.
	sent := make(chan string, 1)
	go func() {
		translated, err := segmented.Translate(context.Background(), draft)
		if err != nil {
			t.Errorf("sending returned %v, want it to stand on its own", err)
		}
		sent <- translated
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()
	close(spy.release)

	select {
	case translated := <-sent:
		if translated == "" {
			t.Error("sending produced nothing, want the translation")
		}
	case <-time.After(3 * time.Second):
		t.Error("sending never finished")
	}
	<-preview
}

type cancelAwareTranslator struct {
	once    sync.Once
	entered chan struct{}
	release chan struct{}
}

func (c *cancelAwareTranslator) Translate(ctx context.Context, text string) (string, error) {
	c.once.Do(func() { close(c.entered) })

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-c.release:
		return "<" + strings.TrimSpace(text) + ">", nil
	}
}
