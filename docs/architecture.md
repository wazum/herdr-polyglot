# How it works

The keybinding runs an action that knows which pane you pressed it in. The
action opens the draft box as a floating popup over that pane and passes the
pane id along in the environment, so the prompt lands where it was written.
Sized popups are only reachable over *herdr*'s socket API, not its CLI, so the
plugin speaks that protocol itself for this one call.

On `alt+enter` the draft goes to a translation service and the result is handed
back to the same pane through the *herdr* CLI: `agent prompt` to send it, or
`pane send-text` to type it without sending.

## The pieces

| Package | Holds |
| --- | --- |
| `promptflow` | The use case: translate a draft, deliver a prompt. Owns nothing about terminals. |
| `translation` | The `Translator` and `Provider` ports, the registry of services, and the sentence-level cache used for previews. |
| `deepl` | One service behind those ports. |
| `herdr` | Talking to *herdr*: the CLI runner, the two delivery targets, the socket. |
| `overlay` | The *Bubble Tea* program: the draft box, the header and footer, the keys. |
| `vimarea` | A text area with modal editing, used by the overlay. |
| `config` | Settings from the environment and the plugin's `.env`. |
| `draft` | An unfinished prompt on disk, one file per pane, written privately and atomically. |

`cmd/herdr-polyglot` is the composition root: it reads the settings, picks the
service, wires the flow to its targets and starts the program. Nothing below it
knows which service is in use or how the popup was opened.

Previews and sends use different translators. A send is one shot and goes to the
service directly; a preview goes through the sentence cache, because writing
means translating the same draft again and again. Both are set up whether live
translation starts on or off, since `ctrl+l` can turn it on at any time.

The draft is stored when the program ends, whichever way it ended: *herdr*
closes a popup by hanging up on it, and `ctrl+c` never passes the close key
either, so the last word on the draft cannot be a keystroke.

## Another translation service

*DeepL* is one implementation of a small interface, not a dependency of the design:

```go
type Provider interface {
	Name() string
	New(Options) (Translator, error)
}

type Translator interface {
	Translate(ctx context.Context, draft string) (string, error)
}
```

A service that can be told what came before a sentence — and does not bill for
it — also implements `ContextualTranslator`, which is what makes live
translation cheap:

```go
type ContextualTranslator interface {
	TranslateWithContext(ctx context.Context, text, preceding string) (string, error)
}
```

Add a package that implements `Provider`, register it in `services()` in
`cmd/herdr-polyglot/main.go`, and it becomes selectable through
`HERDR_POLYGLOT_PROVIDER`. The first registered service is the default. A
service that reports what it has spent can implement `UsageReporter`, and the
header shows the count.

## Working on it

```bash
make qa     # formatting, linting, race tests, vulnerability scan
make build
herdr plugin link .
```

Tests are written from the outside in and named as sentences about behaviour.
The overlay is driven through `teatest`, which means its tests type keys and
read frames rather than reaching into the model.
