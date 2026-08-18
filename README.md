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
│ enter send · alt+enter newline · esc cancel  │
╰──────────────────────────────────────────────╯
```

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

## How it works

The keybinding runs an action that knows which pane you pressed it in, and opens
the overlay above that pane with the pane id in its environment. When you press
enter, the draft goes to a translation service and the result is handed back to
that same pane through the herdr CLI — `agent prompt` to send it, or
`pane send-text` to type it without sending.

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
