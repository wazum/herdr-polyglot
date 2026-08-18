package vimarea_test

import "testing"

// ---------- undo ----------

// vim: one insert session = ONE undo step. "ihello<esc>u" restores "x".
func TestUndoInsertSession(t *testing.T) {
	m := press(box(t, "x"), "i")
	m = press(m, "hello")
	m = press(m, "<esc>")
	value(t, "ihello<esc>", m, "hellox")
	m = press(m, "u")
	value(t, "ihello<esc>u (vim: 'x')", m, "x")
}

// The realistic scenario: type a sentence, esc, u.
func TestUndoAfterTypingASentence(t *testing.T) {
	m := press(box(t, "draft: "), "A")
	m = press(m, "translate this into German")
	m = press(m, "<esc>")
	value(t, "typed sentence", m, "draft: translate this into German")
	m = press(m, "u")
	value(t, "u after a typed sentence (vim: 'draft: ')", m, "draft: ")
}

// Worse: the insert session is not recorded, so u reaches PAST it and undoes
// the edit before it.
func TestUndoReachesPastInsertSession(t *testing.T) {
	m := press(box(t, "one\ntwo"), "dd")
	value(t, "dd", m, "two")
	m = press(m, "A")
	m = press(m, "-tail")
	m = press(m, "<esc>")
	value(t, "dd then A-tail", m, "two-tail")
	m = press(m, "u")
	// vim: u undoes only the insert -> "two"
	value(t, "dd A-tail<esc>u (vim: 'two')", m, "two")
}

// vim: u after x restores the char.
func TestUndoAfterX(t *testing.T) {
	m := press(box(t, "abcdef"), "x")
	value(t, "x", m, "bcdef")
	m = press(m, "u")
	value(t, "xu (vim: 'abcdef')", m, "abcdef")
}

// vim: u after 3x restores all three.
func TestUndoAfterCountedX(t *testing.T) {
	m := press(box(t, "abcdef"), "3x")
	m = press(m, "u")
	value(t, "3xu (vim: 'abcdef')", m, "abcdef")
}

// vim: u after D restores the line.
func TestUndoAfterD(t *testing.T) {
	m := press(box(t, "hello world"), "ll")
	m = press(m, "D")
	m = press(m, "u")
	value(t, "llDu (vim: 'hello world')", m, "hello world")
}

func TestUndoAfterDDollar(t *testing.T) {
	m := press(box(t, "hello world"), "ll")
	m = press(m, "d$")
	m = press(m, "u")
	value(t, "lld$u (vim: 'hello world')", m, "hello world")
}

func TestUndoAfterDZero(t *testing.T) {
	m := press(box(t, "hello world"), "lll")
	m = press(m, "d0")
	m = press(m, "u")
	value(t, "lllld0u (vim: 'hello world')", m, "hello world")
}

func TestUndoAfterDb(t *testing.T) {
	m := press(box(t, "hello world"), "$")
	m = press(m, "db")
	m = press(m, "u")
	value(t, "$dbu (vim: 'hello world')", m, "hello world")
}

// vim: 2dd is ONE undo step.
func TestUndoAfterCountedDD(t *testing.T) {
	m := press(box(t, "a\nb\nc\nd"), "2dd")
	value(t, "2dd", m, "c\nd")
	m = press(m, "u")
	value(t, "2ddu (vim: 'a\\nb\\nc\\nd')", m, "a\nb\nc\nd")
}

// these should already work
func TestUndoAfterDDAndPaste(t *testing.T) {
	m := press(box(t, "a\nb\nc"), "dd")
	m = press(m, "u")
	value(t, "ddu", m, "a\nb\nc")

	m2 := press(box(t, "a\nb"), "yy")
	m2 = press(m2, "p")
	m2 = press(m2, "u")
	value(t, "yypu", m2, "a\nb")

	m3 := press(box(t, "hello world"), "dw")
	m3 = press(m3, "u")
	value(t, "dwu", m3, "hello world")

	m4 := press(box(t, "a\nb"), "o")
	m4 = press(m4, "<esc>")
	m4 = press(m4, "u")
	value(t, "o<esc>u", m4, "a\nb")
}

// ---------- counts ----------

func TestCountsThatWork(t *testing.T) {
	m := press(box(t, "a\nb\nc\nd"), "3j")
	cursor(t, "3j", m, 3, 0)

	m2 := press(box(t, "abcdef"), "3l")
	cursor(t, "3l", m2, 0, 3)

	m3 := press(box(t, "abcdef"), "$")
	m3 = press(m3, "3h")
	cursor(t, "$3h", m3, 0, 2)
}

// vim: 2$ moves down one line then to its end.
func TestCountOnDollar(t *testing.T) {
	m := press(box(t, "ab\ncd\nef"), "2$")
	cursor(t, "2$ (vim: row 1 col 1)", m, 1, 1)
}

// vim: 2^ and 2_ etc. aside, 3G / 2gg are the common ones (covered in motions).

// vim: 3i<text><esc> inserts the text three times.
func TestCountOnInsert(t *testing.T) {
	m := press(box(t, "x"), "3i")
	m = press(m, "ab")
	m = press(m, "<esc>")
	value(t, "3iab<esc> (vim: 'ababab x' -> 'abababx')", m, "abababx")
}

// vim: 2yy then 2p pastes 4 lines (covered in edits for 2p).

// ---------- mode edge cases ----------

func TestEscInNormalModeIsHarmless(t *testing.T) {
	m := press(box(t, "abc"), "<esc>")
	m = press(m, "<esc>")
	if m.Mode().String() != "NORMAL" {
		t.Errorf("esc in normal mode should stay NORMAL, got %s", m.Mode())
	}
	value(t, "esc esc", m, "abc")
}

// vim: `d` then esc CANCELS the pending operator.
func TestEscCancelsPendingOperator(t *testing.T) {
	m := press(box(t, "one\ntwo\nthree"), "d")
	m = press(m, "<esc>")
	m = press(m, "dd")
	value(t, "d<esc>dd (vim: 'two\\nthree' — one line deleted)", m, "two\nthree")
}

// vim: `c` then esc cancels too.
func TestEscCancelsPendingChange(t *testing.T) {
	m := press(box(t, "one two"), "c")
	m = press(m, "<esc>")
	if m.Mode().String() != "NORMAL" {
		t.Errorf("c<esc> should stay NORMAL, got %s", m.Mode())
	}
	m = press(m, "w")
	cursor(t, "c<esc>w should be a plain w motion", m, 0, 4)
}

// vim: `y` then esc cancels.
func TestEscCancelsPendingYank(t *testing.T) {
	m := press(box(t, "one\ntwo"), "y")
	m = press(m, "<esc>")
	m = press(m, "yy")
	m = press(m, "p")
	value(t, "y<esc>yyp (vim: 'one\\none\\ntwo')", m, "one\none\ntwo")
}

// vim: `g` then esc cancels.
func TestEscCancelsPendingG(t *testing.T) {
	m := press(box(t, "a\nb\nc"), "G")
	m = press(m, "g")
	m = press(m, "<esc>")
	m = press(m, "g")
	m = press(m, "g")
	cursor(t, "G g<esc> gg (vim: row 0)", m, 0, 0)
}

// vim: `d` then a non-motion key aborts the operator, and the key is swallowed.
func TestPendingOperatorThenInvalidKey(t *testing.T) {
	m := press(box(t, "one\ntwo"), "d")
	m = press(m, "X")
	value(t, "dX (vim: unchanged)", m, "one\ntwo")
	if m.Mode().String() != "NORMAL" {
		t.Errorf("dX should stay NORMAL, got %s", m.Mode())
	}
	m = press(m, "dd")
	value(t, "dX then dd (vim: 'two')", m, "two")
}

// A count typed after esc-cancelling must not survive either.
func TestCountDoesNotLeakAcrossCancel(t *testing.T) {
	m := press(box(t, "a\nb\nc\nd\ne"), "3")
	m = press(m, "<esc>")
	m = press(m, "j")
	cursor(t, "3<esc>j (vim: row 1)", m, 1, 0)
}

// A dangling count with no command must not leak into the next command.
func TestCountLeaksAcrossInvalidOperator(t *testing.T) {
	m := press(box(t, "a\nb\nc\nd\ne"), "3z")
	m = press(m, "j")
	cursor(t, "3z then j (vim: row 1)", m, 1, 0)
}
