# Global Tree Line Numbers

## Overview
- Add configurable line numbers to the tree view, rendered as a left gutter before the existing tree markers.
- Define line numbers as global tree-order row numbers, not source JSON file lines and not viewport-relative numbers.
- Keep the feature off by default and expose it in the settings dialog with the same preview/save behavior as the existing display settings.

## Context (from discovery)
- files/components involved: [`internal/config/settings.go`](/Users/andrei/PROJECTS/lazy-json/internal/config/settings.go), [`internal/config/settings_test.go`](/Users/andrei/PROJECTS/lazy-json/internal/config/settings_test.go), [`internal/session/rows.go`](/Users/andrei/PROJECTS/lazy-json/internal/session/rows.go), [`internal/session/session.go`](/Users/andrei/PROJECTS/lazy-json/internal/session/session.go), [`internal/session/session_test.go`](/Users/andrei/PROJECTS/lazy-json/internal/session/session_test.go), [`internal/tui/view.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/view.go), [`internal/tui/view_test.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/view_test.go), [`internal/tui/settings.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/settings.go), [`internal/tui/settings_test.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/settings_test.go), [`internal/tui/keymap.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/keymap.go), [`internal/tui/help_test.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/help_test.go), and [`README.md`](/Users/andrei/PROJECTS/lazy-json/README.md).
- related row-building patterns found: [`internal/session/rows.go`](/Users/andrei/PROJECTS/lazy-json/internal/session/rows.go) currently walks the tree once, assigns visible `Row.Index` after collection, and skips collapsed descendants entirely.
- related settings patterns found: [`internal/config/settings.go`](/Users/andrei/PROJECTS/lazy-json/internal/config/settings.go) already persists boolean display settings such as JSON-path visibility, while [`internal/tui/settings.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/settings.go) previews boolean settings immediately and saves them on `s`.
- related rendering constraints found: [`internal/tui/view.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/view.go) renders a row label, then a value, then an optional JSON-path suffix, with special handling for wrapped string rows and continuation-line alignment.
- related documentation/test patterns found: [`internal/tui/help_test.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/help_test.go) keeps help text and README wording synchronized, and [`internal/tui/view_test.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/view_test.go) already validates row-layout details such as wrapping and optional path rendering.
- dependencies identified: session refresh after structural edits, full-tree traversal order, wrapped-row rendering alignment, settings persistence, and `go test ./...` for validation.

## Development Approach
- **testing approach**: Regular (code first, then tests in each task)
- treat line numbers as a display aid for the current logical tree, not as source-file metadata
- count rows in full document preorder so numbering remains stable while scrolling and retains gaps when subtrees are collapsed
- recompute global numbers whenever the session refreshes after edits, expansion changes, or startup initialization
- keep the setting off by default to avoid changing the default visual density of the tree
- render the gutter only when enabled, and keep continuation lines for wrapped strings unnumbered but horizontally aligned with the numbered first line
- preserve existing footer, prompt, search, save, and JSON-path behavior; this feature should not change command semantics
- **CRITICAL: every task MUST include new or updated tests** for changed code paths
- **CRITICAL: all tests must pass before moving to the next task**
- update this plan if implementation scope changes during execution

## Testing Strategy
- **settings tests**: verify the new line-number setting defaults to `false`, previews correctly in the settings modal, persists to `settings.json`, and restores on reload
- **session row tests**: verify visible rows carry full-tree global numbers and preserve numbering gaps when descendants are collapsed
- **view tests**: verify the left gutter renders when enabled, wrapped continuation lines do not repeat the number, gutter width aligns correctly, and line numbers coexist with JSON path on/off states
- **help/README tests**: update assertions for the extra settings row and wording that clarifies these are tree row numbers rather than source file lines
- **manual verification**: open a nested document, toggle line numbers on, collapse and expand containers, and confirm visible numbers keep global gaps and update after edits
- **e2e tests**: none currently; unit/model/view coverage is sufficient for this UI feature

## Progress Tracking
- mark completed items with `[x]` immediately when done
- add newly discovered tasks with `➕` prefix
- document blockers or trade-offs with `⚠️` prefix
- keep this file synchronized with the actual implementation state

## What Goes Where
- **Implementation Steps** (`[ ]` checkboxes): code, tests, and documentation updates inside this repository
- **Post-Completion** (no checkboxes): manual terminal checks and any later UX polish

## Implementation Steps

### Task 1: Add persisted line-number settings state

**Files:**
- Modify: `internal/config/settings.go`
- Modify: `internal/config/settings_test.go`
- Modify: `internal/tui/settings.go`
- Modify: `internal/tui/settings_test.go`

- [x] add `ShowLineNumbers` to persisted settings with default `false` and backward-compatible load behavior
- [x] add a `Line numbers` settings row with immediate preview and normal save semantics
- [x] update saved-settings summary text in the settings dialog to include line-number state
- [x] write tests for default, preview, and persisted-save behavior of the new setting
- [x] write tests for backward-compatible loading when older settings files omit the new field
- [x] run tests: `go test ./...`

### Task 2: Compute global tree-order numbers for visible rows

**Files:**
- Modify: `internal/session/rows.go`
- Modify: `internal/session/session_test.go`

- [x] extend `session.Row` with a dedicated global line-number field that is independent from visible `Row.Index`
- [x] refactor row building so traversal always increments the global number for every logical node in document preorder, even when the subtree is collapsed
- [x] append only visible rows to `Session.Rows` while preserving the global numbers assigned from the full-tree walk
- [x] write tests that collapsed subtrees leave visible numbering gaps instead of renumbering later siblings
- [x] write tests that expanding a subtree reveals the expected previously hidden global numbers
- [x] run tests: `go test ./...`

### Task 3: Render the line-number gutter in the tree view

**Files:**
- Modify: `internal/tui/view.go`
- Modify: `internal/tui/view_test.go`

- [x] add a left gutter renderer that prepends the global line number before the existing tree marker when line numbers are enabled
- [x] size the gutter from the widest visible global number so alignment remains stable within the current view
- [x] keep wrapped string continuation lines unnumbered while preserving the same gutter width for alignment
- [x] ensure line numbers compose cleanly with JSON path visibility on and off
- [x] write view tests for gutter rendering, wrapped-string continuation behavior, and mixed display-setting combinations
- [x] run tests: `go test ./...`

### Task 4: Keep help text and README aligned with the new setting

**Files:**
- Modify: `internal/tui/keymap.go`
- Modify: `internal/tui/help_test.go`
- Modify: `README.md`

- [x] document the new `Line numbers` settings row in help text and README settings documentation
- [x] clarify that the displayed numbers are global tree row numbers, not source JSON line numbers
- [x] update help/README assertions to cover the new setting and wording
- [x] write or update tests that guard the new documentation snippets
- [x] run tests: `go test ./...`

### Task 5: Verify acceptance criteria

**Files:**
- Modify: `docs/plans/20260324-global-tree-line-numbers.md`

- [x] verify line numbers are disabled by default and can be previewed/saved from settings
- [x] verify collapsing a subtree preserves global numbering gaps for later visible rows
- [x] verify wrapped continuations do not repeat the same line number
- [x] run full test suite: `go test ./...`
- [x] record any scope adjustments, follow-up items, or trade-offs in this plan

### Task 6: [Final] Update documentation state

**Files:**
- Modify: `docs/plans/20260324-global-tree-line-numbers.md`

- [x] confirm README/help wording matches the implemented behavior
- [x] note any manual verification still pending
- [x] move this plan to `docs/plans/completed/`

## Technical Details
- **numbering model**: use a global preorder traversal index across the full document tree; root is line `1`
- **collapse behavior**: collapsed descendants still count toward numbering, so visible rows may show gaps such as `1, 2, 7, 12`
- **scroll behavior**: scrolling never renumbers rows because numbering is not derived from the viewport slice
- **edit behavior**: any tree mutation that triggers session refresh recomputes the global numbering from the updated document structure
- **rendering placement**: show the number in a dedicated left gutter before the tree marker and key/index label
- **wrapping behavior**: only the first physical line of a wrapped row shows the number; continuation lines get a blank gutter of identical width
- **scope guardrail**: do not add source-file line parsing, source-offset tracking, or CLI flags as part of this feature

## Post-Completion
*Items requiring manual intervention or external systems - no checkboxes, informational only*

**Manual verification**:
- toggle `Line numbers` on in a nested file and confirm the left gutter appears without disturbing footer/prompt layout
- collapse and expand nested arrays or objects and confirm visible rows keep global numbering gaps
- edit the document structure by adding and deleting nodes and confirm numbers recompute consistently after refresh
- verify wrapped long strings show the number only on the first physical line

**Potential follow-up**:
- if users later want source-file line numbers, treat that as a separate feature requiring parser metadata rather than extending this tree-order gutter
- if gutter density becomes a concern in narrow windows, consider a compact-number mode or optional truncation strategy as a separate UX refinement

## Scope Notes
- The chosen behavior intentionally prioritizes stable logical tree numbering over contiguous visible-row numbering, so gaps after collapse are expected and desirable.
- The setting is scoped to the current display only; clipboard output, saved JSON, and command behavior remain unchanged.
- This plan assumes the feature remains settings-driven only and does not introduce a dedicated runtime command or startup flag.
- Automated verification is complete via `go test ./...`; manual terminal checks for gutter readability in narrow windows and collapse-gap UX are still pending.
