package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anyroad/lazy-json/internal/config"
	"github.com/anyroad/lazy-json/internal/document"
	"github.com/anyroad/lazy-json/internal/source"
	"github.com/anyroad/lazy-json/internal/tui"
)

func TestParseStartupArgs(t *testing.T) {
	testCases := []struct {
		name        string
		args        []string
		wantPath    string
		wantInputs  []string
		wantHelp    bool
		wantVersion bool
		wantErr     string
	}{
		{
			name:       "file only",
			args:       []string{"sample.json"},
			wantInputs: []string{"sample.json"},
		},
		{
			name:       "select before file",
			args:       []string{"--select", "$.items[0].title", "sample.json"},
			wantPath:   "$.items[0].title",
			wantInputs: []string{"sample.json"},
		},
		{
			name:       "select after file",
			args:       []string{"sample.json", "--select", "$.items[0].title"},
			wantPath:   "$.items[0].title",
			wantInputs: []string{"sample.json"},
		},
		{
			name:       "select equals",
			args:       []string{"--select=$.items[0].title"},
			wantPath:   "$.items[0].title",
			wantInputs: nil,
		},
		{
			name:       "help flag",
			args:       []string{"--help"},
			wantHelp:   true,
			wantInputs: []string{},
		},
		{
			name:       "short help flag",
			args:       []string{"-h"},
			wantHelp:   true,
			wantInputs: []string{},
		},
		{
			name:        "version flag",
			args:        []string{"--version"},
			wantVersion: true,
			wantInputs:  []string{},
		},
		{
			name:       "end of flags preserves filename",
			args:       []string{"--", "--select=sample.json"},
			wantInputs: []string{"--select=sample.json"},
		},
		{
			name:       "end of flags after select preserves filename",
			args:       []string{"--select", "$.items[0].title", "--", "--select"},
			wantPath:   "$.items[0].title",
			wantInputs: []string{"--select"},
		},
		{
			name:    "missing value",
			args:    []string{"--select"},
			wantErr: "--select requires a JSON path",
		},
		{
			name:    "missing value before another flag",
			args:    []string{"--select", "--other", "sample.json"},
			wantErr: "--select requires a JSON path",
		},
		{
			name:    "duplicate",
			args:    []string{"--select", "$.name", "--select=$.items"},
			wantErr: "--select may only be provided once",
		},
		{
			name:    "duplicate before value",
			args:    []string{"--select", "--select=$.items", "sample.json"},
			wantErr: "--select may only be provided once",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			parsed, err := parseStartupArgs(testCase.args)
			if testCase.wantErr != "" {
				if err == nil {
					t.Fatalf("parseStartupArgs(%v) error = nil, want %q", testCase.args, testCase.wantErr)
				}
				if got := err.Error(); got != testCase.wantErr {
					t.Fatalf("error = %q, want %q", got, testCase.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseStartupArgs(%v) error = %v", testCase.args, err)
			}
			if got, want := parsed.SelectPath, testCase.wantPath; got != want {
				t.Fatalf("SelectPath = %q, want %q", got, want)
			}
			if got, want := parsed.ShowHelp, testCase.wantHelp; got != want {
				t.Fatalf("ShowHelp = %t, want %t", got, want)
			}
			if got, want := parsed.ShowVersion, testCase.wantVersion; got != want {
				t.Fatalf("ShowVersion = %t, want %t", got, want)
			}
			if got, want := parsed.InputArgs, testCase.wantInputs; len(got) != len(want) {
				t.Fatalf("InputArgs = %v, want %v", got, want)
			} else {
				for index := range want {
					if got[index] != want[index] {
						t.Fatalf("InputArgs[%d] = %q, want %q", index, got[index], want[index])
					}
				}
			}
		})
	}
}

func TestRunHelpWritesUsage(t *testing.T) {
	stdin, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stdin.Close() })

	var stdout bytes.Buffer
	if err := run([]string{"--help"}, stdin, &stdout); err != nil {
		t.Fatalf("run(--help) error = %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, usageLine) {
		t.Fatalf("stdout = %q, want usage line", got)
	}
	if got := stdout.String(); !strings.Contains(got, "--version") {
		t.Fatalf("stdout = %q, want version flag description", got)
	}
}

func TestRunVersionWritesVersion(t *testing.T) {
	stdin, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stdin.Close() })

	originalVersion := version
	version = "v1.2.3"
	t.Cleanup(func() { version = originalVersion })

	var stdout bytes.Buffer
	if err := run([]string{"--version"}, stdin, &stdout); err != nil {
		t.Fatalf("run(--version) error = %v", err)
	}
	if got, want := stdout.String(), "lazy-json v1.2.3\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestApplyStartupSelection(t *testing.T) {
	model := newNestedTestModel(t)

	if err := applyStartupSelection(model, "$.items[0].title"); err != nil {
		t.Fatalf("applyStartupSelection() error = %v", err)
	}
	if got, want := model.Session.SelectedID, model.Doc.Root.Object[0].Value.Array[0].Object[0].Value.ID; got != want {
		t.Fatalf("SelectedID = %d, want %d", got, want)
	}
	if !model.Session.Expanded[model.Doc.Root.Object[0].Value.ID] {
		t.Fatal("items container is not expanded")
	}
	if !model.Session.Expanded[model.Doc.Root.Object[0].Value.Array[0].ID] {
		t.Fatal("first array item is not expanded")
	}
	if got := model.Session.Status; got != "" {
		t.Fatalf("Status = %q, want empty on exact match", got)
	}
	if got := model.Session.Error; got != "" {
		t.Fatalf("Error = %q, want empty on exact match", got)
	}
}

func TestApplyStartupSelectionFallbackMessages(t *testing.T) {
	t.Run("nearest ancestor", func(t *testing.T) {
		model := newNestedTestModel(t)

		if err := applyStartupSelection(model, "$.items[99].title"); err != nil {
			t.Fatalf("applyStartupSelection() error = %v", err)
		}
		if got, want := model.Session.SelectedID, model.Doc.Root.Object[0].Value.ID; got != want {
			t.Fatalf("SelectedID = %d, want %d", got, want)
		}
		if got, want := model.Session.Status, "select path not found: $.items[99].title; opened nearest existing ancestor $.items"; got != want {
			t.Fatalf("Status = %q, want %q", got, want)
		}
		if got := model.Session.Error; got != "" {
			t.Fatalf("Error = %q, want empty for ancestor fallback", got)
		}
	})

	t.Run("root only", func(t *testing.T) {
		model := newNestedTestModel(t)

		if err := applyStartupSelection(model, "$.missing.branch"); err != nil {
			t.Fatalf("applyStartupSelection() error = %v", err)
		}
		if got, want := model.Session.SelectedID, model.Doc.Root.ID; got != want {
			t.Fatalf("SelectedID = %d, want %d", got, want)
		}
		if got, want := model.Session.Error, "select path not found: $.missing.branch; opened root $"; got != want {
			t.Fatalf("Error = %q, want %q", got, want)
		}
		if got := model.Session.Status; got != "" {
			t.Fatalf("Status = %q, want empty for root fallback", got)
		}
	})
}

func TestApplyStartupSelectionRejectsMalformedPath(t *testing.T) {
	model := newNestedTestModel(t)

	err := applyStartupSelection(model, "$.items[")
	if err == nil {
		t.Fatal("applyStartupSelection() error = nil")
	}
	if !strings.Contains(err.Error(), "invalid select path") {
		t.Fatalf("error = %q, want invalid select path prefix", err.Error())
	}
}

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
	if !model.SettingsPersisted {
		t.Fatal("SettingsPersisted = false, want true")
	}
	if model.Session.Status != "" {
		t.Fatalf("Status = %q, want empty", model.Session.Status)
	}
}

func TestLoadModelOptionsFromPathsRestoresThemeAfterModalSaveAndRestart(t *testing.T) {
	paths := config.PathsFromUserConfigDir(t.TempDir())
	nextTheme := tui.BuiltinThemeRegistry().NextTheme(config.DefaultThemeName).Name

	model := newTestModel(t, loadModelOptionsFromPaths(paths))

	updated, _ := model.Update(runeKey("S"))
	model = updated.(*tui.Model)
	updated, _ = model.Update(runeKey("l"))
	model = updated.(*tui.Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(*tui.Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(*tui.Model)
	updated, _ = model.Update(runeKey("s"))
	model = updated.(*tui.Model)

	if got, want := model.Settings.Theme, nextTheme; got != want {
		t.Fatalf("Settings.Theme = %q, want %q after save", got, want)
	}
	if !model.SettingsPersisted {
		t.Fatal("SettingsPersisted = false, want true after save")
	}

	restarted := newTestModel(t, loadModelOptionsFromPaths(paths))

	if got, want := restarted.Settings.Theme, nextTheme; got != want {
		t.Fatalf("restarted Settings.Theme = %q, want %q", got, want)
	}
	if got, want := restarted.Session.ThemeName, nextTheme; got != want {
		t.Fatalf("restarted Session.ThemeName = %q, want %q", got, want)
	}
	if !restarted.SettingsPersisted {
		t.Fatal("restarted SettingsPersisted = false, want true")
	}
	if restarted.Session.Status != "" {
		t.Fatalf("restarted Status = %q, want empty", restarted.Session.Status)
	}
}

func TestLoadModelOptionsFromPathsFallsBackWhenConfiguredThemeMissing(t *testing.T) {
	paths := config.PathsFromUserConfigDir(t.TempDir())
	writeTestFile(t, paths.SettingsFile, []byte("{\n  \"theme\": \"missing-theme\",\n  \"wrap_long_strings\": true,\n  \"save_indent\": {\n    \"kind\": \"tabs\"\n  }\n}\n"))

	model := newTestModel(t, loadModelOptionsFromPaths(paths))

	if got, want := model.Settings.Theme, config.DefaultThemeName; got != want {
		t.Fatalf("Settings.Theme = %q, want %q", got, want)
	}
	if !model.Settings.WrapLongStrings {
		t.Fatal("Settings.WrapLongStrings = false, want true")
	}
	if got, want := model.Settings.SaveIndent.Label(), "tabs"; got != want {
		t.Fatalf("Settings.SaveIndent = %q, want %q", got, want)
	}
	if got, want := model.Session.ThemeName, config.DefaultThemeName; got != want {
		t.Fatalf("Session.ThemeName = %q, want %q", got, want)
	}
	if !strings.Contains(model.Session.Status, `configured theme "missing-theme" not found`) {
		t.Fatalf("Status = %q, want missing-theme warning", model.Session.Status)
	}

	model.Width = 120
	model.Height = 20
	updated, _ := model.Update(runeKey("S"))
	model = updated.(*tui.Model)

	view := model.View()
	if !strings.Contains(view, "Saved theme unavailable; using "+config.DefaultThemeName) {
		t.Fatalf("View() = %q, want fallback saved-theme message", view)
	}
	if !strings.Contains(view, "fallback") {
		t.Fatalf("View() = %q, want fallback state label", view)
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

func TestLoadModelOptionsFromPathsAggregatesSettingsDiscoveryAndRegistryWarnings(t *testing.T) {
	paths := config.PathsFromUserConfigDir(t.TempDir())
	writeTestFile(t, paths.SettingsFile, []byte(`{"theme":`))
	writeTestFile(t, filepath.Join(paths.ThemesDir, "10-broken.json"), []byte(`{"name":`))
	writeTestFile(t, filepath.Join(paths.ThemesDir, "20-mist.json"), []byte(`{"name":"mist","key":{"foreground":"not-a-color"}}`))

	model := newTestModel(t, loadModelOptionsFromPaths(paths))

	if got, want := model.Session.ThemeName, config.DefaultThemeName; got != want {
		t.Fatalf("Session.ThemeName = %q, want %q", got, want)
	}
	if !strings.HasPrefix(model.Session.Status, "warning: ") {
		t.Fatalf("Status = %q, want warning prefix", model.Session.Status)
	}
	if !strings.Contains(model.Session.Status, "parse settings") {
		t.Fatalf("Status = %q, want settings warning", model.Session.Status)
	}
	if !strings.Contains(model.Session.Status, "skip theme 10-broken.json") {
		t.Fatalf("Status = %q, want discovery warning", model.Session.Status)
	}
	if !strings.Contains(model.Session.Status, `theme 20-mist.json key.foreground "not-a-color" is invalid; using default`) {
		t.Fatalf("Status = %q, want registry warning", model.Session.Status)
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

func TestLoadModelOptionsWarningsRemainVisibleAfterOpeningSettings(t *testing.T) {
	paths := config.PathsFromUserConfigDir(t.TempDir())
	writeTestFile(t, paths.SettingsFile, []byte(`{"theme":`))

	model := newTestModel(t, loadModelOptionsFromPaths(paths))
	model.Width = 120
	model.Height = 20

	updated, _ := model.Update(runeKey("S"))
	model = updated.(*tui.Model)

	if view := model.View(); !strings.Contains(view, "warning: parse settings") {
		t.Fatalf("View() = %q, want startup warning after opening settings", view)
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
	if model.SettingsFilePresent {
		t.Fatal("SettingsFilePresent = true, want false without settings.json")
	}
	if model.Session.Status != "" {
		t.Fatalf("Status = %q, want empty", model.Session.Status)
	}

	model.Width = 120
	model.Height = 20
	updated, _ := model.Update(runeKey("S"))
	model = updated.(*tui.Model)

	view := model.View()
	if !strings.Contains(view, "Saved theme: none yet (using "+config.DefaultThemeName+")") {
		t.Fatalf("View() = %q, want not-saved label", view)
	}
	if !strings.Contains(view, "not saved") {
		t.Fatalf("View() = %q, want not-saved state", view)
	}
	if strings.Contains(view, "Saved theme unavailable; using "+config.DefaultThemeName) {
		t.Fatalf("View() = %q, unexpectedly shows fallback warning on first run", view)
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

func TestLoadModelOptionsResolverFailureDisablesSettingsSave(t *testing.T) {
	model := newTestModel(t, loadModelOptionsWithResolver(func() (config.Paths, error) {
		return config.Paths{}, errors.New("boom")
	}))
	model.Width = 120
	model.Height = 20

	updated, _ := model.Update(runeKey("S"))
	model = updated.(*tui.Model)

	view := model.View()
	if !strings.Contains(view, "Save unavailable: config path could not be resolved.") {
		t.Fatalf("View() = %q, want disabled save hint", view)
	}
	if !strings.Contains(view, "Settings file unavailable; using "+config.DefaultThemeName) {
		t.Fatalf("View() = %q, want unavailable saved-theme message", view)
	}
	if !strings.Contains(view, "unavailable") {
		t.Fatalf("View() = %q, want unavailable state label", view)
	}
	if !strings.Contains(view, "up/down row  left/right change  s unavailable") {
		t.Fatalf("View() = %q, want unavailable modal controls", view)
	}
	if !strings.Contains(view, "s unavailable") {
		t.Fatalf("View() = %q, want unavailable footer hint", view)
	}
	if strings.Contains(view, "h/l preview  s save settings.json  esc close") {
		t.Fatalf("View() = %q, unexpectedly advertises save shortcut when unavailable", view)
	}

	updated, _ = model.Update(runeKey("s"))
	model = updated.(*tui.Model)

	if got, want := string(model.Session.Mode), "settings"; got != want {
		t.Fatalf("Mode = %q, want %q", got, want)
	}
	if !strings.Contains(model.Session.Error, "settings path unavailable") {
		t.Fatalf("Error = %q, want settings path error", model.Session.Error)
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

func newNestedTestModel(t *testing.T) *tui.Model {
	t.Helper()
	doc, err := document.Parse([]byte(`{"items":[{"title":"alpha"},{"title":"beta"}],"meta":{"count":2}}`))
	if err != nil {
		t.Fatal(err)
	}
	return tui.NewModel(doc, source.Input{Kind: source.KindFile, Path: "sample.json"}, tui.ModelOptions{})
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

func runeKey(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}
