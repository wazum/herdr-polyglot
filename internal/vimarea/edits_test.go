package vimarea_test

import "testing"

// ---------- insert entries ----------

func TestInsertEntries(t *testing.T) {
	m := press(box(t, "abc"), "i")
	if m.Mode().String() != "INSERT" {
		t.Errorf("i should enter INSERT, got %s", m.Mode())
	}
	m = press(m, "Z")
	value(t, "iZ", m, "Zabc")

	// vim: A appends at end of line
	m2 := press(box(t, "abc"), "A")
	m2 = press(m2, " hello there")
	value(t, "A hello there", m2, "abc hello there")

	// vim: I -> first non-blank
	m3 := press(box(t, "    abc"), "$")
	m3 = press(m3, "I")
	m3 = press(m3, "Z")
	value(t, "I on indented line (vim: '    Zabc')", m3, "    Zabc")
}

// vim: `a` at the last char of a line appends at end of that same line.
func TestAppendAtEndOfLine(t *testing.T) {
	m := press(box(t, "ab\ncd"), "$")
	m = press(m, "a")
	m = press(m, "Z")
	value(t, "$aZ on 'ab' (vim: 'abZ\\ncd')", m, "abZ\ncd")
}

// vim: esc from insert mode moves the cursor one left (never past end of line).
func TestEscFromInsertMovesLeft(t *testing.T) {
	m := press(box(t, "abc"), "i")
	m = press(m, "X")
	m = press(m, "<esc>")
	cursor(t, "iX<esc> (vim: col 0, on 'X')", m, 0, 0)
	value(t, "iX<esc>", m, "Xabc")
}

// The practical fallout of no clamp on esc: `a` then wraps to the next line.
func TestAppendAfterEscAtEndOfLine(t *testing.T) {
	m := press(box(t, "ab\ncd"), "A")
	m = press(m, "<esc>")
	m = press(m, "a")
	m = press(m, "Z")
	value(t, "A<esc>aZ (vim: 'abZ\\ncd')", m, "abZ\ncd")
}

func TestOpenLineBelowAndAbove(t *testing.T) {
	m := press(box(t, "ab\ncd"), "o")
	m = press(m, "X")
	value(t, "oX", m, "ab\nX\ncd")
	cursor(t, "oX cursor", m, 1, 1)

	m2 := press(box(t, "ab\ncd"), "O")
	m2 = press(m2, "X")
	value(t, "OX", m2, "X\nab\ncd")
}

// vim: 3o opens three lines; 2O opens two above.
func TestOpenLineWithCount(t *testing.T) {
	// Vim materialises the repeats when the insert session ends, so the count
	// only shows after esc.
	m := press(box(t, "ab\ncd"), "3o")
	m = press(m, "X<esc>")
	value(t, "3oX<esc> (vim: 'ab\\nX\\nX\\nX\\ncd')", m, "ab\nX\nX\nX\ncd")

	m2 := press(box(t, "ab\ncd"), "2O")
	m2 = press(m2, "X<esc>")
	value(t, "2OX<esc> (vim: 'X\\nX\\nab\\ncd')", m2, "X\nX\nab\ncd")
}

// ---------- x ----------

// vim: x on the last char of a line deletes it and does NOT join the next line.
func TestXAtEndOfLineDoesNotJoin(t *testing.T) {
	m := press(box(t, "ab\ncd"), "l")
	m = press(m, "x")
	value(t, "lx on 'ab\\ncd' (vim: 'a\\ncd')", m, "a\ncd")
	cursor(t, "lx cursor (vim: row0 col0)", m, 0, 0)
}

// vim: x on an empty line is a no-op.
func TestXOnEmptyLineIsNoop(t *testing.T) {
	m := press(box(t, "\ncd"), "x")
	value(t, "x on empty line (vim: '\\ncd')", m, "\ncd")
}

func TestXMidLine(t *testing.T) {
	m := press(box(t, "abcdef"), "x")
	value(t, "x", m, "bcdef")
	cursor(t, "x cursor", m, 0, 0)
}

// vim: 3x deletes 3 chars; a count larger than the rest of the line stops at
// the line end and never joins.
func TestXWithCount(t *testing.T) {
	m := press(box(t, "abcdef"), "3x")
	value(t, "3x", m, "def")

	m2 := press(box(t, "ab\ncd"), "5x")
	value(t, "5x on 'ab' (vim: '\\ncd')", m2, "\ncd")
}

// ---------- D / C / d$ / d0 ----------

func TestDeleteToEndOfLine(t *testing.T) {
	m := press(box(t, "hello world"), "ll")
	m = press(m, "D")
	value(t, "llD", m, "he")
	cursor(t, "llD cursor (vim: col 1, on last remaining char)", m, 0, 1)
}

func TestChangeToEndOfLine(t *testing.T) {
	m := press(box(t, "hello world"), "ll")
	m = press(m, "C")
	if m.Mode().String() != "INSERT" {
		t.Errorf("C should enter INSERT, got %s", m.Mode())
	}
	m = press(m, "X")
	value(t, "llCX", m, "heX")
}

func TestDDollarAndDZero(t *testing.T) {
	m := press(box(t, "hello world"), "lll")
	m = press(m, "d$")
	value(t, "llld$", m, "hel")
	cursor(t, "llld$ cursor (vim: col 2)", m, 0, 2)

	m2 := press(box(t, "hello world"), "lll")
	m2 = press(m2, "d0")
	value(t, "llld0", m2, "lo world")
	cursor(t, "llld0 cursor", m2, 0, 0)
}

// ---------- dw / cw / db ----------

// vim: dw eats the trailing whitespace, cw does not.
func TestDwEatsBlanksCwDoesNot(t *testing.T) {
	m := press(box(t, "hello world"), "dw")
	value(t, "dw (vim: 'world')", m, "world")

	m2 := press(box(t, "hello world"), "cw")
	m2 = press(m2, "Z")
	value(t, "cwZ (vim: 'Z world')", m2, "Z world")

	m3 := press(box(t, "hello  world"), "dw")
	value(t, "dw with two blanks (vim: 'world')", m3, "world")

	m4 := press(box(t, "hello  world"), "cw")
	m4 = press(m4, "Z")
	value(t, "cwZ with two blanks (vim: 'Z  world')", m4, "Z  world")
}

// vim: dw on "foo.bar baz" deletes only "foo" (punctuation is a boundary).
func TestDwStopsAtPunctuation(t *testing.T) {
	m := press(box(t, "foo.bar baz"), "dw")
	value(t, "dw on 'foo.bar baz' (vim: '.bar baz')", m, ".bar baz")
}

// vim: d3w on "one two three four" -> "four"
func TestDwWithCount(t *testing.T) {
	m := press(box(t, "one two three four"), "d3w")
	value(t, "d3w (vim: 'four')", m, "four")
}

// vim: c2w on "one two three" -> "Z three"
func TestCwWithCount(t *testing.T) {
	m := press(box(t, "one two three"), "c2w")
	m = press(m, "Z")
	value(t, "c2wZ (vim: 'Z three')", m, "Z three")
}

func TestDb(t *testing.T) {
	m := press(box(t, "hello world"), "$")
	m = press(m, "db")
	value(t, "$db (vim: 'hello d')", m, "hello d")
}

// ---------- dd / cc ----------

func TestDeleteLine(t *testing.T) {
	m := press(box(t, "one\ntwo\nthree"), "j")
	m = press(m, "dd")
	value(t, "jdd", m, "one\nthree")
	cursor(t, "jdd cursor", m, 1, 0)
}

// vim: dd on the last line moves the cursor up one line.
func TestDeleteLastLineMovesUp(t *testing.T) {
	m := press(box(t, "one\ntwo\nthree"), "jj")
	m = press(m, "dd")
	value(t, "jjdd", m, "one\ntwo")
	cursor(t, "jjdd cursor (vim: row 1)", m, 1, 0)
}

// vim: dd lands on the first non-blank of the line that moves up.
func TestDeleteLineLandsOnFirstNonBlank(t *testing.T) {
	m := press(box(t, "a\n    zzz\nb"), "dd")
	value(t, "dd", m, "    zzz\nb")
	cursor(t, "dd cursor (vim: col 4)", m, 0, 4)
}

func TestDeleteOnlyLine(t *testing.T) {
	m := press(box(t, "x"), "dd")
	value(t, "dd on sole line", m, "")
}

func TestChangeLine(t *testing.T) {
	m := press(box(t, "hello world"), "ll")
	m = press(m, "cc")
	value(t, "llcc", m, "")
	if m.Mode().String() != "INSERT" {
		t.Errorf("cc should enter INSERT, got %s", m.Mode())
	}
}

// vim: 2dd deletes two lines.
func TestDeleteLineWithCount(t *testing.T) {
	m := press(box(t, "a\nb\nc\nd"), "2dd")
	value(t, "2dd", m, "c\nd")
	cursor(t, "2dd cursor", m, 0, 0)
}

// ---------- yank / paste ----------

// vim: yy then p puts the copy BELOW, cursor on first non-blank of the new line.
func TestYankPasteBelow(t *testing.T) {
	m := press(box(t, "  one\n  two"), "yy")
	m = press(m, "p")
	value(t, "yyp", m, "  one\n  one\n  two")
	cursor(t, "yyp cursor (vim: row 1 col 2, first non-blank)", m, 1, 2)
}

func TestYankPasteAbove(t *testing.T) {
	m := press(box(t, "  one\n  two"), "yy")
	m = press(m, "P")
	value(t, "yyP", m, "  one\n  one\n  two")
	cursor(t, "yyP cursor (vim: row 0 col 2)", m, 0, 2)
}

func TestPasteIsLinewise(t *testing.T) {
	m := press(box(t, "a\nb\nc"), "dd")
	m = press(m, "j")
	m = press(m, "p")
	value(t, "dd j p (vim: 'b\\nc\\na')", m, "b\nc\na")
}

func TestYankWithCount(t *testing.T) {
	m := press(box(t, "a\nb\nc"), "2yy")
	m = press(m, "G")
	m = press(m, "p")
	value(t, "2yy G p (vim: 'a\\nb\\nc\\na\\nb')", m, "a\nb\nc\na\nb")
}

// vim: 2p pastes the register twice.
func TestPasteWithCount(t *testing.T) {
	m := press(box(t, "a\nb"), "yy")
	m = press(m, "2p")
	value(t, "yy2p (vim: 'a\\na\\na\\nb')", m, "a\na\na\nb")
}

// vim: yanking an empty line and pasting it inserts an empty line.
func TestYankEmptyLineThenPaste(t *testing.T) {
	m := press(box(t, "\nabc"), "yy")
	m = press(m, "p")
	value(t, "yy p of an empty line (vim: '\\n\\nabc')", m, "\n\nabc")
}

// vim: D fills the unnamed register charwise, so p after D restores the text.
func TestDFillsRegister(t *testing.T) {
	m := press(box(t, "ab\ncd"), "D")
	m = press(m, "p")
	value(t, "Dp (vim: 'ab\\ncd')", m, "ab\ncd")
}
