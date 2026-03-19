package tui

import (
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
