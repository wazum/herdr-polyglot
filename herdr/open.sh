#!/usr/bin/env bash
# Opens the overlay above the pane the keybinding was pressed in, and tells the
# overlay to deliver its prompt back to exactly that pane.
set -euo pipefail

herdr="${HERDR_BIN_PATH:-herdr}"
target="${HERDR_PANE_ID:-}"

if [[ -z $target ]]; then
	echo "deepl-prompt: no invoking pane; open this from an agent pane" >&2
	exit 1
fi

exec "$herdr" plugin pane open \
	--plugin "$HERDR_PLUGIN_ID" \
	--entrypoint overlay \
	--placement overlay \
	--target-pane "$target" \
	--env "HERDR_DEEPL_TARGET=$target" \
	--env "HERDR_DEEPL_ARGS=${DEEPL_PROMPT_ARGS:-}" \
	--focus
