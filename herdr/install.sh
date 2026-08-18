#!/usr/bin/env bash
# Runs on `herdr plugin install`. Builds the overlay binary into the plugin root.
# TODO: prefer a prebuilt binary from the matching GitHub Release once tagged.
set -euo pipefail

cd "$(dirname "$0")/.."
mkdir -p bin

# Without the symbol table and DWARF the binary is a third smaller, which is a
# third less to read from disk the first time a keypress runs it.
go build -trimpath -ldflags "-s -w" -o bin/herdr-polyglot ./cmd/herdr-polyglot

# Run it once so the first real keypress does not wait for the operating system
# to page in and check a binary it has never seen.
./bin/herdr-polyglot warm >/dev/null 2>&1 || true
