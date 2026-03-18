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
}
