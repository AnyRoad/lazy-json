package tui

import (
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
	doc, err := document.Parse([]byte(`{"name":"Ada","items":[1,2]}`))
	if err != nil {
		t.Fatal(err)
	}
	return NewModel(doc, source.Input{Kind: source.KindFile, Path: "sample.json"}, options)
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
		Settings:      config.Settings{Theme: "ember"},
	})
	updated, _ := m.Update(key("j"))
	m = updated.(*Model)
	if m.Session.SelectedID != m.Doc.Root.Object[0].Value.ID {
		t.Fatalf("SelectedID = %d", m.Session.SelectedID)
	}
	updated, _ = m.Update(key("t"))
	m = updated.(*Model)
	if m.Session.ThemeName != "mist" {
		t.Fatalf("ThemeName = %q", m.Session.ThemeName)
	}
	if got, want := m.Settings.Theme, "ember"; got != want {
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
		Settings:      config.Settings{Theme: "ember"},
	})

	runCmd(t, m, m.handleCommand("theme"))

	if got, want := m.Session.ThemeName, "mist"; got != want {
		t.Fatalf("ThemeName = %q, want %q", got, want)
	}
	if got, want := m.Session.Status, "previewing theme mist"; got != want {
		t.Fatalf("Status = %q, want %q", got, want)
	}
	if got, want := m.Settings.Theme, "ember"; got != want {
		t.Fatalf("Settings.Theme = %q, want %q", got, want)
	}
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
