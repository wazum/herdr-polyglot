# Live translation

With `HERDR_POLYGLOT_LIVE=1`, or `ctrl+l` in the popup, the English appears in a
second box and follows the draft about 600ms after you stop typing. Services
charge by the character, so the point of this page is what keeps that from
costing more than translating once.

## Paying for a sentence once

The draft is split into sentences, and each sentence is translated on its own
and remembered together with the sentence in front of it. While you write the
fourth sentence, the first three are already known and cost nothing; editing an
earlier sentence retranslates it and its immediate neighbourhood, not the whole
draft.

Every request carries the text before it as
[context](https://developers.deepl.com/docs/api-reference/translate), which
*DeepL* uses to translate the sentence in place but does not bill. A sentence is
therefore never translated in isolation, and never paid for twice.

The sentence being written is kept in a slot of its own rather than in the
store, so a session does not collect an entry per keystroke. When a newline or a
full stop finishes it, it moves into the store before the next sentence takes
the slot.

Two previews that overlap share one request: the second waits for the first
instead of sending its own. A preview that is abandoned — because you kept
typing — is cancelled, and a caller still waiting simply asks again rather than
inheriting the cancellation.

## Sending costs nothing extra

If the English on screen belongs to the draft as it stands, that text is what is
delivered. A translation you have read is never paid for a second time. While a
preview is out of date it is dimmed, and a newer translation always wins over a
slower older one, so what is delivered is always the English of the draft you
are looking at.

## Nothing is translated by accident

Live translation is off by default, and it switches itself off in the two cases
where a translation would not be something you asked for:

- **A draft that came back** from an earlier session. It might be yesterday's
  thinking, or it might be a cat on the keyboard. The footer says so, and `ctrl+l`
  translates it when you decide it is worth it.
- **Text arriving all at once** — 200 characters or more in a single keystroke is
  a paste, not writing. Pasted logs and code are the expensive kind of text and
  the least worth translating.

Beyond `HERDR_POLYGLOT_MAX_DRAFT` characters (2,000 by default) a draft stops
being translated as you write and the footer says why: this is a box for prompts
you write, not files you paste. Sending still works, and `ctrl+u` throws the
draft away.

A popup is sized for both boxes whether live translation starts on or off, since
a popup cannot be resized once it is open. Turning it on therefore always has
somewhere to show the result — a translation you cannot read would be the one
that really is wasted.

## Watching it work

The circle beside `live` fills and empties while a request is on its way, and
keeps breathing to the end of a cycle so a fast translation is still visible.
`HERDR_POLYGLOT_PULSE=0` leaves it still. The header also shows what the key has
spent this month, when the service reports it — asking costs nothing.
