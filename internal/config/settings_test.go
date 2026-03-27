package config

import (
	"os"
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

func TestResolvePathsUsesUserConfigDir(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, "xdg-config"))

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir() error = %v", err)
	}

	paths, err := ResolvePaths()
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}

	if got, want := paths, PathsFromUserConfigDir(userConfigDir); got != want {
		t.Fatalf("ResolvePaths() = %#v, want %#v", got, want)
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
	if settings.ShowLineNumbers {
		t.Fatal("ShowLineNumbers = true, want false when setting is omitted")
	}
	if !settings.ShowJSONPath {
		t.Fatal("ShowJSONPath = false, want true when setting is omitted")
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

func TestLoadSettingsReadFailureFallsBackToDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), SettingsFileName)
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatalf("Mkdir(%q) error = %v", path, err)
	}

	settings, warnings := LoadSettings(path)

	if settings != DefaultSettings() {
		t.Fatalf("settings = %#v, want %#v", settings, DefaultSettings())
	}
	if len(warnings) != 1 {
		t.Fatalf("warnings = %v, want 1 warning", warnings)
	}
	if !strings.Contains(warnings[0], "load settings") {
		t.Fatalf("warning = %q, want load warning", warnings[0])
	}
}

func TestSaveAndLoadSettings(t *testing.T) {
	paths := PathsFromUserConfigDir(t.TempDir())
	want := Settings{
		Theme:           "custom",
		ShowLineNumbers: true,
		WrapLongStrings: true,
		SaveIndent:      SaveIndent{Kind: IndentKindTabs},
	}.WithShowJSONPath(false).WithDefaults()

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
	if !strings.Contains(data, "\"theme\": \"custom\"") {
		t.Fatalf("settings file = %q, want persisted theme", data)
	}
	if !strings.Contains(data, "\"show_line_numbers\": true") {
		t.Fatalf("settings file = %q, want persisted line-number setting", data)
	}
	if !strings.Contains(data, "\"wrap_long_strings\": true") {
		t.Fatalf("settings file = %q, want persisted wrap setting", data)
	}
	if !strings.Contains(data, "\"show_json_path\": false") {
		t.Fatalf("settings file = %q, want persisted JSON path setting", data)
	}
	if !strings.Contains(data, "\"kind\": \"tabs\"") {
		t.Fatalf("settings file = %q, want persisted tabs indent", data)
	}
}

func TestLoadSettingsNormalizesInvalidSaveIndent(t *testing.T) {
	path := filepath.Join(t.TempDir(), SettingsFileName)
	writeFile(t, path, []byte(`{"theme":"custom","save_indent":{"kind":"spaces","size":9}}`))

	settings, warnings := LoadSettings(path)

	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	if got, want := settings.SaveIndent, DefaultSaveIndent(); got != want {
		t.Fatalf("SaveIndent = %#v, want %#v", got, want)
	}
}

func TestSaveIndentLabelAndString(t *testing.T) {
	if got, want := (SaveIndent{Kind: IndentKindSpaces, Size: 3}).Label(), "spaces:3"; got != want {
		t.Fatalf("Label() = %q, want %q", got, want)
	}
	if got, want := (SaveIndent{Kind: IndentKindSpaces, Size: 3}).String(), "   "; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
	if got, want := (SaveIndent{Kind: IndentKindTabs}).Label(), "tabs"; got != want {
		t.Fatalf("Label() = %q, want %q", got, want)
	}
	if got, want := (SaveIndent{Kind: IndentKindTabs}).String(), "\t"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestSaveSettingsReturnsCreateDirFailure(t *testing.T) {
	blockedPath := filepath.Join(t.TempDir(), "blocked")
	writeFile(t, blockedPath, []byte("blocker"))

	err := SaveSettings(filepath.Join(blockedPath, SettingsFileName), Settings{Theme: "custom"})
	if err == nil {
		t.Fatal("SaveSettings() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "create config dir") {
		t.Fatalf("SaveSettings() error = %q, want create config dir failure", err)
	}
}

func TestSaveSettingsReturnsWriteFailureWhenTargetIsDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings-dir")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatalf("Mkdir(%q) error = %v", path, err)
	}

	err := SaveSettings(path, Settings{Theme: "custom"})
	if err == nil {
		t.Fatal("SaveSettings() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "write settings") {
		t.Fatalf("SaveSettings() error = %q, want write settings failure", err)
	}
}
