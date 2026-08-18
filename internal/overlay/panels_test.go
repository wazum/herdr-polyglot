package overlay_test

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

	"github.com/wazum/herdr-polyglot/internal/overlay"
	"github.com/wazum/herdr-polyglot/internal/promptflow"
)

var paragraph = strings.TrimSpace(strings.Repeat("wort ", 300)) + " ende"

func TestTheDraftScrollsWithTheCursorThroughAWrappedParagraph(t *testing.T) {
	pane := tea.WindowSizeMsg{Width: 87, Height: 15}

	model := overlayWith(t, pane, paragraph)
	atEnd := model.View()

	// Normal mode, then up: the view has to follow the cursor back through the
	// rows of a single wrapped line.
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	for range 4 {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	}
	scrolledBack := model.View()

	if scrolledBack == atEnd {
		t.Error("moving up in normal mode does not scroll the draft")
	}

	for range 4 {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if back := plain(model.View()); !strings.Contains(back, lastWord(paragraph)) {
		t.Errorf("moving back down does not return to where the writing is:\n%s", back)
	}
}

func TestGkWalksAWrappedParagraphTheWayVimDoes(t *testing.T) {
	pane := tea.WindowSizeMsg{Width: 87, Height: 15}

	model := overlayWith(t, pane, paragraph)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	atEnd := model.View()

	for range 4 {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	}
	if model.View() == atEnd {
		t.Error("gk does not move through a wrapped line")
	}
}

// Beside the draft there is only room to glance at the translation, so the panel
// shows where it starts, says there is more, and names the key that shows it all.
func TestTheSmallTranslationPanelShowsItsBeginningAndSaysThereIsMore(t *testing.T) {
	pane := tea.WindowSizeMsg{Width: 87, Height: 15}
	english := "FIRST sentence of the translation. " +
		strings.Repeat("Another sentence in the middle of it. ", 20) + "LAST sentence."

	drawn := strings.Join(laidOut(t, pane, "Bitte behebe den Test.", english), "\n")

	if !strings.Contains(drawn, "FIRST sentence") {
		t.Errorf("the panel does not show the beginning:\n%s", drawn)
	}
	if strings.Contains(drawn, "LAST sentence") {
		t.Error("the panel shows the end, so it is not showing the beginning")
	}
	if !strings.Contains(drawn, "…") {
		t.Error("nothing says the translation goes on")
	}
	if !strings.Contains(lastLine(strings.Split(drawn, "\n")[len(strings.Split(drawn, "\n"))-1]), "tab") {
		t.Errorf("the footer does not name the key that shows the rest: %q",
			lastLine(drawn))
	}
	if strings.Contains(drawn, overlay.ScrollThumb) {
		t.Error("the small panel has a scrollbar, which cannot be used from there")
	}
}

func TestAShortTranslationIsShownWholeWithoutFuss(t *testing.T) {
	pane := tea.WindowSizeMsg{Width: 87, Height: 15}
	drawn := strings.Join(laidOut(t, pane, "Bitte behebe den Test.",
		"Please fix the failing test."), "\n")

	if strings.Contains(drawn, "…") {
		t.Error("a translation that fits is marked as cut off")
	}
	if strings.Contains(lastLine(drawn), "tab") {
		t.Error("the footer invites reading a translation that is already whole")
	}
}

func TestTheDraftKeepsItsBar(t *testing.T) {
	pane := tea.WindowSizeMsg{Width: 87, Height: 15}
	drawn := strings.Join(laidOut(t, pane, paragraph, ""), "\n")

	if !strings.Contains(drawn, overlay.ScrollThumb) {
		t.Errorf("the draft has no bar though it holds more than it shows:\n%s", drawn)
	}
}

func TestNothingIsDrawnWiderThanANarrowPane(t *testing.T) {
	for _, pane := range []tea.WindowSizeMsg{
		{Width: 20, Height: 12},
		{Width: 30, Height: 15},
		{Width: 34, Height: 15},
	} {
		for index, line := range laidOut(t, pane, paragraph, "Please fix the failing test.") {
			if width := lipgloss.Width(line); width > pane.Width {
				t.Errorf("%d columns: row %d is %d wide", pane.Width, index, width)
			}
		}
	}
}

func overlayWith(t *testing.T, pane tea.WindowSizeMsg, draft string) tea.Model {
	t.Helper()
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI)
	t.Cleanup(func() { lipgloss.SetColorProfile(previous) })

	target := &recordingTarget{}
	flow := promptflow.New(stubTranslator{english: english}, target, target)
	var model tea.Model = overlay.New(context.Background(), flow, overlay.Options{
		Service: "deepl", Language: "EN-US", Vim: true,
	})
	model, _ = model.Update(pane)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(draft)})
	return model
}

func lastWord(text string) string {
	words := strings.Fields(text)
	return words[len(words)-1]
}

// The cursor's own styling splits the text it sits on, so what is on screen is
// read without it.
func plain(view string) string {
	return ansi.Strip(view)
}
