# Raise Repo-Wide Coverage To 80% And Refresh The Badge

## Overview
- Raise the reproducible repo-wide Go coverage total to at least `80%` without changing application behavior.
- Standardize coverage measurement on a single workflow that uses `-coverpkg` across production packages and excludes benchmark-only helper code in `internal/perftest`.
- Update the static coverage badge in `README.md` so it reflects the same metric contributors use locally.

## Context (from discovery)
- Files/components involved:
  - `Makefile`
  - `README.md`
  - `docs/badges/coverage.svg`
  - `main.go`
  - `main_test.go`
  - `internal/document/document_test.go`
  - `internal/integration/clipboard_test.go`
  - `internal/integration/editor_test.go`
  - `internal/integration/jq_test.go`
  - `internal/session/session_test.go`
  - `internal/tui/model_test.go`
  - `internal/tui/history_test.go`
  - `internal/tui/view_test.go`
  - `internal/tui/settings_test.go`
- Related patterns found:
  - `README.md` already uses a committed static coverage badge at `docs/badges/coverage.svg`, but the documented refresh command still uses the older single-profile flow.
  - Current package-local coverage from `go test ./... -cover` is not the same as a single repo-wide total.
  - A repo-wide run with `-coverpkg=./...` currently lands below the target at roughly `76.7%`, so the gap is real and measurable.
  - The biggest practical coverage gaps are in startup handling, TUI command/model control flow, session edge helpers, and a few small integration/document branches.
  - `internal/perftest` exists only to support benchmarks and should not drive the user-facing coverage badge metric.
- Dependencies identified:
  - existing Go test tooling via `go test` and `go tool cover`
  - repo-local Go cache usage in `Makefile`
  - existing test suites in `main_test.go`, `internal/session`, `internal/tui`, `internal/document`, and `internal/integration`

## Development Approach
- **testing approach**: Regular (coverage workflow first, then targeted tests, then badge refresh)
- Use one explicit repo-wide coverage metric for all work in this pass:
  - instrument all production packages with `-coverpkg`
  - exclude `internal/perftest` from both the tested package list and the instrumented package list
- Prefer focused behavioral tests that cover currently untested branches users can actually trigger.
- Keep production-code changes minimal and only introduce tiny testability seams when a branch is otherwise hard to exercise deterministically.
- **CRITICAL: every task MUST include new or updated tests** for changed code paths.
- **CRITICAL: all tests must pass before starting the next task**.
- **CRITICAL: the coverage badge must be generated from the same metric used in `make cover`** so there is no mismatch between docs and local verification.
- **CRITICAL: do not add a CI threshold gate in this pass**; first make the metric reproducible and reach the target honestly.

## Testing Strategy
- **coverage workflow validation**:
  - add a `make cover` target that produces a stable profile and prints the repo-wide total
  - add a `make badge-cover` target that regenerates `docs/badges/coverage.svg` from that same profile or total
  - update README instructions to use those Makefile targets instead of ad hoc commands
- **startup tests**:
  - increase `main.go` branch coverage for argument parsing, settings resolution, and non-interactive startup paths
- **TUI/session tests**:
  - cover currently missed command branches such as save/quit, print/quit, copy/replace, prompt handling, busy completion, page movement, and batch/session edge helpers
- **document/integration tests**:
  - cover remaining low-coverage branches in serialization, editor temp-file creation, clipboard/jq wrappers, and document editing helpers where it meaningfully increases control-flow coverage
- **verification**:
  - run `make cover` repeatedly during implementation to measure progress
  - finish with `go test ./...` and a final `make badge-cover`

## Progress Tracking
- Mark completed items with `[x]` immediately when done.
- Add newly discovered tasks with `➕` prefix.
- Document blockers or trade-offs with `⚠️` prefix.
- Keep this file synchronized with the actual implementation state and measured coverage.

## What Goes Where
- **Implementation Steps** (`[ ]` checkboxes): Makefile/scripts changes, tests, minimal production-code seams, and badge updates inside this repository.
- **Post-Completion** (no checkboxes): optional CI threshold enforcement and any later contributor-workflow polish.

## Implementation Steps

### Task 1: Add a reproducible repo-wide coverage workflow and badge refresh path

**Files:**
- Modify: `Makefile`
- Modify: `README.md`
- Modify: `docs/badges/coverage.svg`
- Create: `scripts/update-coverage-badge.sh` (if a script is needed to keep badge generation deterministic)

- [x] add a `make cover` target that measures repo-wide coverage with `-coverpkg` while excluding `internal/perftest`
- [x] make `make cover` write a stable cover profile and print the total percentage from `go tool cover -func`
- [x] add a `make badge-cover` target that regenerates `docs/badges/coverage.svg` from the same measured total
- [x] keep the coverage workflow simple and local, using existing repo-local cache conventions from `Makefile`
- [x] update `README.md` badge-refresh instructions to use the new Makefile targets and metric definition
- [x] run the new coverage target once to capture the baseline before test additions

### Task 2: Raise startup and top-level command-path coverage

**Files:**
- Modify: `main_test.go`
- Modify: `main.go` (only if a tiny test seam is needed)

- [x] add tests for `run(...)` branches that are still unexercised, including unsupported argument combinations and settings-loading paths
- [x] add tests for startup option loading and file/stdin resolution helper branches that currently leave `main.go` coverage low
- [x] keep startup tests non-interactive and deterministic by using existing resolver seams where possible
- [x] write tests for both success and error branches touched in this task
- [x] run focused tests and `make cover` before moving to task 3

### Task 3: Raise TUI and session control-flow coverage

**Files:**
- Modify: `internal/tui/model_test.go`
- Modify: `internal/tui/history_test.go`
- Modify: `internal/tui/view_test.go`
- Modify: `internal/tui/settings_test.go`
- Modify: `internal/session/session_test.go`
- Modify: `internal/tui/commands.go` (only if a small injectable seam is needed)
- Modify: `internal/tui/model.go` (only if a small injectable seam is needed)

- [x] add model tests for currently thin branches in prompt submission, busy completion, page movement, and normal-key handling
- [x] add command-flow tests for save/quit, print/quit, copy document, replace selected, jq command handling, and external-edit success/error paths
- [x] add session tests for `ExpandSelected`, reselection fallbacks, move-to-top/bottom edges, current batch selection, and batch-expansion cleanup
- [x] add view/settings tests only where they unlock real untested control flow instead of superficial getter coverage
- [x] keep any production-code seam changes minimal and scoped to deterministic testing
- [x] run focused tests and `make cover` before moving to task 4

### Task 4: Fill remaining document and integration gaps, then refresh the badge

**Files:**
- Modify: `internal/document/document_test.go`
- Modify: `internal/integration/clipboard_test.go`
- Modify: `internal/integration/editor_test.go`
- Modify: `internal/integration/jq_test.go`
- Modify: `README.md`
- Modify: `docs/badges/coverage.svg`
- Modify: `docs/plans/20260327-coverage-to-80-and-badge.md`

- [x] add targeted tests for remaining meaningful branches in document serialization/edit helpers and integration wrappers
- [x] use `go tool cover -func` output to close the last gap only where it improves real behavioral coverage
- [x] rerun `make cover` until the repo-wide total excluding `internal/perftest` reaches at least `80%`
- [x] regenerate `docs/badges/coverage.svg` from the final measured total
- [x] update `README.md` if the final workflow wording or examples need adjustment after implementation
- [x] run `go test ./...` and record the final measured coverage in this plan
- [x] move this plan to `docs/plans/completed/` once implementation and verification are done

## Technical Details
- **coverage metric definition**:
  - tested packages: all repo packages except `github.com/anyroad/lazy-json/internal/perftest`
  - instrumented packages: the same filtered package set passed through `-coverpkg`
  - output: a single cover profile and total percentage from `go tool cover -func`
- **badge update rule**:
  - the badge percentage must come from the same `make cover` measurement
  - do not manually edit the badge text without regenerating it from the measured total
- **test targeting priority**:
  - prioritize low-coverage branches with meaningful user impact over trivial getters or formatting-only helpers
  - prefer scenario tests that cover multiple branches when they remain readable and deterministic
- **scope guardrails**:
  - do not weaken the metric by excluding normal production packages
  - do not add broad refactors just to make testing easier
  - do not change app behavior as part of this coverage pass

## Post-Completion
*Items requiring manual intervention or follow-up - no checkboxes*

**Manual verification**:
- run `make cover` after a clean checkout and confirm the printed total matches the badge percentage
- spot-check the README coverage instructions and badge link rendering on GitHub

**Potential follow-up**:
- if maintaining `>=80%` becomes important for future work, add a CI coverage threshold in a separate change using the same filtered metric
- if badge generation logic grows beyond a tiny script, consider a small contributor tooling section or dedicated script tests

## Scope Notes
- `make cover` now runs `go test ./... -coverpkg=./...`, stores both raw and filtered profiles under `.coverage/`, and filters `internal/perftest` out of the final profile before computing totals.
- `make badge-cover` regenerates `docs/badges/coverage.svg` from the same filtered coverage profile used by `make cover`.
- Additional unit coverage was added in `main`, `internal/config`, `internal/document`, `internal/integration`, `internal/session`, and `internal/tui` without changing application behavior.
- Final measured coverage after filtering `internal/perftest`: `82.7%` of statements.
