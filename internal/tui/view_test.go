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
	if !strings.Contains(view, "S open settings  t quick preview  ? help") {
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
	paths := config.PathsFromUserConfigDir(t.TempDir())
	m := testModelWithOptions(t, ModelOptions{
		Settings:          config.Settings{Theme: config.DefaultThemeName},
		SettingsPath:      paths.SettingsFile,
		SettingsPersisted: true,
	})
	m.Width = 120
	m.Height = 20
	m.openSettings()

	initialView := m.View()
	if !strings.Contains(initialView, "Theme Settings") {
		t.Fatalf("View() = %q, want settings title", initialView)
	}
	if !strings.Contains(initialView, "saved") {
		t.Fatalf("View() = %q, want saved state hint", initialView)
	}
	if !strings.Contains(initialView, "Saved theme: "+config.DefaultThemeName) {
		t.Fatalf("View() = %q, want saved theme label", initialView)
	}
	if !strings.Contains(initialView, "h/l preview  s save settings.json  esc close") {
		t.Fatalf("View() = %q, want settings footer hint", initialView)
	}
	if !strings.Contains(initialView, "Save writes settings.json on demand.") {
		t.Fatalf("View() = %q, want explicit save hint", initialView)
	}
	if !strings.Contains(initialView, "Built-ins + config themes/*.json appear here.") {
		t.Fatalf("View() = %q, want external theme hint", initialView)
	}

	m.previewSettingsTheme(1)
	previewView := m.View()
	if !strings.Contains(previewView, "preview only") {
		t.Fatalf("View() = %q, want preview-only state hint", previewView)
	}
	if !strings.Contains(previewView, "Saved theme: "+config.DefaultThemeName) {
		t.Fatalf("View() = %q, want original saved theme label during preview", previewView)
	}

	m.saveThemeSettings()
	savedView := m.View()
	if strings.Contains(savedView, "preview only") {
		t.Fatalf("View() = %q, want preview-only hint cleared after save", savedView)
	}
	if !strings.Contains(savedView, "Saved theme: harbor") {
		t.Fatalf("View() = %q, want updated saved theme label", savedView)
	}
}

func TestViewShowsThemePreviewAndPersistMessages(t *testing.T) {
	paths := config.PathsFromUserConfigDir(t.TempDir())
	m := testModelWithOptions(t, ModelOptions{
		Settings:     config.Settings{Theme: config.DefaultThemeName},
		SettingsPath: paths.SettingsFile,
	})
	m.Width = 120
	m.Height = 20

	updated, _ := m.Update(key("t"))
	m = updated.(*Model)

	previewView := m.View()
	if !strings.Contains(previewView, "previewing theme harbor") {
		t.Fatalf("View() = %q, want preview footer message", previewView)
	}
	if !strings.Contains(previewView, "S open settings  t quick preview  ? help") {
		t.Fatalf("View() = %q, want updated footer hint", previewView)
	}

	updated, _ = m.Update(key("S"))
	m = updated.(*Model)
	updated, _ = m.Update(key("s"))
	m = updated.(*Model)

	savedView := m.View()
	if !strings.Contains(savedView, `saved theme "harbor"`) {
		t.Fatalf("View() = %q, want saved footer message", savedView)
	}
	if !strings.Contains(savedView, "Saved theme: harbor") {
		t.Fatalf("View() = %q, want saved theme label", savedView)
	}
}
