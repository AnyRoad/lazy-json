# Theme Settings Dialog

## Overview
- Add a persistent global settings layer for `lazy-json` using the user config directory instead of session-only theme state.
- Expand the theme system with more built-in palettes, including at least one light theme, and support auto-discovered external themes from `~/.config/lazy-json/themes/*.json`.
- Add a modal settings dialog for theme selection with live preview and explicit save, while keeping the existing fast theme-cycle shortcut.
- Keep startup and runtime failures non-fatal: invalid config or theme files should fall back to defaults and surface actionable warnings instead of blocking the editor.

## Context (from discovery)
- files/components involved: [`cmd/lazy-json/main.go`](/Users/andrei/PROJECTS/lazy-json/cmd/lazy-json/main.go), [`internal/tui/theme.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/theme.go), [`internal/tui/model.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/model.go), [`internal/tui/view.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/view.go), [`internal/tui/commands.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/commands.go), [`internal/tui/keymap.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/keymap.go), [`internal/session/session.go`](/Users/andrei/PROJECTS/lazy-json/internal/session/session.go), and [`README.md`](/Users/andrei/PROJECTS/lazy-json/README.md).
- related patterns found: themes are currently two hard-coded palettes in [`internal/tui/theme.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/theme.go), runtime switching is handled by `t` and `:theme`, and the selected theme is stored only as `Session.ThemeName`.
- related UI patterns found: overlays are currently limited to the help screen, while inline data entry uses prompt state in [`internal/tui/prompt.go`](/Users/andrei/PROJECTS/lazy-json/internal/tui/prompt.go).
- dependencies identified: Go standard library config/path APIs, JSON encoding/decoding, Bubble Tea update/view flow, and Lip Gloss style construction.

## Development Approach
- **testing approach**: Regular (code first, then tests in each task)
- complete each task fully before moving to the next
- make small, focused changes and avoid introducing a generic settings framework before there are multiple concrete settings
- **CRITICAL: every task MUST include new or updated tests** for code changed in that task
- tests are not optional; they are a required deliverable of every task
- include both success and error scenarios in tests
- **CRITICAL: all tests must pass before starting the next task**
- keep the theme registry deterministic across built-in and external themes
- keep config and theme-load failures non-fatal so the editor always starts on a valid built-in theme
- update this plan if implementation scope changes during execution

## Testing Strategy
- **unit tests**: required for every task, using `go test ./...`
- **config tests**: use temp directories and injected config paths so settings load/save and theme discovery do not touch the real user environment
- **model/update tests**: exercise modal open/close, key handling, command handling, and save behavior through Bubble Tea update logic
- **render tests**: assert settings modal content, footer messaging, and selected theme effects via string-based view tests
- **manual verification**: confirm light/dark theme preview, explicit save semantics, missing-theme fallback, and external theme discovery in a real terminal session
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

### Task 1: Add persistent settings and external theme discovery

**Files:**
- Create: `internal/config/settings.go`
- Create: `internal/config/settings_test.go`
- Create: `internal/config/themes.go`
- Create: `internal/config/themes_test.go`

- [x] implement config directory resolution and `settings.json` load/save helpers with default values
- [x] define serializable settings and theme-spec structures for persisted theme selection and external theme files
- [x] implement `themes/*.json` discovery with deterministic ordering and warning collection for invalid or duplicate themes
- [x] write tests for missing settings, default fallback, and successful settings load/save
- [x] write tests for invalid theme JSON, duplicate names, partial theme specs, and deterministic discovery order
- [x] run tests: `go test ./...`

### Task 2: Refactor theme handling into a registry and add more built-ins

**Files:**
- Modify: `internal/tui/theme.go`
- Create: `internal/tui/theme_test.go`

- [x] replace the package-global theme slice with registry APIs for listing, lookup, and next-theme cycling
- [x] add additional built-in palettes, including at least one light theme, while preserving the existing built-ins
- [x] convert discovered theme specs into runtime `Theme` values with slot-level fallback to default styles
- [x] write tests for registry lookup, missing-theme fallback, and next-theme cycling across built-in and external themes
- [x] write tests for partial external theme conversion and stable theme ordering
- [x] run tests: `go test ./...`

### Task 3: Load settings at startup and wire registry-backed theme state into the app

**Files:**
- Create: `cmd/lazy-json/main_test.go`
- Modify: `cmd/lazy-json/main.go`
- Modify: `internal/session/session.go`
- Modify: `internal/tui/model.go`
- Modify: `internal/tui/model_test.go`

- [ ] load settings and discovered themes before constructing the Bubble Tea model
- [ ] initialize the active session theme from persisted settings and fall back to the default built-in theme when the configured theme is unavailable
- [ ] inject the theme registry and mutable settings state into the model while keeping `Session.ThemeName` as the active render choice
- [ ] write tests for persisted-theme startup, missing-theme fallback, and non-fatal warning propagation
- [ ] write tests for registry-backed `t` theme cycling and unchanged startup behavior when no config exists
- [ ] run tests: `go test ./...`

### Task 4: Build the settings modal and explicit save flow

**Files:**
- Create: `internal/tui/settings.go`
- Create: `internal/tui/settings_test.go`
- Modify: `internal/tui/model.go`
- Modify: `internal/tui/view.go`
- Modify: `internal/tui/commands.go`
- Modify: `internal/tui/keymap.go`

- [ ] add modal state and rendering for a compact settings overlay that sits above the existing document view
- [ ] implement `S` and `:settings` to open the modal, with modal-local navigation for the theme row
- [ ] preview theme changes immediately in the current session while keeping persistence explicit through a save action
- [ ] implement modal save behavior that writes `settings.json`, keeps the modal open on failure, and reports concrete status/error messages
- [ ] write tests for modal open/close, theme preview, successful save, and `:settings` command entry
- [ ] write tests for save failures, escape behavior without persistence, and modal key handling edge cases
- [ ] run tests: `go test ./...`

### Task 5: Update help text, footer hints, and user documentation

**Files:**
- Modify: `README.md`
- Modify: `internal/tui/keymap.go`
- Modify: `internal/tui/help_test.go`
- Modify: `internal/tui/view_test.go`

- [ ] update help text and footer hints to advertise the settings dialog, theme persistence, and external theme support
- [ ] document the settings file path, external theme directory, theme JSON format, and explicit-save behavior in `README.md`
- [ ] document the retained quick-switch shortcut so runtime cycling and persisted settings are both discoverable
- [ ] write tests for updated help output and settings-related footer/view messaging
- [ ] write tests for render hints covering modal visibility and persisted-theme messaging
- [ ] run tests: `go test ./...`

### Task 6: Verify acceptance criteria

**Files:**
- Modify: `docs/plans/20260319-theme-settings-dialog.md`

- [ ] verify that built-in and external themes are listed deterministically and can be previewed from the settings modal
- [ ] verify that saving from the modal persists the selected theme across app restarts using a temp config directory
- [ ] run full test suite: `go test ./...`
- [ ] perform a manual terminal smoke test for dark/light themes, invalid config/theme files, and modal save/error flows
- [ ] verify that help text and README instructions match the final bindings and file locations

### Task 7: [Final] Update documentation and archive the plan

**Files:**
- Modify: `README.md`
- Modify: `docs/plans/20260319-theme-settings-dialog.md`
- Create: `docs/plans/completed/`

- [ ] update `README.md` if verification changes wording, examples, or file-path guidance
- [ ] update this plan file with final scope notes, warnings, and any follow-up items
- [ ] move this plan to `docs/plans/completed/`

## Technical Details
- **Settings path**: store global settings in `os.UserConfigDir()/lazy-json/settings.json`, with the external theme directory at `os.UserConfigDir()/lazy-json/themes/`.
- **Settings shape**: start with a real struct containing a persisted theme field rather than a bare string so future settings can be added without redesigning the config layer.
- **Theme file format**: each external theme file is JSON with a `name` plus style slots such as `key`, `string`, `number`, `bool`, `null`, `muted`, `selected`, `search_hit`, `status`, `error`, `border`, `help`, and `prompt`. Each slot supports `foreground`, optional `background`, and optional `bold`.
- **Theme fallback**: external theme files may omit slots; missing slots inherit from a built-in default style set so partial themes remain valid.
- **Theme registry order**: register built-ins first, then append discovered external themes sorted by filename to keep listing and cycling deterministic.
- **Duplicate and invalid themes**: skip malformed files or duplicate names, record warnings, and continue startup with the remaining valid themes.
- **Runtime state**: keep `Session.ThemeName` as the active render state for the current session, but back it with a registry and a mutable settings object owned by the TUI model.
- **Settings modal behavior**: opening the modal shows a compact overlay with a single `Theme` row initially; changing the row previews immediately, `s` persists to disk, and `esc` closes without writing. The session keeps the previewed theme until exit even if the user closes without saving.
- **User entry points**: preserve `t` for fast next-theme cycling, add `S` for the settings modal, and add `:settings` as the command-mode entry point.
- **Error handling**: config load failures, theme parse failures, and missing configured themes should never abort startup; instead they should fall back to a safe built-in theme and surface an actionable warning in the UI.

## Post-Completion
*Items requiring manual intervention or external systems - no checkboxes, informational only*

**Manual verification**:
- create or edit a custom theme file under a temp config directory and confirm it appears in the settings modal without restarting the terminal session unexpectedly
- preview several built-in themes, including the light palette, and confirm selection/search/footer styling remain readable
- save a theme, restart `lazy-json`, and confirm the saved theme is restored
- corrupt `settings.json` and a theme file separately, then verify the app still starts on the default built-in theme with a clear warning

**External system updates**:
- consider publishing one or two example external theme files in the repository if user themes become a documented extension point
- revisit whether settings should gain sections or additional categories after the second or third real setting is added
