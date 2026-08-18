package overlay_test

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/wazum/herdr-polyglot/internal/overlay"
	"github.com/wazum/herdr-polyglot/internal/promptflow"
)

func laidOut(t *testing.T, pane tea.WindowSizeMsg, draft, preview string) []string {
	t.Helper()
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI)
	t.Cleanup(func() { lipgloss.SetColorProfile(previous) })

	target := &recordingTarget{}
	flow := promptflow.New(stubTranslator{english: preview}, target, target,
		promptflow.WithPreviewTranslator(stubTranslator{english: preview}))
	var model tea.Model = overlay.New(context.Background(), flow, overlay.Options{
		Service: "deepl", Language: "EN-US", Live: true, Debounce: time.Millisecond,
	})
	model, _ = model.Update(pane)
	if draft != "" {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(draft)})
	}
	if preview != "" {
		model, _ = model.Update(overlay.PreviewShown(draft, preview))
	}
	return strings.Split(model.View(), "\n")
}

var longWord = strings.Repeat("Donaudampfschifffahrtsgesellschaft", 6)

// The pane is the size herdr gave it. A wider line or an extra row is drawn over
// the agent's work or lost off the edge.
func TestThePopupKeepsItsShapeWhateverIsInIt(t *testing.T) {
	pane := tea.WindowSizeMsg{Width: 87, Height: 15}
	empty := laidOut(t, pane, "", "")

	for _, what := range []struct {
		name    string
		draft   string
		preview string
	}{
		{"nothing", "", ""},
		{"one word", "Hallo", ""},
		{"a line that just fits", strings.TrimSpace(strings.Repeat("wort ", 16)), ""},
		{"a line one over", strings.TrimSpace(strings.Repeat("wort ", 17)), ""},
		{"many words", strings.TrimSpace(strings.Repeat("wort ", 200)), ""},
		{"a word longer than the box", longWord, ""},
		{"umlauts at the edge", strings.Repeat("mühsam ", 30), ""},
		{"wide runes", strings.Repeat("日本語のテキスト ", 20), ""},
		{"a long translation", "Bitte behebe den Test.", strings.Repeat("Please fix the failing test. ", 20)},
		{"newlines", strings.Repeat("Zeile\n", 30), ""},
	} {
		lines := laidOut(t, pane, what.draft, what.preview)

		if len(lines) != len(empty) {
			t.Errorf("%s: the popup is %d rows, an empty one is %d",
				what.name, len(lines), len(empty))
		}
		for index, line := range lines {
			if width := lipgloss.Width(line); width > pane.Width {
				t.Errorf("%s: row %d is %d columns in a pane of %d",
					what.name, index, width, pane.Width)
			}
		}
	}
}

// If the area wraps wider than the box shows, the box wraps it again and a word
// lands alone on a line nobody asked for.
func TestNothingIsWrappedTwice(t *testing.T) {
	pane := tea.WindowSizeMsg{Width: 87, Height: 15}
	words := strings.TrimSpace(strings.Repeat("wort ", 60))

	lines := laidOut(t, pane, words, "")
	filled := 0
	for _, line := range lines {
		if strings.Count(line, "wort") > 0 {
			filled++
		}
	}
	if filled == 0 {
		t.Fatal("the draft is not on screen at all")
	}

	// Every row of the draft but the last holds as many words as fit; a row that
	// stops early is a row the box broke.
	widest := 0
	for _, line := range lines {
		if strings.Contains(line, "wort") {
			widest = max(widest, lipgloss.Width(strings.TrimRight(line, " │")))
		}
	}
	for index, line := range lines {
		if !strings.Contains(line, "wort") {
			continue
		}
		written := lipgloss.Width(strings.TrimRight(line, " │"))
		// A row shorter than the widest by more than one word is a second wrap.
		if widest-written > len("wort ")+2 && index < len(lines)-1 {
			t.Errorf("row %d holds %d columns where %d fit: %q", index, written, widest, line)
		}
	}
}

func TestTheLayoutHoldsAtOtherPaneSizes(t *testing.T) {
	for _, pane := range []tea.WindowSizeMsg{
		{Width: 40, Height: 15},
		{Width: 60, Height: 10},
		{Width: 87, Height: 15},
		{Width: 200, Height: 30},
		{Width: 87, Height: 8},
	} {
		empty := laidOut(t, pane, "", "")
		full := laidOut(t, pane, strings.TrimSpace(strings.Repeat("wort ", 300)),
			strings.Repeat("Please fix the failing test. ", 20))

		if len(full) != len(empty) {
			t.Errorf("%dx%d: full popup is %d rows, empty one is %d",
				pane.Width, pane.Height, len(full), len(empty))
		}
		for index, line := range full {
			if width := lipgloss.Width(line); width > pane.Width {
				t.Errorf("%dx%d: row %d is %d columns", pane.Width, pane.Height, index, width)
			}
		}
	}
}

func TestALongDraftGetsAScrollbar(t *testing.T) {
	pane := tea.WindowSizeMsg{Width: 87, Height: 15}

	short := laidOut(t, pane, "Bitte behebe den Test.", "")
	if strings.Contains(strings.Join(short, "\n"), overlay.ScrollThumb) {
		t.Error("a draft that fits has a scrollbar")
	}

	long := laidOut(t, pane, strings.TrimSpace(strings.Repeat("wort ", 200)), "")
	drawn := strings.Join(long, "\n")
	if !strings.Contains(drawn, overlay.ScrollThumb) || !strings.Contains(drawn, overlay.ScrollTrack) {
		t.Errorf("a draft of 200 words has no scrollbar:\n%s", drawn)
	}
}

func TestTheBarIsAColumn(t *testing.T) {
	pane := tea.WindowSizeMsg{Width: 87, Height: 15}
	lines := laidOut(t, pane, "Kurz. "+strings.TrimSpace(strings.Repeat("wort ", 300)), "")

	columns := map[int]int{}
	for _, line := range lines {
		for _, mark := range []string{overlay.ScrollThumb, overlay.ScrollTrack} {
			if at := strings.Index(line, mark); at >= 0 {
				columns[lipgloss.Width(line[:at])]++
			}
		}
	}
	if len(columns) == 0 {
		t.Fatal("no bar was drawn")
	}
	if len(columns) > 1 {
		t.Errorf("the bar wanders between columns %v", columns)
	}
}

// Rounding is where scrollbars go wrong: no thumb, all thumb, or a thumb that
// never reaches the end.
func TestTheBarSurvivesItsEdgeCases(t *testing.T) {
	for _, what := range []struct {
		name  string
		pane  tea.WindowSizeMsg
		draft string
	}{
		{"one row over", tea.WindowSizeMsg{Width: 87, Height: 15}, strings.Repeat("Zeile\n", 12)},
		{"hundreds of rows", tea.WindowSizeMsg{Width: 87, Height: 15}, strings.Repeat("Zeile\n", 400)},
		{"a box of four rows", tea.WindowSizeMsg{Width: 87, Height: 8}, strings.Repeat("Zeile\n", 50)},
		{"a narrow box", tea.WindowSizeMsg{Width: 40, Height: 15}, strings.Repeat("Zeile\n", 50)},
		{
			"a word past the bottom",
			tea.WindowSizeMsg{Width: 87, Height: 15},
			strings.Repeat("Donaudampfschifffahrtsgesellschaft", 40),
		},
	} {
		lines := laidOut(t, what.pane, what.draft, "")
		bar := draftBar(t, lines)
		if bar == "" {
			t.Errorf("%s: no bar for a draft that does not fit", what.name)
			continue
		}

		thumbs := strings.Count(bar, overlay.ScrollThumb)
		switch {
		case thumbs == 0:
			t.Errorf("%s: the bar %q has no thumb", what.name, bar)
		case thumbs == len([]rune(bar)):
			t.Errorf("%s: the bar %q is all thumb, so nothing is out of view", what.name, bar)
		}
		if !strings.HasSuffix(bar, overlay.ScrollThumb) {
			t.Errorf("%s: the bar %q does not reach the cursor at the end", what.name, bar)
		}
	}
}

func draftBar(t *testing.T, lines []string) string {
	t.Helper()

	var bar strings.Builder
	inside := false
	for _, line := range lines {
		switch {
		case strings.Contains(line, "╭") && !inside:
			inside = true
		case strings.Contains(line, "╰") && inside:
			return bar.String()
		case inside && strings.Contains(line, overlay.ScrollThumb):
			bar.WriteString(overlay.ScrollThumb)
		case inside && strings.Contains(line, overlay.ScrollTrack):
			bar.WriteString(overlay.ScrollTrack)
		}
	}
	return bar.String()
}

func TestTheThumbFollowsTheWriting(t *testing.T) {
	pane := tea.WindowSizeMsg{Width: 87, Height: 15}
	bar := draftBar(t, laidOut(t, pane, strings.TrimSpace(strings.Repeat("wort ", 200)), ""))

	if bar == "" {
		t.Fatal("no scrollbar to look at")
	}
	if !strings.HasSuffix(bar, overlay.ScrollThumb) {
		t.Errorf("the bar reads %q, want the thumb at the end where the cursor is", bar)
	}
	if !strings.HasPrefix(bar, overlay.ScrollTrack) {
		t.Errorf("the bar reads %q, want track above a cursor at the end", bar)
	}
	if !strings.Contains(bar, overlay.ScrollThumb+overlay.ScrollThumb) {
		t.Errorf("the bar reads %q, want a thumb showing how much is in view", bar)
	}
}

// A translation too long for its box scrolls to its end, where the newest
// sentence is, and shows a bar for the rest.
