package vimarea_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wazum/herdr-polyglot/internal/vimarea"
)

func keys(area vimarea.Model, sequence string) vimarea.Model {
	for _, r := range sequence {
		area, _ = area.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return area
}

func press(area vimarea.Model, keyType tea.KeyType) vimarea.Model {
	area, _ = area.Update(tea.KeyMsg{Type: keyType})
	return area
}

func normalWith(text string) vimarea.Model {
	area := vimarea.New(vimarea.WithVim(true))
	area.SetValue(text)
	return press(area, tea.KeyEsc)
}

func TestTypingWorksStraightAwayBecauseTheDraftStartsInInsertMode(t *testing.T) {
	t.Parallel()
	area := vimarea.New(vimarea.WithVim(true))

	if area.Mode() != vimarea.Insert {
		t.Fatalf("mode is %v, want insert", area.Mode())
	}

	area = keys(area, "hallo")
	if area.Value() != "hallo" {
		t.Errorf("value is %q, want the typed text", area.Value())
	}
}

func TestEscapeLeavesInsertModeAndLettersStopReachingTheDraft(t *testing.T) {
	t.Parallel()

	area := keys(vimarea.New(vimarea.WithVim(true)), "hallo")
	area = press(area, tea.KeyEsc)

	if area.Mode() != vimarea.Normal {
		t.Fatalf("mode is %v, want normal", area.Mode())
	}

	area = keys(area, "jkl")
	if area.Value() != "hallo" {
		t.Errorf("value is %q, want it untouched in normal mode", area.Value())
	}
}

func TestHjklMovesTheCursor(t *testing.T) {
	t.Parallel()
	area := normalWith("eins\nzwei")

	area = keys(area, "ll")
	if area.Row() != 0 || area.Column() != 2 {
		t.Errorf("cursor at %d:%d, want 0:2 after ll", area.Row(), area.Column())
	}

	area = keys(area, "j")
	if area.Row() != 1 {
		t.Errorf("cursor on row %d, want row 1 after j", area.Row())
	}

	area = keys(area, "h")
	if area.Column() != 1 {
		t.Errorf("cursor at column %d, want 1 after h", area.Column())
	}

	area = keys(area, "k")
	if area.Row() != 0 {
		t.Errorf("cursor on row %d, want row 0 after k", area.Row())
	}
}

func TestZeroAndDollarJumpToTheLineEdges(t *testing.T) {
	t.Parallel()
	area := normalWith("eins zwei")

	area = keys(area, "$")
	if area.Column() != len("eins zwei")-1 {
		t.Errorf("cursor at column %d, want the last character after $", area.Column())
	}

	area = keys(area, "0")
	if area.Column() != 0 {
		t.Errorf("cursor at column %d, want 0 after 0", area.Column())
	}
}

func TestWordMotionsMoveBetweenWords(t *testing.T) {
	t.Parallel()
	area := normalWith("eins zwei drei")

	area = keys(area, "w")
	if area.Column() != 5 {
		t.Errorf("cursor at column %d, want the start of zwei after w", area.Column())
	}

	area = keys(area, "w")
	if area.Column() != 10 {
		t.Errorf("cursor at column %d, want the start of drei after w", area.Column())
	}

	area = keys(area, "b")
	if area.Column() != 5 {
		t.Errorf("cursor at column %d, want the start of zwei after b", area.Column())
	}
}

func TestGgAndGJumpToTheFirstAndLastLine(t *testing.T) {
	t.Parallel()
	area := normalWith("eins\nzwei\ndrei")

	area = keys(area, "G")
	if area.Row() != 2 {
		t.Errorf("cursor on row %d, want the last row after G", area.Row())
	}

	area = keys(area, "gg")
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
			area := keys(normalWith("eins zwei"), entry.sequence)

			if area.Mode() != vimarea.Insert {
				t.Fatalf("mode is %v, want insert after %q", area.Mode(), entry.sequence)
			}

			area = keys(area, entry.typed)
			if area.Value() != entry.want {
				t.Errorf("value is %q, want %q", area.Value(), entry.want)
			}
		})
	}
}

func TestXDeletesTheCharacterUnderTheCursor(t *testing.T) {
	t.Parallel()

	area := keys(normalWith("eins"), "x")

	if area.Value() != "ins" {
		t.Errorf("value is %q, want the first character gone", area.Value())
	}
}

func TestDdDeletesTheWholeLine(t *testing.T) {
	t.Parallel()

	area := keys(normalWith("eins\nzwei\ndrei"), "jdd")

	if area.Value() != "eins\ndrei" {
		t.Errorf("value is %q, want the second line gone", area.Value())
	}
}

func TestDDeletesToTheEndOfTheLine(t *testing.T) {
	t.Parallel()

	area := keys(normalWith("eins zwei"), "wD")

	if area.Value() != "eins " {
		t.Errorf("value is %q, want everything from the cursor gone", area.Value())
	}
}

func TestDwDeletesAWord(t *testing.T) {
	t.Parallel()

	area := keys(normalWith("eins zwei drei"), "wdw")

	if area.Value() != "eins drei" {
		t.Errorf("value is %q, want the second word gone", area.Value())
	}
}

func TestCwReplacesAWord(t *testing.T) {
	t.Parallel()

	area := keys(normalWith("eins zwei"), "wcw")
	if area.Mode() != vimarea.Insert {
		t.Fatalf("mode is %v, want insert after cw", area.Mode())
	}

	area = keys(area, "drei")
	if area.Value() != "eins drei" {
		t.Errorf("value is %q, want the word replaced", area.Value())
	}
}

func TestCcClearsTheLineAndStartsTyping(t *testing.T) {
	t.Parallel()

	area := keys(normalWith("eins\nzwei"), "jcc")
	area = keys(area, "drei")

	if area.Value() != "eins\ndrei" {
		t.Errorf("value is %q, want the second line rewritten", area.Value())
	}
}

func TestYyAndPDuplicateALine(t *testing.T) {
	t.Parallel()

	area := keys(normalWith("eins\nzwei"), "yyp")

	if area.Value() != "eins\neins\nzwei" {
		t.Errorf("value is %q, want the first line pasted below itself", area.Value())
	}
}

func TestCapitalPPastesAboveTheCursor(t *testing.T) {
	t.Parallel()

	area := keys(normalWith("eins\nzwei"), "jyyP")

	if area.Value() != "eins\nzwei\nzwei" {
		t.Errorf("value is %q, want the yanked line pasted above", area.Value())
	}
}

func TestUndoRestoresTheDraftAndTheCursor(t *testing.T) {
	t.Parallel()

	area := keys(normalWith("eins\nzwei\ndrei"), "jdd")
	area = keys(area, "u")

	if area.Value() != "eins\nzwei\ndrei" {
		t.Errorf("value is %q, want the deleted line back", area.Value())
	}
	if area.Row() != 1 {
		t.Errorf("cursor on row %d, want it back on the line that was deleted", area.Row())
	}
}

func TestACountRepeatsAMotionAndAnEdit(t *testing.T) {
	t.Parallel()

	area := keys(normalWith("eins zwei drei"), "3x")
	if area.Value() != "s zwei drei" {
		t.Errorf("value is %q, want three characters gone", area.Value())
	}

	area = keys(normalWith("eins\nzwei\ndrei\nvier"), "3j")
	if area.Row() != 3 {
		t.Errorf("cursor on row %d, want row 3 after 3j", area.Row())
	}
}

func TestEditingKeepsCharactersOutsideAscii(t *testing.T) {
	t.Parallel()

	area := normalWith("prüfe die Übersetzung")
	area = keys(area, "wx")

	if area.Value() != "prüfe ie Übersetzung" {
		t.Errorf("value is %q, want only the d of die gone", area.Value())
	}

	area = keys(area, "0")
	if area.Column() != 0 {
		t.Errorf("cursor at column %d, want 0", area.Column())
	}
	area = keys(area, "$")
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

	area = keys(area, "hallo")
	area = press(area, tea.KeyEsc)
	area = keys(area, "jkl")

	if area.Value() != "hallojkl" {
		t.Errorf("value is %q, want every key typed as text", area.Value())
	}
	if area.Mode() != vimarea.Insert {
		t.Errorf("mode is %v, want to stay in insert", area.Mode())
	}
}
