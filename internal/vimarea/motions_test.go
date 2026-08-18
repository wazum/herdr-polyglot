package vimarea_test

import "testing"

// ---------- $ and 0 / ^ ----------

// vim: `$` on "foo.bar baz" -> col 10 (0-based), ON the last char.
func TestDollarLandsOnLastChar(t *testing.T) {
	m := press(box(t, "foo.bar baz"), "$")
	cursor(t, "$", m, 0, 10)
}

func TestDollarOnEmptyLine(t *testing.T) {
	m := press(box(t, "\nabc"), "$")
	cursor(t, "$ on empty line", m, 0, 0)
}

// vim: `^` on "    indented text" -> col 4; `0` -> col 0.
func TestCaretGoesToFirstNonBlank(t *testing.T) {
	m := press(box(t, "    indented text"), "$")
	m = press(m, "^")
	cursor(t, "^ (vim: col 4)", m, 0, 4)

	m2 := press(box(t, "    indented text"), "$")
	m2 = press(m2, "0")
	cursor(t, "0", m2, 0, 0)
}

// ---------- h / l line boundaries ----------

// vim: `l` at end of line does NOT move to the next line.
func TestLStopsAtEndOfLine(t *testing.T) {
	m := press(box(t, "ab\ncd"), "lll")
	cursor(t, "lll on 'ab' (vim: row 0 col 1)", m, 0, 1)
}

// vim: `h` at col 0 does NOT move to the previous line.
func TestHStopsAtColumnZero(t *testing.T) {
	m := press(box(t, "ab\ncd"), "jh")
	cursor(t, "j then h (vim: row 1 col 0)", m, 1, 0)
}

// ---------- j / k column memory ----------

// vim: from col 4, j over a 1-char line and j again -> col 4 again.
func TestJKPreservesDesiredColumn(t *testing.T) {
	m := press(box(t, "abcdef\nx\nabcdef"), "llll")
	cursor(t, "llll", m, 0, 4)
	m = press(m, "j")
	cursor(t, "llll j (vim: row1 col0)", m, 1, 0)
	m = press(m, "j")
	cursor(t, "llll jj (vim: row2 col4)", m, 2, 4)
}

func TestJKColumnMemoryWithCount(t *testing.T) {
	m := press(box(t, "abcdef\nx\nabcdef"), "llll")
	m = press(m, "2j")
	cursor(t, "llll 2j (vim: row2 col4)", m, 2, 4)
}

// ---------- gg / G ----------

func TestGGAndG(t *testing.T) {
	m := press(box(t, "a\nb\nc\nd"), "G")
	cursor(t, "G", m, 3, 0)
	m = press(m, "gg")
	cursor(t, "gg", m, 0, 0)
}

// vim: 3G -> line 3, 2gg -> line 2.
func TestCountedGGAndG(t *testing.T) {
	m := press(box(t, "a\nb\nc\nd"), "3G")
	cursor(t, "3G (vim: row 2)", m, 2, 0)

	m2 := press(box(t, "a\nb\nc\nd"), "2gg")
	cursor(t, "2gg (vim: row 1)", m2, 1, 0)
}

// vim: G/gg land on the first non-blank of the target line.
func TestGLandsOnFirstNonBlank(t *testing.T) {
	m := press(box(t, "a\n    zzz"), "G")
	cursor(t, "G onto indented line (vim: col 4)", m, 1, 4)
}

// ---------- w / b / e and punctuation ----------

// vim: w on "foo.bar baz" -> 3 ('.'), then 4 ('b'), then 8 ('b' of baz).
func TestWordForwardStopsAtPunctuation(t *testing.T) {
	m := box(t, "foo.bar baz")
	m = press(m, "w")
	cursor(t, "w #1 (vim: col 3, the '.')", m, 0, 3)
	m = press(m, "w")
	cursor(t, "w #2 (vim: col 4)", m, 0, 4)
	m = press(m, "w")
	cursor(t, "w #3 (vim: col 8)", m, 0, 8)
}

// vim: w on "a, b c" -> col 1 (the comma).
func TestWordForwardStopsAtComma(t *testing.T) {
	m := press(box(t, "a, b c"), "w")
	cursor(t, "w on 'a, b c' (vim: col 1)", m, 0, 1)
}

// vim: from $ (col 10) on "foo.bar baz": b -> 8, b -> 4, b -> 3.
func TestWordBackwardStopsAtPunctuation(t *testing.T) {
	m := press(box(t, "foo.bar baz"), "$")
	m = press(m, "b")
	cursor(t, "$b (vim: col 8)", m, 0, 8)
	m = press(m, "b")
	cursor(t, "$bb (vim: col 4)", m, 0, 4)
	m = press(m, "b")
	cursor(t, "$bbb (vim: col 3)", m, 0, 3)
}

// vim: e on "foo.bar baz" -> 2 ('o'), then 3 ('.'), then 6 ('r').
func TestWordEndStopsAtPunctuation(t *testing.T) {
	m := box(t, "foo.bar baz")
	m = press(m, "e")
	cursor(t, "e #1 (vim: col 2)", m, 0, 2)
	m = press(m, "e")
	cursor(t, "e #2 (vim: col 3)", m, 0, 3)
	m = press(m, "e")
	cursor(t, "e #3 (vim: col 6)", m, 0, 6)
}

func TestWordEndOnComma(t *testing.T) {
	m := press(box(t, "a, b c"), "e")
	cursor(t, "e on 'a, b c' (vim: col 1)", m, 0, 1)
}

// vim: b from col 0 of line 2 lands on the START of the last word of line 1.
func TestWordBackwardAcrossLines(t *testing.T) {
	m := press(box(t, "one two\nthree"), "j")
	m = press(m, "b")
	cursor(t, "b from row1 col0 (vim: row0 col4, start of 'two')", m, 0, 4)
}

// plain whitespace words still work
func TestWordMotionsPlainWords(t *testing.T) {
	m := box(t, "alpha beta gamma")
	m = press(m, "w")
	cursor(t, "w", m, 0, 6)
	m = press(m, "w")
	cursor(t, "ww", m, 0, 11)
	m = press(m, "b")
	cursor(t, "wwb", m, 0, 6)
	m = press(m, "e")
	cursor(t, "wwbe", m, 0, 9)
	m = press(m, "2w")
	cursor(t, "2w from col9 -> last word already; clamp", m, 0, 15)
}
