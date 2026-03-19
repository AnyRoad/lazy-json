package tui

import (
	"strings"
	"testing"

	"github.com/anyroad/lazy-json/internal/config"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
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
	if !strings.Contains(view, "S settings") {
		t.Fatalf("View() missing settings hint: %q", view)
	}
}

func TestViewUsesRegistryThemeForRendering(t *testing.T) {
	profile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() {
		lipgloss.SetColorProfile(profile)
	})

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
	m.Width = 120
	m.Height = 20
	m.Session.SelectedID = m.Doc.Root.Object[0].Value.ID

	mistView := m.View()
	if want := registry.ThemeByName("mist").Key.Render("name"); !strings.Contains(mistView, want) {
		t.Fatalf("View() = %q, want mist key styling", mistView)
	}

	m.Session.ThemeName = config.DefaultThemeName
	forestView := m.View()

	if mistView == forestView {
		t.Fatalf("View() did not change after theme switch: %q", mistView)
	}
	if strings.Contains(forestView, registry.ThemeByName("mist").Key.Render("name")) {
		t.Fatalf("View() = %q, unexpectedly contains mist key styling", forestView)
	}
}

func TestViewShowsSettingsOverlayHints(t *testing.T) {
	m := testModelWithOptions(t, ModelOptions{
		Settings: config.Settings{Theme: config.DefaultThemeName},
	})
	m.Width = 120
	m.Height = 20
	m.openSettings()

	view := m.View()
	if !strings.Contains(view, "Theme Settings") {
		t.Fatalf("View() = %q, want settings title", view)
	}
	if !strings.Contains(view, "preview only") && !strings.Contains(view, "saved") {
		t.Fatalf("View() = %q, want settings state hint", view)
	}
	if !strings.Contains(view, "h/l preview  s save  esc close") {
		t.Fatalf("View() = %q, want settings footer hint", view)
	}
}
