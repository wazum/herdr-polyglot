package overlay_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/wazum/herdr-polyglot/internal/overlay"
	"github.com/wazum/herdr-polyglot/internal/promptflow"
)

func composer(t *testing.T, options overlay.Options, translator promptflow.Translator) (tea.Model, *recordingTarget) {
	t.Helper()
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI)
	t.Cleanup(func() { lipgloss.SetColorProfile(previous) })

	target := &recordingTarget{}
	flow := promptflow.New(translator, target, target, promptflow.WithPreviewTranslator(translator))
	var model tea.Model = overlay.New(context.Background(), flow, options)
	model, _ = model.Update(tea.WindowSizeMsg{Width: 87, Height: 15})
	return model, target
}

type asWritten struct{}

func (asWritten) Translate(_ context.Context, draft string) (string, error) { return draft, nil }

// With no service there is nothing to translate with, so the popup is a draft box
// and says as much rather than naming a service it does not have.
func TestWithoutAServiceThePopupIsADraftBox(t *testing.T) {
	model, target := composer(t, overlay.Options{
		Language: "EN-US", Debounce: time.Millisecond, WithoutService: true,
	}, asWritten{})

	shown := plain(model.View())
	if strings.Contains(shown, "→ EN-US") {
		t.Errorf("the header names a language nothing translates into:\n%s", shown)
	}
	if !strings.Contains(shown, "no translation") {
		t.Errorf("the header does not say translation is off:\n%s", shown)
	}
	for _, key := range []string{"ctrl+l", "tab"} {
		if strings.Contains(lastLine(shown), key) {
			t.Errorf("the footer offers %s with nothing to translate: %q", key, lastLine(shown))
		}
	}

	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Bitte behebe den Test")})
	drive(model, cmd)
	if strings.Count(plain(model.View()), "╭") > 1 {
		t.Error("a second panel is drawn for a translation that cannot happen")
	}

	model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter, Alt: true})
	drive(model, cmd)
	if len(target.inserted) != 1 || target.inserted[0] != "Bitte behebe den Test" {
		t.Errorf("target received %v, want the draft as it was written", target.inserted)
	}
}

func TestAskingToTranslateWithoutAServiceSaysSo(t *testing.T) {
	model, _ := composer(t, overlay.Options{Language: "EN-US", WithoutService: true}, asWritten{})

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlL})
	if footer := lastLine(plain(model.View())); !strings.Contains(footer, "no translation service") {
		t.Errorf("ctrl+l says %q, want it to say there is no service", footer)
	}

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	if footer := lastLine(plain(model.View())); !strings.Contains(footer, "no translation service") {
		t.Errorf("ctrl+t says %q, want it to say there is no service", footer)
	}
}

// A service that is configured but broken is not the same as none: the reason is
// said out loud, and the draft is not delivered as it was written.
func TestABrokenServiceIsReportedAndNothingIsDelivered(t *testing.T) {
	trouble := errors.New("deepl: no API key; configure it in /somewhere/.env")
	model, target := composer(t, overlay.Options{
		Service: "deepl", Language: "EN-US", Trouble: trouble, Debounce: time.Millisecond,
	}, brokenTranslator{trouble})

	// The trouble is said as the popup opens, which is one of its first commands.
	model = driveOnce(model, model.Init())
	if !strings.Contains(plain(model.View()), "no API key") {
		t.Errorf("the popup does not say what is wrong:\n%s", plain(model.View()))
	}

	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Bitte behebe den Test")})
	drive(model, cmd)
	model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter, Alt: true})
	drive(model, cmd)

	if len(target.inserted) != 0 {
		t.Errorf("target received %v, want nothing while the service is broken", target.inserted)
	}
}

type brokenTranslator struct{ why error }

func (b brokenTranslator) Translate(context.Context, string) (string, error) { return "", b.why }
