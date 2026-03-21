# Footer Prefix Help Menu

## Overview
- Replace the current `pending: <prefix>` footer hint with a single-line which-key style menu that shows valid second-key options for active prefix families.
- Keep the interaction lightweight by reusing the existing footer line instead of introducing a popup or multi-line panel.
- Make prefix guidance easier to scan during normal-mode navigation, especially for the new `y`, `z`, `g`, and `]` families.

## Context (from discovery)
- files/components involved: [`internal/tui/model.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/model.go), [`internal/tui/view.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/view.go), [`internal/tui/keymap.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/keymap.go), [`internal/tui/view_test.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/view_test.go), and [`internal/tui/model_test.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/model_test.go).
- related patterns found: pending-prefix state already exists in [`internal/tui/model.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/model.go), and the footer hint logic is centralized in [`internal/tui/view.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/view.go) through `renderFooter()` and `footerHint()`.
- related rendering constraints found: the document view currently renders JSON rows, then one footer line, then an optional prompt line; there is no transient bottom panel today, so the new guidance must fit within the existing footer width-trimming behavior.
- related test coverage found: [`internal/tui/view_test.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/view_test.go) already verifies default footer hints and pending-prefix behavior, and [`internal/tui/model_test.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/model_test.go) already covers prefix activation and reset behavior.
- dependencies identified: Bubble Tea update flow, current prefix-dispatch logic, Lip Gloss theme styles, and `go test ./...` for validation.

## Development Approach
- **testing approach**: Regular (code first, then tests in each task)
- keep the change narrowly scoped to the footer and prefix metadata; do not introduce a popup, overlay, or multi-line layout in this pass
- make the prefix help menu data-driven so rendering and key resolution cannot drift apart
- keep labels short and stable so the menu remains readable within a single trimmed footer line
- prefer a semantic group tag for each prefix family, for example `[y:copy]`, instead of echoing a raw `pending:` marker
- use a readability-first exception for the `]` family and render it as `[jump]` rather than trying to force the literal prefix into awkward bracket syntax
- **CRITICAL: every task MUST include new or updated tests** for changed code paths
- **CRITICAL: all tests must pass before moving to the next task**
- update this plan if implementation scope changes during execution

## Testing Strategy
- **view tests**: assert the exact single-line prefix menus shown for active prefix families and confirm the default footer hint disappears while a prefix is pending
- **model tests**: keep existing prefix dispatch tests passing to prove the help-menu refactor does not break keyboard behavior
- **width behavior tests**: rely on existing trimmed rendering path rather than adding special wrapping logic in this pass
- **manual verification**: spot-check the footer in a real terminal on both a reasonably wide and narrow window to confirm the menu remains legible enough for transient use
- **e2e tests**: none currently; unit/model/view coverage is sufficient for this UI refinement

## Progress Tracking
- mark completed items with `[x]` immediately when done
- add newly discovered tasks with `➕` prefix
- document blockers or trade-offs with `⚠️` prefix
- keep this file synchronized with the actual implementation state

## What Goes Where
- **Implementation Steps** (`[ ]` checkboxes): code, tests, and lightweight help-text updates inside this repository
- **Post-Completion** (no checkboxes): manual terminal checks and any later polish decisions

## Implementation Steps

### Task 1: Define single-line prefix menu metadata

**Files:**
- Modify: `internal/tui/model.go`

- [x] add a small prefix-menu definition structure for the supported families
- [x] capture the agreed labels and items for `g`, `y`, `z`, and `]`
- [x] keep the rendering metadata aligned with the currently valid second-key actions
- [x] write tests for any new metadata helper logic if it is non-trivial
- [x] run tests: `go test ./...`

### Task 2: Render which-key style menus in the footer

**Files:**
- Modify: `internal/tui/view.go`
- Modify: `internal/tui/view_test.go`

- [x] replace `pending: <prefix>` with a formatted single-line menu such as `[y:copy] p:path k:key v:value s:subtree j:json`
- [x] keep settings-mode and prompt-mode footer behavior unchanged
- [x] style the group tag distinctly using existing theme styles without adding new theme slots
- [x] write view tests for `y`, `z`, and default-footer behavior while a prefix is pending
- [x] run tests: `go test ./...`

### Task 3: Keep prefix behavior and help text consistent

**Files:**
- Modify: `internal/tui/model.go`
- Modify: `internal/tui/model_test.go`
- Modify: `internal/tui/keymap.go`

- [x] verify the prefix help metadata stays consistent with `resolvePendingPrefix()` behavior
- [x] keep prefix clearing, unknown-shortcut handling, and existing shortcut execution unchanged
- [x] update any help text that still describes the footer as a generic pending-prefix hint
- [x] write tests covering no-regression for prefix activation and execution after the footer-menu change
- [x] run tests: `go test ./...`

### Task 4: Verify acceptance criteria

**Files:**
- Modify: `docs/plans/20260321-footer-prefix-help-menu.md`

- [x] verify the footer shows the agreed single-line format for active prefix families
- [x] verify the `]` family uses the readability-first tag instead of awkward raw bracket syntax
- [x] run full test suite: `go test ./...`
- [ ] perform a manual terminal smoke test for wide and narrow footer rendering
- [x] record any scope adjustments or follow-up polish items in this plan

## Technical Details
- **agreed footer format**: render single-line menus like `[y:copy] p:path k:key v:value s:subtree j:json` instead of `pending: y`
- **supported families**: `g`, `y`, `z`, and `]` should each have a compact semantic tag and ordered second-key entries
- **menu placement**: the transient menu replaces the normal footer hint and does not add extra lines or a separate panel
- **status coexistence**: status or error messages still appear first in the footer; the prefix menu is appended as the transient hint portion
- **width handling**: rely on the existing `trimWidth()` path for now; do not add wrapping or alternate compact modes in this pass
- **scope guardrail**: do not refactor the full keybinding system or add new theme config surface just to support this footer improvement

## Post-Completion
*Items requiring manual intervention or external systems - no checkboxes, informational only*

**Manual verification**:
- press `y`, `z`, `g`, and `]` in a real terminal and confirm the correct menu appears immediately
- confirm the menu disappears after a valid second key, invalid second key, or `esc`
- check a narrow terminal width to confirm truncation is acceptable for transient use

**Potential follow-up**:
- if more prefix families are added later, consider moving dispatch and menu definitions into one shared table instead of keeping parallel structures
- if narrow-window readability becomes a real issue, add abbreviated labels rather than switching to a multi-line panel

## Scope Notes
- The footer menus now render from the same prefix metadata used to resolve `gg`, `y*`, `z*`, and `]p`, so the displayed continuations and active shortcuts stay aligned.
- Help text gained a short discovery hint in the `Other` section instead of broader README changes, because this refinement only affects transient footer guidance and not the underlying bindings.
- Automated verification is complete via `go test ./...`; a real interactive terminal check for wide and narrow window behavior is still pending.
