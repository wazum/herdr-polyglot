# polyglot

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
│ INSERT  ctrl+d send · esc normal · enter …   │
╰──────────────────────────────────────────────╯
```

Sending is deliberate: `ctrl+d` (or `alt+enter`) translates and delivers, while
`enter` stays what it should be inside a text box — a new line.

## Install

```bash
herdr plugin install wazum/herdr-polyglot
```

Building from source needs Go; the install script compiles the overlay binary
into the plugin root.

## Configure

Credentials live in the plugin's own config directory, which herdr can print:

```bash
echo "HERDR_POLYGLOT_API_KEY=your-deepl-key" \
  >> "$(herdr plugin config-dir wazum.polyglot)/.env"
```

| Setting | Meaning |
| --- | --- |
| `HERDR_POLYGLOT_API_KEY` | Credentials for the translation service |
| `HERDR_POLYGLOT_PROVIDER` | Which service to use: `deepl` (default) or `dry-run` |
| `HERDR_POLYGLOT_LANGUAGE` | Target language, `EN-US` by default |
| `HERDR_POLYGLOT_ENDPOINT` | Override the service endpoint |
| `HERDR_POLYGLOT_SUBMIT` | `0` types the prompt without sending it |
| `HERDR_POLYGLOT_VIM` | `1` turns on the vim bindings described below |
| `HERDR_POLYGLOT_LIVE` | `1` translates while you write, see below |

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
key = "prefix+shift+t"
type = "plugin_action"
command = "wazum.polyglot.compose"
description = "write a prompt, review before sending"
```

`prompt` sends the translated prompt straight to the agent. `compose` types it
into the agent's input and leaves the final keystroke to you.

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
│ INSERT  ctrl+d send · esc normal               │
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
| Leaving | `ctrl+d` sends, `q` closes from normal mode, `ctrl+c` always closes |

Without vim, the box is an ordinary text area and `esc` closes it.

Pasting works in either mode and in the middle of a draft: the text is inserted
where the cursor is and you keep writing after it. In normal mode a paste is
still text, never a sequence of commands, the way bracketed paste behaves in
nvim.

## How it works

The keybinding runs an action that knows which pane you pressed it in, and opens
the overlay above that pane with the pane id in its environment. On `ctrl+d` the
draft goes to a translation service and the result is handed back to that same
pane through the herdr CLI — `agent prompt` to send it, or `pane send-text` to
type it without sending.

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
