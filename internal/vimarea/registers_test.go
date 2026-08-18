package vimarea_test

import "testing"

// ---------- pending operator left dangling after esc ----------

// The value check in TestEscCancelsPendingOperator passes by accident: the esc
// is ignored, the first following `d` completes the *old* pending `d` as a dd,
// and the second `d` is left pending. This test shows the dangling state.
func TestEscLeavesOperatorPendingDangling(t *testing.T) {
	m := press(box(t, "one\ntwo\nthree\nfour"), "d")
	m = press(m, "<esc>")
	// vim: the esc cancelled `d`, so this is a plain `j` motion.
	m = press(m, "j")
	value(t, "d<esc>j (vim: unchanged text)", m, "one\ntwo\nthree\nfour")
	cursor(t, "d<esc>j (vim: row 1)", m, 1, 0)
}

func TestEscLeavesGPendingDangling(t *testing.T) {
	m := press(box(t, "a\nb\nc"), "G")
	m = press(m, "g")
	m = press(m, "<esc>")
	// vim: esc cancelled `g`, so `j` is a plain motion (already on last line).
	m = press(m, "g")
	cursor(t, "G g<esc> g (vim: still row 2, a lone g does nothing)", m, 2, 0)
}

// ---------- sticky $ column ----------

// vim: after `$`, j/k keep going to the end of each line.
func TestDollarColumnIsSticky(t *testing.T) {
	m := press(box(t, "abcdef\nxy\nabcdef"), "$")
	m = press(m, "j")
	cursor(t, "$j (vim: row1 col1, end of 'xy')", m, 1, 1)
	m = press(m, "j")
	cursor(t, "$jj (vim: row2 col5, end of 'abcdef')", m, 2, 5)
}

// ---------- w across lines ----------

func TestWordForwardAcrossLines(t *testing.T) {
	m := press(box(t, "one\ntwo"), "w")
	cursor(t, "w across lines", m, 1, 0)
}

// vim: w stops on an empty line.
func TestWordForwardStopsOnEmptyLine(t *testing.T) {
	m := press(box(t, "one two\n\nthree"), "ww")
	cursor(t, "ww (vim: row1 col0, the empty line)", m, 1, 0)
}

// ---------- multi-byte ----------

func TestMultiByteColumns(t *testing.T) {
	m := press(box(t, "schön wörld"), "w")
	cursor(t, "w over umlaut (vim: rune 6)", m, 0, 6)

	m2 := press(box(t, "schön wörld"), "$")
	cursor(t, "$ over umlaut (vim: rune 10)", m2, 0, 10)

	m3 := press(box(t, "schön wörld"), "x")
	value(t, "x on multi-byte line", m3, "chön wörld")
}

// ---------- the past-end cursor after esc, other fallout ----------

// vim: A<esc>x deletes the last char of the line and never joins lines.
func TestXAfterAppendEsc(t *testing.T) {
	m := press(box(t, "ab\ncd"), "A")
	m = press(m, "<esc>")
	m = press(m, "x")
	value(t, "A<esc>x (vim: 'a\\ncd')", m, "a\ncd")
}

// vim: A<esc>$ is a no-op; the cursor is already on the last char.
func TestDollarAfterAppendEsc(t *testing.T) {
	m := press(box(t, "abc"), "A")
	m = press(m, "<esc>")
	cursor(t, "A<esc> (vim: col 2, on 'c')", m, 0, 2)
}

// ---------- register / paste corner cases ----------

func TestPasteWithoutYankIsNoop(t *testing.T) {
	m := press(box(t, "abc"), "p")
	value(t, "p with an empty register", m, "abc")
}

// vim: dw fills the register so p pastes the word back (charwise).
func TestDwFillsRegister(t *testing.T) {
	m := press(box(t, "hello world"), "dw")
	m = press(m, "p")
	value(t, "dw p (vim: 'whello orld')", m, "whello orld")
}

// vim: x fills the register too.
func TestXFillsRegister(t *testing.T) {
	m := press(box(t, "ab"), "x")
	m = press(m, "p")
	value(t, "xp (vim: 'ba')", m, "ba")
}

// ---------- bracketed paste ----------

func TestBracketedPasteIsText(t *testing.T) {
	m := box(t, "abc")
	m, _ = m.Update(pasteMsg("dd"))
	value(t, "bracketed paste of 'dd' in NORMAL mode", m, "ddabc")
	if m.Mode().String() != "NORMAL" {
		t.Errorf("paste should not change mode, got %s", m.Mode())
	}
}

// ---------- counts around operators ----------

// vim: 3dw == d3w
func TestCountBeforeOperator(t *testing.T) {
	m := press(box(t, "one two three four"), "3dw")
	value(t, "3dw (vim: 'four')", m, "four")
}

// vim: 2d3w deletes 2*3 = 6 words.
func TestCountBothSidesOfOperator(t *testing.T) {
	m := press(box(t, "a b c d e f g h"), "2d3w")
	value(t, "2d3w (vim: 'g h')", m, "g h")
}
