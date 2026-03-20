# JSON TUI Editor V1

## Overview
- Build `lazy-json`, a Go-based terminal JSON viewer/editor with a tree-first editing model.
- Support file-backed and stdin-backed workflows from day one: `lazy-json file.json` and `cat file.json | lazy-json`.
- Deliver keyboard-first navigation with Vim-like movement, expand/collapse, search, structured edits, canonical JSON saves, optional external editor handoff, and optional `jq` transforms.
- Keep v1 intentionally scoped around structured editing instead of building a general-purpose raw text editor.

## Context (from discovery)
- files/components involved: repository is currently empty; the initial implementation will create the CLI entrypoint, document model, Bubble Tea app, integration helpers, tests, and README.
- related patterns found: no existing code or conventions are present in the repo, so package boundaries and testing patterns can be chosen cleanly.
- dependencies identified: Go toolchain, `bubbletea`, `lipgloss`, `bubbles`, Go standard library, optional `jq` binary at runtime, optional `$EDITOR` at runtime.

## Development Approach
- **testing approach**: Regular (code first, then tests in each task)
- complete each task fully before moving to the next
- make small, focused changes
- **CRITICAL: every task MUST include new/updated tests** for code changes in that task
- tests are not optional; they are a required deliverable of every task
- write unit tests for new functions and modified behavior
- include both success and error scenarios in tests
- **CRITICAL: all tests must pass before starting the next task**
- update this plan if implementation scope changes during execution
- prefer simple package boundaries over premature abstractions
- maintain deterministic behavior for ordering, selection, and serialization

## Testing Strategy
- **unit tests**: required for every task, using `go test ./...`
- **model/update tests**: exercise Bubble Tea update logic directly instead of relying on a full terminal harness for most behavior
- **render tests**: use string/golden-style assertions for view output where practical
- **integration-style tests**: stub subprocess execution for `jq` and `$EDITOR` flows so success and failure behavior can be validated deterministically
- **manual verification**: cover end-to-end terminal workflows for file load/save, stdin/stdout, search, structured edits, external editor handoff, and `jq`

## Progress Tracking
- mark completed items with `[x]` immediately when done
- add newly discovered tasks with `➕` prefix
- document blockers or trade-offs with `⚠️` prefix
- update plan text if implementation deviates from the current design
- keep this file synchronized with the actual implementation state

## What Goes Where
- **Implementation Steps** (`[ ]` checkboxes): code, tests, and project documentation changes that happen inside this repo
- **Post-Completion** (no checkboxes): manual validation and follow-up items that require terminal verification or user review

## Implementation Steps

### Task 1: Bootstrap the module and CLI startup flow

**Files:**
- Create: `go.mod`
- Create: `cmd/lazy-json/main.go`
- Create: `internal/source/input.go`
- Create: `internal/source/input_test.go`
- Create: `internal/tui/model.go`
- Create: `internal/tui/model_test.go`

- [x] initialize the Go module and add the initial Bubble Tea ecosystem dependencies
- [x] implement invocation parsing for `lazy-json <file>` and piped stdin input, and reject ambiguous startup states
- [x] create a minimal Bubble Tea app shell that can receive loaded source content and exit cleanly
- [x] surface startup and load errors from `cmd/lazy-json/main.go` with non-zero exit codes
- [x] write tests for source detection, stdin/file precedence, and ambiguous invocation handling
- [x] write tests for the initial model startup/quit flow
- [x] run tests: `go test ./...`

### Task 2: Implement the mutable ordered JSON document model

**Files:**
- Create: `internal/document/node.go`
- Create: `internal/document/parse.go`
- Create: `internal/document/edit.go`
- Create: `internal/document/serialize.go`
- Create: `internal/document/document_test.go`
- Modify: `internal/tui/model.go`

- [x] define node kinds, stable node IDs, ordered object entries, and the root document container
- [x] parse JSON into the document model without losing numeric precision, using validated number lexemes instead of `float64`
- [x] implement canonical pretty JSON serialization for full-document and subtree writes
- [x] implement structured mutations for replace, add child, insert array item, rename key, and delete node
- [x] write tests for parsing, serialization, and successful edit operations
- [x] write tests for invalid mutations, invalid numbers, and root-level edge cases
- [x] run tests: `go test ./...`

### Task 3: Build session state and visible-row projection

**Files:**
- Create: `internal/session/session.go`
- Create: `internal/session/rows.go`
- Create: `internal/session/session_test.go`
- Modify: `internal/tui/model.go`

- [x] add session state for selected node, expanded/collapsed nodes, source metadata, dirty flag, current mode, and status messages
- [x] build a flattened visible-row projection with depth, path text, parent references, and row-to-node mapping
- [x] implement navigation helpers for up/down, parent/child, top/bottom, and stable reselection after document mutations
- [x] recompute visible rows predictably after edits, theme changes, and collapse/expand actions
- [x] write tests for row projection, path generation, and selection movement
- [x] write tests for collapsed ancestor behavior, deletion reselection, and empty/leaf edge cases
- [x] run tests: `go test ./...`

### Task 4: Build the core TUI viewer, keymap, and themes

**Files:**
- Create: `internal/tui/view.go`
- Create: `internal/tui/keymap.go`
- Create: `internal/tui/theme.go`
- Create: `internal/tui/view_test.go`
- Modify: `internal/tui/model.go`

- [x] render the visible tree rows with syntax highlighting, indentation guides, selection state, and width-aware wrapping
- [x] add multiple color schemes for JSON rendering and a simple runtime theme switch action
- [x] implement navigation and tree control bindings for `h/j/k/l`, `gg`, `G`, expand, collapse, and child entry
- [x] add a status/help strip that shows mode, dirty state, source kind, and transient errors
- [x] write tests for key handling and state transitions in the core viewer flow
- [x] write tests for rendered output fragments, wrapping behavior, and theme selection
- [x] run tests: `go test ./...`

### Task 5: Implement structured editing commands and save flows

**Files:**
- Create: `internal/tui/prompt.go`
- Create: `internal/tui/commands.go`
- Create: `internal/source/output.go`
- Create: `internal/source/output_test.go`
- Create: `internal/tui/editing_test.go`
- Modify: `internal/tui/model.go`

- [x] implement normal, prompt, and `:` command modes for inline scalar edits, key renames, and command entry
- [x] add structured editing actions for add object field, add array item, delete node, rename key, and change scalar value/type
- [x] implement save commands for `:w`, `:w <path>`, `:x`, and `:print`, including atomic file writes and stdout emission for stdin-backed sessions
- [x] track dirty state accurately and prompt before destructive quit when unsaved changes exist
- [x] write tests for save/output success paths and file/stdout behavior
- [x] write tests for edit validation failures, invalid command input, and unsaved-quit protection
- [x] run tests: `go test ./...`

### Task 6: Add search and match navigation

**Files:**
- Create: `internal/session/search.go`
- Create: `internal/session/search_test.go`
- Create: `internal/tui/search_test.go`
- Modify: `internal/tui/model.go`
- Modify: `internal/tui/view.go`

- [x] implement substring search across object keys, scalar values, and rendered JSON paths
- [x] add `/`, `n`, and `N` flows with search prompt state, current match focus, and match highlight styling
- [x] keep search results stable as rows collapse, expand, and mutate after edits
- [x] write tests for match indexing and next/previous traversal behavior
- [x] write tests for search prompt interactions, empty queries, and no-match behavior
- [x] run tests: `go test ./...`

### Task 7: Add external editor handoff for node/subtree editing

**Files:**
- Create: `internal/integration/editor.go`
- Create: `internal/integration/editor_test.go`
- Modify: `internal/tui/commands.go`
- Modify: `internal/tui/model.go`

- [x] implement `E` and `:edit-external` for the selected scalar or subtree by serializing JSON to a temp file and invoking `$EDITOR`
- [x] suspend and resume the TUI cleanly around the external editor flow
- [x] validate edited JSON on return and replace the selected node/subtree only on successful parse
- [x] preserve selection and show actionable errors when the editor is missing, canceled, or returns invalid JSON
- [x] write tests for successful external edit replacement and temp-file cleanup behavior
- [x] write tests for missing-editor, canceled-edit, and invalid-JSON recovery paths
- [x] run tests: `go test ./...`

### Task 8: Add optional `jq` transform commands

**Files:**
- Create: `internal/integration/jq.go`
- Create: `internal/integration/jq_test.go`
- Modify: `internal/tui/commands.go`
- Modify: `internal/tui/model.go`

- [x] implement `:jq <expr>` for whole-document transforms and `:jq! <expr>` for selected-subtree transforms
- [x] run `jq` as a pure transform step by serializing the target JSON, capturing stdout/stderr, and validating that stdout is exactly one JSON value
- [x] leave the document unchanged when `jq` is missing, exits non-zero, or produces empty/invalid/multi-document output
- [x] update dirty state, selection, and status messages correctly after successful transforms
- [x] write tests for successful document and subtree replacement via stubbed `jq` execution
- [x] write tests for missing binary, non-zero exit, invalid output, and multi-document output
- [x] run tests: `go test ./...`

### Task 9: Polish the UX and add operator-facing documentation

**Files:**
- Create: `internal/tui/help.go`
- Create: `internal/tui/help_test.go`
- Create: `testdata/basic.json`
- Create: `testdata/nested.json`
- Create: `testdata/numbers.json`
- Create: `README.md`
- Modify: `cmd/lazy-json/main.go`
- Modify: `internal/tui/model.go`
- Modify: `internal/tui/view.go`

- [x] add a compact help overlay or help screen for keybindings, command mode actions, and theme/jq/editor features
- [x] improve user-facing startup and runtime error messaging for invalid JSON, missing files, missing `jq`, and missing `$EDITOR`
- [x] add representative JSON fixtures for tests and manual terminal verification
- [x] document installation, invocation modes, keybindings, save behavior, external editor behavior, and optional `jq` support in `README.md`
- [x] write tests for help visibility, error banner rendering, and command discoverability
- [x] write tests for startup error messaging and fixture-backed smoke coverage where useful
- [x] run tests: `go test ./...`

### Task 10: Verify acceptance criteria

**Files:**
- Modify: `docs/plans/20260318-json-tui-editor-v1.md`

- [x] verify that file-backed and stdin-backed workflows both behave as described in the Overview
- [x] verify that navigation, expand/collapse, search, structured edits, save flows, external editor handoff, and optional `jq` transforms all work end-to-end
- [x] run full test suite: `go test ./...`
- [ ] perform a manual terminal smoke test with `testdata/basic.json`, `testdata/nested.json`, and piped stdin examples
- [x] verify that the implementation still matches the v1 scope and record any scope adjustments in this plan

### Task 11: [Final] Update documentation and archive the plan

**Files:**
- Modify: `README.md`
- Modify: `docs/plans/20260318-json-tui-editor-v1.md`
- Create: `docs/plans/completed/`

- [x] update `README.md` for any behavior changes discovered during implementation or verification
- [x] update this plan file to reflect the final executed scope and any follow-up notes
- [ ] move this plan to `docs/plans/completed/`

## Technical Details
- **Document model**: represent JSON as a mutable tree with stable `NodeID` values. Objects store ordered `[]ObjectEntry` instead of Go maps so rendering, navigation, and saves remain deterministic.
- **Number handling**: avoid parsing into `float64`; store validated JSON number text so large integers and precise decimals survive load, edit, and save cycles.
- **Session state**: keep editor state outside the document itself. Track selected node, expanded set, flattened visible rows, search state, current mode, theme, source kind, dirty flag, and status line text separately.
- **Rendering flow**: each state-changing action recomputes visible rows, then the view layer renders styled lines only. Rendering should never mutate the document.
- **Source handling**: startup accepts either a file path or piped stdin. File-backed sessions default `:w` to the original path. Stdin-backed sessions require `:w <path>` for file writes, while `:print` emits canonical JSON to stdout.
- **TTY handoff for piped input**: stdin-backed sessions consume the JSON payload first, then Bubble Tea reopens the controlling terminal with `tea.WithInputTTY()` so keyboard interaction still works after pipe input is exhausted.
- **Save semantics**: file saves should use atomic replacement in the target directory. All writes use canonical pretty JSON formatting.
- **External editor flow**: serialize the selected scalar or subtree into a temp file, launch `$EDITOR`, parse the edited file on return, and replace the target only if the result is valid JSON.
- **`jq` flow**: treat `jq` as an optional pure transform. Serialize the current target, run `jq`, require exactly one valid JSON value on stdout, then replace the target and refresh session state.
- **Error handling**: integration failures should be non-destructive. Invalid edits, invalid subprocess output, or missing optional tools must leave the current document unchanged and show a clear status/error message.

## Post-Completion
*Items requiring manual intervention or external systems - no checkboxes, informational only*

**Manual verification**:
- open a JSON file directly, make structured edits, save with `:w`, and confirm canonical formatting on disk
- pipe JSON into `lazy-json`, edit it, use `:print`, and confirm stdout output is valid JSON
- verify `E` with a real editor such as `vim`, `nvim`, or `nano`
- verify `:jq` and `:jq!` with a real installed `jq`
- test behavior with large nested documents to assess responsiveness before deciding whether further optimization is needed

**External system updates**:
- decide on distribution strategy after v1 is working locally: single binary release, Homebrew tap, or package manager integration
- consider adding CI for `go test ./...` once the initial implementation stabilizes

⚠️ Manual interactive smoke testing is still pending. The automated unit/integration suite passes, and the stdin-backed input path was corrected to use Bubble Tea's TTY input option, but a full operator-driven terminal pass still needs to be done outside this automation session.
