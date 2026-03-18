# lazy-json

`lazy-json` is a keyboard-first JSON viewer and editor for the terminal, built with Go and Bubble Tea.

## Current v1 scope

- tree-first navigation with `h/j/k/l`, `gg`, and `G`
- ordered object rendering with syntax highlighting
- collapse and expand for objects and arrays
- substring search with `/`, `n`, and `N`
- structured editing for scalars, object keys, object fields, and array items
- canonical pretty-printed JSON saves
- file-backed and stdin-backed sessions
- optional subtree editing through `$EDITOR`
- optional `jq` transforms through `:jq` and `:jq!`
- switchable color themes

## Usage

Open a file:

```bash
lazy-json data.json
```

Read from stdin:

```bash
cat data.json | lazy-json
```

## Keybindings

### Navigation

- `j` / `k`: move selection
- `h` / `l`: collapse or expand / move to parent or child
- `gg` / `G`: jump to top or bottom
- `?`: toggle help

### Editing

- `e`: edit selected scalar value as JSON
- `E`: edit selected node or subtree in `$EDITOR`
- `a`: add an object field or array item
- `r`: rename the selected object key
- `d`: delete the selected node

### Search and commands

- `/`: search keys, scalar values, and paths
- `n` / `N`: next or previous search hit
- `:`: open command mode
- `:w`: save to the current file
- `:w path.json`: save to a specific path
- `:x`: save and quit, or print to stdout for stdin-backed sessions
- `:print`: print canonical JSON to stdout and quit
- `:q` / `:q!`: quit / force quit
- `:jq EXPR`: apply `jq` to the whole document
- `:jq! EXPR`: apply `jq` to the selected subtree
- `:theme`: switch theme

## Notes

- saves always rewrite the document as canonical pretty JSON
- `jq` is optional; the editor stays usable without it
- `$EDITOR` is optional; `E` reports an error if it is not configured

## Development

Run tests with a writable Go cache:

```bash
GOCACHE=/tmp/lazy-json-gocache GOMODCACHE=/tmp/lazy-json-gomodcache go test ./...
```
