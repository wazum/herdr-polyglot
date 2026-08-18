package vimarea_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/wazum/herdr-polyglot/internal/vimarea"
)

func TestTypingWorksStraightAwayBecauseTheDraftStartsInInsertMode(t *testing.T) {
	t.Parallel()
	area := vimarea.New(vimarea.WithVim(true))

	if area.Mode() != vimarea.Insert {
		t.Fatalf("mode is %v, want insert", area.Mode())
	}

	area = press(area, "hallo")
	if area.Value() != "hallo" {
		t.Errorf("value is %q, want the typed text", area.Value())
	}
}

func TestEscapeLeavesInsertModeAndLettersStopReachingTheDraft(t *testing.T) {
	t.Parallel()

	area := press(vimarea.New(vimarea.WithVim(true)), "hallo")
	area = press(area, "<esc>")

	if area.Mode() != vimarea.Normal {
		t.Fatalf("mode is %v, want normal", area.Mode())
	}

	area = press(area, "jkl")
	if area.Value() != "hallo" {
		t.Errorf("value is %q, want it untouched in normal mode", area.Value())
	}
}

func TestHjklMovesTheCursor(t *testing.T) {
	t.Parallel()
	area := boxed(t, "eins\nzwei")

	area = press(area, "ll")
	if area.Row() != 0 || area.Column() != 2 {
		t.Errorf("cursor at %d:%d, want 0:2 after ll", area.Row(), area.Column())
	}

	area = press(area, "j")
	if area.Row() != 1 {
		t.Errorf("cursor on row %d, want row 1 after j", area.Row())
	}

	area = press(area, "h")
	if area.Column() != 1 {
		t.Errorf("cursor at column %d, want 1 after h", area.Column())
	}

	area = press(area, "k")
	if area.Row() != 0 {
		t.Errorf("cursor on row %d, want row 0 after k", area.Row())
	}
}

func TestZeroAndDollarJumpToTheLineEdges(t *testing.T) {
	t.Parallel()
	area := boxed(t, "eins zwei")

	area = press(area, "$")
	if area.Column() != len("eins zwei")-1 {
		t.Errorf("cursor at column %d, want the last character after $", area.Column())
	}

	area = press(area, "0")
	if area.Column() != 0 {
		t.Errorf("cursor at column %d, want 0 after 0", area.Column())
	}
}

func TestWordMotionsMoveBetweenWords(t *testing.T) {
	t.Parallel()
	area := boxed(t, "eins zwei drei")

	area = press(area, "w")
	if area.Column() != 5 {
		t.Errorf("cursor at column %d, want the start of zwei after w", area.Column())
	}

	area = press(area, "w")
	if area.Column() != 10 {
		t.Errorf("cursor at column %d, want the start of drei after w", area.Column())
	}

	area = press(area, "b")
	if area.Column() != 5 {
		t.Errorf("cursor at column %d, want the start of zwei after b", area.Column())
	}
}

func TestGgAndGJumpToTheFirstAndLastLine(t *testing.T) {
	t.Parallel()
	area := boxed(t, "eins\nzwei\ndrei")

	area = press(area, "G")
	if area.Row() != 2 {
		t.Errorf("cursor on row %d, want the last row after G", area.Row())
	}

	area = press(area, "gg")
	if area.Row() != 0 {
		t.Errorf("cursor on row %d, want the first row after gg", area.Row())
	}
}

func TestInsertEntriesStartTypingWhereVimWould(t *testing.T) {
	t.Parallel()

	for _, entry := range []struct {
		name     string
		sequence string
		typed    string
		want     string
	}{
		{"i inserts before the cursor", "i", "X", "Xeins zwei"},
		{"a inserts after the cursor", "a", "X", "eXins zwei"},
		{"I inserts at the line start", "$I", "X", "Xeins zwei"},
		{"A appends at the line end", "0A", "X", "eins zweiX"},
		{"o opens a line below", "o", "X", "eins zwei\nX"},
		{"O opens a line above", "O", "X", "X\neins zwei"},
	} {
		t.Run(entry.name, func(t *testing.T) {
			t.Parallel()
			area := press(boxed(t, "eins zwei"), entry.sequence)

			if area.Mode() != vimarea.Insert {
				t.Fatalf("mode is %v, want insert after %q", area.Mode(), entry.sequence)
			}

			area = press(area, entry.typed)
			if area.Value() != entry.want {
				t.Errorf("value is %q, want %q", area.Value(), entry.want)
			}
		})
	}
}

func TestXDeletesTheCharacterUnderTheCursor(t *testing.T) {
	t.Parallel()

	area := press(boxed(t, "eins"), "x")

	if area.Value() != "ins" {
		t.Errorf("value is %q, want the first character gone", area.Value())
	}
}

func TestDdDeletesTheWholeLine(t *testing.T) {
	t.Parallel()

	area := press(boxed(t, "eins\nzwei\ndrei"), "jdd")

	if area.Value() != "eins\ndrei" {
		t.Errorf("value is %q, want the second line gone", area.Value())
	}
}

func TestDDeletesToTheEndOfTheLine(t *testing.T) {
	t.Parallel()

	area := press(boxed(t, "eins zwei"), "wD")

	if area.Value() != "eins " {
		t.Errorf("value is %q, want everything from the cursor gone", area.Value())
	}
}

func TestDwDeletesAWord(t *testing.T) {
	t.Parallel()

	area := press(boxed(t, "eins zwei drei"), "wdw")

	if area.Value() != "eins drei" {
		t.Errorf("value is %q, want the second word gone", area.Value())
	}
}

func TestCwReplacesAWord(t *testing.T) {
	t.Parallel()

	area := press(boxed(t, "eins zwei"), "wcw")
	if area.Mode() != vimarea.Insert {
		t.Fatalf("mode is %v, want insert after cw", area.Mode())
	}

	area = press(area, "drei")
	if area.Value() != "eins drei" {
		t.Errorf("value is %q, want the word replaced", area.Value())
	}
}

func TestCcClearsTheLineAndStartsTyping(t *testing.T) {
	t.Parallel()

	area := press(boxed(t, "eins\nzwei"), "jcc")
	area = press(area, "drei")

	if area.Value() != "eins\ndrei" {
		t.Errorf("value is %q, want the second line rewritten", area.Value())
	}
}

func TestYyAndPDuplicateALine(t *testing.T) {
	t.Parallel()

	area := press(boxed(t, "eins\nzwei"), "yyp")

	if area.Value() != "eins\neins\nzwei" {
		t.Errorf("value is %q, want the first line pasted below itself", area.Value())
	}
}

func TestCapitalPPastesAboveTheCursor(t *testing.T) {
	t.Parallel()

	area := press(boxed(t, "eins\nzwei"), "jyyP")

	if area.Value() != "eins\nzwei\nzwei" {
		t.Errorf("value is %q, want the yanked line pasted above", area.Value())
	}
}

func TestUndoRestoresTheDraftAndTheCursor(t *testing.T) {
	t.Parallel()

	area := press(boxed(t, "eins\nzwei\ndrei"), "jdd")
	area = press(area, "u")

	if area.Value() != "eins\nzwei\ndrei" {
		t.Errorf("value is %q, want the deleted line back", area.Value())
	}
	if area.Row() != 1 {
		t.Errorf("cursor on row %d, want it back on the line that was deleted", area.Row())
	}
}

func TestACountRepeatsAMotionAndAnEdit(t *testing.T) {
	t.Parallel()

	area := press(boxed(t, "eins zwei drei"), "3x")
	if area.Value() != "s zwei drei" {
		t.Errorf("value is %q, want three characters gone", area.Value())
	}

	area = press(boxed(t, "eins\nzwei\ndrei\nvier"), "3j")
	if area.Row() != 3 {
		t.Errorf("cursor on row %d, want row 3 after 3j", area.Row())
	}
}

func TestEditingKeepsCharactersOutsideAscii(t *testing.T) {
	t.Parallel()

	area := boxed(t, "prüfe die Übersetzung")
	area = press(area, "wx")

	if area.Value() != "prüfe ie Übersetzung" {
		t.Errorf("value is %q, want only the d of die gone", area.Value())
	}

	area = press(area, "0")
	if area.Column() != 0 {
		t.Errorf("cursor at column %d, want 0", area.Column())
	}
	area = press(area, "$")
	if area.Column() != len([]rune("prüfe ie Übersetzung"))-1 {
		t.Errorf("cursor at column %d, want the last rune", area.Column())
	}
}

func TestWithoutVimBindingsEveryKeyGoesIntoTheDraft(t *testing.T) {
	t.Parallel()
	area := vimarea.New(vimarea.WithVim(false))

	if area.Modal() {
		t.Fatal("Modal is true, want plain editing")
	}

	area = press(area, "hallo")
	area = press(area, "<esc>")
	area = press(area, "jkl")

	if area.Value() != "hallojkl" {
		t.Errorf("value is %q, want every key typed as text", area.Value())
	}
	if area.Mode() != vimarea.Insert {
		t.Errorf("mode is %v, want to stay in insert", area.Mode())
	}
}

func TestPastedTextLandsInTheDraftAndTypingContinuesAfterIt(t *testing.T) {
	t.Parallel()
	area := press(vimarea.New(vimarea.WithVim(true)), "Kontext: ")

	area = pasted(area, "erste Zeile\nzweite Zeile mit Übersetzung")
	area = press(area, " – weiter")

	const want = "Kontext: erste Zeile\nzweite Zeile mit Übersetzung – weiter"
	if area.Value() != want {
		t.Errorf("value is %q, want %q", area.Value(), want)
	}
}

func TestPastingInNormalModeInsertsTextInsteadOfRunningItAsCommands(t *testing.T) {
	t.Parallel()
	area := boxed(t, "Kontext")

	// Every character here is also a normal-mode command: d deletes, x cuts,
	// p pastes. None of them may fire.
	area = pasted(area, " dxp")

	if area.Value() != " dxpKontext" {
		t.Errorf("value is %q, want the pasted text inserted verbatim", area.Value())
	}
	if area.Mode() != vimarea.Normal {
		t.Errorf("mode is %v, want to stay in normal as nvim does", area.Mode())
	}
}

func TestPastingWorksWithoutVimBindings(t *testing.T) {
	t.Parallel()

	area := pasted(vimarea.New(vimarea.WithVim(false)), "eins\nzwei")

	if area.Value() != "eins\nzwei" {
		t.Errorf("value is %q, want the pasted text", area.Value())
	}
}

// The text area keeps an internal pointer to whichever style set is active, so
// styles must be applied before it is focused or they are silently ignored.
func TestTheConfiguredStylesReachTheRenderedDraft(t *testing.T) {
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(previous)

	area := vimarea.New(
		vimarea.WithPlaceholder("Schreib etwas …"),
		vimarea.WithStyles(
			lipgloss.NewStyle().Foreground(lipgloss.Color("#E6E6F0")),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#8A8FA3")),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#B79BFF")),
		),
	)
	area.SetWidth(40)
	area.SetHeight(3)

	if rendered := area.View(); strings.Contains(rendered, "\x1b[40m") {
		t.Errorf("rendered draft paints a black background:\n%q", rendered)
	}
}
