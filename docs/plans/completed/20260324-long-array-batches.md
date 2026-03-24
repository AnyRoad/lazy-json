# Long Array Batches

## Overview
- Batch long arrays into synthetic navigation rows of 100 items each so expanding very large arrays does not dump every element directly into the visible tree.
- Only arrays with more than 100 items should batch; arrays with 100 or fewer items keep the current direct child-row behavior.
- Render synthetic batch rows with the agreed minimalist label style such as `[0-99]`, and treat them as navigation-only containers rather than real JSON nodes.

## Context (from discovery)
- files/components involved: [`internal/session/rows.go`](/Users/andrei/PROJECTS/lazy-json/internal/session/rows.go), [`internal/session/session.go`](/Users/andrei/PROJECTS/lazy-json/internal/session/session.go), [`internal/session/search.go`](/Users/andrei/PROJECTS/lazy-json/internal/session/search.go), [`internal/session/session_test.go`](/Users/andrei/PROJECTS/lazy-json/internal/session/session_test.go), [`internal/tui/model.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/model.go), [`internal/tui/commands.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/commands.go), [`internal/tui/view.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/view.go), [`internal/tui/view_test.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/view_test.go), [`internal/tui/model_test.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/model_test.go), [`internal/tui/search_test.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/search_test.go), and [`README.md`](/Users/andrei/PROJECTS/lazy-json/README.md).
- related row-building patterns found: [`internal/session/rows.go`](/Users/andrei/PROJECTS/lazy-json/internal/session/rows.go) currently walks real document nodes only and emits one visible `Row` per visible node, with expansion state stored only by real `NodeID`.
- related selection/navigation constraints found: [`internal/session/session.go`](/Users/andrei/PROJECTS/lazy-json/internal/session/session.go) indexes visible rows by `NodeID` and stores selection as `SelectedID`, which cannot represent synthetic batch rows without refactoring row identity.
- related command constraints found: [`internal/tui/model.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/model.go) and [`internal/tui/commands.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/commands.go) assume the selected row is always a real node for edit, rename, delete, subtree copy, and external-editor actions.
- related reveal/search patterns found: [`internal/session/search.go`](/Users/andrei/PROJECTS/lazy-json/internal/session/search.go) searches the full real-node tree already, while [`internal/session/session.go`](/Users/andrei/PROJECTS/lazy-json/internal/session/session.go) reveals hits by expanding ancestor nodes only.
- related rendering patterns found: [`internal/tui/view.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/view.go) derives labels from `Row` depth, key, array index, and container state, so batch rendering can stay local to row metadata once session emits synthetic rows.
- dependencies identified: row identity, expansion state, reveal logic, command gating, view labels, and `go test ./...` for validation.

## Development Approach
- **testing approach**: Regular (code first, then tests in each task)
- keep batching as a session/view concern; do not add synthetic nodes to the document model or change JSON serialization behavior
- batch only when an array has more than 100 elements; exactly 100 elements should still expand directly into item rows
- represent synthetic batch rows explicitly in `session.Row` so selection, rendering, and navigation can distinguish real nodes from batch containers
- move selection/indexing from plain `NodeID` assumptions toward a row identity that can address both real node rows and batch rows
- keep existing commands unchanged for real node rows, while batch rows become navigation-only with narrow exceptions such as `a` targeting the parent array
- make search reveal open both the real ancestor array and the matching synthetic batch row automatically
- preserve current behavior for objects, short arrays, save/load, clipboard output, JSON paths, and line numbers
- **CRITICAL: every task MUST include new or updated tests** for changed code paths
- **CRITICAL: all tests must pass before moving to the next task**
- update this plan if implementation scope changes during execution

## Testing Strategy
- **session row tests**: verify arrays of 100 do not batch, arrays of 101 batch into two synthetic ranges, and larger arrays produce the expected visible batch rows and row identities
- **navigation tests**: verify `l` expands long arrays into batch rows, `l` on a batch row opens its real child items, `h` collapses batch rows, and selection remains valid when visible rows rebuild
- **command tests**: verify edit/delete/rename/value-copy/subtree-copy reject batch rows with clear errors, while add-on-batch targets the parent array
- **search tests**: verify a search hit inside a long collapsed array expands the ancestor array and the correct batch automatically
- **view tests**: verify batch rows render with labels like `[0-99]`, keep existing tree markers and line-number gutter alignment, and do not break path rendering
- **manual verification**: open a file with a very large array, expand it, confirm batch rows appear and can be navigated fluidly, then search for a deep item and verify the correct batch opens
- **e2e tests**: none currently; session/model/view coverage is sufficient for this TUI behavior change

## Progress Tracking
- mark completed items with `[x]` immediately when done
- add newly discovered tasks with `➕` prefix
- document blockers or trade-offs with `⚠️` prefix
- keep this file synchronized with the actual implementation state

## What Goes Where
- **Implementation Steps** (`[ ]` checkboxes): code, tests, and documentation updates inside this repository
- **Post-Completion** (no checkboxes): manual terminal checks and any later UX refinements such as configurable batch size

## Implementation Steps

### Task 1: Introduce row identity and synthetic batch metadata

**Files:**
- Modify: `internal/session/rows.go`
- Modify: `internal/session/session.go`
- Modify: `internal/session/session_test.go`

- [x] extend `session.Row` with explicit row kind and batch range metadata for synthetic long-array rows
- [x] introduce a row-selection/indexing key that can represent either a real node row or a synthetic batch row
- [x] update session refresh and reselection logic to preserve valid selection across rebuilt visible rows
- [x] write tests for row identity, selection persistence, and visible row indexing with synthetic batch rows
- [x] write tests for selection fallback when a previously selected batch row disappears after collapse or edit
- [x] run tests: `go test ./...`

### Task 2: Build batch rows for long arrays and support fold navigation

**Files:**
- Modify: `internal/session/rows.go`
- Modify: `internal/session/session.go`
- Modify: `internal/session/session_test.go`
- Modify: `internal/tui/model.go`
- Modify: `internal/tui/model_test.go`

- [x] emit synthetic batch rows in `BuildRows` when an expanded array has more than 100 items, using contiguous 100-item ranges
- [x] keep arrays with 100 or fewer items on the existing direct child-row path
- [x] add batch expansion state so synthetic batch rows behave like collapsible containers under `h` and `l`
- [x] update movement and fold behavior so `l` on a long array enters the first batch and `h` on a collapsed batch returns to the parent array row
- [x] write tests for 100-item, 101-item, and larger-array batching boundaries plus fold navigation behavior
- [x] run tests: `go test ./...`

### Task 3: Make search reveal and commands batch-aware

**Files:**
- Modify: `internal/session/session.go`
- Modify: `internal/session/session_test.go`
- Modify: `internal/tui/commands.go`
- Modify: `internal/tui/model.go`
- Modify: `internal/tui/model_test.go`
- Modify: `internal/tui/search_test.go`

- [x] extend reveal logic so a searched or selected descendant inside a long array expands both the real array ancestor and the matching synthetic batch row
- [x] keep search hit collection on real nodes only while mapping each hit to the correct visible batch during reveal
- [x] guard batch rows from scalar edit, external edit, rename, delete, key copy, value copy, and subtree copy with clear navigation-only errors
- [x] make add-on-batch target the parent array so appending from a selected batch row still works
- [x] write tests for batch-aware reveal, rejected commands on batch rows, and add behavior from a selected batch row
- [x] run tests: `go test ./...`

### Task 4: Render synthetic batch rows and document the behavior

**Files:**
- Modify: `internal/tui/view.go`
- Modify: `internal/tui/view_test.go`
- Modify: `README.md`

- [x] render synthetic batch rows with the agreed label style `[start-end]` while preserving existing tree markers, indentation, and optional line-number gutter
- [x] ensure batch rows do not attempt to render real-node values or misleading JSON-path suffixes
- [x] write view tests for batch label rendering, gutter alignment, and coexistence with existing display settings
- [x] update README.md to document that arrays longer than 100 expand into 100-item batch rows
- [x] update any affected documentation assertions if README text is covered by tests
- [x] run tests: `go test ./...`

### Task 5: Verify acceptance criteria

**Files:**
- Modify: `docs/plans/20260324-long-array-batches.md`

- [x] verify arrays with 100 or fewer elements still expand directly without synthetic rows
- [x] verify arrays with more than 100 elements expand into `[0-99]`, `[100-199]`, and later ranges as needed
- [x] verify batch rows are navigation-only and search reveal opens the correct batch automatically
- [x] run full test suite: `go test ./...`
- [x] record any scope adjustments, trade-offs, or follow-up items in this plan

### Task 6: [Final] Update documentation state

**Files:**
- Modify: `docs/plans/20260324-long-array-batches.md`

- [x] confirm README wording matches the implemented behavior
- [x] note any manual verification still pending
- [x] move this plan to `docs/plans/completed/`

## Technical Details
- **batch threshold**: only arrays with `len(array) > 100` batch; `100` remains unbatched
- **batch size**: synthetic rows cover contiguous ranges of exactly 100 items except for the final partial range
- **row identity**: selection must distinguish between real node rows and synthetic batch rows, so visible-row indexing can no longer rely on `NodeID` alone
- **batch rendering**: synthetic batch rows should reuse container markers (`▸` / `▾`) and indentation but render only the range label, for example `[0-99]`
- **search reveal**: search hit storage remains node-based, while reveal logic must derive which long-array batch contains the hit and expand that synthetic row
- **command semantics**: batch rows are navigation-only placeholders, not editable JSON values; `a` is the only command that should resolve through them to the parent array
- **scope guardrail**: do not add configurable batch sizes, source-model synthetic nodes, or separate paging commands in this feature

## Post-Completion
*Items requiring manual intervention or external systems - no checkboxes, informational only*

**Manual verification**:
- open a JSON document with an array longer than 100 items and confirm expanding the array shows `[0-99]`, `[100-199]`, and later batches instead of raw child rows
- expand and collapse batch rows with `h` and `l` and confirm selection behaves predictably when moving between array, batch, and real item rows
- run a search that lands inside a deep long-array element and confirm the correct batch opens automatically
- append a new item while a batch row is selected and confirm it targets the parent array rather than failing

**Potential follow-up**:
- configurable batch size if users later want larger or smaller chunks
- richer batch labels such as counts or path hints if the minimalist `[0-99]` label proves too opaque in practice
- jump-to-batch or jump-to-index commands if large-array navigation still needs more speed after batching lands

## Scope Notes
- This plan intentionally keeps batching as a UI/session abstraction, so the saved JSON document and parser stay unchanged.
- The feature applies only to long arrays; objects and shorter arrays keep their current expansion behavior.
- Search and selection remain node-centric internally where possible, with batch rows acting as reveal/navigation scaffolding rather than new JSON entities.
- `ExpandAll` and `zR` continue to expand real JSON containers only; long arrays still open into batch rows instead of auto-expanding every batch.
- Automated verification is complete via `go test ./...`; manual terminal checks for very large arrays and narrow viewport readability are still pending.
