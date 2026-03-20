# Prefix Shortcuts for Clipboard and Tree Actions

## Overview
- Add scalable prefix-based normal-mode shortcuts for clipboard-oriented actions and tree-wide navigation in `lazy-json`.
- Support copying the selected JSON path, key, compact value, pretty subtree, and whole document directly to the system clipboard.
- Add expand-all, collapse-all, and "next parent sibling" navigation as both shortcuts and `:` commands.
- Keep the interaction model consistent with the existing keyboard-first editor by reusing centralized action helpers, precise status/error messages, and testable Bubble Tea update flows.

## Context (from discovery)
- files/components involved: [`internal/tui/model.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/model.go), [`internal/tui/commands.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/commands.go), [`internal/tui/keymap.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/keymap.go), [`internal/tui/view.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/view.go), [`internal/tui/model_test.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/model_test.go), [`internal/tui/help_test.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/help_test.go), [`internal/tui/view_test.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/view_test.go), [`internal/session/session.go`](/Users/andrei/PROJECTS/lazy-json/internal/session/session.go), [`internal/session/rows.go`](/Users/andrei/PROJECTS/lazy-json/internal/session/rows.go), [`internal/session/session_test.go`](/Users/andrei/PROJECTS/lazy-json/internal/session/session_test.go), [`internal/document/serialize.go`](/Users/andrei/PROJECTS/lazy-json/internal/document/serialize.go), [`README.md`](/Users/andrei/PROJECTS/lazy-json/README.md), and [`go.mod`](/Users/andrei/PROJECTS/lazy-json/go.mod).
- related patterns found: normal-mode key handling currently lives in [`internal/tui/model.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/model.go) and already has a small multi-key precedent via `gg` using `lastKey`; command aliases are centralized in `handleCommand` inside [`internal/tui/commands.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/commands.go).
- related data/rendering patterns found: visible rows already carry JSON-path text via `session.Row.Path` in [`internal/session/rows.go`](/Users/andrei/PROJECTS/lazy-json/internal/session/rows.go), and the view/footer pipeline in [`internal/tui/view.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/view.go) is the right place to surface pending-prefix hints and action status messages.
- related serialization/integration patterns found: subtree and full-document JSON formatting already exist in [`internal/document/serialize.go`](/Users/andrei/PROJECTS/lazy-json/internal/document/serialize.go), and optional external tool behavior is already isolated behind integration helpers such as [`internal/integration/editor.go`](/Users/andrei/PROJECTS/lazy-json/internal/integration/editor.go) and [`internal/integration/jq.go`](/Users/andrei/PROJECTS/lazy-json/internal/integration/jq.go).
- dependencies identified: Bubble Tea update flow, existing document/session abstractions, `github.com/atotto/clipboard` for system clipboard writes, and `go test ./...` for validation.

## Development Approach
- **testing approach**: Regular (code first, then tests in each task)
- complete each task fully before moving to the next
- make small, focused changes rather than introducing a generic keybinding framework before there is a second concrete need
- **CRITICAL: every task MUST include new or updated tests** for code changed in that task
- tests are not optional; they are a required deliverable of every task
- include both success and error scenarios in tests
- **CRITICAL: all tests must pass before starting the next task**
- keep the shortcut families mnemonic and deterministic: `y*` for copy/yank, `z*` for fold state, `]p` for forward structural navigation
- keep command-mode aliases thin by routing them through the same helpers used by normal-mode shortcuts
- update this plan if implementation scope changes during execution

## Testing Strategy
- **unit tests**: required for every task, using `go test ./...`
- **model/update tests**: drive Bubble Tea updates directly to verify prefix handling, command dispatch, pending-prefix clearing, and error/status messaging
- **session tests**: validate tree-wide expand/collapse behavior and next-parent-sibling navigation independently from the UI layer where possible
- **integration-style tests**: stub clipboard writes through an injected adapter so copy commands can be verified without touching the real clipboard
- **render/help tests**: assert footer hints, help output, and README snippets for the new bindings and commands
- **manual verification**: confirm real clipboard behavior in a terminal session and spot-check that prefix input feels responsive and unambiguous
- **e2e tests**: none currently; rely on unit/model/view coverage plus manual terminal smoke testing

## Progress Tracking
- mark completed items with `[x]` immediately when done
- add newly discovered tasks with `➕` prefix
- document blockers or trade-offs with `⚠️` prefix
- update plan text if implementation deviates from the current design
- keep this file synchronized with the actual implementation state

## What Goes Where
- **Implementation Steps** (`[ ]` checkboxes): code, tests, and documentation changes inside this repository
- **Post-Completion** (no checkboxes): manual verification and external follow-up outside the automated implementation steps

## Implementation Steps

### Task 1: Add clipboard integration and reusable copy helpers

**Files:**
- Modify: `go.mod`
- Create: `internal/integration/clipboard.go`
- Create: `internal/integration/clipboard_test.go`
- Modify: `internal/tui/commands.go`
- Modify: `internal/tui/model.go`
- Modify: `internal/tui/model_test.go`

- [x] make `github.com/atotto/clipboard` a direct dependency and add a small clipboard adapter that can be replaced in tests
- [x] inject clipboard access into the TUI model without coupling tests to the host clipboard environment
- [x] implement reusable helpers for copying path, key, compact value, pretty subtree, and whole-document JSON
- [x] write tests for successful copy payloads and clipboard write failures
- [x] write tests for error cases such as copying a key from a node without an object key
- [x] run tests: `go test ./...`

### Task 2: Generalize normal-mode key handling for prefix families

**Files:**
- Modify: `internal/tui/model.go`
- Modify: `internal/tui/view.go`
- Modify: `internal/tui/model_test.go`
- Modify: `internal/tui/view_test.go`

- [x] replace the single-purpose `lastKey` handling with a small pending-prefix state that supports `g`, `y`, `z`, and `]`
- [x] implement prefix resolution for `yp`, `yk`, `yv`, `ys`, `yj`, `zR`, `zM`, and `]p` while preserving existing single-key behavior
- [x] clear pending prefixes correctly on invalid continuations, prompt entry, help/settings transitions, and non-prefix commands
- [x] surface pending-prefix feedback in the footer so partial input is visible to the user
- [x] write tests for prefix success paths, unknown second-key errors, and prefix reset behavior
- [x] run tests: `go test ./...`

### Task 3: Add structural navigation and tree-wide fold helpers

**Files:**
- Modify: `internal/session/session.go`
- Modify: `internal/session/session_test.go`
- Modify: `internal/tui/commands.go`
- Modify: `internal/tui/model_test.go`

- [x] add session helpers to expand all containers and collapse all containers except the root
- [x] implement next-parent-sibling traversal that climbs ancestors until a valid next sibling target is found
- [x] wire `:expand-all`, `:collapse-all`, and `:next-parent-sibling` to the same helpers used by the new shortcuts
- [x] write tests for expand-all/collapse-all row visibility and selection stability
- [x] write tests for immediate-parent and ancestor-climbing `]p` navigation, including the no-target error case
- [x] run tests: `go test ./...`

### Task 4: Finish command aliases, help text, and user documentation

**Files:**
- Modify: `internal/tui/commands.go`
- Modify: `internal/tui/keymap.go`
- Modify: `internal/tui/help_test.go`
- Modify: `internal/tui/view_test.go`
- Modify: `README.md`

- [x] add command-mode aliases for `:copy-path`, `:copy-key`, `:copy-value`, `:copy-subtree`, `:copy-json`, `:expand-all`, `:collapse-all`, and `:next-parent-sibling`
- [x] update help text to document the new prefix families and command names clearly
- [x] document the copy semantics and structural navigation behavior in `README.md`
- [x] write tests for updated help output, footer hints, and README command/shortcut documentation
- [x] write tests that command aliases route to the expected actions and report precise status/error messages
- [x] run tests: `go test ./...`

### Task 5: Verify acceptance criteria

**Files:**
- Modify: `docs/plans/20260321-prefix-shortcuts-for-clipboard-and-tree-actions.md`

- [x] verify that all requested shortcuts and command aliases are implemented with the agreed semantics
- [x] verify that compact vs pretty JSON copy formats match the design for value, subtree, and whole-document actions
- [x] run full test suite: `go test ./...`
- [ ] perform a manual terminal smoke test for prefix input, clipboard behavior, and tree navigation
- [x] record any scope adjustments, follow-up items, or implementation trade-offs in this plan

### Task 6: [Final] Update documentation and archive the plan

**Files:**
- Modify: `README.md`
- Modify: `docs/plans/20260321-prefix-shortcuts-for-clipboard-and-tree-actions.md`
- Create: `docs/plans/completed/`

- [x] update `README.md` if verification changes wording, examples, or binding descriptions
- [x] update this plan file with final scope notes, warnings, and any follow-up items
- [ ] move this plan to `docs/plans/completed/`
⚠️ Automated coverage is in place for clipboard payloads, prefix handling, structural navigation, footer/help rendering, and command aliases. A real terminal smoke test for system clipboard integration is still pending.

## Technical Details
- **shortcut families**: use a compact pending-prefix state in the TUI model instead of a growing set of one-off booleans. Supported families should include `g` for existing `gg`, `y` for copy/yank actions, `z` for fold-state actions, and `]` for forward structural navigation.
- **clipboard outputs**: `yp` copies the rendered JSON path, `yk` copies the raw object key, `yv` copies the selected node as compact JSON, `ys` copies the selected node as pretty JSON with indentation, and `yj` copies the entire document as pretty JSON.
- **command aliases**: `:copy-path`, `:copy-key`, `:copy-value`, `:copy-subtree`, `:copy-json`, `:expand-all`, `:collapse-all`, and `:next-parent-sibling` should call the same helpers as the keyboard shortcuts so behavior stays identical.
- **next parent sibling semantics**: starting from the selected node, inspect the parent; if that parent has a next sibling, jump there. If not, keep climbing ancestors until a next sibling is found. If no ancestor has a next sibling, report `no next parent sibling`.
- **fold semantics**: `zR` expands every container in the document tree; `zM` collapses every container except the root so the user keeps a stable top-level overview.
- **status and error messaging**: copy actions should report what was copied, clipboard failures should surface the underlying error, invalid prefix continuations should report the full attempted shortcut, and key-copying on non-object entries should return `selected node does not have an object key`.
- **rendering behavior**: while waiting for a second key in a prefix family, the footer should make the pending prefix visible so input does not appear to be swallowed.

## Post-Completion
*Items requiring manual intervention or external systems - no checkboxes, informational only*

**Manual verification**:
- open a nested JSON fixture and verify `yp`, `yk`, `yv`, `ys`, and `yj` place the expected payloads on the real system clipboard
- confirm `zR` and `zM` behave predictably on both deep objects and arrays, with selection remaining valid
- verify `]p` on a deep node, on the last child of a branch, and on the final branch in the document
- check that invalid second keys after `y`, `z`, or `]` clear the prefix and show a concise error without leaving the model stuck

**External system updates**:
- consider whether future shortcut families justify extracting a reusable keymap table after this feature lands
- revisit clipboard support on unusual terminal/desktop environments if real-world reports show platform-specific failures
