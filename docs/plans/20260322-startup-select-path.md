# Startup `--select` Path Resolution

## Overview
- Add a `--select <path>` startup flag so `lazy-json` can open a file-backed or stdin-backed JSON document with a specific node selected immediately.
- Support deep startup selection using the app's existing displayed path format, including object keys, array indices, and quoted keys.
- Resolve missing paths by falling back to the nearest existing ancestor, expanding ancestor containers so the resolved node is visible on first render.

## Context (from discovery)
- files/components involved: [`cmd/lazy-json/main.go`](/Users/andrei/PROJECTS/lazy-json/cmd/lazy-json/main.go), [`cmd/lazy-json/main_test.go`](/Users/andrei/PROJECTS/lazy-json/cmd/lazy-json/main_test.go), [`internal/source/input.go`](/Users/andrei/PROJECTS/lazy-json/internal/source/input.go), [`internal/document/node.go`](/Users/andrei/PROJECTS/lazy-json/internal/document/node.go), [`internal/session/session.go`](/Users/andrei/PROJECTS/lazy-json/internal/session/session.go), [`internal/session/rows.go`](/Users/andrei/PROJECTS/lazy-json/internal/session/rows.go), and [`README.md`](/Users/andrei/PROJECTS/lazy-json/README.md).
- related patterns found: startup currently accepts either one file path or piped stdin in [`internal/source/input.go`](/Users/andrei/PROJECTS/lazy-json/internal/source/input.go), and [`cmd/lazy-json/main.go`](/Users/andrei/PROJECTS/lazy-json/cmd/lazy-json/main.go) constructs the document and TUI model before starting Bubble Tea.
- related selection/rendering patterns found: [`internal/session/session.go`](/Users/andrei/PROJECTS/lazy-json/internal/session/session.go) currently starts with only the root expanded and defaults selection to the first visible row, while [`internal/session/rows.go`](/Users/andrei/PROJECTS/lazy-json/internal/session/rows.go) already defines the canonical displayed path format such as `$.items[0].name` and `$["two words"]`.
- related testing patterns found: [`cmd/lazy-json/main_test.go`](/Users/andrei/PROJECTS/lazy-json/cmd/lazy-json/main_test.go) already covers startup configuration behavior, and [`internal/session/session_test.go`](/Users/andrei/PROJECTS/lazy-json/internal/session/session_test.go) validates expansion and selection behavior independently from the TUI.
- dependencies identified: strict CLI argument parsing, document tree traversal, session expansion state, Bubble Tea TTY handoff for stdin-backed sessions, and `go test ./...` for validation.

## Development Approach
- **testing approach**: Regular (code first, then tests in each task)
- keep `--select` parsing outside `source.Load(...)` so file-vs-stdin loading remains unchanged and well-scoped
- add a document-level path parser/resolver rather than relying on currently visible rows, because deep startup selection must work even when ancestors start collapsed
- support only the app's existing displayed path syntax, not generic JSONPath features such as wildcards, filters, or recursive descent
- expand ancestors needed to reveal the resolved node, but do not auto-expand the selected container's children unless they are part of the ancestor chain
- use strict syntax validation, but forgiving node resolution:
  - malformed path aborts startup
  - valid path with missing tail falls back to the nearest existing ancestor
  - root-only fallback opens successfully but surfaces an error in the footer
- **CRITICAL: every task MUST include new or updated tests** for changed code paths
- **CRITICAL: all tests must pass before moving to the next task**
- update this plan if implementation scope changes during execution

## Testing Strategy
- **CLI parsing tests**: verify `--select` is parsed correctly, rejects duplicates, rejects missing values, and preserves existing file/stdin input handling
- **document path resolver tests**: cover `$`, simple keys, array indices, quoted keys, malformed syntax, exact matches, nearest-ancestor fallback, and root-only fallback
- **session/startup application tests**: verify ancestor expansion, initial `SelectedID`, and startup status/error messaging based on the resolved path
- **README/help validation**: document `--select` usage for both file and stdin invocation forms
- **manual verification**: open nested JSON from both a file and piped stdin with exact, partial, and root-only fallback paths to confirm first-render selection feels correct
- **e2e tests**: none currently; unit and startup tests should be sufficient for this CLI/startup feature

## Progress Tracking
- mark completed items with `[x]` immediately when done
- add newly discovered tasks with `➕` prefix
- document blockers or trade-offs with `⚠️` prefix
- keep this file synchronized with the actual implementation state

## What Goes Where
- **Implementation Steps** (`[ ]` checkboxes): code, tests, and documentation updates inside this repository
- **Post-Completion** (no checkboxes): manual terminal checks and any later CLI polish

## Implementation Steps

### Task 1: Parse `--select` separately from input source arguments

**Files:**
- Modify: `cmd/lazy-json/main.go`
- Modify: `cmd/lazy-json/main_test.go`

- [x] add startup argument parsing that extracts `--select <path>` and returns the remaining positional args for `source.Load(...)`
- [x] reject `--select` when provided without a value or more than once
- [x] preserve the existing file-vs-stdin behavior and Bubble Tea `WithInputTTY()` handling for stdin-backed sessions
- [x] write tests for valid `--select` parsing with file and stdin forms
- [x] write tests for duplicate and missing-value flag errors
- [x] run tests: `go test ./...`

### Task 2: Add a document-level resolver for displayed row paths

**Files:**
- Create: `internal/document/path.go`
- Create: `internal/document/path_test.go`
- Modify: `internal/document/node.go`

- [x] implement parsing for the app's displayed path syntax: `$`, `.key`, `[index]`, and `["quoted key"]`
- [x] walk the document tree and return the deepest existing node match plus the ancestor IDs required to reveal it
- [x] distinguish malformed syntax from valid-but-missing path segments
- [x] write tests for exact matches across object keys, arrays, and quoted-key paths
- [x] write tests for malformed syntax and fallback to the nearest existing ancestor
- [x] run tests: `go test ./...`

### Task 3: Apply resolved startup selection to the initial session

**Files:**
- Modify: `cmd/lazy-json/main.go`
- Modify: `internal/session/session.go`
- Modify: `internal/session/session_test.go`
- Modify: `cmd/lazy-json/main_test.go`

- [x] add a startup-selection helper that expands ancestor containers, refreshes session rows, and sets `SelectedID` before Bubble Tea starts
- [x] keep exact matches silent, show a status message for non-root ancestor fallback, and show an error message for root-only fallback
- [x] avoid expanding the selected node's children unless they are needed as ancestors of the resolved path
- [x] write tests that deep startup selection lands on the expected node with the expected expansion state
- [x] write tests for ancestor-fallback and root-only fallback messaging behavior
- [x] run tests: `go test ./...`

### Task 4: Document CLI usage and acceptance criteria

**Files:**
- Modify: `README.md`
- Modify: `cmd/lazy-json/main_test.go`
- Modify: `docs/plans/20260322-startup-select-path.md`

- [x] document `--select` usage for file-backed and stdin-backed workflows in `README.md`
- [x] add or update tests that guard startup-usage strings and example behavior where practical
- [x] verify exact selection, nearest-ancestor fallback, and root-only fallback behavior against the agreed design
- [x] run full test suite: `go test ./...`
- [x] record any scope adjustments, follow-up items, or trade-offs in this plan

## Technical Details
- **flag shape**: use `--select <path>` with the existing displayed path format, for example `lazy-json --select '$.items[0].name' file.json` and `cat file.json | lazy-json --select '$.items[0].name'`
- **accepted path syntax**: support only the app's own row-path dialect:
  - `$`
  - `$.name`
  - `$.items[0]`
  - `$["two words"]`
  - nested combinations of the above
- **unsupported syntax**: do not accept generic JSONPath features such as `*`, `..`, filters, slices, or single-quoted keys
- **resolution behavior**:
  - malformed path: abort startup with a parse error
  - exact match: select the target and show no startup message
  - nearest existing ancestor below root: open successfully there and show a status message
  - root-only fallback: open successfully at `$` and show an error message in the footer
- **visibility behavior**: expand ancestor containers needed to reveal the resolved node before first render; do not fully expand unrelated branches
- **scope guardrail**: do not add a generic query language or change row-path rendering semantics as part of this feature

## Post-Completion
*Items requiring manual intervention or external systems - no checkboxes, informational only*

**Manual verification**:
- run `lazy-json --select '$.items[0].name' file.json` on a nested file and confirm the node is selected immediately
- run `cat file.json | lazy-json --select '$.items[0].name'` and confirm stdin-backed startup still works with interactive TTY input
- verify a partial miss such as `$.items[999].name` falls back to the nearest existing ancestor and shows a status message
- verify a root-only miss such as `$.totally.missing.branch` opens at `$` and shows an error message in the footer
- verify malformed syntax such as `$.items[` fails startup before the TUI opens

**Potential follow-up**:
- if path-based startup proves useful elsewhere, consider reusing the resolver for future commands such as `:select-path`
- if users ask for broader JSONPath compatibility later, treat that as a separate feature instead of expanding this minimal displayed-path resolver

## Scope Notes
- The implementation supports both `--select <path>` and `--select=<path>` while keeping the agreed path syntax and fallback semantics unchanged.
- Startup selection is applied directly to the initial session state before Bubble Tea starts, so deep paths work for both file-backed and stdin-backed launches without depending on currently visible rows.
- Automated verification is complete via `go test ./...`; manual terminal checks for exact match, ancestor fallback, root-only fallback, and stdin-backed startup are still pending.
