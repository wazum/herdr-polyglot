package overlay_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/wazum/herdr-polyglot/internal/overlay"
	"github.com/wazum/herdr-polyglot/internal/promptflow"
)

type failingOnceTranslator struct {
	mu    sync.Mutex
	calls int
}

func (f *failingOnceTranslator) Translate(context.Context, string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.calls == 1 {
		return "", errors.New("deepl did not answer in time")
	}
	return english, nil
}

func (f *failingOnceTranslator) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func TestAFailedTranslationCanBeTriedAgain(t *testing.T) {
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI)
	t.Cleanup(func() { lipgloss.SetColorProfile(previous) })

	service := &failingOnceTranslator{}
	target := &recordingTarget{}
	flow := promptflow.New(service, target, target, promptflow.WithPreviewTranslator(service))

	var model tea.Model = overlay.New(context.Background(), flow, overlay.Options{
		Service: "deepl", Language: "EN-US", Live: true, Debounce: time.Millisecond,
	})
	model, _ = model.Update(tea.WindowSizeMsg{Width: 87, Height: 15})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Bitte behebe den Test.")})
	model, _ = model.Update(overlay.PreviewFailed("deepl did not answer in time"))

	shown := plain(model.View())
	if !strings.Contains(shown, "did not answer in time") {
		t.Errorf("the panel does not say what went wrong:\n%s", shown)
	}
	if !strings.Contains(lastLine(shown), "ctrl+t") {
		t.Errorf("nothing says how to try again: %q", lastLine(shown))
	}

	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	drive(model, cmd)
	if service.count() < 1 {
		t.Error("ctrl+t did not ask the service again")
	}
}

// With live translation off there is no preview at all, so the same key is how a
// draft gets translated without being sent.
func TestCtrlTTranslatesOnDemandWithLiveOff(t *testing.T) {
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI)
	t.Cleanup(func() { lipgloss.SetColorProfile(previous) })

	service := &countingTranslator{english: english}
	target := &recordingTarget{}
	flow := promptflow.New(service, target, target, promptflow.WithPreviewTranslator(service))

	var model tea.Model = overlay.New(context.Background(), flow, overlay.Options{
		Service: "deepl", Language: "EN-US", Debounce: time.Millisecond,
	})
	model, _ = model.Update(tea.WindowSizeMsg{Width: 87, Height: 15})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Bitte behebe den Test.")})

	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	model = drive(model, cmd)

	if service.count() == 0 {
		t.Error("ctrl+t translated nothing with live off")
	}
	if !strings.Contains(plain(model.View()), english) {
		t.Errorf("the translation is not shown:\n%s", plain(model.View()))
	}
	if len(target.inserted) != 0 {
		t.Errorf("ctrl+t delivered %v, want it only translated", target.inserted)
	}
}
