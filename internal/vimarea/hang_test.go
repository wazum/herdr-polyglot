package vimarea_test

import (
	"strings"
	"testing"
	"time"

	"github.com/wazum/herdr-polyglot/internal/vimarea"
)

// finishes runs keys with a deadline: a motion that cannot reach its target must
// give up rather than spin, which would freeze the whole popup.
func finishes(t *testing.T, area vimarea.Model, sequence string) vimarea.Model {
	t.Helper()

	done := make(chan vimarea.Model, 1)
	go func() { done <- press(area, sequence) }()

	select {
	case result := <-done:
		return result
	case <-time.After(2 * time.Second):
		t.Fatalf("%q never finished", sequence)
		return area
	}
}

// A line exactly as wide as the box wraps to a phantom second row, which used to
// make moving down loop for ever.
func TestMovingBetweenLinesFinishesWhenALineFillsTheWidth(t *testing.T) {
	for _, width := range []int{32, 40, 96} {
		full := strings.Repeat("a", width)

		area := vimarea.New(vimarea.WithVim(true))
		area.SetWidth(width)
		area.SetHeight(6)
		area.SetValue(full + "\nzweite Zeile")
		area = press(area, "<esc>")

		for _, sequence := range []string{"j", "G", "gg", "k", "dd", "yyp"} {
			moved := finishes(t, area, sequence)
			if moved.Value() == "" && sequence != "dd" {
				t.Errorf("width %d: %q emptied the draft", width, sequence)
			}
		}
	}
}

// A count typed by accident must not stall the popup or exhaust memory.
func TestAnAbsurdCountFinishesQuickly(t *testing.T) {
	area := vimarea.New(vimarea.WithVim(true))
	area.SetWidth(40)
	area.SetHeight(6)
	area.SetValue("eine Zeile\nzweite Zeile")
	area = press(area, "<esc>")

	for _, sequence := range []string{
		"999999999l", "999999999h", "999999999x", "999999999j", "999999999w",
		"yy999999999999999999p", "999999999dd", "99999999999d99999999w",
	} {
		start := time.Now()
		finishes(t, area, sequence)
		if elapsed := time.Since(start); elapsed > time.Second {
			t.Errorf("%q took %s, want it bounded", sequence, elapsed)
		}
	}
}
