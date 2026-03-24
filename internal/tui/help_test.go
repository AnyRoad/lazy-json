package tui

import (
	"os"
	"strings"
	"testing"
)

func TestHelpView(t *testing.T) {
	m := testModel(t)
	m.Width = 120
	m.Height = 80
	help := m.helpView()
	if !strings.Contains(help, "Navigation") {
		t.Fatalf("helpView() = %q", help)
	}
	if !strings.Contains(help, "t         quick preview next theme") {
		t.Fatalf("helpView() = %q, want quick preview shortcut", help)
	}
	if !strings.Contains(help, "S         open settings dialog") {
		t.Fatalf("helpView() = %q, want settings shortcut", help)
	}
	if !strings.Contains(help, "]p        next parent sibling") {
		t.Fatalf("helpView() = %q, want parent sibling shortcut", help)
	}
	if !strings.Contains(help, "zR / zM   expand all / collapse all") {
		t.Fatalf("helpView() = %q, want expand/collapse all shortcuts", help)
	}
	if !strings.Contains(help, "za        expand array elements one level") {
		t.Fatalf("helpView() = %q, want array expansion shortcut", help)
	}
	if !strings.Contains(help, "zA        collapse array elements") {
		t.Fatalf("helpView() = %q, want array collapse shortcut", help)
	}
	if !strings.Contains(help, "yp        copy JSON path") {
		t.Fatalf("helpView() = %q, want copy path shortcut", help)
	}
	if !strings.Contains(help, "yj        copy whole document") {
		t.Fatalf("helpView() = %q, want copy whole document shortcut", help)
	}
	if !strings.Contains(help, ":theme    quick preview next theme") {
		t.Fatalf("helpView() = %q, want quick preview command", help)
	}
	if !strings.Contains(help, ":copy-*   copy path/key/value/subtree/document") {
		t.Fatalf("helpView() = %q, want copy command family", help)
	}
	if !strings.Contains(help, ":expand-all / :collapse-all") {
		t.Fatalf("helpView() = %q, want expand/collapse commands", help)
	}
	if !strings.Contains(help, ":next-parent-sibling") {
		t.Fatalf("helpView() = %q, want next-parent-sibling command", help)
	}
	if !strings.Contains(help, "s         save settings to settings.json") {
		t.Fatalf("helpView() = %q, want explicit save help", help)
	}
	if !strings.Contains(help, "persist   saved settings restore on next launch") {
		t.Fatalf("helpView() = %q, want persisted theme help", help)
	}
	if !strings.Contains(help, "themes    built-ins + config themes/*.json") {
		t.Fatalf("helpView() = %q, want external theme help", help)
	}
	if !strings.Contains(help, "lines     global tree row-number gutter can be toggled") {
		t.Fatalf("helpView() = %q, want line-number settings help", help)
	}
	if !strings.Contains(help, "paths     JSON path display can be toggled") {
		t.Fatalf("helpView() = %q, want JSON path settings help", help)
	}
	if !strings.Contains(help, ":settings open settings dialog") {
		t.Fatalf("helpView() = %q, want settings command", help)
	}
	if !strings.Contains(help, ":select-path P  select node by JSON path") {
		t.Fatalf("helpView() = %q, want select-path command", help)
	}
	if !strings.Contains(help, "wrap      long strings setting affects display only") {
		t.Fatalf("helpView() = %q, want wrap settings help", help)
	}
	if !strings.Contains(help, "indent    pretty save/print/copy uses selected indent") {
		t.Fatalf("helpView() = %q, want indent settings help", help)
	}
	if strings.Contains(help, "theme picker") {
		t.Fatalf("helpView() = %q, unexpectedly mentions theme picker", help)
	}
	if !strings.Contains(help, "prefixes  footer shows next-key menu") {
		t.Fatalf("helpView() = %q, want prefix footer help", help)
	}
}

func TestReadmeDocumentsThemeSettingsBindingsAndPaths(t *testing.T) {
	data, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatalf("ReadFile(README.md) error = %v", err)
	}

	readme := string(data)
	snippets := []string{
		"- `t`: quick-preview the next theme for the current session",
		"- `S`: open the settings dialog",
		"### Open a file with a startup selection",
		"lazy-json --select '$.items[0].name' data.json",
		"### Open stdin with a startup selection",
		"cat data.json | lazy-json --select '$.items[0].name'",
		"You can add `--select '$.path.to.node'` to either startup form to open with a specific node selected.",
		"If the full path does not exist, `lazy-json` falls back to the nearest existing ancestor; if only `$` exists, it still opens and shows an error in the footer.",
		"- `]p`: jump to the next parent sibling node, climbing ancestors until a next sibling is found",
		"- `zR` / `zM`: expand all containers / collapse all containers except the root",
		"- `za`: expand the selected array, or nearest array ancestor, so each container element opens one level",
		"- `zA`: collapse the selected array, or nearest array ancestor, so all element containers close",
		"- `yp`: copy the selected JSON path",
		"- `yk`: copy the selected object key",
		"- `yv`: copy the selected value as compact JSON",
		"- `ys`: copy the selected subtree as pretty JSON",
		"- `yj`: copy the whole document as pretty JSON",
		"- `:theme`: quick-preview the next theme without saving",
		"- `:settings`: open the settings dialog",
		"- `:select-path $.items[0].name`: select a node by JSON path",
		"- `:copy-path`: copy the selected JSON path",
		"- `:copy-json`: copy the whole document as pretty JSON",
		"- `:expand-all`: expand every object and array in the document",
		"- `:next-parent-sibling`: jump to the next sibling of the selected node's parent, climbing ancestors as needed",
		"- `s`: save the current settings to `settings.json`",
		"`lazy-json` stores theme, line-number visibility, JSON-path visibility, wrapping, and save-indent settings under `os.UserConfigDir()/lazy-json/settings.json` and discovers external themes from `os.UserConfigDir()/lazy-json/themes/*.json`.",
		"Theme, line-number visibility, JSON-path visibility, wrap, and save-indent changes are not persisted until you press `s` in the settings dialog, so the quick `t` / `:theme` shortcuts remain preview-only switches.",
		"- `left` / `right` or `h` / `l`: change the selected setting",
		"- `up` / `down` or `j` / `k`: move between settings rows",
		"- `enter` / `space`: cycle the selected setting",
		"- `Line numbers`: toggle a global tree row-number gutter; collapsed rows still count, so visible numbering may have gaps",
		"- `JSON path`: toggle whether rendered rows show the selected node path",
		"- `Long strings`: toggle wrapping for displayed string scalar values only",
		"- `Save indent`: choose `spaces:2`, `spaces:3`, `spaces:4`, or `tabs` for pretty JSON output",
	}

	for _, snippet := range snippets {
		if !strings.Contains(readme, snippet) {
			t.Fatalf("README.md missing %q", snippet)
		}
	}
}
