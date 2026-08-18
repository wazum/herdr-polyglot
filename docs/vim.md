# Vim bindings

Off by default, since modal editing is a matter of taste. `HERDR_POLYGLOT_VIM=1`
makes the draft box modal and the footer shows which mode you are in.

This covers what makes sense inside a text box. There are no files, buffers or
windows here, so nothing that acts on them exists.

| | |
| --- | --- |
| Modes | `esc` to normal, `i` `a` `I` `A` `o` `O` back to insert |
| Motions | `h` `j` `k` `l`, `w` `b` `e`, `0` `^` `$`, `gg` `G`, arrow keys |
| Delete | `x`, `dd`, `D`, `dw`, `db`, `d$`, `d0` |
| Change | `cw`, `cc`, `C` |
| Yank and paste | `yy`, `p`, `P` |
| Undo | `u` |
| Counts | `3j`, `2dd`, `3x` and so on |
| Leaving | `alt+enter` sends, `esc` closes from normal mode, `ctrl+c` always closes |

Escape is the whole way out: once to leave insert mode, once to close. Nothing
is lost by closing, because the draft is kept for next time.

Without vim bindings the box is an ordinary text area and `esc` closes it.

Pasting works in either mode and in the middle of a draft: the text is inserted
where the cursor is and you keep writing after it. In normal mode a paste is
still text, never a sequence of commands, the way bracketed paste behaves in
nvim.

Counts are bounded at 1,000, so a mistyped `999999999l` cannot hang the popup.
Motions that act on characters work on runes, which means an emoji built from
several code points can be split by `x` — the same as in vim itself.
