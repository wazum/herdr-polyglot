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

func numberedEnglish(sentences int) string {
	var written strings.Builder
	for number := 1; number <= sentences; number++ {
		written.WriteString("Sentence number ")
		written.WriteString(strings.Repeat("x", 40))
		written.WriteString(" ")
		written.WriteString(marker(number))
		written.WriteString(". ")
	}
	return written.String()
}

func marker(number int) string {
	return "END" + strings.Repeat("0", 3-len(itoa(number))) + itoa(number)
}

func itoa(number int) string {
	if number < 10 {
		return string(rune('0' + number))
	}
	return string(rune('0'+number/10)) + string(rune('0'+number%10))
}

func reader(t *testing.T) tea.Model {
	t.Helper()
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI)
	t.Cleanup(func() { lipgloss.SetColorProfile(previous) })

	target := &recordingTarget{}
	english := numberedEnglish(20)
	flow := promptflow.New(stubTranslator{english: english}, target, target,
		promptflow.WithPreviewTranslator(stubTranslator{english: english}))
	var model tea.Model = overlay.New(context.Background(), flow, overlay.Options{
		Service: "deepl", Language: "EN-US", Live: true, Debounce: time.Millisecond,
	})
	model, _ = model.Update(tea.WindowSizeMsg{Width: 87, Height: 15})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Bitte behebe den Test.")})
	model, _ = model.Update(overlay.PreviewShown("Bitte behebe den Test.", english))
	return model
}

func TestTabTurnsTheTranslationIntoSomethingReadable(t *testing.T) {
	model := reader(t)

	writing := model.View()
	if !strings.Contains(writing, marker(1)) {
		t.Errorf("while writing, the translation does not start at its beginning:\n%s", writing)
	}

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	reading := model.View()

	if !strings.Contains(reading, marker(1)) {
		t.Errorf("reading does not start at the beginning:\n%s", reading)
	}
	if strings.Contains(reading, "Bitte behebe den Test.") {
		t.Error("the draft is still on screen while reading, so there is no room to read")
	}
	if rows := strings.Count(reading, "\n"); rows != strings.Count(writing, "\n") {
		t.Errorf("reading is %d rows and writing %d, want the popup unchanged", rows,
			strings.Count(writing, "\n"))
	}
}

func TestReadingScrollsWithTheArrowsAndBack(t *testing.T) {
	model := reader(t)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})

	first := model.View()
	for range 3 {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	scrolled := model.View()
	if scrolled == first {
		t.Error("the arrows do not scroll the translation")
	}

	for range 10 {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	}
	if back := model.View(); back != first {
		t.Error("scrolling back up does not return to the beginning")
	}

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	if paged := model.View(); paged == first {
		t.Error("page down does not move")
	}
}

func TestReadingReachesTheEndAndStops(t *testing.T) {
	model := reader(t)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})

	for range 200 {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	atEnd := model.View()
	if !strings.Contains(atEnd, marker(20)) {
		t.Errorf("scrolling down does not reach the end:\n%s", atEnd)
	}

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	if past := model.View(); past != atEnd {
		t.Error("the translation scrolls past its own end")
	}
}

// Reading is not a place to get stuck: writing a letter goes back to the draft and
// types it, and so does escape without typing anything.
func TestWritingAnythingGoesBackToTheDraft(t *testing.T) {
	model := reader(t)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("X")})

	back := model.View()
	if !strings.Contains(back, "Bitte behebe den Test.X") {
		t.Errorf("the letter did not reach the draft:\n%s", back)
	}

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if !strings.Contains(model.View(), "Bitte behebe den Test.X") {
		t.Error("escape did not come back from reading")
	}
}

func TestTheReadingFooterSaysHowToGetAroundAndOut(t *testing.T) {
	model := reader(t)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})

	footer := lastLine(model.View())
	for _, key := range []string{"tab", "esc"} {
		if !strings.Contains(footer, key) {
			t.Errorf("the reading footer does not mention %s: %q", key, footer)
		}
	}
	if lipgloss.Width(footer) > 87 {
		t.Errorf("the reading footer is %d columns: %q", lipgloss.Width(footer), footer)
	}
}
