package tui

import (
	"os"
	"strings"
	"testing"
)

func TestHelpView(t *testing.T) {
	m := testModel(t)
	help := m.helpView()
	if !strings.Contains(help, "Navigation") {
		t.Fatalf("helpView() = %q", help)
	}
	if !strings.Contains(help, "t         quick preview next theme") {
		t.Fatalf("helpView() = %q, want quick preview shortcut", help)
	}
	if !strings.Contains(help, "S         open theme settings dialog") {
		t.Fatalf("helpView() = %q, want settings shortcut", help)
	}
	if !strings.Contains(help, ":theme    quick preview next theme") {
		t.Fatalf("helpView() = %q, want quick preview command", help)
	}
	if !strings.Contains(help, "s         save preview to settings.json") {
		t.Fatalf("helpView() = %q, want explicit save help", help)
	}
	if !strings.Contains(help, "persist   saved theme restores on next launch") {
		t.Fatalf("helpView() = %q, want persisted theme help", help)
	}
	if !strings.Contains(help, "themes    built-ins + config themes/*.json") {
		t.Fatalf("helpView() = %q, want external theme help", help)
	}
	if !strings.Contains(help, ":settings open theme settings dialog") {
		t.Fatalf("helpView() = %q, want settings command", help)
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
		"- `S`: open the theme settings dialog",
		"- `:theme`: quick-preview the next theme without saving",
		"- `:settings`: open the theme settings dialog",
		"- `s`: save the current preview to `settings.json`",
		"`lazy-json` stores its saved theme under `os.UserConfigDir()/lazy-json/settings.json` and discovers external themes from `os.UserConfigDir()/lazy-json/themes/*.json`.",
		"They are not persisted until you press `s` in the settings dialog, so the quick `t` / `:theme` shortcuts remain preview-only switches.",
	}

	for _, snippet := range snippets {
		if !strings.Contains(readme, snippet) {
			t.Fatalf("README.md missing %q", snippet)
		}
	}
}
