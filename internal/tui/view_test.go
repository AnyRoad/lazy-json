package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/anyroad/lazy-json/internal/config"
)

func TestViewContainsKeyAndFooter(t *testing.T) {
	m := testModel(t)
	m.Width = 120
	m.Height = 20
	view := m.View()
	if !strings.Contains(view, "name") {
		t.Fatalf("View() missing key: %q", view)
	}
	if !strings.Contains(view, "sample.json") {
		t.Fatalf("View() missing footer source: %q", view)
	}
}

func TestThemeUsesRegistryThemeForRendering(t *testing.T) {
	registry, warnings := NewThemeRegistry([]config.DiscoveredTheme{
		{
			Path: "10-mist.json",
			Spec: config.ThemeSpec{
				Name: "mist",
				Key: &config.StyleSpec{
					Foreground: "#112233",
				},
			},
		},
	})
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}

	m := testModelWithOptions(t, ModelOptions{
		ThemeRegistry: registry,
		Settings:      config.Settings{Theme: "mist"},
	})

	if got, want := m.theme().Name, "mist"; got != want {
		t.Fatalf("theme().Name = %q, want %q", got, want)
	}
	if got, want := m.theme().Key.GetForeground(), lipgloss.Color("#112233"); got != want {
		t.Fatalf("theme().Key.GetForeground() = %#v, want %#v", got, want)
	}
}
