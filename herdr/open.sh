#!/usr/bin/env bash
# Opens the overlay above the pane the keybinding was pressed in, and tells the
# overlay to deliver its prompt back to exactly that pane.
#
# Usage: open.sh [submit|compose]
#   submit  (default) hand the prompt to the agent and submit it
#   compose type the prompt into the input and leave sending to the user
set -euo pipefail

herdr="${HERDR_BIN_PATH:-herdr}"
target="${HERDR_PANE_ID:-}"
mode="${1:-submit}"

if [[ -z $target ]]; then
	echo "deepl-prompt: no invoking pane; open this from an agent pane" >&2
	exit 1
fi

submit=1
if [[ $mode == compose ]]; then
	submit=0
fi

exec "$herdr" plugin pane open \
	--plugin "$HERDR_PLUGIN_ID" \
	--entrypoint overlay \
	--placement overlay \
	--env "HERDR_DEEPL_TARGET=$target" \
	--env "HERDR_DEEPL_SUBMIT=$submit" \
	--focus
