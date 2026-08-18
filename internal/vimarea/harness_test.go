package vimarea_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wazum/herdr-polyglot/internal/vimarea"
)

// box builds a modal text area seeded with text, cursor at row 0 col 0, in
// NORMAL mode (SetValue leaves INSERT, so esc first).
func box(t *testing.T, text string) vimarea.Model {
	t.Helper()
	m := vimarea.New(vimarea.WithVim(true))
	m.SetWidth(200)
	m.SetHeight(40)
	m.SetValue(text)
	m = press(m, "<esc>")
	m = press(m, "gg")
	return m
}

func press(m vimarea.Model, seq string) vimarea.Model {
	for _, k := range keys(seq) {
		m, _ = m.Update(k)
	}
	return m
}

func keys(seq string) []tea.KeyMsg {
	var out []tea.KeyMsg
	runes := []rune(seq)
	for i := 0; i < len(runes); i++ {
		if runes[i] == '<' {
			if end := strings.IndexRune(string(runes[i:]), '>'); end > 0 {
				token := string(runes[i+1 : i+end])
				out = append(out, special(token))
				i += end
				continue
			}
		}
		out = append(out, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{runes[i]}})
	}
	return out
}

func special(token string) tea.KeyMsg {
	switch token {
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "cr":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "bs":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
	}
	panic("unknown token " + token)
}

func cursor(t *testing.T, label string, m vimarea.Model, wantRow, wantCol int) {
	t.Helper()
	if m.Row() != wantRow || m.Column() != wantCol {
		t.Errorf("%s: got row=%d col=%d, want row=%d col=%d", label, m.Row(), m.Column(), wantRow, wantCol)
	}
}

func value(t *testing.T, label string, m vimarea.Model, want string) {
	t.Helper()
	if m.Value() != want {
		t.Errorf("%s: got %q, want %q", label, m.Value(), want)
	}
}

func pasteMsg(text string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text), Paste: true}
}

// boxed is box for tests that only need a seeded normal-mode area.
func boxed(t *testing.T, text string) vimarea.Model {
	t.Helper()
	return box(t, text)
}

func pasted(area vimarea.Model, text string) vimarea.Model {
	area, _ = area.Update(pasteMsg(text))
	return area
}
