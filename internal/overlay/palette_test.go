package overlay_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/wazum/herdr-polyglot/internal/overlay"
	"github.com/wazum/herdr-polyglot/internal/promptflow"
)

// Herdr does not tell plugins which theme is active, but it does paint the
// terminal palette. Sticking to that palette is what makes the overlay follow
// the theme instead of fighting it.
func TestTheOverlayPaintsWithTheTerminalPaletteOnly(t *testing.T) {
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(previous)

	flow := promptflow.New(stubTranslator{english: english}, &recordingTarget{})
	var model tea.Model = overlay.New(context.Background(), flow, overlay.Options{
		Service: "deepl", Language: "EN-US", Vim: true, Live: true,
	})
	model, _ = model.Update(tea.WindowSizeMsg{Width: 100, Height: 24})

	rendered := []byte(model.View())
	for _, fixed := range []string{"38;2;", "48;2;", "38;5;", "48;5;"} {
		if bytes.Contains(rendered, []byte(fixed)) {
			t.Errorf("overlay paints fixed colour %q instead of using the terminal palette", fixed)
		}
	}
}

func TestThePopupIsTallEnoughToLeaveTheDraftItsFullHeight(t *testing.T) {
	for _, live := range []bool{false, true} {
		flow := promptflow.New(stubTranslator{english: english}, &recordingTarget{})
		var model tea.Model = overlay.New(context.Background(), flow, overlay.Options{
			Service: "deepl", Language: "EN-US", Vim: true, Live: live,
		})
		model, _ = model.Update(tea.WindowSizeMsg{
			Width:  overlay.PopupWidth - overlay.PopupBorder,
			Height: overlay.PopupHeight(live) - overlay.PopupBorder,
		})

		if rows := overlay.DraftRows(model.(overlay.Model)); rows != overlay.DraftHeight {
			t.Errorf("live=%v: the draft got %d rows in a popup of %d, want its full %d",
				live, rows, overlay.PopupHeight(live), overlay.DraftHeight)
		}
	}
}

// Whatever herdr paints behind the popup shows through any cell the overlay
// leaves untouched, which reads as a stripe down one side.
func TestTheOverlayCoversEveryCellOfItsPane(t *testing.T) {
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.Ascii)
	defer lipgloss.SetColorProfile(previous)

	for _, pane := range []struct{ width, height int }{
		{88, 15}, {90, 17}, {70, 12}, {120, 20},
	} {
		flow := promptflow.New(stubTranslator{english: english}, &recordingTarget{})
		var model tea.Model = overlay.New(context.Background(), flow, overlay.Options{
			Service: "deepl", Language: "EN-US", Vim: true, Live: true,
		})
		model, _ = model.Update(tea.WindowSizeMsg{Width: pane.width, Height: pane.height})

		lines := strings.Split(model.View(), "\n")
		if len(lines) != pane.height {
			t.Errorf("pane %dx%d: drew %d rows, want every row covered",
				pane.width, pane.height, len(lines))
		}
		for row, line := range lines {
			if width := lipgloss.Width(line); width != pane.width {
				t.Errorf("pane %dx%d: row %d is %d columns, want %d",
					pane.width, pane.height, row, width, pane.width)
			}
		}
	}
}
