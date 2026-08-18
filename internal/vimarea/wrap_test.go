package vimarea_test

import (
	"testing"

	"github.com/wazum/herdr-polyglot/internal/vimarea"
)

// A narrow popup soft-wraps. vim's `j` moves by LOGICAL line (gj moves by
// screen row), so j on a wrapped line should jump to the next real line.
func TestJOnSoftWrappedLine(t *testing.T) {
	m := vimarea.New(vimarea.WithVim(true))
	m.SetWidth(10)
	m.SetHeight(10)
	m.SetValue("aaaaaaaaaaaaaaaaaaaaaaaa\nsecond")
	m = press(m, "<esc>")
	m = press(m, "gg")
	m = press(m, "j")
	cursor(t, "j on a soft-wrapped line (vim: row 1, the 'second' line)", m, 1, 0)
}

// `$` on a wrapped line: vim goes to the end of the logical line.
func TestDollarOnSoftWrappedLine(t *testing.T) {
	m := vimarea.New(vimarea.WithVim(true))
	m.SetWidth(10)
	m.SetHeight(10)
	m.SetValue("aaaaaaaaaaaaaaaaaaaaaaaa\nsecond")
	m = press(m, "<esc>")
	m = press(m, "gg")
	m = press(m, "$")
	cursor(t, "$ on a soft-wrapped line (vim: col 23)", m, 0, 23)
}

// `dd` on a wrapped line deletes the whole logical line.
func TestDDOnSoftWrappedLine(t *testing.T) {
	m := vimarea.New(vimarea.WithVim(true))
	m.SetWidth(10)
	m.SetHeight(10)
	m.SetValue("aaaaaaaaaaaaaaaaaaaaaaaa\nsecond")
	m = press(m, "<esc>")
	m = press(m, "gg")
	m = press(m, "dd")
	value(t, "dd on a soft-wrapped line", m, "second")
}
