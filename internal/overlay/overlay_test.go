package overlay_test

import (
	"bytes"
	"context"
	"errors"
	"io"
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
) *teatest.TestModel {
	t.Helper()
	return teatest.NewTestModel(
		t,
		overlay.New(context.Background(), promptflow.New(translator, target), options),
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

func TestQClosesFromNormalMode(t *testing.T) {
	t.Parallel()
	target := &recordingTarget{}

	overlayUnderTest := newOverlay(t, stubTranslator{english: english}, target)
	overlayUnderTest.Type("hallo")
	overlayUnderTest.Send(tea.KeyMsg{Type: tea.KeyEsc})
	overlayUnderTest.Type("q")
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

func TestAltEnterAlsoSends(t *testing.T) {
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
