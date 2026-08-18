#!/usr/bin/env bash
# Runs on `herdr plugin install`. Builds the overlay binary into the plugin root.
# TODO: prefer a prebuilt binary from the matching GitHub Release once tagged.
set -euo pipefail

cd "$(dirname "$0")/.."
mkdir -p bin
go build -o bin/herdr-polyglot ./cmd/herdr-polyglot
