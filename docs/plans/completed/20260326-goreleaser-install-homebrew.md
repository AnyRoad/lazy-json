# Go Install And Homebrew Releases With GoReleaser

## Overview
- make `lazy-json` installable with `go install github.com/anyroad/lazy-json@latest`
- replace the hand-rolled release packaging path with GoReleaser while keeping GitHub Actions as the tag-triggered release entrypoint
- publish a Homebrew formula to `anyroad/homebrew-apps` so users can install with `brew install anyroad/apps/lazy-json`
- preserve current release functionality: cross-platform archives, checksums, and GitHub release uploads

## Context (from discovery)
- files/components involved:
  - `cmd/lazy-json/main.go`
  - `cmd/lazy-json/main_test.go`
  - `Makefile`
  - `README.md`
  - `.github/workflows/release.yml`
  - `AGENT.md`
- related patterns found:
  - the current executable lives under `./cmd/lazy-json`, so the repo root is not directly installable with `go install ...@latest`
  - release packaging is currently implemented by a custom GitHub Actions matrix in `.github/workflows/release.yml`
  - `Makefile` build targets and Go file discovery are hardcoded to `./cmd/lazy-json` and `cmd internal`
  - README already documents manual build and GitHub release behavior, but not `go install` or Homebrew
- dependencies identified:
  - GoReleaser config and binary are new release-time dependencies
  - GitHub Actions needs a tap publishing token for `anyroad/homebrew-apps`
  - brew formula smoke testing needs a stable non-interactive CLI path

## Development Approach
- **testing approach**: Regular (code first, then tests)
- complete each task fully before moving to the next
- make small, focused changes
- **CRITICAL: every task MUST include new/updated tests** for code changes in that task
  - tests are not optional - they are a required part of the checklist
  - write unit tests for new functions/methods
  - write unit tests for modified functions/methods
  - add new test cases for new code paths
  - update existing test cases if behavior changes
  - tests cover both success and error scenarios
- **CRITICAL: all tests must pass before starting next task** - no exceptions
- **CRITICAL: update this plan file when scope changes during implementation**
- run tests after each change
- maintain backward compatibility for existing runtime behavior and release artifact naming where practical

## Testing Strategy
- **unit tests**:
  - move and update existing startup tests so the root package entrypoint keeps current CLI behavior
  - add tests for new non-interactive CLI metadata flags used by packaging, if introduced
- **build/release validation**:
  - run `go test ./...`
  - run `make build`
  - run `goreleaser check`
  - run `goreleaser release --snapshot --clean`
- **workflow validation**:
  - keep release publishing logic simple enough that local snapshot validation covers config correctness
  - do not attempt live tap publishing from local development

## Progress Tracking
- mark completed items with `[x]` immediately when done
- add newly discovered tasks with ➕ prefix
- document issues/blockers with ⚠️ prefix
- update plan if implementation deviates from original scope
- keep plan in sync with actual work done

## What Goes Where
- **Implementation Steps** (`[ ]` checkboxes): code/config/doc changes that can be completed in this repo
- **Post-Completion** (no checkboxes): manual follow-up and external repo setup

## Implementation Steps

### Task 1: Move the CLI entrypoint to the repo root

**Files:**
- Create: `main.go`
- Create: `main_test.go`
- Delete: `cmd/lazy-json/main.go`
- Delete: `cmd/lazy-json/main_test.go`

- [x] move the current executable entrypoint from `cmd/lazy-json/main.go` to root `main.go` so `go install github.com/anyroad/lazy-json@latest` works
- [x] keep existing startup behavior intact for file input, stdin input, `--select`, alt-screen launch, and stdout-on-exit flows
- [x] add a stable non-interactive CLI metadata path such as `--help` and/or `--version` for packaging smoke tests if the current CLI does not provide one
- [x] move and update startup tests into `main_test.go`, including new coverage for any added metadata flag behavior
- [x] run `go test ./...` and confirm the root package builds correctly before changing release surfaces

### Task 2: Update local build and developer workflow for the root package

**Files:**
- Modify: `Makefile`
- Modify: `README.md`
- Modify: `AGENT.md`

- [x] update `Makefile` targets to build from `.` instead of `./cmd/lazy-json`
- [x] update Go file discovery in `Makefile` so format targets include the new root `main.go` and `main_test.go`
- [x] add a local GoReleaser validation target such as `make release-snapshot` for `goreleaser release --snapshot --clean`
- [x] update developer and installation documentation to prefer `go install github.com/anyroad/lazy-json@latest`
- [x] run `make build` and `go test ./...` to verify the local workflow still works after the path migration

### Task 3: Add GoReleaser as the packaging source of truth

**Files:**
- Create: `.goreleaser.yml`
- Modify: `README.md`

- [x] define a GoReleaser build that packages the root command as `lazy-json` for the existing release target set: Linux amd64/arm64, macOS amd64/arm64, and Windows amd64
- [x] preserve current archive naming and checksum generation closely enough that release artifacts remain recognizable on the GitHub release page
- [x] add GoReleaser metadata for GitHub release uploads and the Homebrew formula targeting `anyroad/homebrew-apps`
- [x] configure the Homebrew formula to install the prebuilt binary and use a stable non-interactive smoke test command
- [x] run `goreleaser check` and `goreleaser release --snapshot --clean`, then inspect `dist/` output for expected archives and formula metadata

### Task 4: Replace the manual release matrix with a hybrid GoReleaser workflow

**Files:**
- Modify: `.github/workflows/release.yml`
- Modify: `README.md`

- [x] replace the hand-written archive matrix in `.github/workflows/release.yml` with a GoReleaser-driven release job triggered by `v*` tags
- [x] keep GitHub release artifact uploads enabled through GoReleaser instead of custom `gh release upload` scripting
- [x] wire tap publishing through a dedicated token environment variable for `anyroad/homebrew-apps` and document the required secret name in repo docs
- [x] keep the workflow minimal: checkout, setup Go, run GoReleaser, and rely on GoReleaser for archives, checksums, release assets, and brew formula updates
- [x] rerun `goreleaser check`, `goreleaser release --snapshot --clean`, and `go test ./...` after the workflow/config changes

### Task 5: Verify acceptance criteria

**Files:**
- Modify: `README.md`

- [x] verify `go install github.com/anyroad/lazy-json@latest` is the documented canonical source install path
- [x] verify README documents `brew install anyroad/apps/lazy-json`
- [x] verify the release config still produces cross-platform archives, checksums, GitHub release uploads, and Homebrew formula updates
- [x] run full test suite: `go test ./...`
- [x] run build/release validation: `make build` and `goreleaser release --snapshot --clean`

### Task 6: [Final] Update documentation and workflow notes

**Files:**
- Modify: `README.md`
- Modify: `AGENT.md`
- Modify: `docs/plans/20260326-goreleaser-install-homebrew.md`

- [x] update README release instructions and install sections to match the final GoReleaser-based flow
- [x] update AGENT.md release guidance if the preferred release procedure changes from manual packaging to GoReleaser
- [x] confirm any new token names, tap repo assumptions, or release commands are documented clearly
- [x] move this plan to `docs/plans/completed/`
- [x] rerun the final verification commands before closing the task

## Technical Details
- root entrypoint:
  - the repo root becomes the only `main` package so `go install github.com/anyroad/lazy-json@latest` works without a `/cmd/...` suffix
  - startup behavior must stay identical to the current command path, including stdin-backed TTY reopening
- CLI metadata:
  - add a stable non-interactive command path for packaging tests if needed, preferably `--help` and/or `--version`
  - if version output is added, wire it through GoReleaser ldflags so release binaries report the tagged version
- GoReleaser:
  - build from `main.go` at the repo root
  - keep release targets aligned with the current workflow
  - generate archives plus a checksum file
  - publish GitHub release assets
  - publish a Homebrew formula to `anyroad/homebrew-apps`
- Homebrew:
  - formula name: `lazy-json`
  - install command: `brew install anyroad/apps/lazy-json`
  - tap publishing should use a dedicated token environment variable, not the default `GITHUB_TOKEN`, because the tap repo is separate
- local workflow:
  - keep `make build` as the standard local binary build
  - add a repeatable snapshot release command for validating `.goreleaser.yml` without publishing

## Post-Completion
*Items requiring manual intervention or external systems - no checkboxes, informational only*

**Manual verification**:
- install from source with `go install github.com/anyroad/lazy-json@latest` on a clean machine or fresh Go bin path
- install from Homebrew with `brew install anyroad/apps/lazy-json` after the first tagged release
- smoke-test the installed binary on macOS and Linux outside the repo checkout

**External system updates**:
- create or confirm a repository secret for tap publishing, such as `HOMEBREW_TAP_GITHUB_TOKEN`
- grant that token write access to `anyroad/homebrew-apps`
- confirm the tap repo accepts GoReleaser formula updates on the target branch
