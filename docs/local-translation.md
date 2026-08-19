# Translation on your own machine

`HERDR_POLYGLOT_COMMAND` points the plugin at a program instead of a service. The
draft is written to that program's input, and what it writes back is the
translation. No key, no account, no network.

```bash
ENV_FILE="$(herdr plugin config-dir wazum.polyglot)/.env"
echo 'HERDR_POLYGLOT_COMMAND=/Applications/translateLocally.app/Contents/MacOS/translateLocally -m de-en-base' >> "$ENV_FILE"
```

That is the whole configuration: a command and no key means the command
translates, so `HERDR_POLYGLOT_PROVIDER` can stay unset.

## translateLocally

[translateLocally](https://translatelocally.com) runs the *Bergamot* models
*Firefox* uses for its own translation, and it is quick enough for live
translation — a prompt of a couple of sentences comes back in well under a second
on an M-series Mac. Install it, then download the model for your language:

```bash
translateLocally --available-models
translateLocally --download-model de-en-base
translateLocally --list-models
```

The model name carries the direction, so `-m de-en-base` says what
`HERDR_POLYGLOT_LANGUAGE` says for a service. The setting still names the target
language for the header, and the command can read it as
`$POLYGLOT_TARGET_LANGUAGE` if it would rather be told.

On macOS the binary lives inside the application bundle, at
`/Applications/translateLocally.app/Contents/MacOS/translateLocally`. On Linux it
is on the path.

## Anything else that reads and writes text

The command line goes to a shell, so a pipeline, flags and quoting all work:

```bash
HERDR_POLYGLOT_COMMAND=curl -s -X POST http://localhost:5000/translate -H 'Content-Type: application/json' --data-binary @- ...
HERDR_POLYGLOT_COMMAND=trans -brief -no-warn :en
HERDR_POLYGLOT_COMMAND=ollama run translategemma
```

Only two rules: read the draft from standard input, and write the translation to
standard output. Anything on standard error is what the popup shows when the
command fails, so a program that explains itself there explains itself in the
footer.

## What the popup says when it goes wrong

| It says | It means |
| --- | --- |
| `translateLocally could not be started — is it installed?` | The shell could not find or execute it; check the path |
| `translateLocally failed: <what it complained about>` | It ran and refused — a missing model, usually |
| `translateLocally did not answer in time` | A minute passed with no answer; it is killed, and the rest of its pipeline with it |
| `translateLocally returned nothing` | It succeeded and wrote nothing, which would deliver an empty prompt |

`ctrl+t` tries again once the reason is fixed, without retyping the draft.

## What it costs

Nothing, and nothing leaves the machine. The sentence cache and the code
protection still work the same way, so a long draft is not re-translated word by
word while you write — see [live translation](live-translation.md).

The trade is quality: these models are small, and a long sentence with subclauses
comes back flatter than *DeepL* would put it. For prompts to a coding agent that
is usually enough, and you read the English beside your draft anyway.
