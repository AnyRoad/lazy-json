package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anyroad/lazy-json/internal/config"
)

func TestSettingsModalOpensPreviewsThemeAndEscClosesWithoutSaving(t *testing.T) {
	paths := config.PathsFromUserConfigDir(t.TempDir())
	nextTheme := nextThemeName(t, BuiltinThemeRegistry(), config.DefaultThemeName)
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
	if view := m.View(); !strings.Contains(view, "Settings") {
		t.Fatalf("View() = %q, want settings overlay", view)
	}

	updated, _ = m.Update(specialKey(tea.KeyRight))
	m = updated.(*Model)

	if got, want := m.Session.ThemeName, nextTheme; got != want {
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
	if got, want := m.Session.ThemeName, nextTheme; got != want {
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
	nextTheme := nextThemeName(t, BuiltinThemeRegistry(), config.DefaultThemeName)
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

	if got, want := m.Session.ThemeName, nextTheme; got != want {
		t.Fatalf("ThemeName = %q, want %q", got, want)
	}
	if got, want := m.Settings.Theme, nextTheme; got != want {
		t.Fatalf("Settings.Theme = %q, want %q", got, want)
	}
	if !m.SettingsPersisted {
		t.Fatal("SettingsPersisted = false, want true after save")
	}
	if !m.settingsOpen() {
		t.Fatal("settings dialog is closed after save, want open")
	}
	if got, want := m.Session.Status, "saved settings"; got != want {
		t.Fatalf("Status = %q, want %q", got, want)
	}

	data, err := os.ReadFile(paths.SettingsFile)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", paths.SettingsFile, err)
	}
	if !strings.Contains(string(data), `"theme": "`+nextTheme+`"`) {
		t.Fatalf("settings file = %q, want saved theme", string(data))
	}
}

func TestSettingsModalPreviewsAndSavesJSONPath(t *testing.T) {
	paths := config.PathsFromUserConfigDir(t.TempDir())
	m := testModelWithOptions(t, ModelOptions{
		Settings:     config.DefaultSettings(),
		SettingsPath: paths.SettingsFile,
	})
	m.Width = 120
	m.Height = 20

	updated, _ := m.Update(key("S"))
	m = updated.(*Model)
	updated, _ = m.Update(specialKey(tea.KeyDown))
	m = updated.(*Model)
	updated, _ = m.Update(specialKey(tea.KeyRight))
	m = updated.(*Model)

	if m.ActiveSettings.WithDefaults().ShowJSONPath {
		t.Fatal("ActiveSettings.ShowJSONPath = true, want false during preview")
	}
	if !m.Settings.WithDefaults().ShowJSONPath {
		t.Fatal("Settings.ShowJSONPath = false, want true before save")
	}
	if strings.Contains(stripANSI(m.View()), "$.name") {
		t.Fatalf("View() = %q, want hidden JSON path during preview", stripANSI(m.View()))
	}

	updated, _ = m.Update(key("s"))
	m = updated.(*Model)

	if m.Settings.WithDefaults().ShowJSONPath {
		t.Fatal("Settings.ShowJSONPath = true, want false after save")
	}

	data, err := os.ReadFile(paths.SettingsFile)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", paths.SettingsFile, err)
	}
	if !strings.Contains(string(data), `"show_json_path": false`) {
		t.Fatalf("settings file = %q, want saved JSON path setting", string(data))
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
	updated, _ = m.Update(specialKey(tea.KeyDown))
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
	wantTheme := previousThemeName(t, BuiltinThemeRegistry(), config.DefaultThemeName)
	selectedBefore := m.Session.SelectedID

	updated, _ := m.Update(key("S"))
	m = updated.(*Model)
	updated, _ = m.Update(specialKey(tea.KeyLeft))
	m = updated.(*Model)

	if got, want := m.Session.SelectedID, selectedBefore; got != want {
		t.Fatalf("SelectedID = %d, want %d", got, want)
	}
	if got, want := m.Session.ThemeName, wantTheme; got != want {
		t.Fatalf("ThemeName = %q, want %q", got, want)
	}
	if !m.settingsOpen() {
		t.Fatal("settings dialog is closed, want open")
	}
}

func TestSettingsModalListsBuiltInAndExternalThemesInOrder(t *testing.T) {
	builtins := BuiltinThemeRegistry().Themes()
	builtinCount := len(builtins)
	lastBuiltin := builtins[builtinCount-1].Name
	registry, warnings := NewThemeRegistry([]config.DiscoveredTheme{
		{Path: "10-mist.json", Spec: config.ThemeSpec{Name: "mist"}},
		{Path: "20-aurora.json", Spec: config.ThemeSpec{Name: "aurora"}},
	})
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}

	m := testModelWithOptions(t, ModelOptions{
		ThemeRegistry: registry,
		Settings:      config.Settings{Theme: lastBuiltin},
	})
	m.Width = 120
	m.Height = 20

	updated, _ := m.Update(key("S"))
	m = updated.(*Model)

	initialView := m.View()
	if !strings.Contains(initialView, " "+lastBuiltin+" ") {
		t.Fatalf("View() = %q, want last built-in selected", initialView)
	}
	if !strings.Contains(initialView, fmt.Sprintf("%d/%d", builtinCount, builtinCount+2)) {
		t.Fatalf("View() = %q, want built-in position", initialView)
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
	if !strings.Contains(externalView, fmt.Sprintf("%d/%d", builtinCount+2, builtinCount+2)) {
		t.Fatalf("View() = %q, want aurora position", externalView)
	}
}
