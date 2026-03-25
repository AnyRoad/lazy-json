# Vim-Style Undo and Redo

## Overview
- Add Vim-style edit history to `lazy-json` with `u` for undo and both `U` and `ctrl+r` for redo.
- Limit history scope to document mutations only: scalar edits, key renames, object/array inserts, deletes, external-editor replacements, and `jq` transforms.
- Keep folds, selection movement, search mode, theme previews, and settings changes outside the undo system so the feature stays predictable and aligned with editor-style expectations.
- Implement inverse-operation history instead of full-document snapshots so large files do not pay a full-copy cost for every edit.

## Context (from discovery)
- files/components involved: [`internal/tui/model.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/model.go), [`internal/tui/commands.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/commands.go), [`internal/tui/keymap.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/keymap.go), [`internal/tui/model_test.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/model_test.go), [`internal/tui/help_test.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/help_test.go), [`internal/document/edit.go`](/Users/andrei/PROJECTS/lazy-json/internal/document/edit.go), [`internal/document/node.go`](/Users/andrei/PROJECTS/lazy-json/internal/document/node.go), [`internal/document/document_test.go`](/Users/andrei/PROJECTS/lazy-json/internal/document/document_test.go), and [`README.md`](/Users/andrei/PROJECTS/lazy-json/README.md).
- related patterns found: all document mutations already funnel through a small set of helpers in [`internal/document/edit.go`](/Users/andrei/PROJECTS/lazy-json/internal/document/edit.go): `Replace`, `Delete`, `AddObjectEntry`, `AddArrayItem`, and `RenameKey`.
- related UI integration patterns found: normal-mode key dispatch and asynchronous edit completions both converge in [`internal/tui/model.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/model.go), while command aliases and mutation helpers are centralized in [`internal/tui/commands.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/commands.go).
- related selection/rendering patterns found: selection is already expressed in terms of `session.RowID`, which covers both real nodes and synthetic batch rows, so history can restore logical selection anchors without adding new session concepts.
- related save-state patterns found: the current dirty flag is toggled directly by edit handlers and cleared on save; undo/redo will need to replace that ad hoc toggling with revision-based dirty tracking.
- dependencies identified: Bubble Tea update flow, current session refresh/reveal behavior, the document mutation API, and `go test ./...` for validation.

## Development Approach
- **testing approach**: Regular (code first, then tests in each task)
- keep history state in the TUI model, not in `Document` or `Session`
- use typed inverse operations rather than closures, patches, or JSON reparse snapshots
- add only the document helpers required to replay inverse ops cleanly: deep node clone with preserved IDs plus indexed insert helpers for objects and arrays
- store both `beforeSelection` and `afterSelection` as `session.RowID` on history entries so batch-row append and other long-array flows restore sensible selection
- treat save points as revision markers: saving updates the saved revision but does not clear undo or redo stacks
- clear the redo stack on any new edit after an undo, unconditionally
- **CRITICAL: every task MUST include new or updated tests** for the changed code paths
- **CRITICAL: all tests must pass before moving to the next task**
- update this plan if implementation details or scope change during execution

## Testing Strategy
- **document tests**: verify deep clones preserve IDs without aliasing and indexed inserts restore entries/items at the exact original position
- **history unit tests**: exercise inverse-operation helpers independently where practical so replay semantics do not rely entirely on end-to-end UI tests
- **model/update tests**: drive Bubble Tea updates directly to verify `u`, `U`, `ctrl+r`, `:undo`, and `:redo` behavior
- **edit-flow tests**: cover scalar replace, rename, object add, array add, delete, batch-row append, external editor completion, and `jq` completion
- **dirty-state tests**: verify undoing back to the saved revision clears the dirty marker and redoing or making a new edit sets it again
- **help/docs tests**: update help and README assertions for the new bindings and command aliases
- **manual verification**: spot-check undo/redo on a real terminal session, especially after long-array batch navigation and after saving
- **e2e tests**: none currently; rely on unit and model coverage plus manual terminal smoke testing

## Progress Tracking
- mark completed items with `[x]` immediately when done
- add newly discovered work with `➕` prefix
- record blockers, trade-offs, or deferred items with `⚠️` prefix
- keep the plan synchronized with the actual implementation state

## What Goes Where
- **Implementation Steps** (`[ ]` checkboxes): code, tests, and documentation changes inside this repository
- **Post-Completion** (no checkboxes): manual terminal checks and any follow-up product decisions

## Implementation Steps

### Task 1: Add document helpers needed for reversible edits

**Files:**
- Modify: `internal/document/edit.go`
- Modify: `internal/document/node.go`
- Modify: `internal/document/document_test.go`

- [x] add a deep-clone helper for `document.Node` that preserves IDs while avoiding aliasing between history entries and the live document
- [x] add indexed object-entry and array-item insertion helpers so undo can restore deleted content at the original position instead of appending
- [x] keep existing append-style add helpers compatible by routing them through the new indexed insertion logic where appropriate
- [x] write tests for clone behavior, indexed object insertion, and indexed array insertion
- [x] write tests for edge cases such as invalid parent kind or out-of-range insert positions
- [x] run tests: `go test ./internal/document`
⚠️ Task 1 also added a preserve-IDs `Restore` helper in the document layer. Redo chains that target descendants of a replaced container require subtree IDs to stay stable across undo and redo; indexed reinsertion alone was not enough.

### Task 2: Add inverse-operation history primitives to the TUI layer

**Files:**
- Create: `internal/tui/history.go`
- Create: `internal/tui/history_test.go`
- Modify: `internal/tui/model.go`

- [x] define history entry types for replace, rename, object insert, array insert, and delete operations
- [x] store `beforeSelection` and `afterSelection` as `session.RowID` on history entries so logical selection can be restored after replay
- [x] add undo stack, redo stack, current revision, and saved revision fields to `Model`
- [x] implement shared helpers to push a new history entry, apply undo, apply redo, refresh session state, and keep `Session.Dirty` synchronized with revision state
- [x] write tests for history stack behavior, redo clearing after new edits, and revision-based dirty tracking
- [x] run tests: `go test ./internal/tui`

### Task 3: Record history for every mutating edit path

**Files:**
- Modify: `internal/tui/commands.go`
- Modify: `internal/tui/model.go`
- Modify: `internal/tui/model_test.go`

- [x] wrap scalar replace, key rename, object add, array add, and delete flows so each mutation records the correct inverse and forward operation data
- [x] capture history for asynchronous completion paths from external-editor replacement and `jq` / `jq!` transforms
- [x] restore the intended selection anchor after undo and redo, relying on the existing session refresh/reselection behavior when the exact row is no longer visible
- [x] write tests for undo/redo of scalar edit, rename, object add, array add, delete, and batch-row append
- [x] write tests for undo/redo through external-editor and `jq` completion messages
- [x] run tests: `go test ./internal/tui`

### Task 4: Add normal-mode bindings and command aliases

**Files:**
- Modify: `internal/tui/model.go`
- Modify: `internal/tui/commands.go`
- Modify: `internal/tui/keymap.go`
- Modify: `internal/tui/model_test.go`
- Modify: `internal/tui/help_test.go`

- [x] add `u` as a normal-mode undo binding
- [x] add both `U` and `ctrl+r` as normal-mode redo bindings without affecting prompt, help, or settings behavior
- [x] add `:undo` and `:redo` command aliases that route through the same history helpers as the key bindings
- [x] surface precise status and error messages for success, empty undo stack, and empty redo stack
- [x] write tests for key bindings, command aliases, and empty-history error cases
- [x] run tests: `go test ./...`

### Task 5: Verify save-point semantics and acceptance criteria

**Files:**
- Modify: `internal/tui/commands.go`
- Modify: `internal/tui/model_test.go`
- Modify: `docs/plans/completed/20260325-vim-style-undo-redo.md`

- [x] update save flows so successful saves mark the current revision as the saved revision without clearing edit history
- [x] verify undoing back to the saved revision clears the dirty marker and redoing past it restores the dirty marker
- [x] verify a fresh edit after undo clears redo history
- [x] run full test suite: `go test ./...`
- [ ] perform a focused manual terminal smoke test for undo/redo after save, after long-array batch append, and after external edit
- [x] record any scope adjustments, edge cases, or follow-up items in this plan

### Task 6: [Final] Update documentation and archive the plan

**Files:**
- Modify: `README.md`
- Modify: `docs/plans/completed/20260325-vim-style-undo-redo.md`
- Create: `docs/plans/completed/`

- [x] update `README.md` to document undo/redo bindings, command aliases, and edit-history scope
- [x] update this plan with final scope notes, testing results, and any deferred follow-ups
- [x] move this plan to `docs/plans/completed/`

## Technical Details
- **history scope**: only document mutations participate in undo/redo. Navigation, fold state, search query changes, theme preview, and settings changes remain outside the history system.
- **supported operations**: the first pass should cover scalar replace, subtree replace from external editor, whole-document or subtree replace from `jq`, key rename, object insert, array insert, and delete.
- **selection anchors**: history entries should store `session.RowID` before and after the edit. On replay, set the stored row selection, refresh, and rely on existing `Session.Refresh()` fallback behavior when the exact row is no longer visible.
- **inverse-op data**:
  - `replace`: target node ID plus deep clones of the old and new subtree
  - `rename`: target node ID plus old and new key
  - `object insert`: parent object ID, insertion index, key, and inserted subtree clone
  - `array insert`: parent array ID, insertion index, and inserted subtree clone
  - `delete`: parent ID, parent kind, original index, deleted key if needed, and deleted subtree clone
- **dirty tracking**: replace ad hoc `Session.Dirty = true` edit toggles with revision-based dirty synchronization so the saved point is preserved across undo and redo.
- **redo semantics**: any new edit after an undo must discard the redo stack immediately.
- **scope guardrails**: do not add history persistence, configurable history limits, multi-edit coalescing, or UI-history for non-document actions in this pass.

## Post-Completion
*Items requiring manual intervention or external systems - no checkboxes, informational only*

**Manual verification**:
- edit a scalar, undo it with `u`, and redo it with both `U` and `ctrl+r`
- add and delete object/array items, then confirm undo restores original order rather than appending restored nodes at the end
- append from a selected long-array batch row, then undo and redo while confirming selection stays sensible
- save the document, undo back to the saved state, and confirm the dirty indicator clears
- apply an external edit and a `jq` transform, then confirm both are reversible

**Potential follow-up**:
- consider compact per-entry status labels if users later want more descriptive undo messages than `undid change` / `redid change`
- consider a bounded history limit only if large real-world documents show memory pressure after the inverse-op implementation lands

## Scope Notes
- The final implementation added a preserve-IDs `Restore` helper alongside deep clone and indexed insert helpers in the document layer. Container replace redo chains need stable descendant IDs so later redo entries targeting children keep working.
- Edit history now lives in `internal/tui/history.go` as typed inverse operations with row-based selection anchors and revision-based dirty tracking. Saving marks the current revision as the saved checkpoint instead of clearing undo/redo stacks.
- Automated coverage now includes history stack behavior, scalar/object/array edits, delete restore, save-point dirty semantics, long-array batch append undo/redo, async external-editor and `jq` completion replay, and help/README assertions.
- `go test ./...` passes. A real terminal smoke test for `u`, `U`, `ctrl+r`, and save-point behavior is still pending.
