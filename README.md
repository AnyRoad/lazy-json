# lazy-json

[![CI](https://github.com/anyroad/lazy-json/actions/workflows/ci.yml/badge.svg?branch=release)](https://github.com/anyroad/lazy-json/actions/workflows/ci.yml?query=branch%3Arelease)
[![Release](https://img.shields.io/github/v/release/anyroad/lazy-json)](https://github.com/anyroad/lazy-json/releases)
[![Go](https://img.shields.io/badge/go-1.25-00ADD8?logo=go)](https://go.dev/dl/)

`lazy-json` is a keyboard-first JSON viewer and editor for the terminal, built with Go and Bubble Tea. It focuses on structured JSON editing instead of raw text editing: move around a tree, expand and collapse nodes, search, edit scalars, add and remove nodes, hand subtrees to your external editor, and optionally run `jq` transforms without leaving the app.

## Features

- tree-first navigation with `h/j/k/l`, `gg`, and `G`
- prefix shortcuts for clipboard actions, structural jumps, and tree-wide expand/collapse
- ordered object rendering with syntax highlighting, built-in themes, and persistent theme settings
- collapse and expand for objects and arrays
- substring search with `/`, `n`, and `N`
- structured editing for scalars, object keys, object fields, and array items
- canonical pretty-printed JSON saves
- file-backed and stdin-backed sessions
- optional subtree editing through `$EDITOR`
- optional `jq` transforms through `:jq` and `:jq!`

## Quick Start

### Install from source

```bash
git clone https://github.com/anyroad/lazy-json.git
cd lazy-json
make build
./dist/lazy-json testdata/basic.json
```

### Build directly with Go

```bash
go build -o dist/lazy-json ./cmd/lazy-json
./dist/lazy-json testdata/basic.json
```

### Open a file

```bash
lazy-json data.json
```

### Open a file with a startup selection

```bash
lazy-json --select '$.items[0].name' data.json
```

### Open JSON from stdin

```bash
cat data.json | lazy-json
```

### Open stdin with a startup selection

```bash
cat data.json | lazy-json --select '$.items[0].name'
```

### Optional tools

- `jq` enables `:jq` and `:jq!` transform commands
- `$EDITOR` enables subtree editing with `E`

## User Guide

### Session types

`lazy-json` starts in one of two modes:

- file-backed: `lazy-json data.json`
- stdin-backed: `cat data.json | lazy-json`

File-backed sessions save back to the original path with `:w`. Stdin-backed sessions do not have a default file target, so use `:w path.json` to save to disk or `:print` to write the current document to stdout.

You can add `--select '$.path.to.node'` to either startup form to open with a specific node selected. The accepted syntax matches the paths shown in the tree, such as `$.items[0].name` and `$["two words"]`. If the full path does not exist, `lazy-json` falls back to the nearest existing ancestor; if only `$` exists, it still opens and shows an error in the footer.

### Navigation

- `j` / `k`: move the selection up or down through visible rows
- `h`: collapse the current container, or move to the parent row
- `l`: expand the current container, or move into the first child
- `gg` / `G`: jump to the first or last visible row
- `]p`: jump to the next parent sibling node, climbing ancestors until a next sibling is found
- `zR` / `zM`: expand all containers / collapse all containers except the root
- `?`: open the built-in help screen
- `t`: quick-preview the next theme for the current session
- `S`: open the theme settings dialog

### Editing model

The editor is structured, not freeform. You operate on the selected node:

- `e`: edit the selected scalar value as JSON, such as `"text"`, `42`, `true`, or `null`
- `a`: add a new field to an object or append a new value to an array
- `r`: rename the selected object key
- `d`: delete the selected node
- `E`: serialize the selected node or subtree into a temp file, open it in `$EDITOR`, and replace the node only if the edited JSON parses successfully

This keeps edits valid and avoids the complexity of embedding a full text editor into the TUI.

### Clipboard

- `yp`: copy the selected JSON path
- `yk`: copy the selected object key
- `yv`: copy the selected value as compact JSON
- `ys`: copy the selected subtree as pretty JSON
- `yj`: copy the whole document as pretty JSON

All clipboard copies use structured JSON output rather than the rendered screen text. `yv` preserves valid JSON scalars and compact containers, while `ys` and `yj` use the same canonical pretty formatting as saves.

### Search

- `/`: open search
- `n`: jump to the next match
- `N`: jump to the previous match

Search matches visible rows based on keys, scalar values, and rendered JSON paths. If a subtree is collapsed, rows hidden inside that subtree are not searchable until expanded.

### Commands

- `:w`: save to the current file path
- `:w path.json`: save to a specific path
- `:x`: save and quit for file-backed sessions; for stdin-backed sessions with no file path, print to stdout and quit
- `:print`: print canonical JSON to stdout and quit
- `:q`: quit if there are no unsaved changes
- `:q!`: quit without saving
- `:theme`: quick-preview the next theme without saving
- `:settings`: open the theme settings dialog
- `:copy-path`: copy the selected JSON path
- `:copy-key`: copy the selected object key
- `:copy-value`: copy the selected value as compact JSON
- `:copy-subtree`: copy the selected subtree as pretty JSON
- `:copy-json`: copy the whole document as pretty JSON
- `:expand-all`: expand every object and array in the document
- `:collapse-all`: collapse every container except the root
- `:next-parent-sibling`: jump to the next sibling of the selected node's parent, climbing ancestors as needed
- `:edit-external`: same behavior as `E`
- `:jq EXPR`: apply a `jq` expression to the whole document
- `:jq! EXPR`: apply a `jq` expression to the selected subtree

### Theme Settings

Press `S` or run `:settings` to open the theme settings dialog. The dialog lists built-in themes first and then valid external themes discovered from your config directory. Inside the dialog:

- `h` / `left`: preview the previous theme
- `l` / `right`: preview the next theme
- `s`: save the current preview to `settings.json`
- `esc`: close the dialog without writing to disk

Theme previews apply immediately to the current session. They are not persisted until you press `s` in the settings dialog, so the quick `t` / `:theme` shortcuts remain preview-only switches. A theme saved from the dialog is restored automatically on the next launch.

`lazy-json` stores its saved theme under `os.UserConfigDir()/lazy-json/settings.json` and discovers external themes from `os.UserConfigDir()/lazy-json/themes/*.json`. The exact base directory follows `os.UserConfigDir()` for your platform; for example, on Linux this is typically `~/.config/lazy-json/settings.json` and `~/.config/lazy-json/themes/`.

External theme files are JSON objects with a required `name` plus optional style slots such as `key`, `string`, `number`, `bool`, `null`, `muted`, `selected`, `search_hit`, `status`, `error`, `border`, `help`, and `prompt`. Each slot supports `foreground`, optional `background`, and optional `bold`. For example:

```json
{
  "name": "mist",
  "key": {
    "foreground": "#112233"
  },
  "selected": {
    "background": "#ddeeff",
    "bold": true
  }
}
```

### Examples

Edit a file and save it back:

```bash
lazy-json config.json
```

Pipe JSON in, modify it, then print the result:

```bash
cat config.json | lazy-json
```

Transform a whole document with `jq` inside the editor:

```text
:jq .items |= map(select(.enabled == true))
```

Transform just the selected subtree:

```text
:jq! .version = "2"
```

### Save behavior

All saves rewrite the current document as canonical pretty JSON. The tool does not preserve the original whitespace layout.

## Developer Guide

### Make targets

- `make fmt`: rewrite Go files with `gofmt`
- `make fmt-check`: fail if formatting is not clean
- `make vet`: run `go vet ./...`
- `make test`: run `go test ./...`
- `make build`: build `dist/lazy-json` for the current platform
- `make build-all`: cross-compile release binaries for the supported target set
- `make check`: run format check, vet, and tests
- `make clean`: remove `dist/`

### Local workflow

Recommended local check before pushing:

```bash
make check
make build
```

If you need writable Go cache directories in a restricted environment:

```bash
GOCACHE=/tmp/lazy-json-gocache GOMODCACHE=/tmp/lazy-json-gomodcache make check
```

### GitHub Actions

- `CI`: runs on pushes and pull requests targeting the `release` branch, and executes `make check` plus `make build`
- `Release`: runs when a tag matching `v*` is pushed, cross-compiles release archives for Linux, macOS, and Windows, generates checksums, and uploads them to GitHub Releases

Create a release tag:

```bash
git tag v0.1.0
git push origin v0.1.0
```

## Current Limitations

- saves always rewrite canonical JSON formatting
- search only indexes visible rows
- `jq` is optional; commands fail cleanly when it is missing
- `$EDITOR` is optional; external edit fails cleanly when it is not configured
