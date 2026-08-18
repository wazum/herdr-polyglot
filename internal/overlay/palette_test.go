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

// Herdr paints the popup; a space of ours would cover that with this terminal's
// own background and show as a seam. So no line is padded out.
func TestTheOverlayDoesNotPaintOverWhatHerdrDrew(t *testing.T) {
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.Ascii)
	defer lipgloss.SetColorProfile(previous)

	flow := promptflow.New(stubTranslator{english: english}, &recordingTarget{})
	var model tea.Model = overlay.New(context.Background(), flow, overlay.Options{
		Service: "deepl", Language: "EN-US", Vim: true, Live: true,
	})
	model, _ = model.Update(tea.WindowSizeMsg{Width: 87, Height: 17})

	lines := strings.Split(model.View(), "\n")
	if len(lines) > 17 {
		t.Errorf("drew %d rows into a pane of 17", len(lines))
	}
	for row, line := range lines {
		if width := lipgloss.Width(line); width > 87 {
			t.Errorf("row %d is %d columns wide, the pane is 87", row, width)
		}
	}
	if trailing := lines[len(lines)-1]; strings.HasSuffix(trailing, "  ") {
		t.Errorf("the last row ends in padding: %q", trailing)
	}
}

// Herdr paints the popup in its own colour and reports it when the terminal is
// asked. Every cell the overlay draws has to carry that colour, or the content
// reads as a patch of a different shade.
func TestEveryCellCarriesTheColourHerdrPainted(t *testing.T) {
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(previous)

	const painted = "#282c34"
	flow := promptflow.New(stubTranslator{english: english}, &recordingTarget{})
	var model tea.Model = overlay.New(context.Background(), flow, overlay.Options{
		Service: "deepl", Language: "EN-US", Vim: true, Live: true, Pulse: true,
		Background: painted,
	})
	model, _ = model.Update(tea.WindowSizeMsg{Width: 87, Height: 17})

	// 40;2;40;44;52 is that colour as a background.
	const asBackground = "48;2;40;44;52"
	lines := strings.Split(model.View(), "\n")
	if len(lines) != 17 {
		t.Errorf("drew %d rows, want the pane's 17 filled", len(lines))
	}
	for row, line := range lines {
		if !strings.Contains(line, asBackground) {
			t.Errorf("row %d carries no background: %q", row, line)
		}
		if width := lipgloss.Width(line); width != 87 {
			t.Errorf("row %d is %d columns, want the pane's 87", row, width)
		}
	}
}
