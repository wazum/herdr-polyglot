# polyglot

[![CI](https://github.com/wazum/herdr-polyglot/actions/workflows/ci.yml/badge.svg)](https://github.com/wazum/herdr-polyglot/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/wazum/herdr-polyglot.svg)](https://pkg.go.dev/github.com/wazum/herdr-polyglot)
[![Go](https://img.shields.io/github/go-mod/go-version/wazum/herdr-polyglot)](go.mod)
[![herdr](https://img.shields.io/badge/herdr-%E2%89%A5%200.8.0-6C3EF5)](https://herdr.dev)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A [herdr](https://herdr.dev) plugin for people who think faster in their own
language than in English. Press a key on any agent pane, an overlay opens above
it, you write the prompt in your language, and the translated prompt lands in
the agent's input.

Works with every agent herdr manages — Claude Code, Codex, opencode and the
rest — because the prompt is delivered through herdr, not typed into a specific
tool.

```
╭──────────────────────────────────────────────╮
│ ✳ polyglot              deepl → EN-US · send │
│ ╭──────────────────────────────────────────╮ │
│ │ Bitte behebe den fehlschlagenden Test    │ │
│ ╰──────────────────────────────────────────╯ │
│ INSERT  alt+enter send · esc → normal        │
╰──────────────────────────────────────────────╯
```

Sending is deliberate: `alt+enter` translates and delivers, while `enter` stays
what it should be inside a text box — a new line.

## Install

```bash
herdr plugin install wazum/herdr-polyglot
```

Building from source needs Go; the install script compiles the overlay binary
into the plugin root.

## Configure

Credentials live in the plugin's own config directory, which herdr can print.
Create the file so that only you can read it — a plain redirect leaves it
readable by everyone on the machine:

```bash
ENV_FILE="$(herdr plugin config-dir wazum.polyglot)/.env"
touch "$ENV_FILE" && chmod 600 "$ENV_FILE"
echo "HERDR_POLYGLOT_API_KEY=your-deepl-key" >> "$ENV_FILE"
```

[DeepL's free tier](https://www.deepl.com/pro-api) covers 500,000 characters a
month. Free keys end in `:fx`, and the plugin sends those to DeepL's free host
by itself.

| Setting | Meaning |
| --- | --- |
| `HERDR_POLYGLOT_API_KEY` | Credentials for the translation service |
| `HERDR_POLYGLOT_PROVIDER` | Which service to use: `deepl` (default) or `dry-run` |
| `HERDR_POLYGLOT_LANGUAGE` | Target language, `EN-US` by default |
| `HERDR_POLYGLOT_ENDPOINT` | Override the service endpoint |
| `HERDR_POLYGLOT_SUBMIT` | `0` types the prompt without sending it |
| `HERDR_POLYGLOT_VIM` | `1` turns on the vim bindings described below |
| `HERDR_POLYGLOT_LIVE` | `1` translates while you write, see below |
| `HERDR_POLYGLOT_CONFIRM` | `1` shows the English and waits for a second `alt+enter` |
| `HERDR_POLYGLOT_KEEP_DRAFT` | `0` starts from an empty box instead of resuming |
| `HERDR_POLYGLOT_MAX_DRAFT` | Characters before the box says the draft is too long, `2000` by default |
| `HERDR_POLYGLOT_PULSE` | `0` stops the live circle from breathing |

Every setting can also be passed as an environment variable, which wins over
the `.env` file. With `HERDR_POLYGLOT_PROVIDER=dry-run` the overlay marks the
draft instead of translating it, so you can check the wiring without a key.

To keep keys for several services side by side, scope them by service name:
`HERDR_POLYGLOT_DEEPL_API_KEY`.

## Bind a key

In `~/.config/herdr/config.toml`:

```toml
[[keys.command]]
key = "prefix+t"
type = "plugin_action"
command = "wazum.polyglot.prompt"
description = "write a prompt in your own language"

[[keys.command]]
key = "prefix+d"
type = "plugin_action"
command = "wazum.polyglot.compose"
description = "write a prompt, review before sending"
```

`prompt` sends the translated prompt straight to the agent. `compose` types it
into the agent's input and leaves the final keystroke to you. Either way `ctrl+r`
switches to the other one while you write, so one keybinding is enough — the
header says which it will be.

`t` as in translate: herdr already uses `prefix+p` for the previous tab and
`prefix+shift+p` for renaming a pane. Free letters are `a`, `d`, `f`, `i`, `m`,
`t`, `u` and `y` — if you want `p`, move herdr's `previous_tab` first.

## In the popup

| | |
| --- | --- |
| `alt+enter` | translate and hand the prompt over — `ctrl+d` does the same |
| `ctrl+r` | switch between `send` and `paste` — the header says which one is next |
| `ctrl+l` | turn live translation on or off for this prompt |
| `enter` | a new line, because a prompt is often more than one |
| `esc` | close — with vim bindings on, first to normal mode, then close |
| `ctrl+u` | throw the draft away, as it clears a line in a shell |
| `ctrl+c` | close, always |

When something did not work — a blank draft, a service that says no — the footer
says so, `esc` takes the message away, and it goes by itself after a few seconds.

The header says what `ctrl+d` will do: `sends to agent` hands the prompt over and
the agent starts working, `fills the input` only types it there and leaves the
final keystroke to you.

With vim bindings on, `esc` goes to normal mode and a second `esc` closes. Nothing
is lost either way, because the draft is kept.

## An unfinished prompt is kept

Closing the popup does not lose the draft, however it closes — `esc`, `q`,
`ctrl+c`, or herdr taking the popup away. It is written to the plugin's own state
directory, one file per pane, readable only by you, and comes back the next time
you open the popup there — the header says `resumed` until you type, and `ctrl+u`
throws it away. A sent prompt is forgotten immediately.

Since a draft is unfinished thinking about your code, it sits on disk until sent
or discarded. `HERDR_POLYGLOT_KEEP_DRAFT=0` turns that off and always starts from
an empty box. Herdr decides where the files go and tells the plugin through
`HERDR_PLUGIN_STATE_DIR`, alongside the config directory that `herdr plugin
config-dir wazum.polyglot` prints.

## Reading the English before it goes

`HERDR_POLYGLOT_CONFIRM=1` puts a stop between translating and sending: `alt+enter`
shows the English, a second `alt+enter` delivers it, and `esc` goes back to
writing. It costs one translation, not two, and pairs well with live off.

## Live translation

Off by default: the draft is translated once, when you send it. With
`HERDR_POLYGLOT_LIVE=1` the English appears in a second pane and follows what
you write, roughly 600ms after you stop typing.

```
╭────────────────────────────────────────────────╮
│ ✳ polyglot        deepl → EN-US · live · send  │
│ ╭────────────────────────────────────────────╮ │
│ │ Bitte behebe den fehlschlagenden Test      │ │
│ ╰────────────────────────────────────────────╯ │
│ ╭────────────────────────────────────────────╮ │
│ │ Please fix the failing test                │ │
│ ╰────────────────────────────────────────────╯ │
│ INSERT  alt+enter send · esc → normal          │
╰────────────────────────────────────────────────╯
```

Translating on every pause would mean paying for the whole draft again and
again, so live mode does two things about it. The draft is split into sentences
and each one is translated once — while you write the fourth sentence, the first
three are already known and cost nothing. And every sentence is sent with the
rest of the draft as [context](https://developers.deepl.com/docs/api-reference/translate),
which informs the translation without being billed, so a sentence is not
translated in isolation.

Sending costs nothing extra either: if the English on screen belongs to the
draft as it stands, that text is delivered as it is. A translation you have read
is never paid for twice. While a preview is out of date it is dimmed, and a
newer one always wins over a slower older one.

The circle beside `live` fills and empties while a translation is on its way, so
you can see it working without watching the text.

A draft longer than 2,000 characters stops being translated as you write, and the
box says so: this is a place for prompts you write, not files you paste. Sending
still works, and `ctrl+u` throws the draft away.

## Vim bindings

Off by default, since modal editing is a matter of taste. With
`HERDR_POLYGLOT_VIM=1` the draft box becomes modal, and the footer shows which
mode you are in. It covers what makes sense inside a text box — there are no
files, buffers or windows here, so nothing that acts on them exists.

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

Without vim, the box is an ordinary text area and `esc` closes it.

Pasting works in either mode and in the middle of a draft: the text is inserted
where the cursor is and you keep writing after it. In normal mode a paste is
still text, never a sequence of commands, the way bracketed paste behaves in
nvim.

## What leaves your machine

The draft is sent to the translation service, so treat it the way you treat
anything you paste into a web translator. Prompts for a coding agent tend to
carry file paths, code and occasionally a secret, and in live mode the draft
goes out again after every pause in typing. Each sentence also travels with the
text before it as context.

Nothing else leaves: the API key goes to the translation service only, never to
the agent, the herdr socket, a command line, or a child process. Translated text
is stripped of control characters before it is typed into a pane, so neither a
line break nor an escape sequence can reach the agent's terminal.

Keep a draft off the network entirely with `HERDR_POLYGLOT_PROVIDER=dry-run`,
which marks the text instead of translating it.

## Colours

The overlay names palette slots rather than fixed colours, so it takes on
whatever herdr theme is active — nord looks like nord, gruvbox like gruvbox —
including a light terminal. Herdr does not expose the theme to plugins, but it
does paint the terminal palette, which is what the overlay draws with.

## How it works

The keybinding runs an action that knows which pane you pressed it in. It opens
the draft box as a floating popup sized to exactly what the box needs, so the
agent's output stays readable around it, and passes that pane id along in the
environment. On `alt+enter` the draft goes to a translation service and the result
is handed back to the same pane through the herdr CLI — `agent prompt` to send
it, or `pane send-text` to type it without sending.

Sized popups are only reachable over herdr's socket API, not its CLI, so the
plugin binary speaks that protocol itself for this one call.

## Another translation service

DeepL is one implementation of a small interface, not a dependency of the design:

```go
type Provider interface {
	Name() string
	New(Options) (Translator, error)
}

type Translator interface {
	Translate(ctx context.Context, draft string) (string, error)
}
```

Add a package that implements it, register it in `services()` in
`cmd/herdr-polyglot/main.go`, and it becomes selectable through
`HERDR_POLYGLOT_PROVIDER`. The first registered service is the default.

## Development

```bash
make qa     # formatting, linting, race tests, vulnerability scan
make build
herdr plugin link .
```

## License

MIT
