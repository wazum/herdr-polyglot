package overlay_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
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

type recordingTranslator struct {
	english   string
	seenDraft string
}

func (r *recordingTranslator) Translate(_ context.Context, draft string) (string, error) {
	r.seenDraft = draft
	return r.english, nil
}

type recordingTarget struct{ inserted []string }

func (r *recordingTarget) Insert(_ context.Context, text string) error {
	r.inserted = append(r.inserted, text)
	return nil
}

func newOverlay(t *testing.T, translator promptflow.Translator, target promptflow.Target) *teatest.TestModel {
	t.Helper()
	return newOverlayWith(t, translator, target, overlay.Options{
		Service:  "deepl",
		Language: "EN-US",
		Vim:      true,
	})
}

func newOverlayWith(
	t *testing.T,
	translator promptflow.Translator,
	target promptflow.Target,
	options overlay.Options,
	flowOptions ...promptflow.Option,
) *teatest.TestModel {
	t.Helper()
	return teatest.NewTestModel(
		t,
		overlay.New(context.Background(),
			promptflow.New(translator, target, target, flowOptions...), options),
		teatest.WithInitialTermSize(80, 20),
	)
}

func TestTheOverlayShowsWhichServiceAndLanguageItWillUse(t *testing.T) {
	t.Parallel()

	overlayUnderTest := newOverlay(t, stubTranslator{english: english}, &recordingTarget{})
	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("deepl")) && bytes.Contains(out, []byte("EN-US"))
	}, teatest.WithDuration(2*time.Second))

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestSubmittingADraftInsertsTheEnglishTranslationIntoTheTarget(t *testing.T) {
	t.Parallel()
	target := &recordingTarget{}

	overlayUnderTest := newOverlay(t, stubTranslator{english: english}, target)
	overlayUnderTest.Type(draft)
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlD})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	if len(target.inserted) != 1 || target.inserted[0] != english {
		t.Errorf("target received %v, want one insert of %q", target.inserted, english)
	}
}

func TestEscapeSwitchesToNormalModeInsteadOfClosing(t *testing.T) {
	t.Parallel()
	target := &recordingTarget{}

	overlayUnderTest := newOverlay(t, stubTranslator{english: english}, target)
	overlayUnderTest.Type("hallo")
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyEsc})

	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("NORMAL"))
	}, teatest.WithDuration(2*time.Second))

	if len(target.inserted) != 0 {
		t.Errorf("target received %v, want nothing sent", target.inserted)
	}

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

// Escape is the only way back out: once to leave insert mode, once to close.
func TestEscapeTwiceClosesFromNormalMode(t *testing.T) {
	t.Parallel()
	target := &recordingTarget{}

	overlayUnderTest := newOverlay(t, stubTranslator{english: english}, target)
	overlayUnderTest.Type("hallo")
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyEsc})
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyEsc})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	if len(target.inserted) != 0 {
		t.Errorf("target received %v, want nothing sent", target.inserted)
	}
}

func TestQIsJustTextWhileTyping(t *testing.T) {
	t.Parallel()

	overlayUnderTest := newOverlay(t, stubTranslator{english: english}, &recordingTarget{})
	overlayUnderTest.Type("quatsch")

	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("quatsch"))
	}, teatest.WithDuration(2*time.Second))

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestCancellingLeavesTheTargetUntouched(t *testing.T) {
	t.Parallel()
	target := &recordingTarget{}

	overlayUnderTest := newOverlay(t, stubTranslator{english: english}, target)
	overlayUnderTest.Type(draft)
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
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
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlD})

	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("deepl unreachable"))
	}, teatest.WithDuration(2*time.Second))

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestABlankDraftIsNotSentAnywhere(t *testing.T) {
	t.Parallel()
	translatorThatMustNotRun := stubTranslator{err: errors.New("translator was called")}
	target := &recordingTarget{}

	overlayUnderTest := newOverlay(t, translatorThatMustNotRun, target)
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlD})
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	if len(target.inserted) != 0 {
		t.Errorf("target received %v, want nothing inserted", target.inserted)
	}
}

func TestEnterAddsALineInsteadOfSending(t *testing.T) {
	t.Parallel()
	target := &recordingTarget{}

	overlayUnderTest := newOverlay(t, stubTranslator{english: english}, target)
	overlayUnderTest.Type("erste Zeile")
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyEnter})
	overlayUnderTest.Type("zweite Zeile")

	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("erste Zeile")) && bytes.Contains(out, []byte("zweite Zeile"))
	}, teatest.WithDuration(2*time.Second))

	if len(target.inserted) != 0 {
		t.Errorf("target received %v, want nothing sent by enter alone", target.inserted)
	}

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestAltEnterSendsLikeCtrlD(t *testing.T) {
	t.Parallel()
	target := &recordingTarget{}

	overlayUnderTest := newOverlay(t, stubTranslator{english: english}, target)
	overlayUnderTest.Type(draft)
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyEnter, Alt: true})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	if len(target.inserted) != 1 || target.inserted[0] != english {
		t.Errorf("target received %v, want one insert of %q", target.inserted, english)
	}
}

// A terminal writes an alt chord as an escape and then the key. Two presses stay
// two messages, so leaving insert mode and pressing enter is not a send.
func TestEscapeThenEnterIsNotASend(t *testing.T) {
	t.Parallel()
	target := &recordingTarget{}

	overlayUnderTest := newOverlay(t, stubTranslator{english: english}, target)
	overlayUnderTest.Type(draft)
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyEsc})
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyEnter})
	time.Sleep(200 * time.Millisecond)

	if len(target.inserted) != 0 {
		t.Errorf("target received %v, want nothing sent", target.inserted)
	}

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestADraftKeepsCharactersOutsideAscii(t *testing.T) {
	t.Parallel()
	const umlauts = "Bitte prüfe die Übersetzung: äöü ß — fertig"
	translator := &recordingTranslator{english: english}

	overlayUnderTest := newOverlay(t, translator, &recordingTarget{})
	// Typed byte by byte, teatest would mangle multibyte runes.
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(umlauts)})
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlD})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	if translator.seenDraft != umlauts {
		t.Errorf("translator saw %q, want %q", translator.seenDraft, umlauts)
	}
}

func TestWithoutVimEscapeClosesTheOverlay(t *testing.T) {
	t.Parallel()
	target := &recordingTarget{}

	overlayUnderTest := newOverlayWith(t, stubTranslator{english: english}, target, overlay.Options{
		Service:  "deepl",
		Language: "EN-US",
	})
	overlayUnderTest.Type("hallo")
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyEsc})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	if len(target.inserted) != 0 {
		t.Errorf("target received %v, want nothing sent", target.inserted)
	}
}

func TestAnEmptyDraftShowsWhatToDo(t *testing.T) {
	t.Parallel()

	overlayUnderTest := newOverlay(t, stubTranslator{english: english}, &recordingTarget{})

	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("own language"))
	}, teatest.WithDuration(2*time.Second))

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

type gatedTranslator struct {
	mu      sync.Mutex
	calls   int
	started chan struct{}
	release chan struct{}
}

func (g *gatedTranslator) Translate(_ context.Context, _ string) (string, error) {
	g.mu.Lock()
	g.calls++
	call := g.calls
	g.mu.Unlock()

	if call == 1 {
		close(g.started)
		<-g.release
		return "FIRST", nil
	}
	return "SECOND", nil
}

func liveOverlay(t *testing.T, translator promptflow.Translator, target promptflow.Target) *teatest.TestModel {
	t.Helper()
	return newOverlayWith(t, translator, target, overlay.Options{
		Service:  "deepl",
		Language: "EN-US",
		Live:     true,
		Debounce: 20 * time.Millisecond,
	})
}

func TestLiveModeShowsTheEnglishWhileYouWrite(t *testing.T) {
	t.Parallel()
	target := &recordingTarget{}

	overlayUnderTest := liveOverlay(t, stubTranslator{english: english}, target)
	overlayUnderTest.Type(draft)

	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte(english))
	}, teatest.WithDuration(3*time.Second))

	if len(target.inserted) != 0 {
		t.Errorf("target received %v, want a preview only", target.inserted)
	}

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestWithoutLiveModeNothingIsTranslatedUntilYouSend(t *testing.T) {
	t.Parallel()
	translator := &recordingTranslator{english: english}

	overlayUnderTest := newOverlay(t, translator, &recordingTarget{})
	overlayUnderTest.Type(draft)
	time.Sleep(200 * time.Millisecond)

	if translator.seenDraft != "" {
		t.Errorf("translator was called with %q, want no call before sending", translator.seenDraft)
	}

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestALatePreviewNeverOverwritesANewerOne(t *testing.T) {
	t.Parallel()
	translator := &gatedTranslator{started: make(chan struct{}), release: make(chan struct{})}

	overlayUnderTest := liveOverlay(t, translator, &recordingTarget{})
	overlayUnderTest.Type("erste Fassung")
	<-translator.started

	overlayUnderTest.Type(" zweite Fassung")
	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("SECOND"))
	}, teatest.WithDuration(3*time.Second))

	close(translator.release)
	time.Sleep(300 * time.Millisecond)

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	shown, err := io.ReadAll(overlayUnderTest.Output())
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	if bytes.Contains(shown, []byte("FIRST")) {
		t.Error("the stale translation was shown, want it discarded")
	}
}

func TestSendingAfterAPreviewDeliversItWithoutTranslatingAgain(t *testing.T) {
	t.Parallel()
	translator := &countingTranslator{english: english}
	target := &recordingTarget{}

	overlayUnderTest := liveOverlay(t, translator, target)
	overlayUnderTest.Type(draft)

	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte(english))
	}, teatest.WithDuration(3*time.Second))

	// Typing may well have cost more than one translation on the way; what
	// matters is that sending costs none.
	beforeSending := translator.count()

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlD})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	if len(target.inserted) != 1 || target.inserted[0] != english {
		t.Errorf("target received %v, want the previewed translation delivered", target.inserted)
	}
	if calls := translator.count(); calls != beforeSending {
		t.Errorf("sending cost %d more translations, want the preview reused",
			calls-beforeSending)
	}
}

type countingTranslator struct {
	mu      sync.Mutex
	calls   int
	english string
}

func (c *countingTranslator) Translate(_ context.Context, _ string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	return c.english, nil
}

func (c *countingTranslator) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

// A preview belongs to the draft it was made from. Pairing it with the draft
// that merely started the newest request delivers the wrong prompt: the author
// reads one thing and the agent receives another.
func TestSendingWhileANewPreviewIsInFlightNeverDeliversTheOlderEnglish(t *testing.T) {
	t.Parallel()
	translator := &slowSecondTranslator{
		first:   "Do not delete the database",
		second:  "Do not delete the database, delete it after all",
		blocked: make(chan struct{}),
		release: make(chan struct{}),
	}
	target := &recordingTarget{}

	overlayUnderTest := newOverlayWith(t, translator, target, overlay.Options{
		Service:  "deepl",
		Language: "EN-US",
		Live:     true,
		Debounce: 20 * time.Millisecond,
	})

	overlayUnderTest.Type("Loesche die Datenbank nicht")
	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Do not delete the database"))
	}, teatest.WithDuration(3*time.Second))

	// The draft changes; the second translation starts but has not answered.
	overlayUnderTest.Type(". Loesche sie doch")
	<-translator.blocked

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlD})
	close(translator.release)
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	if len(target.inserted) != 1 {
		t.Fatalf("target received %v, want exactly one delivery", target.inserted)
	}
	if delivered := target.inserted[0]; delivered == translator.first {
		t.Errorf("delivered %q, which was translated from the earlier draft", delivered)
	}
}

type slowSecondTranslator struct {
	mu      sync.Mutex
	calls   int
	first   string
	second  string
	blocked chan struct{}
	release chan struct{}
}

func (s *slowSecondTranslator) Translate(_ context.Context, _ string) (string, error) {
	s.mu.Lock()
	s.calls++
	call := s.calls
	s.mu.Unlock()

	if call == 1 {
		return s.first, nil
	}
	if call == 2 {
		close(s.blocked)
		<-s.release
	}
	return s.second, nil
}

// A draft that moves on makes the request it started pointless; letting it run
// spends characters and request budget on a translation nobody will read.
func TestANewDraftCancelsTheTranslationAlreadyRunning(t *testing.T) {
	t.Parallel()
	translator := &cancellingTranslator{
		entered:  make(chan struct{}),
		observed: make(chan error, 4),
		release:  make(chan struct{}),
	}

	overlayUnderTest := newOverlayWith(t, translator, &recordingTarget{}, overlay.Options{
		Service:  "deepl",
		Language: "EN-US",
		Live:     true,
		Debounce: 20 * time.Millisecond,
	})

	overlayUnderTest.Type("Erste Fassung")
	select {
	case <-translator.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("the first translation never started")
	}

	overlayUnderTest.Type(" zweite Fassung")

	select {
	case err := <-translator.observed:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("the abandoned request saw %v, want it cancelled", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("the abandoned request was never cancelled")
	}

	close(translator.release)
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestACancelledTranslationIsNotShownAsAFailure(t *testing.T) {
	t.Parallel()
	translator := &cancellingTranslator{
		entered:  make(chan struct{}),
		observed: make(chan error, 4),
		release:  make(chan struct{}),
	}

	overlayUnderTest := newOverlayWith(t, translator, &recordingTarget{}, overlay.Options{
		Service:  "deepl",
		Language: "EN-US",
		Live:     true,
		Debounce: 20 * time.Millisecond,
	})

	overlayUnderTest.Type("Erste Fassung")
	select {
	case <-translator.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("the first translation never started")
	}

	overlayUnderTest.Type(" mehr")
	select {
	case <-translator.observed:
	case <-time.After(2 * time.Second):
	}
	close(translator.release)
	time.Sleep(200 * time.Millisecond)

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	shown, err := io.ReadAll(overlayUnderTest.Output())
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	if bytes.Contains(shown, []byte("context canceled")) {
		t.Error("a cancelled request was reported to the author, want it passed over")
	}
}

type cancellingTranslator struct {
	once     sync.Once
	entered  chan struct{}
	observed chan error
	release  chan struct{}
}

func (c *cancellingTranslator) Translate(ctx context.Context, _ string) (string, error) {
	c.once.Do(func() { close(c.entered) })

	select {
	case <-ctx.Done():
		c.observed <- ctx.Err()
		return "", ctx.Err()
	case <-c.release:
		return english, nil
	}
}

func TestEscapeClosesAnEmptyDraftFromNormalMode(t *testing.T) {
	t.Parallel()
	target := &recordingTarget{}

	overlayUnderTest := newOverlay(t, stubTranslator{english: english}, target)
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyEsc}) // to normal mode
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyEsc}) // nothing to lose, so close
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	if len(target.inserted) != 0 {
		t.Errorf("target received %v, want nothing sent", target.inserted)
	}
}

// Escape closes on written work too, because closing keeps the draft.
func TestEscapeClosesAWrittenDraftAndKeepsIt(t *testing.T) {
	t.Parallel()
	drafts := &fakeDrafts{}

	overlayUnderTest := newOverlayWith(t, stubTranslator{english: english}, &recordingTarget{},
		overlay.Options{Service: "deepl", Language: "EN-US", Vim: true, Drafts: drafts})
	overlayUnderTest.Type("Bitte behebe")
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyEsc})
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyEsc})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	if len(drafts.saved) == 0 || drafts.saved[len(drafts.saved)-1] != "Bitte behebe" {
		t.Errorf("the store was given %v, want the draft kept on the way out", drafts.saved)
	}
}

type fakeDrafts struct {
	kept      string
	loadError error
	saved     []string
	cleared   int
	saveError error
}

func (f *fakeDrafts) Load() (string, error) { return f.kept, f.loadError }

func (f *fakeDrafts) Save(text string) error {
	if f.saveError != nil {
		return f.saveError
	}
	f.saved = append(f.saved, text)
	return nil
}

func (f *fakeDrafts) Clear() error {
	f.cleared++
	return nil
}

func TestAKeptDraftIsThereAgainWhenTheOverlayOpens(t *testing.T) {
	t.Parallel()
	drafts := &fakeDrafts{kept: "Bitte behebe den Test"}

	overlayUnderTest := newOverlayWith(t, stubTranslator{english: english}, &recordingTarget{},
		overlay.Options{Service: "deepl", Language: "EN-US", Drafts: drafts})

	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Bitte behebe den Test"))
	}, teatest.WithDuration(2*time.Second))

	// Typing continues after the kept text rather than in front of it.
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" gründlich")})
	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Test gründlich"))
	}, teatest.WithDuration(2*time.Second))

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestClosingKeepsTheDraftForNextTime(t *testing.T) {
	t.Parallel()
	drafts := &fakeDrafts{}

	overlayUnderTest := newOverlayWith(t, stubTranslator{english: english}, &recordingTarget{},
		overlay.Options{Service: "deepl", Language: "EN-US", Vim: true, Drafts: drafts})
	overlayUnderTest.Type("Bitte behebe den Test")
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyEsc})
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyEsc})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	if len(drafts.saved) != 1 || drafts.saved[0] != "Bitte behebe den Test" {
		t.Errorf("the store was given %v, want the draft kept once", drafts.saved)
	}
}

func TestASentDraftIsNotKept(t *testing.T) {
	t.Parallel()
	drafts := &fakeDrafts{}
	target := &recordingTarget{}

	overlayUnderTest := newOverlayWith(t, stubTranslator{english: english}, target,
		overlay.Options{Service: "deepl", Language: "EN-US", Drafts: drafts})
	overlayUnderTest.Type(draft)
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlD})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	if len(target.inserted) != 1 {
		t.Fatalf("target received %v, want the prompt delivered", target.inserted)
	}
	if drafts.cleared != 1 {
		t.Errorf("the store was cleared %d times, want the sent draft forgotten", drafts.cleared)
	}
	if len(drafts.saved) != 0 {
		t.Errorf("the store was given %v, want a sent draft not kept", drafts.saved)
	}
}

func TestADraftThatCannotBeKeptIsNotSilentlyLost(t *testing.T) {
	t.Parallel()
	drafts := &fakeDrafts{saveError: errors.New("disk full")}

	overlayUnderTest := newOverlayWith(t, stubTranslator{english: english}, &recordingTarget{},
		overlay.Options{Service: "deepl", Language: "EN-US", Vim: true, Drafts: drafts})
	overlayUnderTest.Type("Bitte behebe den Test")
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyEsc})
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyEsc})

	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("disk full"))
	}, teatest.WithDuration(2*time.Second))

	// ctrl+c is the way out when even keeping the draft fails.
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

// Services charge by the character, and a draft that comes back may be anything —
// yesterday's thought, or a cat on the keyboard. Live translation starts off, so
// resuming costs nothing until ctrl+l asks for it.
func TestAResumedDraftArrivesWithLiveTranslationOff(t *testing.T) {
	t.Parallel()
	translator := &countingTranslator{english: english}
	drafts := &fakeDrafts{kept: "Bitte behebe den Test"}

	overlayUnderTest := newOverlayWith(t, translator, &recordingTarget{},
		overlay.Options{
			Service: "deepl", Language: "EN-US", Live: true,
			Debounce: 10 * time.Millisecond, Drafts: drafts,
		})

	// Live translation turning itself off is worth saying, or it looks broken.
	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Bitte behebe den Test")) &&
			bytes.Contains(out, []byte("translates it"))
	}, teatest.WithDuration(2*time.Second))

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" gründlich")})
	time.Sleep(300 * time.Millisecond)

	if calls := translator.count(); calls != 0 {
		t.Errorf("the service was asked %d times for a draft that came back", calls)
	}

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlL})
	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte(english))
	}, teatest.WithDuration(2*time.Second))

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

// Herdr closes a popup by hanging up on it, and ctrl+c never reaches the close
// key either. Whatever ends the session, the writing has to survive it.
func TestADraftSurvivesAnEndingNobodyAskedFor(t *testing.T) {
	t.Parallel()
	drafts := &fakeDrafts{}

	target := &recordingTarget{}
	var model tea.Model = overlay.New(context.Background(),
		promptflow.New(stubTranslator{english: english}, target, target),
		overlay.Options{Service: "deepl", Language: "EN-US", Vim: true, Drafts: drafts})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Bitte behebe den Test")})

	if err := model.(overlay.Model).KeepUnfinished(); err != nil {
		t.Fatalf("keeping the draft: %v", err)
	}
	if len(drafts.saved) != 1 || drafts.saved[0] != "Bitte behebe den Test" {
		t.Errorf("the store was given %v, want the draft as it stood", drafts.saved)
	}
}

func TestASentDraftIsNotWrittenBackOnTheWayOut(t *testing.T) {
	t.Parallel()
	drafts := &fakeDrafts{}

	target := &recordingTarget{}
	var model tea.Model = overlay.New(context.Background(),
		promptflow.New(stubTranslator{english: english}, target, target),
		overlay.Options{Service: "deepl", Language: "EN-US", Vim: true, Drafts: drafts})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Bitte behebe den Test")})
	model, _ = model.Update(overlay.PromptDelivered())

	if err := model.(overlay.Model).KeepUnfinished(); err != nil {
		t.Fatalf("keeping the draft: %v", err)
	}
	if len(drafts.saved) != 0 {
		t.Errorf("the store was given %v, want a sent draft left forgotten", drafts.saved)
	}
}

func TestARestoredDraftSaysThatItWasResumed(t *testing.T) {
	t.Parallel()
	drafts := &fakeDrafts{kept: "Bitte behebe den Test"}

	overlayUnderTest := newOverlayWith(t, stubTranslator{english: english}, &recordingTarget{},
		overlay.Options{Service: "deepl", Language: "EN-US", Drafts: drafts})

	// Text that reappears without explanation is a surprise, so say where it
	// came from until the author touches it.
	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("resumed draft"))
	}, teatest.WithDuration(2*time.Second))

	overlayUnderTest.Type("!")
	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		tail := out[max(0, len(out)-400):]
		return bytes.Contains(tail, []byte("Test!")) && !bytes.Contains(tail, []byte("resumed"))
	}, teatest.WithDuration(2*time.Second))

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestADraftCanBeThrownAwayWithOneKey(t *testing.T) {
	t.Parallel()
	drafts := &fakeDrafts{kept: "Ein alter Entwurf"}

	overlayUnderTest := newOverlayWith(t, stubTranslator{english: english}, &recordingTarget{},
		overlay.Options{Service: "deepl", Language: "EN-US", Drafts: drafts})
	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Ein alter Entwurf"))
	}, teatest.WithDuration(2*time.Second))

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlU})
	overlayUnderTest.Type("Etwas Neues")

	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		tail := out[max(0, len(out)-500):]
		return bytes.Contains(tail, []byte("Etwas Neues")) && !bytes.Contains(tail, []byte("alter Entwurf"))
	}, teatest.WithDuration(2*time.Second))

	if drafts.cleared != 1 {
		t.Errorf("the store was cleared %d times, want the thrown-away draft forgotten", drafts.cleared)
	}

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func confirmingOverlay(t *testing.T, translator promptflow.Translator, target promptflow.Target) *teatest.TestModel {
	t.Helper()
	return newOverlayWith(t, translator, target, overlay.Options{
		Service:  "deepl",
		Language: "EN-US",
		Confirm:  true,
	})
}

func TestWithConfirmationTheEnglishIsShownBeforeItIsSent(t *testing.T) {
	t.Parallel()
	translator := &countingTranslator{english: english}
	target := &recordingTarget{}

	overlayUnderTest := confirmingOverlay(t, translator, target)
	overlayUnderTest.Type(draft)
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlD})

	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte(english))
	}, teatest.WithDuration(3*time.Second))

	if len(target.inserted) != 0 {
		t.Fatalf("target received %v, want nothing until it is confirmed", target.inserted)
	}

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlD})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	if len(target.inserted) != 1 || target.inserted[0] != english {
		t.Errorf("target received %v, want the confirmed translation", target.inserted)
	}
	if calls := translator.count(); calls != 1 {
		t.Errorf("the translator was called %d times, want the shown translation reused", calls)
	}
}

func TestConfirmationCanBeTurnedDownToKeepWriting(t *testing.T) {
	t.Parallel()
	target := &recordingTarget{}

	overlayUnderTest := confirmingOverlay(t, stubTranslator{english: english}, target)
	overlayUnderTest.Type(draft)
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlD})
	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte(english))
	}, teatest.WithDuration(3*time.Second))

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyEsc})
	overlayUnderTest.Type(" bitte")

	// Back in the draft, with the writing intact and nothing sent.
	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Test bitte"))
	}, teatest.WithDuration(3*time.Second))
	if len(target.inserted) != 0 {
		t.Errorf("target received %v, want nothing sent", target.inserted)
	}

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestWithoutConfirmationSendingStaysOneKey(t *testing.T) {
	t.Parallel()
	target := &recordingTarget{}

	overlayUnderTest := newOverlay(t, stubTranslator{english: english}, target)
	overlayUnderTest.Type(draft)
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlD})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	if len(target.inserted) != 1 {
		t.Errorf("target received %v, want it delivered on the first key", target.inserted)
	}
}

type spendingService struct{ spent promptflow.Usage }

func (s spendingService) Usage(context.Context) (promptflow.Usage, error) {
	return s.spent, nil
}

func TestTheHeaderShowsWhatTheKeyHasSpent(t *testing.T) {
	t.Parallel()
	overlayUnderTest := newOverlayWith(t, stubTranslator{english: english}, &recordingTarget{},
		overlay.Options{Service: "deepl", Language: "EN-US", Vim: true},
		promptflow.WithUsageReporter(spendingService{
			spent: promptflow.Usage{Used: 12345, Limit: 1_000_000},
		}))

	// Compact, because the header is narrow: 12.3k of 1M.
	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("12.3k/1M chars"))
	}, teatest.WithDuration(3*time.Second))

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestAServiceWithoutAnAllowanceShowsNoCount(t *testing.T) {
	t.Parallel()

	overlayUnderTest := newOverlay(t, stubTranslator{english: english}, &recordingTarget{})
	overlayUnderTest.Type("Bitte behebe")
	time.Sleep(300 * time.Millisecond)

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	shown, err := io.ReadAll(overlayUnderTest.Output())
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	if bytes.Contains(shown, []byte("/1M")) || bytes.Contains(shown, []byte("0/0")) {
		t.Error("the header shows an allowance the service never reported")
	}
}

func longOverlay(t *testing.T, translator promptflow.Translator, target promptflow.Target) *teatest.TestModel {
	t.Helper()
	return newOverlayWith(t, translator, target, overlay.Options{
		Service:  "deepl",
		Language: "EN-US",
		Live:     true,
		Debounce: 20 * time.Millisecond,
		MaxDraft: 200,
	})
}

func TestPastingFarMoreThanAPromptSaysSo(t *testing.T) {
	t.Parallel()

	overlayUnderTest := longOverlay(t, stubTranslator{english: english}, &recordingTarget{})
	overlayUnderTest.Send(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune(strings.Repeat("sehr langer Text ", 40)),
		Paste: true,
	})

	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("680 characters"))
	}, teatest.WithDuration(2*time.Second))

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestTheWarningGoesWhenTheDraftIsShortAgain(t *testing.T) {
	t.Parallel()

	overlayUnderTest := longOverlay(t, stubTranslator{english: english}, &recordingTarget{})
	overlayUnderTest.Send(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune(strings.Repeat("x", 400)),
		Paste: true,
	})
	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("characters"))
	}, teatest.WithDuration(2*time.Second))

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlU})
	overlayUnderTest.Type("kurz")
	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("kurz"))
	}, teatest.WithDuration(2*time.Second))

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	last, err := io.ReadAll(overlayUnderTest.FinalOutput(t))
	if err != nil {
		t.Fatalf("reading the last frame: %v", err)
	}
	if bytes.Contains(last, []byte("characters")) {
		t.Error("the warning is still shown for a draft that is short again")
	}
}

// Translating a pasted wall of text again after every pause would spend the
// allowance on something the tool is not for.
func TestNothingIsTranslatedWhileTheDraftIsTooLong(t *testing.T) {
	t.Parallel()
	translator := &countingTranslator{english: english}

	overlayUnderTest := longOverlay(t, translator, &recordingTarget{})
	overlayUnderTest.Send(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune(strings.Repeat("y", 500)),
		Paste: true,
	})
	time.Sleep(400 * time.Millisecond)

	if calls := translator.count(); calls != 0 {
		t.Errorf("the translator was called %d times, want the long draft left alone", calls)
	}

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestALongDraftCanStillBeSent(t *testing.T) {
	t.Parallel()
	target := &recordingTarget{}

	overlayUnderTest := longOverlay(t, stubTranslator{english: english}, target)
	overlayUnderTest.Send(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune(strings.Repeat("z", 500)),
		Paste: true,
	})
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlD})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	if len(target.inserted) != 1 {
		t.Errorf("target received %v, want the prompt sent anyway", target.inserted)
	}
}

func TestLiveModeShowsAPulseWhileItIsTranslating(t *testing.T) {
	t.Parallel()
	translator := &gatedTranslator{started: make(chan struct{}), release: make(chan struct{})}

	overlayUnderTest := newOverlayWith(t, translator, &recordingTarget{}, overlay.Options{
		Service:  "deepl",
		Language: "EN-US",
		Live:     true,
		Pulse:    true,
		Debounce: 20 * time.Millisecond,
	})
	overlayUnderTest.Type("Bitte behebe")
	<-translator.started

	// The circle fills and empties again, so two different states show up.
	seen := map[string]bool{}
	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		for _, glyph := range []string{"○", "◔", "◑", "◕", "●"} {
			if bytes.Contains(out, []byte(glyph)) {
				seen[glyph] = true
			}
		}
		return len(seen) >= 2
	}, teatest.WithDuration(3*time.Second))

	close(translator.release)
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestWithoutThePulseLiveModeSaysSoQuietly(t *testing.T) {
	t.Parallel()
	translator := &gatedTranslator{started: make(chan struct{}), release: make(chan struct{})}

	overlayUnderTest := newOverlayWith(t, translator, &recordingTarget{}, overlay.Options{
		Service:  "deepl",
		Language: "EN-US",
		Live:     true,
		Debounce: 20 * time.Millisecond,
	})
	overlayUnderTest.Type("Bitte behebe")
	<-translator.started
	time.Sleep(400 * time.Millisecond)

	close(translator.release)
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	shown, err := io.ReadAll(overlayUnderTest.Output())
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	if !bytes.Contains(shown, []byte("live")) {
		t.Error("live mode is not mentioned at all")
	}
	for _, glyph := range []string{"◔", "◑", "◕"} {
		if bytes.Contains(shown, []byte(glyph)) {
			t.Errorf("the header pulses with %q although the pulse is off", glyph)
		}
	}
}

// Nothing is being translated, so the circle rests instead of drawing attention.
func TestThePulseRestsWhenNothingIsBeingTranslated(t *testing.T) {
	t.Parallel()

	overlayUnderTest := newOverlayWith(t, stubTranslator{english: english}, &recordingTarget{},
		overlay.Options{Service: "deepl", Language: "EN-US", Live: true, Pulse: true})
	time.Sleep(700 * time.Millisecond)

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	shown, err := io.ReadAll(overlayUnderTest.Output())
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	for _, filling := range []string{"◔", "◑", "◕", "●"} {
		if bytes.Contains(shown, []byte(filling)) {
			t.Errorf("the circle filled to %q with nothing to translate", filling)
		}
	}
}

// A translation that answers in a blink would otherwise flash once and stop, so
// the circle finishes the breath it started.
func TestAFastTranslationStillShowsAWholeBreath(t *testing.T) {
	t.Parallel()

	overlayUnderTest := newOverlayWith(t, stubTranslator{english: english}, &recordingTarget{},
		overlay.Options{
			Service:  "deepl",
			Language: "EN-US",
			Live:     true,
			Pulse:    true,
			Debounce: 20 * time.Millisecond,
		})
	overlayUnderTest.Type("Bitte behebe")

	seen := map[string]bool{}
	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		for _, glyph := range []string{"·", "○", "◔", "◑", "◕", "●"} {
			if bytes.Contains(out, []byte(glyph)) {
				seen[glyph] = true
			}
		}
		return len(seen) >= 5
	}, teatest.WithDuration(4*time.Second))

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func switchableOverlay(t *testing.T, sending, typing promptflow.Target, options overlay.Options) *teatest.TestModel {
	t.Helper()
	flow := promptflow.New(stubTranslator{english: english}, sending, typing)
	options.Service, options.Language = "deepl", "EN-US"
	return teatest.NewTestModel(t, overlay.New(context.Background(), flow, options),
		teatest.WithInitialTermSize(87, 17))
}

// Whether a prompt is sent or only typed is easier to decide once the draft is
// written, so the key is in the popup rather than in the keybinding.
func TestCtrlRSwitchesToTypingWithoutSending(t *testing.T) {
	t.Parallel()
	sending, typing := &recordingTarget{}, &recordingTarget{}

	overlayUnderTest := switchableOverlay(t, sending, typing, overlay.Options{})
	overlayUnderTest.Type(draft)
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlR})

	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("fills the input"))
	}, teatest.WithDuration(2*time.Second))

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlD})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	if len(typing.inserted) != 1 || typing.inserted[0] != english {
		t.Errorf("the typing target received %v, want the prompt", typing.inserted)
	}
	if len(sending.inserted) != 0 {
		t.Errorf("the sending target received %v, want nothing", sending.inserted)
	}
}

func TestCtrlRSwitchesBackToSending(t *testing.T) {
	t.Parallel()
	sending, typing := &recordingTarget{}, &recordingTarget{}

	overlayUnderTest := switchableOverlay(t, sending, typing, overlay.Options{Review: true})
	overlayUnderTest.Type(draft)
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlR})
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlD})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	if len(sending.inserted) != 1 {
		t.Errorf("the sending target received %v, want the prompt", sending.inserted)
	}
	if len(typing.inserted) != 0 {
		t.Errorf("the typing target received %v, want nothing", typing.inserted)
	}
}

// Live translation costs characters, so it can be turned on for one prompt.
func TestCtrlLTurnsLiveTranslationOnForThisPrompt(t *testing.T) {
	t.Parallel()
	translator := &countingTranslator{english: english}
	flow := promptflow.New(translator, &recordingTarget{}, &recordingTarget{})

	overlayUnderTest := teatest.NewTestModel(t,
		overlay.New(context.Background(), flow, overlay.Options{
			Service: "deepl", Language: "EN-US", Debounce: 20 * time.Millisecond,
		}),
		teatest.WithInitialTermSize(87, 17))

	overlayUnderTest.Type("Bitte behebe")
	time.Sleep(300 * time.Millisecond)
	if calls := translator.count(); calls != 0 {
		t.Fatalf("the translator ran %d times with live off", calls)
	}

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlL})
	teatest.WaitFor(t, overlayUnderTest.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte(english))
	}, teatest.WithDuration(3*time.Second))

	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	overlayUnderTest.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}
