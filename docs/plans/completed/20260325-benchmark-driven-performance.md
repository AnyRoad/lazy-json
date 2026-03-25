# Benchmark-Driven Performance Improvements

## Overview
- Improve interaction latency for large JSON documents without changing user-visible behavior.
- Add a repeatable benchmark workflow so performance work is measured, compared, and reviewed instead of guessed.
- Focus this pass on `Session.Refresh`, search-hit rebuilding/highlighting, and `Model.View()` using the agreed external fixtures:
  - `~/PROJECTS/react-json-view-lite-benchmark/src/hugeArray.json`
  - `~/PROJECTS/react-json-view-lite-benchmark/src/hugeJson.json`

## Context (from discovery)
- Files/components involved:
  - `internal/session/rows.go`
  - `internal/session/search.go`
  - `internal/session/session.go`
  - `internal/tui/view.go`
  - `internal/tui/model.go`
  - `internal/tui/view_test.go`
  - `internal/session/session_test.go`
  - `Makefile`
  - `.gitignore`
- Related patterns found:
  - `Session.Refresh()` rebuilds all rows on each refresh and currently walks hidden descendants to preserve global row numbering.
  - `Session.UpdateSearchHits()` rebuilds `SearchHits` as a slice, and `HasSearchHit()` does a linear scan per rendered row.
  - `visibleDocumentLines()` eagerly renders all rows to determine viewport slicing.
  - `renderRowLines()` does `Document.Find()` for every non-batch row during rendering.
- Dependencies identified:
  - Existing Go test/benchmark tooling via `go test`
  - Optional `benchstat` for better before/after comparisons
  - External benchmark fixtures from the user’s benchmark repo, with env-var override support required for portability

## Development Approach
- **testing approach**: Regular (benchmark harness first, then measured optimizations, then validation)
- Complete each task fully before moving to the next.
- Make small, focused changes and validate behavior after each task.
- **CRITICAL: every task MUST include new/updated tests** for code changes in that task.
- **CRITICAL: all tests must pass before starting the next task**.
- **CRITICAL: benchmark work must preserve current UI behavior**:
  - same row numbering behavior
  - same search results and highlighting semantics
  - same wrapped-string rendering semantics
  - same long-array batching behavior
- **CRITICAL: update this plan file when scope changes during implementation**.

## Testing Strategy
- **unit tests**:
  - add targeted correctness tests for any rendering fast path introduced in `internal/tui`
  - add tests for set-backed search-hit lookup behavior in `internal/session`
  - keep existing TUI/session/document tests passing unchanged unless behavior is intentionally preserved via equivalent assertions
- **benchmarks**:
  - add Go benchmarks for `Session.Refresh`, search-hit rebuilding, and `Model.View()`
  - benchmark both `hugeArray.json` and `hugeJson.json`
  - benchmark representative view states, not only initial load
  - run benchmarks with `-benchmem`
- **comparison workflow**:
  - `make perf` runs the benchmark suite
  - `make perf-save` stores a baseline in a local `.perf/` directory
  - `make perf-compare` compares current results to the saved baseline using `benchstat` when available, otherwise falls back to printing both files with a clear notice

## Progress Tracking
- Mark completed items with `[x]` immediately when done.
- Add newly discovered tasks with `➕` prefix.
- Document issues/blockers with `⚠️` prefix.
- Keep benchmark notes tied to concrete commands and result files.

## What Goes Where
- **Implementation Steps** (`[x]` checkboxes): code changes, tests, benchmark helpers, Makefile targets, and local tooling support.
- **Post-Completion** (no checkboxes): manual benchmark review and optional deeper profiling if the first pass is insufficient.

## Implementation Steps

### Task 1: Add a repeatable benchmark harness for large fixtures

**Files:**
- Create: `internal/session/perf_test.go`
- Create: `internal/tui/perf_test.go`
- Create: `internal/perftest/fixtures.go`
- Modify: `.gitignore`

- [x] create fixture-loading helpers in `internal/perftest/fixtures.go` that default to the agreed benchmark paths and allow env-var overrides
- [x] make fixture helpers skip benchmarks cleanly when files are unavailable instead of failing normal `go test` runs
- [x] add `internal/session/perf_test.go` benchmarks for `Session.Refresh()` and `Session.UpdateSearchHits()` on `hugeArray.json` and `hugeJson.json`
- [x] add `internal/tui/perf_test.go` benchmarks for `Model.View()` in representative interaction states (collapsed, expanded long-array batch, active search)
- [x] update `.gitignore` to exclude `.perf/` benchmark baseline output
- [x] run targeted benchmark commands to verify the harness executes successfully before optimization work

### Task 2: Add Makefile performance targets and comparison workflow

**Files:**
- Modify: `Makefile`
- Create: `scripts/perf-compare.sh`

- [x] add `make perf` to run the benchmark suite with `-benchmem`
- [x] add `make perf-save` to write a baseline result file under `.perf/`
- [x] add `make perf-compare` to compare current benchmark output against the saved baseline
- [x] implement comparison logic in `scripts/perf-compare.sh` so `benchstat` is used when installed and a readable fallback is shown otherwise
- [x] document benchmark file naming and baseline expectations inside the script or target comments for maintainability
- [x] run the new Makefile targets end to end and verify they produce usable output on the benchmark fixtures

### Task 3: Remove avoidable rendering overhead from the common view path

**Files:**
- Modify: `internal/session/rows.go`
- Modify: `internal/session/session.go`
- Modify: `internal/tui/view.go`
- Modify: `internal/tui/view_test.go`

- [x] extend `session.Row` with the node data needed for common rendering so `renderRowLines()` no longer calls `Document.Find()` per visible row
- [x] add a fast path in `internal/tui/view.go` for the non-wrapping case so viewport slicing does not require eagerly rendering every row
- [x] compute view-level repeated values once per render pass, including line-number gutter width and defaulted settings used by row rendering
- [x] preserve the existing wrapped-string rendering path and long-array batch row rendering semantics
- [x] write/update tests in `internal/tui/view_test.go` covering equivalent output for non-wrapping views after the fast path change
- [x] run focused tests and benchmarks to verify view-path improvements without regressions

### Task 4: Make search highlighting and lookup scale better

**Files:**
- Modify: `internal/session/search.go`
- Modify: `internal/session/session.go`
- Modify: `internal/session/session_test.go`
- Modify: `internal/tui/view.go`

- [x] change search-hit storage so the existing ordered slice is preserved for navigation while a set/map is maintained for constant-time `HasSearchHit()` lookup
- [x] ensure search-hit state is rebuilt and cleared correctly across refresh, search submit, and navigation flows
- [x] update `internal/tui/view.go` to use the new constant-time lookup without changing highlight behavior
- [x] write/update tests in `internal/session/session_test.go` for hit rebuilding, empty-query handling, and lookup correctness
- [x] write/update any TUI assertions needed to confirm search highlighting still appears on matching rows
- [x] rerun focused benchmarks for search-heavy view scenarios and compare against the saved baseline

### Task 5: Validate results and document the measured gains

**Files:**
- Modify: `README.md` (only if benchmark workflow needs user-facing documentation)
- Modify: `docs/plans/20260325-benchmark-driven-performance.md`

- [x] run `go test ./...` after all changes
- [x] run `make perf-save` before the optimization set if not already captured, then run `make perf-compare` after implementation
- [x] record the meaningful benchmark deltas and any remaining hot spots in this plan file
- [x] update `README.md` only if the benchmark workflow should be discoverable for contributors
- [x] move this plan to `docs/plans/completed/` once implementation and verification are done

## Technical Details
- Benchmark fixture loading:
  - default to the user-provided benchmark repo paths under `$HOME/PROJECTS/react-json-view-lite-benchmark/src/`
  - allow overrides through environment variables such as `LAZY_JSON_BENCH_HUGE_ARRAY` and `LAZY_JSON_BENCH_HUGE_JSON`
  - skip, not fail, when fixtures are missing
- Benchmark scenario shape:
  - `Session.Refresh()` with root-only expansion and representative expanded states
  - `Session.UpdateSearchHits()` with realistic queries that hit nested keys/values
  - `Model.View()` with width/height set to realistic terminal sizes and stable view state setup
- Rendering fast path:
  - optimize only the common non-wrapping path in this pass
  - keep the wrapped-string path behaviorally identical, even if it remains slower
- Search-hit optimization:
  - preserve current navigation order via `SearchHits []NodeID`
  - add a companion lookup map for `HasSearchHit`
- Baseline storage:
  - store raw benchmark output under `.perf/`
  - do not commit machine-specific benchmark outputs
- Measured results from `.perf/perf.baseline.txt` to `.perf/perf.current.txt`:
  - `Session.Refresh/huge-array/root`: `374507 ns/op` -> `19954 ns/op`
  - `Session.Refresh/huge-json/root`: `1075242 ns/op` -> `96630 ns/op`
  - `Session.SearchHits/huge-array/address`: `1367614 ns/op` -> `735065 ns/op`
  - `Session.SearchHits/huge-json/trainStation`: `3447487 ns/op` -> `2182288 ns/op`
  - `Model.View/huge-array/expanded-first-batch`: `648574 ns/op` -> `194876 ns/op`
  - `Model.View/huge-json/default`: `2632653 ns/op` -> `258151 ns/op`
  - `Model.View/huge-json/expanded-first-container`: `3103974 ns/op` -> `265034 ns/op`
  - remaining hot spots are still full-tree search traversal and wrapped-string rendering, which were intentionally left behavior-preserving in this pass

## Post-Completion
**Manual verification**:
- run `make perf` and `make perf-compare` on both benchmark fixtures after a cold start and after a second run
- manually open the large fixtures in `lazy-json` and verify navigation, search, batching, and wrapping still behave identically
- inspect whether the remaining dominant time is in row building, search traversal, or wrapped rendering before considering deeper architectural work

**External system updates**:
- none expected for this pass
