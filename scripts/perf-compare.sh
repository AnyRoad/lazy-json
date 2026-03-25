#!/bin/sh
set -eu

baseline=${1:-.perf/perf.baseline.txt}
current=${2:-.perf/perf.current.txt}
repo_root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)

if [ ! -f "$baseline" ]; then
  echo "baseline not found: $baseline" >&2
  echo "run 'make perf-save' first" >&2
  exit 1
fi

mkdir -p "$(dirname "$current")"

GOCACHE=${GOCACHE:-$repo_root/.gocache} \
go test ./internal/session ./internal/tui -run '^$' -bench . -benchmem -count=1 | tee "$current"

if command -v benchstat >/dev/null 2>&1; then
  echo
  benchstat "$baseline" "$current"
  exit 0
fi

echo
echo "benchstat not installed; showing benchmark lines from both files."
echo "baseline: $baseline"
grep '^Benchmark' "$baseline" || cat "$baseline"
echo
echo "current: $current"
grep '^Benchmark' "$current" || cat "$current"
