package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anyroad/lazy-json/internal/config"
)

func TestSettingsModalOpensPreviewsThemeAndEscClosesWithoutSaving(t *testing.T) {
	paths := config.PathsFromUserConfigDir(t.TempDir())
	m := testModelWithOptions(t, ModelOptions{
		Settings:     config.Settings{Theme: config.DefaultThemeName},
		SettingsPath: paths.SettingsFile,
	})
	m.Width = 120
	m.Height = 20

	updated, _ := m.Update(key("S"))
	m = updated.(*Model)

	if !m.settingsOpen() {
		t.Fatal("settings dialog is closed, want open")
	}
	if got, want := m.Session.Mode, "settings"; string(got) != want {
		t.Fatalf("Mode = %q, want %q", got, want)
	}
	if view := m.View(); !strings.Contains(view, "Theme Settings") {
		t.Fatalf("View() = %q, want settings overlay", view)
	}

	updated, _ = m.Update(specialKey(tea.KeyRight))
	m = updated.(*Model)

	if got, want := m.Session.ThemeName, "harbor"; got != want {
		t.Fatalf("ThemeName = %q, want %q", got, want)
	}
	if got, want := m.Settings.Theme, config.DefaultThemeName; got != want {
		t.Fatalf("Settings.Theme = %q, want %q", got, want)
	}

	updated, _ = m.Update(specialKey(tea.KeyEsc))
	m = updated.(*Model)

	if m.settingsOpen() {
		t.Fatal("settings dialog is open, want closed")
	}
	if got, want := m.Session.ThemeName, "harbor"; got != want {
		t.Fatalf("ThemeName = %q, want %q after esc", got, want)
	}
	if got, want := m.Session.Mode, "normal"; string(got) != want {
		t.Fatalf("Mode = %q, want %q", got, want)
	}
	if _, err := os.Stat(paths.SettingsFile); err == nil || !os.IsNotExist(err) {
		t.Fatalf("settings file error = %v, want not-exist", err)
	}
}

func TestSettingsCommandSavesPreviewedTheme(t *testing.T) {
	paths := config.PathsFromUserConfigDir(t.TempDir())
	m := testModelWithOptions(t, ModelOptions{
		Settings:     config.Settings{Theme: config.DefaultThemeName},
		SettingsPath: paths.SettingsFile,
	})

	updated, _ := m.Update(key(":"))
	m = updated.(*Model)
	m.prompt.SetValue("settings")
	runCmd(t, m, m.submitPrompt())

	if !m.settingsOpen() {
		t.Fatal("settings dialog is closed after :settings, want open")
	}

	updated, _ = m.Update(key("l"))
	m = updated.(*Model)
	updated, _ = m.Update(key("s"))
	m = updated.(*Model)

	if got, want := m.Session.ThemeName, "harbor"; got != want {
		t.Fatalf("ThemeName = %q, want %q", got, want)
	}
	if got, want := m.Settings.Theme, "harbor"; got != want {
		t.Fatalf("Settings.Theme = %q, want %q", got, want)
	}
	if !m.SettingsPersisted {
		t.Fatal("SettingsPersisted = false, want true after save")
	}
	if !m.settingsOpen() {
		t.Fatal("settings dialog is closed after save, want open")
	}
	if got, want := m.Session.Status, `saved theme "harbor"`; got != want {
		t.Fatalf("Status = %q, want %q", got, want)
	}

	data, err := os.ReadFile(paths.SettingsFile)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", paths.SettingsFile, err)
	}
	if !strings.Contains(string(data), `"theme": "harbor"`) {
		t.Fatalf("settings file = %q, want harbor theme", string(data))
	}
}

func TestSettingsSaveFailureKeepsDialogOpen(t *testing.T) {
	blockedPath := filepath.Join(t.TempDir(), "blocked")
	if err := os.Mkdir(blockedPath, 0o755); err != nil {
		t.Fatalf("Mkdir(%q) error = %v", blockedPath, err)
	}

	m := testModelWithOptions(t, ModelOptions{
		Settings:     config.Settings{Theme: config.DefaultThemeName},
		SettingsPath: blockedPath,
	})

	updated, _ := m.Update(key("S"))
	m = updated.(*Model)
	updated, _ = m.Update(key("l"))
	m = updated.(*Model)
	updated, _ = m.Update(key("s"))
	m = updated.(*Model)

	if !m.settingsOpen() {
		t.Fatal("settings dialog is closed after failed save, want open")
	}
	if got, want := m.Settings.Theme, config.DefaultThemeName; got != want {
		t.Fatalf("Settings.Theme = %q, want %q", got, want)
	}
	if !strings.Contains(m.Session.Error, "write settings") {
		t.Fatalf("Error = %q, want write settings failure", m.Session.Error)
	}
}

func TestSettingsModalConsumesNavigationAndCyclesBackward(t *testing.T) {
	m := testModelWithOptions(t, ModelOptions{
		Settings: config.Settings{Theme: config.DefaultThemeName},
	})
	selectedBefore := m.Session.SelectedID

	updated, _ := m.Update(key("S"))
	m = updated.(*Model)
	updated, _ = m.Update(key("j"))
	m = updated.(*Model)
	updated, _ = m.Update(specialKey(tea.KeyLeft))
	m = updated.(*Model)

	if got, want := m.Session.SelectedID, selectedBefore; got != want {
		t.Fatalf("SelectedID = %d, want %d", got, want)
	}
	if got, want := m.Session.ThemeName, "ember"; got != want {
		t.Fatalf("ThemeName = %q, want %q", got, want)
	}
	if !m.settingsOpen() {
		t.Fatal("settings dialog is closed, want open")
	}
}

func TestSettingsModalListsBuiltInAndExternalThemesInOrder(t *testing.T) {
	registry, warnings := NewThemeRegistry([]config.DiscoveredTheme{
		{Path: "10-mist.json", Spec: config.ThemeSpec{Name: "mist"}},
		{Path: "20-aurora.json", Spec: config.ThemeSpec{Name: "aurora"}},
	})
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}

	m := testModelWithOptions(t, ModelOptions{
		ThemeRegistry: registry,
		Settings:      config.Settings{Theme: "ember"},
	})
	m.Width = 120
	m.Height = 20

	updated, _ := m.Update(key("S"))
	m = updated.(*Model)

	initialView := m.View()
	if !strings.Contains(initialView, " ember ") {
		t.Fatalf("View() = %q, want ember selected", initialView)
	}
	if !strings.Contains(initialView, "4/6") {
		t.Fatalf("View() = %q, want ember position", initialView)
	}

	updated, _ = m.Update(specialKey(tea.KeyRight))
	m = updated.(*Model)
	updated, _ = m.Update(specialKey(tea.KeyRight))
	m = updated.(*Model)

	if got, want := m.Session.ThemeName, "aurora"; got != want {
		t.Fatalf("ThemeName = %q, want %q", got, want)
	}

	externalView := m.View()
	if !strings.Contains(externalView, " aurora ") {
		t.Fatalf("View() = %q, want aurora selected", externalView)
	}
	if !strings.Contains(externalView, "6/6") {
		t.Fatalf("View() = %q, want aurora position", externalView)
	}
}
