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

// The braille the plugin signs the empty box with. Only the first line is needed
// to tell whether it is there.
const logoLine = "⢀⣼⣿⣿⣿⡟⠛⢻⡿⣧"

func drawn(t *testing.T, options overlay.Options) (empty, written string) {
	t.Helper()
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI)
	t.Cleanup(func() { lipgloss.SetColorProfile(previous) })

	flow := promptflow.New(stubTranslator{english: english}, &recordingTarget{}, &recordingTarget{})
	var model tea.Model = overlay.New(context.Background(), flow, options)
	model, _ = model.Update(tea.WindowSizeMsg{Width: 87, Height: 15})
	empty = model.View()

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Bitte behebe den Test")})
	return empty, model.View()
}

// An empty box has room nobody is using.
func TestTheEmptyDraftBoxIsSigned(t *testing.T) {
	empty, written := drawn(t, overlay.Options{Service: "deepl", Language: "EN-US", Logo: true})

	if !strings.Contains(empty, logoLine) {
		t.Errorf("the empty box is unsigned:\n%s", empty)
	}
	if strings.Contains(written, logoLine) {
		t.Errorf("the signature is still there over the writing:\n%s", written)
	}
	if !strings.Contains(empty, "own language") {
		t.Error("the signature took the placeholder's place, want both")
	}
}

func TestTheSignatureCanBeTurnedOff(t *testing.T) {
	empty, _ := drawn(t, overlay.Options{Service: "deepl", Language: "EN-US"})

	if strings.Contains(empty, logoLine) {
		t.Errorf("the box is signed with the setting off:\n%s", empty)
	}
}

// Whatever it draws, the box stays the same shape.
func TestTheSignatureChangesNoLineWidth(t *testing.T) {
	signed, _ := drawn(t, overlay.Options{Service: "deepl", Language: "EN-US", Logo: true})
	plain, _ := drawn(t, overlay.Options{Service: "deepl", Language: "EN-US"})

	signedLines, plainLines := strings.Split(signed, "\n"), strings.Split(plain, "\n")
	if len(signedLines) != len(plainLines) {
		t.Fatalf("the signed box is %d lines, the plain one %d", len(signedLines), len(plainLines))
	}
	for index := range signedLines {
		if lipgloss.Width(signedLines[index]) != lipgloss.Width(plainLines[index]) {
			t.Errorf("line %d is %d columns signed and %d plain",
				index, lipgloss.Width(signedLines[index]), lipgloss.Width(plainLines[index]))
		}
	}
}

// A box too short for the signature simply goes without it.
func TestAShortBoxIsLeftUnsigned(t *testing.T) {
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI)
	t.Cleanup(func() { lipgloss.SetColorProfile(previous) })

	flow := promptflow.New(stubTranslator{english: english}, &recordingTarget{}, &recordingTarget{})
	var model tea.Model = overlay.New(context.Background(), flow, overlay.Options{
		Service: "deepl", Language: "EN-US", Logo: true,
	})
	model, _ = model.Update(tea.WindowSizeMsg{Width: 87, Height: 8})

	if view := model.View(); strings.Contains(view, logoLine) {
		t.Errorf("a box with no room for it was signed anyway:\n%s", view)
	}
}
