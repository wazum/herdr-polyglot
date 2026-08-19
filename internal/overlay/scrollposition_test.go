package overlay_test

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/wazum/herdr-polyglot/internal/overlay"
	"github.com/wazum/herdr-polyglot/internal/promptflow"
)

var manyLines = strings.TrimSpace(strings.Repeat("Zeile\n", 30))

func TestTheBarSaysWhereTheViewActuallyIs(t *testing.T) {
	pane := tea.WindowSizeMsg{Width: 87, Height: 15}
	model := overlayWith(t, pane, manyLines)

	bar := draftBar(t, strings.Split(model.View(), "\n"))
	if !strings.HasSuffix(bar, overlay.ScrollThumb) {
		t.Errorf("at the end of the draft the bar reads %q, want the thumb at the bottom", bar)
	}

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})

	bar = draftBar(t, strings.Split(model.View(), "\n"))
	if !strings.HasPrefix(bar, overlay.ScrollThumb) {
		t.Errorf("at the top of the draft the bar reads %q, want the thumb at the top", bar)
	}

	// Far enough down that the view has to move, which is when rows go above it.
	for range 14 {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if bar = draftBar(t, strings.Split(model.View(), "\n")); strings.HasPrefix(bar, overlay.ScrollThumb) {
		t.Errorf("with rows hidden above, the bar still reads %q", bar)
	}
}

func TestAResumedDraftOpensAtItsBeginning(t *testing.T) {
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI)
	t.Cleanup(func() { lipgloss.SetColorProfile(previous) })

	kept := "Erste Zeile.\n" + strings.Repeat("Mittlere Zeile.\n", 20) + "Letzte Zeile."
	drafts := &fakeDrafts{kept: kept}

	target := &recordingTarget{}
	flow := promptflow.New(stubTranslator{english: english}, target, target)
	var model tea.Model = overlay.New(context.Background(), flow, overlay.Options{
		Service: "deepl", Language: "EN-US", Vim: true, Drafts: drafts,
	})
	model, _ = model.Update(tea.WindowSizeMsg{Width: 87, Height: 15})

	shown := plain(model.View())
	if !strings.Contains(shown, "Erste Zeile.") {
		t.Errorf("a draft that came back does not open at its beginning:\n%s", shown)
	}
	if strings.Contains(shown, "Letzte Zeile.") {
		t.Error("a draft that came back opens at its end")
	}

	bar := draftBar(t, strings.Split(model.View(), "\n"))
	if !strings.HasPrefix(bar, overlay.ScrollThumb) {
		t.Errorf("the bar reads %q, want the thumb at the top", bar)
	}
}
