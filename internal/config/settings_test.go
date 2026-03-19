package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestPathsFromUserConfigDir(t *testing.T) {
	paths := PathsFromUserConfigDir("/tmp/user-config")

	if got, want := paths.RootDir, filepath.Join("/tmp/user-config", AppDirName); got != want {
		t.Fatalf("RootDir = %q, want %q", got, want)
	}
	if got, want := paths.SettingsFile, filepath.Join("/tmp/user-config", AppDirName, SettingsFileName); got != want {
		t.Fatalf("SettingsFile = %q, want %q", got, want)
	}
	if got, want := paths.ThemesDir, filepath.Join("/tmp/user-config", AppDirName, ThemesDirName); got != want {
		t.Fatalf("ThemesDir = %q, want %q", got, want)
	}
}

func TestLoadSettingsMissingReturnsDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), SettingsFileName)

	settings, warnings := LoadSettings(path)

	if settings != DefaultSettings() {
		t.Fatalf("settings = %#v, want %#v", settings, DefaultSettings())
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
}

func TestLoadSettingsDefaultFallback(t *testing.T) {
	path := filepath.Join(t.TempDir(), SettingsFileName)
	writeFile(t, path, []byte(`{"theme":"   "}`))

	settings, warnings := LoadSettings(path)

	if got, want := settings.Theme, DefaultThemeName; got != want {
		t.Fatalf("Theme = %q, want %q", got, want)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
}

func TestLoadSettingsInvalidJSONFallsBackToDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), SettingsFileName)
	writeFile(t, path, []byte(`{"theme":`))

	settings, warnings := LoadSettings(path)

	if settings != DefaultSettings() {
		t.Fatalf("settings = %#v, want %#v", settings, DefaultSettings())
	}
	if len(warnings) != 1 {
		t.Fatalf("warnings = %v, want 1 warning", warnings)
	}
	if !strings.Contains(warnings[0], "parse settings") {
		t.Fatalf("warning = %q, want parse warning", warnings[0])
	}
}

func TestSaveAndLoadSettings(t *testing.T) {
	paths := PathsFromUserConfigDir(t.TempDir())
	want := Settings{Theme: "harbor"}

	if err := SaveSettings(paths.SettingsFile, want); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	settings, warnings := LoadSettings(paths.SettingsFile)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	if settings != want {
		t.Fatalf("settings = %#v, want %#v", settings, want)
	}

	data := string(readFile(t, paths.SettingsFile))
	if !strings.Contains(data, "\"theme\": \"harbor\"") {
		t.Fatalf("settings file = %q, want persisted theme", data)
	}
}
