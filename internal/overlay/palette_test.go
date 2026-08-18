package overlay_test

import (
	"bytes"
	"context"
	"strconv"
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

// Herdr paints the popup in whatever its theme says. A background of ours would
// replace that colour cell by cell, so the overlay sets none.
func TestTheOverlayNeverSetsABackground(t *testing.T) {
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(previous)

	flow := promptflow.New(stubTranslator{english: english}, &recordingTarget{})
	var model tea.Model = overlay.New(context.Background(), flow, overlay.Options{
		Service: "deepl", Language: "EN-US", Vim: true, Live: true, Pulse: true,
	})
	model, _ = model.Update(tea.WindowSizeMsg{Width: 87, Height: 17})

	for _, sequence := range sgrSequences(model.View()) {
		for _, parameter := range strings.Split(sequence, ";") {
			if setsBackground(parameter) {
				t.Errorf("a style sets background %q, in \x1b[%sm", parameter, sequence)
			}
		}
	}
}

func sgrSequences(rendered string) []string {
	var found []string
	for _, after := range strings.Split(rendered, "\x1b[")[1:] {
		if end := strings.IndexRune(after, 'm'); end >= 0 {
			found = append(found, after[:end])
		}
	}
	return found
}

// 40-47 and 100-107 are the palette backgrounds, 48 introduces an exact one.
func setsBackground(parameter string) bool {
	number, err := strconv.Atoi(parameter)
	if err != nil {
		return false
	}
	switch {
	case number >= 40 && number <= 48,
		number >= 100 && number <= 107:
		return true
	default:
		return false
	}
}
