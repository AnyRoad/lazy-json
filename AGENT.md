# Agent Notes

This file is for future Codex sessions working in this repo.

## Project Summary

- Project: `lazy-json`
- Module path: `github.com/anyroad/lazy-json`
- Language: Go 1.25
- App type: terminal JSON viewer/editor built with Bubble Tea
- Primary branch convention: `release`
- Release convention: push a tag matching `v*` to trigger the GitHub release workflow

The editor is intentionally **tree-first and structured**. Do not drift it toward a raw text editor unless the user explicitly asks for that change.

## Repo Layout

- `main.go`
  CLI entrypoint. Loads input, parses JSON, starts the Bubble Tea program, and handles stdout-on-exit behavior.
- `internal/document/`
  Mutable ordered JSON tree.
  Critical invariants:
  - object fields are stored as ordered `[]ObjectEntry`, not maps
  - numbers are stored as validated JSON number strings, not `float64`
  - nodes use stable `NodeID` values for selection and subtree replacement
- `internal/session/`
  Editor/session state separate from the JSON tree.
  Contains row projection, search state, selection, expanded/collapsed state, dirty flag, and source metadata.
- `internal/tui/`
  Bubble Tea model, view rendering, prompt handling, command parsing, theme logic, and keymaps.
- `internal/source/`
  File/stdin loading and atomic file writes.
- `internal/integration/`
  Optional external integrations:
  - `$EDITOR` subtree editing
  - `jq` subtree/document transforms
- `testdata/`
  JSON fixtures used for smoke-style checks and examples.
- `.github/workflows/`
  - `ci.yml`: quality/build checks on pushes and PRs to `release`
  - `release.yml`: GoReleaser-based GitHub release and Homebrew tap publishing on `v*` tags
- `.goreleaser.yml`
  Release packaging config for archives, checksums, GitHub release assets, and the `anyroad/homebrew-apps` formula update

## Core Behavior To Preserve

- File-backed sessions:
  - `lazy-json file.json`
  - `:w` saves back to the original path
- Stdin-backed sessions:
  - `cat file.json | lazy-json`
  - `:w <path>` saves to a new file
  - `:print` prints current JSON to stdout and quits
- All saves rewrite canonical pretty JSON.
- Search currently indexes **visible rows only**.
- Integration failures must be non-destructive.
  - missing `jq`
  - invalid `jq` output
  - missing `$EDITOR`
  - invalid JSON returned from external editor

## Non-Obvious Implementation Detail

For stdin-backed sessions, the app reads the JSON payload from stdin first and then reopens terminal input via Bubble Tea's TTY input option. If you touch startup/input code, do not break this behavior or pipe-first interactive sessions will stop working.

## Commands For Local Work

Preferred repo commands:

- `make fmt`
- `make fmt-check`
- `make vet`
- `make test`
- `make release-check`
- `make release-snapshot`
- `make build`
- `make build-all`
- `make check`
- `make clean`

In restricted/sandboxed environments, use writable Go caches:

```bash
GOCACHE=/tmp/lazy-json-gocache GOMODCACHE=/tmp/lazy-json-gomodcache make check
GOCACHE=/tmp/lazy-json-gocache GOMODCACHE=/tmp/lazy-json-gomodcache make build-all
```

## Testing Expectations

- Run `make check` for normal code changes.
- If you touch build/release surfaces, also run `make build` and the relevant GoReleaser validation targets.
- Prefer unit/model/update tests over fragile terminal automation.
- Keep integration tests stubbed where possible for `jq` and editor behavior.

## Editing Guidance

- Keep document state and UI state separate.
- Preserve deterministic ordering and stable selection behavior after edits.
- When replacing subtrees, keep failure modes safe and explicit.
- Avoid adding hidden formatting-preservation behavior. Current contract is canonical rewrite.
- Keep GoReleaser as the source of truth for release packaging and Homebrew formula generation unless the user explicitly asks to move away from it.

## Documentation And Workflow Notes

- `README.md` already contains:
  - badges
  - quick start
  - user guide
  - developer guide
- Badge and clone URLs currently assume the repo slug `anyroad/lazy-json`.
  If a Git remote is configured later and differs from that slug, update the README URLs.

## Known Pending Work

- Manual interactive smoke testing is still not recorded as completed in the plan file.
- The older plan file in `docs/plans/20260318-json-tui-editor-v1.md` was not moved to `docs/plans/completed/`.

## Recommended First Read For Future Sessions

If you start a non-trivial task, read these files first:

1. `README.md`
2. `AGENT.md`
3. `main.go`
4. `internal/document/`
5. `internal/session/`
6. `internal/tui/model.go`
7. `internal/tui/commands.go`
