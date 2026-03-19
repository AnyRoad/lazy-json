package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anyroad/lazy-json/internal/config"
	"github.com/anyroad/lazy-json/internal/document"
	"github.com/anyroad/lazy-json/internal/source"
	"github.com/anyroad/lazy-json/internal/tui"
)

func TestLoadModelOptionsFromPathsUsesPersistedTheme(t *testing.T) {
	paths := config.PathsFromUserConfigDir(t.TempDir())
	writeTestFile(t, paths.SettingsFile, []byte("{\n  \"theme\": \"mist\"\n}\n"))
	writeTestFile(t, filepath.Join(paths.ThemesDir, "10-mist.json"), []byte(`{"name":"mist","status":{"foreground":"#999999"}}`))

	model := newTestModel(t, loadModelOptionsFromPaths(paths))

	if got, want := model.Settings.Theme, "mist"; got != want {
		t.Fatalf("Settings.Theme = %q, want %q", got, want)
	}
	if got, want := model.Session.ThemeName, "mist"; got != want {
		t.Fatalf("Session.ThemeName = %q, want %q", got, want)
	}
	if got := model.ThemeRegistry.ThemeByName("mist").Name; got != "mist" {
		t.Fatalf("ThemeRegistry.ThemeByName(mist).Name = %q, want mist", got)
	}
	if got, want := model.SettingsPath, paths.SettingsFile; got != want {
		t.Fatalf("SettingsPath = %q, want %q", got, want)
	}
	if model.Session.Status != "" {
		t.Fatalf("Status = %q, want empty", model.Session.Status)
	}
}

func TestLoadModelOptionsFromPathsFallsBackWhenConfiguredThemeMissing(t *testing.T) {
	paths := config.PathsFromUserConfigDir(t.TempDir())
	writeTestFile(t, paths.SettingsFile, []byte("{\n  \"theme\": \"missing-theme\"\n}\n"))

	model := newTestModel(t, loadModelOptionsFromPaths(paths))

	if got, want := model.Settings.Theme, config.DefaultThemeName; got != want {
		t.Fatalf("Settings.Theme = %q, want %q", got, want)
	}
	if got, want := model.Session.ThemeName, config.DefaultThemeName; got != want {
		t.Fatalf("Session.ThemeName = %q, want %q", got, want)
	}
	if !strings.Contains(model.Session.Status, `configured theme "missing-theme" not found`) {
		t.Fatalf("Status = %q, want missing-theme warning", model.Session.Status)
	}
}

func TestLoadModelOptionsFromPathsPropagatesWarningsWithoutBlockingStartup(t *testing.T) {
	paths := config.PathsFromUserConfigDir(t.TempDir())
	writeTestFile(t, filepath.Join(paths.ThemesDir, "10-broken.json"), []byte(`{"name":`))

	model := newTestModel(t, loadModelOptionsFromPaths(paths))

	if got, want := model.Session.ThemeName, config.DefaultThemeName; got != want {
		t.Fatalf("Session.ThemeName = %q, want %q", got, want)
	}
	if !strings.Contains(model.Session.Status, "skip theme 10-broken.json") {
		t.Fatalf("Status = %q, want broken theme warning", model.Session.Status)
	}
	if got, want := model.ThemeRegistry.ThemeByName(config.DefaultThemeName).Name, config.DefaultThemeName; got != want {
		t.Fatalf("ThemeRegistry.ThemeByName(default).Name = %q, want %q", got, want)
	}
}

func TestLoadModelOptionsFromPathsPropagatesSettingsWarningsWithoutBlockingStartup(t *testing.T) {
	paths := config.PathsFromUserConfigDir(t.TempDir())
	writeTestFile(t, paths.SettingsFile, []byte(`{"theme":`))

	model := newTestModel(t, loadModelOptionsFromPaths(paths))

	if got, want := model.Settings.Theme, config.DefaultThemeName; got != want {
		t.Fatalf("Settings.Theme = %q, want %q", got, want)
	}
	if got, want := model.Session.ThemeName, config.DefaultThemeName; got != want {
		t.Fatalf("Session.ThemeName = %q, want %q", got, want)
	}
	if !strings.Contains(model.Session.Status, "parse settings") {
		t.Fatalf("Status = %q, want parse settings warning", model.Session.Status)
	}
}

func TestLoadModelOptionsFromPathsKeepsDefaultStartupWithoutConfig(t *testing.T) {
	paths := config.PathsFromUserConfigDir(t.TempDir())

	model := newTestModel(t, loadModelOptionsFromPaths(paths))

	if got, want := model.Settings.Theme, config.DefaultThemeName; got != want {
		t.Fatalf("Settings.Theme = %q, want %q", got, want)
	}
	if got, want := model.Session.ThemeName, config.DefaultThemeName; got != want {
		t.Fatalf("Session.ThemeName = %q, want %q", got, want)
	}
	if model.Session.Status != "" {
		t.Fatalf("Status = %q, want empty", model.Session.Status)
	}
}

func TestLoadModelOptionsWarnsWhenConfigPathsCannotBeResolved(t *testing.T) {
	model := newTestModel(t, loadModelOptionsWithResolver(func() (config.Paths, error) {
		return config.Paths{}, errors.New("boom")
	}))

	if got, want := model.Settings.Theme, config.DefaultThemeName; got != want {
		t.Fatalf("Settings.Theme = %q, want %q", got, want)
	}
	if got, want := model.Session.ThemeName, config.DefaultThemeName; got != want {
		t.Fatalf("Session.ThemeName = %q, want %q", got, want)
	}
	if !strings.Contains(model.Session.Status, "resolve config paths: boom") {
		t.Fatalf("Status = %q, want resolve config paths warning", model.Session.Status)
	}
}

func newTestModel(t *testing.T, options tui.ModelOptions) *tui.Model {
	t.Helper()
	doc, err := document.Parse([]byte(`{"name":"Ada"}`))
	if err != nil {
		t.Fatal(err)
	}
	return tui.NewModel(doc, source.Input{Kind: source.KindFile, Path: "sample.json"}, options)
}

func writeTestFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}
