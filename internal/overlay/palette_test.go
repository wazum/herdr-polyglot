package overlay_test

import (
	"bytes"
	"context"
	"fmt"
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

	flow := promptflow.New(stubTranslator{english: english}, &recordingTarget{}, &recordingTarget{})
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

// One popup height serves both states: live translation can be switched on at any
// time and a popup cannot be resized, so the room for the English is always there
// — taken by the draft while there is nothing to show in it.
func TestThePopupIsTallEnoughForTheDraftAndTheTranslation(t *testing.T) {
	for _, live := range []bool{false, true} {
		flow := promptflow.New(stubTranslator{english: english}, &recordingTarget{}, &recordingTarget{})
		var model tea.Model = overlay.New(context.Background(), flow, overlay.Options{
			Service: "deepl", Language: "EN-US", Vim: true, Live: live,
		})
		model, _ = model.Update(tea.WindowSizeMsg{
			Width:  overlay.PopupWidth - overlay.PopupBorder,
			Height: overlay.PopupHeight() - overlay.PopupBorder,
		})

		rows := overlay.DraftRows(model.(overlay.Model))
		if live && rows != overlay.DraftHeight {
			t.Errorf("with live on the draft got %d rows, want its full %d", rows, overlay.DraftHeight)
		}
		if !live && rows < overlay.DraftHeight {
			t.Errorf("with live off the draft got %d rows, want at least %d", rows, overlay.DraftHeight)
		}
		if !overlay.ShowsEnglish(model.(overlay.Model), true) {
			t.Error("the popup has no room for the translation, so ctrl+l would pay for nothing")
		}
	}
}

// A popup can end up shorter than asked for. One line is not a box to write a
// prompt in, so the translation steps aside before the draft is squeezed.
func TestAShortPopupGivesTheDraftTheRoomFirst(t *testing.T) {
	flow := promptflow.New(stubTranslator{english: english}, &recordingTarget{}, &recordingTarget{})
	var model tea.Model = overlay.New(context.Background(), flow, overlay.Options{
		Service: "deepl", Language: "EN-US", Live: true,
	})
	model, _ = model.Update(tea.WindowSizeMsg{Width: 87, Height: 10})

	rows := overlay.DraftRows(model.(overlay.Model))
	if rows < overlay.MinDraftRows {
		t.Errorf("the draft got %d rows in a pane of 10, want at least %d",
			rows, overlay.MinDraftRows)
	}
	if boxes := strings.Count(model.View(), "╭"); boxes > 1 {
		t.Errorf("the translation is still shown in a pane too short for both: %d boxes", boxes)
	}
}

// Herdr paints the popup in whatever its theme says. A background of ours would
// replace that colour cell by cell, so the overlay sets none.
func TestTheOverlayNeverSetsABackground(t *testing.T) {
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(previous)

	flow := promptflow.New(stubTranslator{english: english}, &recordingTarget{}, &recordingTarget{})
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

// Every key the popup answers to belongs on the footer line. If they stop
// fitting, they need shorter names, not a help screen to hide behind.
func TestTheFooterListsEveryKeyAndStillFits(t *testing.T) {
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.Ascii)
	defer lipgloss.SetColorProfile(previous)

	const paneWidth = 87
	for _, vim := range []bool{false, true} {
		for _, normalMode := range []bool{false, true} {
			flow := promptflow.New(stubTranslator{english: english}, &recordingTarget{}, &recordingTarget{})
			var model tea.Model = overlay.New(context.Background(), flow, overlay.Options{
				Service: "deepl", Language: "EN-US", Vim: vim, Live: true,
			})
			model, _ = model.Update(tea.WindowSizeMsg{Width: paneWidth, Height: 17})
			if vim && normalMode {
				model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
			}

			// Normal mode has vim's own way to clear a line, so ctrl+u steps aside
			// there to leave room for i.
			expected := []string{"alt+enter", "ctrl+r", "ctrl+l", "ctrl+u", "esc"}
			if vim && normalMode {
				expected = []string{"alt+enter", "ctrl+r", "ctrl+l", "i ", "esc"}
			}

			footer := lastLine(model.View())
			for _, key := range expected {
				if !strings.Contains(footer, key) {
					t.Errorf("vim=%v normal=%v: the footer does not mention %s: %q",
						vim, normalMode, key, footer)
				}
			}
			if width := lipgloss.Width(footer); width > paneWidth {
				t.Errorf("vim=%v normal=%v: the footer is %d columns, the pane is %d: %q",
					vim, normalMode, width, paneWidth, footer)
			}
		}
	}
}

func lastLine(rendered string) string {
	lines := strings.Split(rendered, "\n")
	return lines[len(lines)-1]
}

type keptDraft struct{}

func (keptDraft) Load() (string, error) { return "ein alter Entwurf", nil }
func (keptDraft) Save(string) error     { return nil }
func (keptDraft) Clear() error          { return nil }

// The heading grows with what it has to say; wrapping it onto the draft would
// break the whole popup, so it is cut instead.
func TestTheHeadingNeverOutgrowsThePane(t *testing.T) {
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.Ascii)
	defer lipgloss.SetColorProfile(previous)

	const paneWidth = 87
	flow := promptflow.New(stubTranslator{english: english}, &recordingTarget{}, &recordingTarget{})
	var model tea.Model = overlay.New(context.Background(), flow, overlay.Options{
		Service: "deepl", Language: "EN-US", Vim: true, Live: true, Pulse: true,
		Review: true, Drafts: keptDraft{},
	})
	model, _ = model.Update(tea.WindowSizeMsg{Width: paneWidth, Height: 17})
	model, _ = model.Update(overlay.UsageSeen(promptflow.Usage{Used: 999_900, Limit: 1_000_000}))

	heading := strings.Split(model.View(), "\n")[0]
	if width := lipgloss.Width(heading); width > paneWidth-1 {
		t.Errorf("the heading is %d columns in a pane of %d: %q", width, paneWidth, heading)
	}
	if !strings.Contains(heading, "chars") {
		t.Errorf("the heading does not say what the number counts: %q", heading)
	}
	t.Logf("heading at its longest: %q (%d columns)", heading, lipgloss.Width(heading))
}

// Toggling live translation or the delivery must not shuffle the rest of the
// heading along the line.
func TestTheHeadingKeepsItsWidthWhenSettingsChange(t *testing.T) {
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.Ascii)
	defer lipgloss.SetColorProfile(previous)

	widths := map[string]int{}
	for _, live := range []bool{true, false} {
		for _, review := range []bool{true, false} {
			flow := promptflow.New(stubTranslator{english: english}, &recordingTarget{}, &recordingTarget{})
			var model tea.Model = overlay.New(context.Background(), flow, overlay.Options{
				Service: "deepl", Language: "EN-US", Live: live, Review: review, Pulse: true,
			})
			model, _ = model.Update(tea.WindowSizeMsg{Width: 87, Height: 17})

			heading := strings.Split(model.View(), "\n")[0]
			widths[fmt.Sprintf("live=%v review=%v", live, review)] = lipgloss.Width(heading)
		}
	}

	var first int
	for label, width := range widths {
		if first == 0 {
			first = width
			continue
		}
		if width != first {
			t.Errorf("%s makes the heading %d columns, another combination gives %d: %v",
				label, width, first, widths)
		}
	}
}
