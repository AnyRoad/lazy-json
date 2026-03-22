package tui

import (
	"errors"
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anyroad/lazy-json/internal/config"
	"github.com/anyroad/lazy-json/internal/document"
	"github.com/anyroad/lazy-json/internal/source"
)

func testModel(t *testing.T) *Model {
	t.Helper()
	return testModelWithOptions(t, ModelOptions{})
}

func testModelWithOptions(t *testing.T, options ModelOptions) *Model {
	t.Helper()
	if options.Clipboard == nil {
		options.Clipboard = &stubClipboard{}
	}
	doc, err := document.Parse([]byte(`{"name":"Ada","items":[1,2]}`))
	if err != nil {
		t.Fatal(err)
	}
	return NewModel(doc, source.Input{Kind: source.KindFile, Path: "sample.json"}, options)
}

func builtinThemeNameAt(t *testing.T, index int) string {
	t.Helper()
	themes := BuiltinThemeRegistry().Themes()
	if index < 0 || index >= len(themes) {
		t.Fatalf("builtin theme index %d out of range %d", index, len(themes))
	}
	return themes[index].Name
}

func lastBuiltinThemeName(t *testing.T) string {
	t.Helper()
	themes := BuiltinThemeRegistry().Themes()
	if len(themes) == 0 {
		t.Fatal("builtin themes are empty")
	}
	return themes[len(themes)-1].Name
}

func nextThemeName(t *testing.T, registry ThemeRegistry, current string) string {
	t.Helper()
	next := registry.NextTheme(current)
	if next.Name == "" {
		t.Fatalf("NextTheme(%q) returned empty theme", current)
	}
	return next.Name
}

func previousThemeName(t *testing.T, registry ThemeRegistry, current string) string {
	t.Helper()
	themes := registry.Themes()
	for index, theme := range themes {
		if normalizeThemeName(theme.Name) == normalizeThemeName(current) {
			return themes[(index-1+len(themes))%len(themes)].Name
		}
	}
	t.Fatalf("theme %q not found in registry", current)
	return ""
}

type stubClipboard struct {
	writes []string
	err    error
}

func (c *stubClipboard) WriteAll(text string) error {
	if c.err != nil {
		return c.err
	}
	c.writes = append(c.writes, text)
	return nil
}

func clipboardForModel(t *testing.T, m *Model) *stubClipboard {
	t.Helper()
	clipboard, ok := m.Clipboard.(*stubClipboard)
	if !ok {
		t.Fatalf("Clipboard = %T, want *stubClipboard", m.Clipboard)
	}
	return clipboard
}

func key(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func specialKey(keyType tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: keyType}
}

func runCmd(t *testing.T, m *Model, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		return
	}
	msg := cmd()
	if msg == nil {
		return
	}
	updated, next := m.Update(msg)
	model, ok := updated.(*Model)
	if !ok {
		t.Fatalf("Update() returned %T", updated)
	}
	*m = *model
	runCmd(t, m, next)
}

func TestNavigationAndThemeSwitch(t *testing.T) {
	registry, warnings := NewThemeRegistry([]config.DiscoveredTheme{
		{Path: "10-mist.json", Spec: config.ThemeSpec{Name: "mist"}},
	})
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}

	m := testModelWithOptions(t, ModelOptions{
		ThemeRegistry: registry,
		Settings:      config.Settings{Theme: lastBuiltinThemeName(t)},
	})
	updated, _ := m.Update(key("j"))
	m = updated.(*Model)
	if m.Session.SelectedID != m.Doc.Root.Object[0].Value.ID {
		t.Fatalf("SelectedID = %d", m.Session.SelectedID)
	}
	updated, _ = m.Update(key("t"))
	m = updated.(*Model)
	if got, want := m.Session.ThemeName, "mist"; got != want {
		t.Fatalf("ThemeName = %q", m.Session.ThemeName)
	}
	if got, want := m.Settings.Theme, lastBuiltinThemeName(t); got != want {
		t.Fatalf("Settings.Theme = %q, want %q", got, want)
	}
}

func TestCommandThemeSwitch(t *testing.T) {
	registry, warnings := NewThemeRegistry([]config.DiscoveredTheme{
		{Path: "10-mist.json", Spec: config.ThemeSpec{Name: "mist"}},
	})
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}

	m := testModelWithOptions(t, ModelOptions{
		ThemeRegistry: registry,
		Settings:      config.Settings{Theme: lastBuiltinThemeName(t)},
	})

	runCmd(t, m, m.handleCommand("theme"))

	if got, want := m.Session.ThemeName, "mist"; got != want {
		t.Fatalf("ThemeName = %q, want %q", got, want)
	}
	if got, want := m.Session.Status, "previewing theme mist"; got != want {
		t.Fatalf("Status = %q, want %q", got, want)
	}
	if got, want := m.Settings.Theme, lastBuiltinThemeName(t); got != want {
		t.Fatalf("Settings.Theme = %q, want %q", got, want)
	}
}

func TestCommandSelectPathExact(t *testing.T) {
	m := testModel(t)

	runCmd(t, m, m.handleCommand("select-path $.items[1]"))

	items := m.Doc.Root.Object[1].Value
	if got, want := m.Session.SelectedID, items.Array[1].ID; got != want {
		t.Fatalf("SelectedID = %d, want %d", got, want)
	}
	if !m.Session.Expanded[items.ID] {
		t.Fatal("items container is not expanded")
	}
	if got, want := m.Session.Status, "selected path $.items[1]"; got != want {
		t.Fatalf("Status = %q, want %q", got, want)
	}
	if got := m.Session.Error; got != "" {
		t.Fatalf("Error = %q, want empty", got)
	}
}

func TestCommandSelectPathFallbackMessages(t *testing.T) {
	t.Run("nearest ancestor", func(t *testing.T) {
		m := testModel(t)

		runCmd(t, m, m.handleCommand("select-path $.items[99].title"))

		if got, want := m.Session.SelectedID, m.Doc.Root.Object[1].Value.ID; got != want {
			t.Fatalf("SelectedID = %d, want %d", got, want)
		}
		if got, want := m.Session.Status, "select path not found: $.items[99].title; opened nearest existing ancestor $.items"; got != want {
			t.Fatalf("Status = %q, want %q", got, want)
		}
		if got := m.Session.Error; got != "" {
			t.Fatalf("Error = %q, want empty", got)
		}
	})

	t.Run("root only", func(t *testing.T) {
		m := testModel(t)

		runCmd(t, m, m.handleCommand("select-path $.missing.branch"))

		if got, want := m.Session.SelectedID, m.Doc.Root.ID; got != want {
			t.Fatalf("SelectedID = %d, want %d", got, want)
		}
		if got, want := m.Session.Error, "select path not found: $.missing.branch; opened root $"; got != want {
			t.Fatalf("Error = %q, want %q", got, want)
		}
		if got := m.Session.Status; got != "" {
			t.Fatalf("Status = %q, want empty", got)
		}
	})
}

func TestCommandSelectPathRejectsInvalidInput(t *testing.T) {
	t.Run("missing path", func(t *testing.T) {
		m := testModel(t)

		runCmd(t, m, m.handleCommand("select-path"))

		if got, want := m.Session.Error, "select-path requires a JSON path"; got != want {
			t.Fatalf("Error = %q, want %q", got, want)
		}
	})

	t.Run("malformed path", func(t *testing.T) {
		m := testModel(t)

		runCmd(t, m, m.handleCommand("select-path $.items["))

		if got := m.Session.Error; got == "" {
			t.Fatal("Error = empty, want invalid select path")
		}
	})
}

func TestHelpToggle(t *testing.T) {
	m := testModel(t)
	updated, _ := m.Update(key("?"))
	m = updated.(*Model)
	if !m.Session.Help {
		t.Fatal("Help = false, want true")
	}
	updated, _ = m.Update(key("?"))
	m = updated.(*Model)
	if m.Session.Help {
		t.Fatal("Help = true, want false")
	}
}

func TestEditScalarPrompt(t *testing.T) {
	m := testModel(t)
	m.Session.SelectedID = m.Doc.Root.Object[0].Value.ID
	cmd := m.startScalarEdit()
	runCmd(t, m, cmd)
	m.prompt.SetValue(`"Grace"`)
	cmd = m.submitPrompt()
	runCmd(t, m, cmd)

	if got := m.Doc.Root.Object[0].Value.String; got != "Grace" {
		t.Fatalf("string = %q", got)
	}
	if !m.Session.Dirty {
		t.Fatal("Dirty = false, want true")
	}
}

func TestCommandSaveAndPrint(t *testing.T) {
	m := testModel(t)
	updated, cmd := m.Update(key(":"))
	m = updated.(*Model)
	m.prompt.SetValue("print")
	runCmd(t, m, m.submitPrompt())
	if len(m.ExitOutput) == 0 {
		t.Fatal("ExitOutput is empty")
	}

	m = testModel(t)
	m.Session.SourcePath = t.TempDir() + "/saved.json"
	cmd = m.handleCommand("w")
	runCmd(t, m, cmd)
	if m.Session.Error != "" {
		t.Fatalf("Error = %q", m.Session.Error)
	}
}

func TestSavePrintAndPrettyCopyUseActiveSaveIndent(t *testing.T) {
	clipboard := &stubClipboard{}
	m := testModelWithOptions(t, ModelOptions{Clipboard: clipboard})
	m.ActiveSettings = config.Settings{
		Theme:      config.DefaultThemeName,
		SaveIndent: config.SaveIndent{Kind: config.IndentKindTabs},
	}.WithDefaults()

	runCmd(t, m, m.handleCommand("copy-json"))
	if got, want := clipboard.writes[len(clipboard.writes)-1], "{\n\t\"name\": \"Ada\",\n\t\"items\": [\n\t\t1,\n\t\t2\n\t]\n}"; got != want {
		t.Fatalf("copy-json = %q, want %q", got, want)
	}

	runCmd(t, m, m.handleCommand("print"))
	if got, want := string(m.ExitOutput), "{\n\t\"name\": \"Ada\",\n\t\"items\": [\n\t\t1,\n\t\t2\n\t]\n}\n"; got != want {
		t.Fatalf("ExitOutput = %q, want %q", got, want)
	}

	m = testModelWithOptions(t, ModelOptions{Clipboard: clipboard})
	m.ActiveSettings = config.Settings{
		Theme:      config.DefaultThemeName,
		SaveIndent: config.SaveIndent{Kind: config.IndentKindSpaces, Size: 3},
	}.WithDefaults()
	path := t.TempDir() + "/saved.json"
	if err := m.save(path); err != nil {
		t.Fatalf("save() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	if got, want := string(data), "{\n   \"name\": \"Ada\",\n   \"items\": [\n      1,\n      2\n   ]\n}\n"; got != want {
		t.Fatalf("saved file = %q, want %q", got, want)
	}
}

func TestDeleteAndSearch(t *testing.T) {
	m := testModel(t)
	m.Session.SelectedID = m.Doc.Root.Object[0].Value.ID
	runCmd(t, m, m.deleteSelected())
	if len(m.Doc.Root.Object) != 1 {
		t.Fatalf("object length = %d", len(m.Doc.Root.Object))
	}

	m.openPrompt(promptSearch, "/ ", "")
	m.prompt.SetValue("items")
	runCmd(t, m, m.submitPrompt())
	if len(m.Session.SearchHits) != 1 {
		t.Fatalf("SearchHits = %d", len(m.Session.SearchHits))
	}
}

func TestQuitWithDirtyRequiresForce(t *testing.T) {
	m := testModel(t)
	m.Session.Dirty = true
	updated, cmd := m.Update(key("q"))
	m = updated.(*Model)
	if cmd != nil {
		t.Fatal("cmd != nil, want nil")
	}
	if m.Session.Error == "" {
		t.Fatal("Error is empty")
	}
}

func TestPromptEscapes(t *testing.T) {
	m := testModel(t)
	m.openPrompt(promptCommand, ": ", "")
	updated, _ := m.Update(specialKey(tea.KeyEsc))
	m = updated.(*Model)
	if m.promptKind != promptNone {
		t.Fatalf("promptKind = %q", m.promptKind)
	}
}

func TestCopyShortcutAndCommandAliases(t *testing.T) {
	clipboard := &stubClipboard{}
	m := testModelWithOptions(t, ModelOptions{Clipboard: clipboard})
	m.Session.SelectedID = m.Doc.Root.Object[0].Value.ID

	updated, cmd := m.Update(key("y"))
	m = updated.(*Model)
	runCmd(t, m, cmd)
	if got, want := m.pendingPrefix, "y"; got != want {
		t.Fatalf("pendingPrefix = %q, want %q", got, want)
	}

	updated, cmd = m.Update(key("p"))
	m = updated.(*Model)
	runCmd(t, m, cmd)
	if got, want := clipboard.writes[len(clipboard.writes)-1], "$.name"; got != want {
		t.Fatalf("clipboard write = %q, want %q", got, want)
	}
	if got, want := m.Session.Status, "copied path to clipboard"; got != want {
		t.Fatalf("Status = %q, want %q", got, want)
	}
	if m.pendingPrefix != "" {
		t.Fatalf("pendingPrefix = %q, want empty", m.pendingPrefix)
	}

	runCmd(t, m, m.handleCommand("copy-value"))
	if got, want := clipboard.writes[len(clipboard.writes)-1], `"Ada"`; got != want {
		t.Fatalf("clipboard write = %q, want %q", got, want)
	}

	runCmd(t, m, m.handleCommand("copy-json"))
	if got, want := clipboard.writes[len(clipboard.writes)-1], "{\n  \"name\": \"Ada\",\n  \"items\": [\n    1,\n    2\n  ]\n}"; got != want {
		t.Fatalf("clipboard write = %q, want %q", got, want)
	}
}

func TestCopyKeyShortcutRequiresObjectKey(t *testing.T) {
	m := testModel(t)
	m.Session.SelectedID = m.Doc.Root.Object[0].Value.ID

	updated, cmd := m.Update(key("y"))
	m = updated.(*Model)
	runCmd(t, m, cmd)

	updated, cmd = m.Update(key("k"))
	m = updated.(*Model)
	runCmd(t, m, cmd)

	if got, want := m.Session.Status, "copied key to clipboard"; got != want {
		t.Fatalf("Status = %q, want %q", got, want)
	}

	m.Session.SelectedID = m.Doc.Root.ID
	runCmd(t, m, m.handleCommand("copy-key"))
	if got, want := m.Session.Error, "selected node does not have an object key"; got != want {
		t.Fatalf("Error = %q, want %q", got, want)
	}
}

func TestCopySubtreeAndClipboardFailure(t *testing.T) {
	clipboard := &stubClipboard{err: errors.New("clipboard offline")}
	m := testModelWithOptions(t, ModelOptions{Clipboard: clipboard})
	m.Session.SelectedID = m.Doc.Root.Object[0].Value.ID

	runCmd(t, m, m.handleCommand("copy-subtree"))
	if got, want := m.Session.Error, "could not copy to clipboard: clipboard offline"; got != want {
		t.Fatalf("Error = %q, want %q", got, want)
	}

	clipboard.err = nil
	runCmd(t, m, m.handleCommand("copy-subtree"))
	if got, want := clipboard.writes[len(clipboard.writes)-1], `"Ada"`; got != want {
		t.Fatalf("clipboard write = %q, want %q", got, want)
	}
}

func TestSettingsModalPreviewsWrapAndIndentWithoutSaving(t *testing.T) {
	m := testModel(t)

	updated, _ := m.Update(key("S"))
	m = updated.(*Model)
	updated, _ = m.Update(specialKey(tea.KeyDown))
	m = updated.(*Model)
	updated, _ = m.Update(specialKey(tea.KeyRight))
	m = updated.(*Model)

	if !m.ActiveSettings.WrapLongStrings {
		t.Fatal("ActiveSettings.WrapLongStrings = false, want true")
	}
	if m.Settings.WrapLongStrings {
		t.Fatal("Settings.WrapLongStrings = true, want false before save")
	}

	updated, _ = m.Update(specialKey(tea.KeyDown))
	m = updated.(*Model)
	updated, _ = m.Update(specialKey(tea.KeyRight))
	m = updated.(*Model)

	if got, want := m.ActiveSettings.SaveIndent.Label(), "spaces:3"; got != want {
		t.Fatalf("ActiveSettings.SaveIndent = %q, want %q", got, want)
	}
	if got, want := m.Settings.WithDefaults().SaveIndent.Label(), "spaces:2"; got != want {
		t.Fatalf("Settings.SaveIndent = %q, want %q before save", got, want)
	}

	updated, _ = m.Update(specialKey(tea.KeyEsc))
	m = updated.(*Model)

	if got, want := m.Session.Status, "kept preview settings for this session"; got != want {
		t.Fatalf("Status = %q, want %q", got, want)
	}
	if !m.ActiveSettings.WrapLongStrings {
		t.Fatal("ActiveSettings.WrapLongStrings = false after close, want true")
	}
}

func TestPrefixErrorsAndEscape(t *testing.T) {
	m := testModel(t)

	updated, cmd := m.Update(key("y"))
	m = updated.(*Model)
	runCmd(t, m, cmd)

	updated, cmd = m.Update(key("x"))
	m = updated.(*Model)
	runCmd(t, m, cmd)
	if got, want := m.Session.Error, "unknown shortcut: yx"; got != want {
		t.Fatalf("Error = %q, want %q", got, want)
	}
	if m.pendingPrefix != "" {
		t.Fatalf("pendingPrefix = %q, want empty", m.pendingPrefix)
	}

	updated, cmd = m.Update(key("z"))
	m = updated.(*Model)
	runCmd(t, m, cmd)
	updated, cmd = m.Update(specialKey(tea.KeyEsc))
	m = updated.(*Model)
	runCmd(t, m, cmd)
	if m.pendingPrefix != "" {
		t.Fatalf("pendingPrefix = %q, want empty after esc", m.pendingPrefix)
	}
	if m.Session.Error != "" {
		t.Fatalf("Error = %q, want empty after esc", m.Session.Error)
	}
}

func TestGoFoldAndJumpPrefixShortcuts(t *testing.T) {
	m := testModel(t)
	m.Session.MoveToBottom()

	updated, cmd := m.Update(key("g"))
	m = updated.(*Model)
	runCmd(t, m, cmd)
	if got, want := m.pendingPrefix, "g"; got != want {
		t.Fatalf("pendingPrefix = %q, want %q", got, want)
	}

	updated, cmd = m.Update(key("g"))
	m = updated.(*Model)
	runCmd(t, m, cmd)
	if got, want := m.Session.SelectedID, m.Doc.Root.ID; got != want {
		t.Fatalf("SelectedID = %d, want %d after gg", got, want)
	}
	if m.pendingPrefix != "" {
		t.Fatalf("pendingPrefix = %q, want empty after gg", m.pendingPrefix)
	}

	initialRows := len(m.Session.Rows)
	updated, cmd = m.Update(key("z"))
	m = updated.(*Model)
	runCmd(t, m, cmd)
	updated, cmd = m.Update(key("R"))
	m = updated.(*Model)
	runCmd(t, m, cmd)
	if got := len(m.Session.Rows); got <= initialRows {
		t.Fatalf("rows = %d, want more than %d after zR", got, initialRows)
	}

	updated, cmd = m.Update(key("z"))
	m = updated.(*Model)
	runCmd(t, m, cmd)
	updated, cmd = m.Update(key("M"))
	m = updated.(*Model)
	runCmd(t, m, cmd)
	if got, want := len(m.Session.Rows), initialRows; got != want {
		t.Fatalf("rows = %d, want %d after zM", got, want)
	}

	doc, err := document.Parse([]byte(`{"name":"Ada","items":[{"title":"alpha"},{"title":"beta"}],"meta":{"count":2}}`))
	if err != nil {
		t.Fatal(err)
	}
	m = NewModel(doc, source.Input{Kind: source.KindFile, Path: "sample.json"}, ModelOptions{
		Clipboard: &stubClipboard{},
	})

	runCmd(t, m, m.expandAll())
	m.Session.SelectedID = doc.Root.Object[1].Value.Array[0].Object[0].Value.ID

	updated, cmd = m.Update(key("]"))
	m = updated.(*Model)
	runCmd(t, m, cmd)
	if got, want := m.pendingPrefix, "]"; got != want {
		t.Fatalf("pendingPrefix = %q, want %q", got, want)
	}

	updated, cmd = m.Update(key("p"))
	m = updated.(*Model)
	runCmd(t, m, cmd)
	if got, want := m.Session.SelectedID, doc.Root.Object[1].Value.Array[1].ID; got != want {
		t.Fatalf("SelectedID = %d, want %d after ]p", got, want)
	}
	if got, want := m.Session.Status, "moved to next parent sibling"; got != want {
		t.Fatalf("Status = %q, want %q", got, want)
	}
}

func TestExpandCollapseAndNextParentSiblingCommands(t *testing.T) {
	doc, err := document.Parse([]byte(`{"name":"Ada","items":[{"title":"alpha"},{"title":"beta"}],"meta":{"count":2},"tail":true}`))
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(doc, source.Input{Kind: source.KindFile, Path: "sample.json"}, ModelOptions{
		Clipboard: &stubClipboard{},
	})

	runCmd(t, m, m.handleCommand("expand-all"))
	if got, want := len(m.Session.Rows), 10; got != want {
		t.Fatalf("rows = %d, want %d after expand-all", got, want)
	}

	titleID := doc.Root.Object[1].Value.Array[0].Object[0].Value.ID
	m.Session.SelectedID = titleID
	runCmd(t, m, m.handleCommand("next-parent-sibling"))
	if got, want := m.Session.SelectedID, doc.Root.Object[1].Value.Array[1].ID; got != want {
		t.Fatalf("SelectedID = %d, want %d", got, want)
	}
	if got, want := m.Session.Status, "moved to next parent sibling"; got != want {
		t.Fatalf("Status = %q, want %q", got, want)
	}

	lastBranchTitleID := doc.Root.Object[1].Value.Array[1].Object[0].Value.ID
	m.Session.SelectedID = lastBranchTitleID
	runCmd(t, m, m.handleCommand("next-parent-sibling"))
	if got, want := m.Session.SelectedID, doc.Root.Object[2].Value.ID; got != want {
		t.Fatalf("SelectedID = %d, want %d after climb", got, want)
	}

	m.Session.SelectedID = doc.Root.Object[3].Value.ID
	runCmd(t, m, m.handleCommand("next-parent-sibling"))
	if got, want := m.Session.Error, "no next parent sibling"; got != want {
		t.Fatalf("Error = %q, want %q", got, want)
	}

	runCmd(t, m, m.handleCommand("collapse-all"))
	if got, want := len(m.Session.Rows), 5; got != want {
		t.Fatalf("rows = %d, want %d after collapse-all", got, want)
	}
	if got, want := m.Session.Status, "collapsed all nodes"; got != want {
		t.Fatalf("Status = %q, want %q", got, want)
	}
}
