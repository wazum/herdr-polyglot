# Changelog

All notable changes to this project are written down here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and
the versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- The plugin signs the empty draft box with a small braille mark, in the corner
  furthest from the writing. It goes as soon as there is anything in the box, and
  `HERDR_POLYGLOT_LOGO=0` leaves the box bare.

### Changed

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

[Unreleased]: https://github.com/wazum/herdr-polyglot/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/wazum/herdr-polyglot/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/wazum/herdr-polyglot/releases/tag/v0.1.0
