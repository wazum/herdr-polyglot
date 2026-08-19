# Changelog

All notable changes to this project are written down here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and
the versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `tab` flips between writing and reading. Reading gives the translation the whole
  popup, scrolled with `↑ ↓ PageUp PageDown` — or `j k g G` with vim bindings on —
  and shows how far down it you are. Any letter, `tab` or `esc` goes back to the
  draft, so it cannot be got stuck in.

### Added

- The panel being written in or read is drawn with the accent border, the other in
  the frame's grey, so it is clear which one has the keys.
- How far through a panel you are is written into its bottom border rather than
  the footer.

### Fixed

- The scrollbar says where the view actually is. It was worked out from the cursor
  on the assumption that the last row was on screen, so it claimed the top while
  rows were hidden above.
- A draft that comes back opens at its beginning, where it can be read, instead of
  scrolled to its end.
- The draft can be walked through when it holds more than it shows: the arrows
  move by row, and `gj`/`gk` do the same with vim bindings on, where `j`/`k` keep
  vim's meaning of a whole line.
- Pasting a long draft no longer leaves the view at the top with the cursor out of
  sight below.
- Nothing is drawn wider than the pane. A popup narrower than 34 columns used to
  be drawn at 34 and wrap every line, stacking frame on frame while scrolling.
- The translation beside the draft shows its beginning and ends in `…` when there
  is more, instead of a scrollbar that could not be used from there. `tab` reads
  the rest, and the footer says so while there is more to read.
- Text no longer wraps twice. The draft box was two columns wider than what it
  showed, so a line was wrapped once by the text area and again by the box, which
  dropped words onto lines of their own.
- The popup keeps its shape whatever is written in it. A long translation used to
  grow its box and push the footer out of the pane; both boxes now scroll, with a
  bar showing how much is out of view, and the keys drop off the footer one at a
  time when a pane is too narrow for them.

## [0.3.0] - 2026-08-18

### Added

- The plugin signs the empty draft box with a small braille mark, in the corner
  furthest from the writing. It goes as soon as there is anything in the box, and
  `HERDR_POLYGLOT_LOGO=0` leaves the box bare.

### Changed

- A delivered prompt keeps its line breaks instead of being flattened onto one
  line, so a protected code block arrives as a code block and a `//` comment no
  longer swallows the rest of it. Tabs arrive as four spaces; escape sequences are
  still stripped.
- Code is no longer translated. Backticked spans and fenced blocks are taken out
  of the draft before it is sent and put back exactly as they were, so identifiers
  keep their names, comments and string literals stay as written, and indentation
  survives. The code never reaches the service, so it costs nothing to translate
  and stays on your machine.

## [0.2.0] - 2026-08-18

### Added

- *Google Cloud Translation* as a second service, chosen with
  `HERDR_POLYGLOT_PROVIDER=google` and a key in `HERDR_POLYGLOT_GOOGLE_API_KEY`.
  The language setting is shared: `EN-US` reaches *DeepL* as it stands and
  *Google* as `en`, and a region is kept only where the region is the point.

## [0.1.0] - 2026-08-18

First release. Settings, keys and behaviour may still change while the plugin
meets other people's terminals — that is what the 0.x is for.

### Added

- An overlay over any agent pane, opened by a keybinding. Write the prompt in
  your own language; `alt+enter` translates it and hands it over.
- Two ways to deliver it: sent to the agent, or typed into its input for you to
  send. `ctrl+r` switches between them while you write, and the header says which
  it will be.
- *DeepL* as the translation service, with `dry-run` for checking the wiring
  without a key. The service sits behind an interface, so another one is a
  package away.
- Live translation, off by default: the English follows the draft as you write.
  Each sentence is paid for once, and a translation you have already read is
  delivered without being translated again. It stays off for a draft that came
  back from an earlier session and for pasted text until `ctrl+l` asks for it.
- Confirmation mode: see the English and press `alt+enter` again to deliver it.
- Unfinished drafts kept privately, one per pane, and restored the next time the
  popup opens — however it was closed, including *herdr* closing it.
- Vim bindings in the draft box, off by default: the motions, edits, counts and
  registers that make sense inside a text box.
- The month's character count in the header, when the service reports it, and a
  warning when a draft grows past prompt size.
- Settings from the plugin's `.env` or the environment. A value that is neither
  on nor off is refused rather than guessed at.
- Prebuilt binaries for macOS and Linux on arm64 and amd64, published with
  checksums and build provenance. Installing needs no Go toolchain.

[Unreleased]: https://github.com/wazum/herdr-polyglot/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/wazum/herdr-polyglot/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/wazum/herdr-polyglot/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/wazum/herdr-polyglot/releases/tag/v0.1.0
