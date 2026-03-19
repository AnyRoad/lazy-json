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
	if !strings.Contains(help, "S         open theme settings dialog") {
		t.Fatalf("helpView() = %q, want settings shortcut", help)
	}
	if !strings.Contains(help, ":settings open theme settings dialog") {
		t.Fatalf("helpView() = %q, want settings command", help)
	}
}
