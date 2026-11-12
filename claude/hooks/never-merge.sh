#!/usr/bin/env bash
# The hook is the Go binary `harness` (sources in harness/); this file only starts it.
exec "${HARNESS:-$HOME/.local/bin/harness}" hook never-merge "$@"
