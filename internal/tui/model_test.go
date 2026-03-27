package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anyroad/lazy-json/internal/config"
	"github.com/anyroad/lazy-json/internal/document"
	"github.com/anyroad/lazy-json/internal/session"
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

func testLongArrayDoc(t *testing.T, count int) *document.Document {
	t.Helper()

	var raw strings.Builder
	raw.WriteString(`{"items":[`)
	for idx := 0; idx < count; idx++ {
		if idx > 0 {
			raw.WriteByte(',')
		}
		raw.WriteString(fmt.Sprintf("%d", idx))
	}
	raw.WriteString(`],"tail":true}`)

	doc, err := document.Parse([]byte(raw.String()))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func testLongObjectArrayDoc(t *testing.T, count int) *document.Document {
	t.Helper()

	var raw strings.Builder
	raw.WriteString(`{"items":[`)
	for idx := 0; idx < count; idx++ {
		if idx > 0 {
			raw.WriteByte(',')
		}
		raw.WriteString(fmt.Sprintf(`{"title":"item-%d"}`, idx))
	}
	raw.WriteString(`],"tail":true}`)

	doc, err := document.Parse([]byte(raw.String()))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func testFlatObjectDoc(t *testing.T, count int) *document.Document {
	t.Helper()

	var raw strings.Builder
	raw.WriteString("{")
	for idx := 0; idx < count; idx++ {
		if idx > 0 {
			raw.WriteByte(',')
		}
		raw.WriteString(fmt.Sprintf(`"item_%d":%d`, idx, idx))
	}
	raw.WriteString("}")

	doc, err := document.Parse([]byte(raw.String()))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func selectFirstBatch(t *testing.T, m *Model, arrayID document.NodeID) {
	t.Helper()

	m.Session.SelectNode(arrayID)
	m.Session.MoveInto(m.Doc)
	m.refresh()
	m.Session.MoveInto(m.Doc)

	row, ok := m.Session.CurrentRow()
	if !ok {
		t.Fatal("CurrentRow() = false after selecting first batch")
	}
	if !row.IsBatch() || row.BatchLabel() != "[0-99]" {
		t.Fatalf("CurrentRow() = %#v, want first batch row", row)
	}
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

type blockingJQRunner struct {
	started chan struct{}
	release chan struct{}
	stdout  []byte
	stderr  []byte
	err     error
}

func newBlockingJQRunner(stdout []byte, stderr []byte, err error) *blockingJQRunner {
	return &blockingJQRunner{
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
		stdout:  stdout,
		stderr:  stderr,
		err:     err,
	}
}

func (r *blockingJQRunner) Run(name string, args []string, stdin []byte) ([]byte, []byte, error) {
	select {
	case r.started <- struct{}{}:
	default:
	}
	<-r.release
	return r.stdout, r.stderr, r.err
}

func waitForChannelSignal[T any](t *testing.T, ch <-chan T, label string) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", label)
	}
	var zero T
	return zero
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

func applyMsg(t *testing.T, m *Model, msg tea.Msg) {
	t.Helper()
	updated, cmd := m.Update(msg)
	model, ok := updated.(*Model)
	if !ok {
		t.Fatalf("Update() returned %T", updated)
	}
	*m = *model
	runCmd(t, m, cmd)
}

func mustParseNode(t *testing.T, raw string) *document.Node {
	t.Helper()
	node, err := document.ParseNode([]byte(raw))
	if err != nil {
		t.Fatalf("ParseNode(%q) error = %v", raw, err)
	}
	return node
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

func TestModelInitSummaryStringAndMovePage(t *testing.T) {
	m := testModel(t)
	if cmd := m.Init(); cmd == nil {
		t.Fatal("Init() = nil, want blink command")
	}
	if got, want := m.Summary(), "3 rows"; got != want {
		t.Fatalf("Summary() = %q, want %q", got, want)
	}
	if got, want := m.String(), strings.TrimSpace(m.View()); got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}

	doc := testFlatObjectDoc(t, 20)
	m = NewModel(doc, source.Input{Kind: source.KindFile, Path: "sample.json"}, ModelOptions{
		Clipboard: &stubClipboard{},
	})
	m.Width = 120
	m.Height = 6
	m.movePage(1)

	row, ok := m.Session.CurrentRow()
	if !ok {
		t.Fatal("CurrentRow() = false after movePage")
	}
	if got, want := row.Index, 5; got != want {
		t.Fatalf("row.Index after movePage = %d, want %d", got, want)
	}
}

func TestPageNavigationShortcuts(t *testing.T) {
	doc := testFlatObjectDoc(t, 20)
	m := NewModel(doc, source.Input{Kind: source.KindFile, Path: "sample.json"}, ModelOptions{})
	m.Width = 100
	m.Height = 8

	applyMsg(t, m, specialKey(tea.KeyCtrlF))
	if got, want := m.Session.SelectedID, doc.Root.Object[6].Value.ID; got != want {
		t.Fatalf("SelectedID after ctrl+f = %d, want %d", got, want)
	}

	applyMsg(t, m, specialKey(tea.KeyPgDown))
	if got, want := m.Session.SelectedID, doc.Root.Object[13].Value.ID; got != want {
		t.Fatalf("SelectedID after pgdown = %d, want %d", got, want)
	}

	applyMsg(t, m, specialKey(tea.KeyCtrlB))
	if got, want := m.Session.SelectedID, doc.Root.Object[6].Value.ID; got != want {
		t.Fatalf("SelectedID after ctrl+b = %d, want %d", got, want)
	}

	applyMsg(t, m, specialKey(tea.KeyPgUp))
	if got, want := m.Session.SelectedID, doc.Root.ID; got != want {
		t.Fatalf("SelectedID after pgup = %d, want %d", got, want)
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

func TestUndoRedoScalarEditKeys(t *testing.T) {
	m := testModel(t)
	nameID := m.Doc.Root.Object[0].Value.ID
	m.Session.SelectNode(nameID)

	runCmd(t, m, m.startScalarEdit())
	m.prompt.SetValue(`"Grace"`)
	runCmd(t, m, m.submitPrompt())

	if got, want := m.Doc.Root.Object[0].Value.String, "Grace"; got != want {
		t.Fatalf("string after edit = %q, want %q", got, want)
	}
	if !m.Session.Dirty {
		t.Fatal("Dirty = false after edit, want true")
	}

	applyMsg(t, m, key("u"))
	if got, want := m.Doc.Root.Object[0].Value.String, "Ada"; got != want {
		t.Fatalf("string after undo = %q, want %q", got, want)
	}
	if got, want := m.Session.Status, "undid change"; got != want {
		t.Fatalf("Status after undo = %q, want %q", got, want)
	}
	if m.Session.Dirty {
		t.Fatal("Dirty = true after undo to initial state, want false")
	}

	applyMsg(t, m, specialKey(tea.KeyCtrlR))
	if got, want := m.Doc.Root.Object[0].Value.String, "Grace"; got != want {
		t.Fatalf("string after ctrl+r = %q, want %q", got, want)
	}
	if got, want := m.Session.Status, "redid change"; got != want {
		t.Fatalf("Status after ctrl+r = %q, want %q", got, want)
	}

	applyMsg(t, m, key("u"))
	applyMsg(t, m, key("U"))
	if got, want := m.Doc.Root.Object[0].Value.String, "Grace"; got != want {
		t.Fatalf("string after U = %q, want %q", got, want)
	}
	if got, want := m.Session.Status, "redid change"; got != want {
		t.Fatalf("Status after U = %q, want %q", got, want)
	}
}

func TestUndoRedoCommandsAndRedoClearing(t *testing.T) {
	m := testModel(t)
	nameID := m.Doc.Root.Object[0].Value.ID
	m.Session.SelectNode(nameID)

	runCmd(t, m, m.startScalarEdit())
	m.prompt.SetValue(`"Grace"`)
	runCmd(t, m, m.submitPrompt())

	runCmd(t, m, m.handleCommand("undo"))
	if got, want := m.Doc.Root.Object[0].Value.String, "Ada"; got != want {
		t.Fatalf("string after :undo = %q, want %q", got, want)
	}

	runCmd(t, m, m.startScalarEdit())
	m.prompt.SetValue(`"Marie"`)
	runCmd(t, m, m.submitPrompt())
	if got, want := m.Doc.Root.Object[0].Value.String, "Marie"; got != want {
		t.Fatalf("string after second edit = %q, want %q", got, want)
	}

	runCmd(t, m, m.handleCommand("redo"))
	if got, want := m.Session.Error, "nothing to redo"; got != want {
		t.Fatalf("Error after :redo = %q, want %q", got, want)
	}
	if got, want := m.Doc.Root.Object[0].Value.String, "Marie"; got != want {
		t.Fatalf("string after failed :redo = %q, want %q", got, want)
	}
}

func TestUndoRedoTracksSavedRevision(t *testing.T) {
	m := testModel(t)
	nameID := m.Doc.Root.Object[0].Value.ID
	m.Session.SelectNode(nameID)

	runCmd(t, m, m.startScalarEdit())
	m.prompt.SetValue(`"Grace"`)
	runCmd(t, m, m.submitPrompt())

	path := t.TempDir() + "/saved.json"
	if err := m.save(path); err != nil {
		t.Fatalf("save() error = %v", err)
	}
	if m.Session.Dirty {
		t.Fatal("Dirty = true after save, want false")
	}

	runCmd(t, m, m.startScalarEdit())
	m.prompt.SetValue(`"Marie"`)
	runCmd(t, m, m.submitPrompt())
	if !m.Session.Dirty {
		t.Fatal("Dirty = false after second edit, want true")
	}

	runCmd(t, m, m.handleCommand("undo"))
	if got, want := m.Doc.Root.Object[0].Value.String, "Grace"; got != want {
		t.Fatalf("string after undo to saved revision = %q, want %q", got, want)
	}
	if m.Session.Dirty {
		t.Fatal("Dirty = true after undo to saved revision, want false")
	}

	runCmd(t, m, m.handleCommand("redo"))
	if got, want := m.Doc.Root.Object[0].Value.String, "Marie"; got != want {
		t.Fatalf("string after redo from saved revision = %q, want %q", got, want)
	}
	if !m.Session.Dirty {
		t.Fatal("Dirty = false after redo away from saved revision, want true")
	}
}

func TestUndoRedoTracksSavedRevisionAcrossBranchEdits(t *testing.T) {
	m := testModel(t)
	nameID := m.Doc.Root.Object[0].Value.ID
	m.Session.SelectNode(nameID)

	runCmd(t, m, m.startScalarEdit())
	m.prompt.SetValue(`"Grace"`)
	runCmd(t, m, m.submitPrompt())

	path := t.TempDir() + "/saved.json"
	if err := m.save(path); err != nil {
		t.Fatalf("save() error = %v", err)
	}

	runCmd(t, m, m.handleCommand("undo"))
	if got, want := m.Doc.Root.Object[0].Value.String, "Ada"; got != want {
		t.Fatalf("string after undo away from saved revision = %q, want %q", got, want)
	}
	if !m.Session.Dirty {
		t.Fatal("Dirty = false after undo away from saved revision, want true")
	}

	runCmd(t, m, m.startScalarEdit())
	m.prompt.SetValue(`"Marie"`)
	runCmd(t, m, m.submitPrompt())

	if got, want := m.Doc.Root.Object[0].Value.String, "Marie"; got != want {
		t.Fatalf("string after branch edit = %q, want %q", got, want)
	}
	if !m.Session.Dirty {
		t.Fatal("Dirty = false after branch edit, want true")
	}
}

func TestUndoRedoRenameKey(t *testing.T) {
	m := testModel(t)
	nameID := m.Doc.Root.Object[0].Value.ID
	m.Session.SelectNode(nameID)

	runCmd(t, m, m.startRename())
	m.prompt.SetValue("full_name")
	runCmd(t, m, m.submitPrompt())

	if got, want := m.Doc.Root.Object[0].Key, "full_name"; got != want {
		t.Fatalf("key after rename = %q, want %q", got, want)
	}

	runCmd(t, m, m.undoChange())
	if got, want := m.Doc.Root.Object[0].Key, "name"; got != want {
		t.Fatalf("key after undo rename = %q, want %q", got, want)
	}

	runCmd(t, m, m.redoChange())
	if got, want := m.Doc.Root.Object[0].Key, "full_name"; got != want {
		t.Fatalf("key after redo rename = %q, want %q", got, want)
	}
}

func TestUndoRedoDeleteAndAddOperations(t *testing.T) {
	t.Run("delete", func(t *testing.T) {
		m := testModel(t)
		nameID := m.Doc.Root.Object[0].Value.ID
		m.Session.SelectNode(nameID)

		runCmd(t, m, m.deleteSelected())
		if got, want := len(m.Doc.Root.Object), 1; got != want {
			t.Fatalf("object length after delete = %d, want %d", got, want)
		}

		runCmd(t, m, m.undoChange())
		if got, want := len(m.Doc.Root.Object), 2; got != want {
			t.Fatalf("object length after undo delete = %d, want %d", got, want)
		}
		if got, want := m.Doc.Root.Object[0].Key, "name"; got != want {
			t.Fatalf("restored key = %q, want %q", got, want)
		}
		if got, want := m.Session.SelectedID, nameID; got != want {
			t.Fatalf("SelectedID after undo delete = %d, want %d", got, want)
		}

		runCmd(t, m, m.redoChange())
		if got, want := len(m.Doc.Root.Object), 1; got != want {
			t.Fatalf("object length after redo delete = %d, want %d", got, want)
		}
	})

	t.Run("object add", func(t *testing.T) {
		m := testModel(t)
		m.Session.SelectNode(m.Doc.Root.ID)

		runCmd(t, m, m.startAdd())
		m.prompt.SetValue(`active=true`)
		runCmd(t, m, m.submitPrompt())

		if got, want := len(m.Doc.Root.Object), 3; got != want {
			t.Fatalf("object length after add = %d, want %d", got, want)
		}
		if got, want := m.Doc.Root.Object[2].Key, "active"; got != want {
			t.Fatalf("added key = %q, want %q", got, want)
		}

		runCmd(t, m, m.undoChange())
		if got, want := len(m.Doc.Root.Object), 2; got != want {
			t.Fatalf("object length after undo add = %d, want %d", got, want)
		}

		runCmd(t, m, m.redoChange())
		if got, want := len(m.Doc.Root.Object), 3; got != want {
			t.Fatalf("object length after redo add = %d, want %d", got, want)
		}
		if got, want := m.Doc.Root.Object[2].Key, "active"; got != want {
			t.Fatalf("redo key = %q, want %q", got, want)
		}
	})

	t.Run("array add", func(t *testing.T) {
		m := testModel(t)
		itemsID := m.Doc.Root.Object[1].Value.ID
		m.Session.SelectNode(itemsID)

		runCmd(t, m, m.startAdd())
		m.prompt.SetValue(`3`)
		runCmd(t, m, m.submitPrompt())

		if got, want := len(m.Doc.Root.Object[1].Value.Array), 3; got != want {
			t.Fatalf("array length after add = %d, want %d", got, want)
		}
		if got, want := m.Doc.Root.Object[1].Value.Array[2].Number, "3"; got != want {
			t.Fatalf("added number = %q, want %q", got, want)
		}

		runCmd(t, m, m.undoChange())
		if got, want := len(m.Doc.Root.Object[1].Value.Array), 2; got != want {
			t.Fatalf("array length after undo add = %d, want %d", got, want)
		}

		runCmd(t, m, m.redoChange())
		if got, want := len(m.Doc.Root.Object[1].Value.Array), 3; got != want {
			t.Fatalf("array length after redo add = %d, want %d", got, want)
		}
		if got, want := m.Doc.Root.Object[1].Value.Array[2].Number, "3"; got != want {
			t.Fatalf("redo number = %q, want %q", got, want)
		}
	})

	t.Run("array delete", func(t *testing.T) {
		m := testModel(t)
		items := m.Doc.Root.Object[1].Value
		deletedID := items.Array[1].ID
		if !m.Session.RevealNode(m.Doc, deletedID) {
			t.Fatalf("RevealNode(%d) = false", deletedID)
		}

		runCmd(t, m, m.deleteSelected())
		if got, want := len(items.Array), 1; got != want {
			t.Fatalf("array length after delete = %d, want %d", got, want)
		}
		if got, want := items.Array[0].Number, "1"; got != want {
			t.Fatalf("remaining array item = %q, want %q", got, want)
		}

		runCmd(t, m, m.undoChange())
		if got, want := len(items.Array), 2; got != want {
			t.Fatalf("array length after undo delete = %d, want %d", got, want)
		}
		if got, want := items.Array[1].ID, deletedID; got != want {
			t.Fatalf("restored array item ID = %d, want %d", got, want)
		}
		if got, want := m.Session.SelectedID, deletedID; got != want {
			t.Fatalf("SelectedID after undo delete = %d, want %d", got, want)
		}

		runCmd(t, m, m.redoChange())
		if got, want := len(items.Array), 1; got != want {
			t.Fatalf("array length after redo delete = %d, want %d", got, want)
		}
		if got, want := items.Array[0].Number, "1"; got != want {
			t.Fatalf("remaining array item after redo = %q, want %q", got, want)
		}
	})
}

func TestUndoRedoEmptyHistoryErrors(t *testing.T) {
	m := testModel(t)

	runCmd(t, m, m.undoChange())
	if got, want := m.Session.Error, "nothing to undo"; got != want {
		t.Fatalf("Error after undo = %q, want %q", got, want)
	}

	runCmd(t, m, m.redoChange())
	if got, want := m.Session.Error, "nothing to redo"; got != want {
		t.Fatalf("Error after redo = %q, want %q", got, want)
	}
}

func TestUndoRedoAsyncCompletionMessages(t *testing.T) {
	t.Run("external editor", func(t *testing.T) {
		m := testModel(t)
		targetID := m.Doc.Root.Object[0].Value.ID
		m.Session.SelectNode(targetID)

		applyMsg(t, m, editorFinishedMsg{
			Node:            mustParseNode(t, `"Grace"`),
			TargetID:        targetID,
			Before:          document.CloneNode(m.Doc.Root.Object[0].Value),
			BeforeSelection: m.Session.SelectedRow(),
		})

		if got, want := m.Doc.Root.Object[0].Value.String, "Grace"; got != want {
			t.Fatalf("string after editor msg = %q, want %q", got, want)
		}

		runCmd(t, m, m.undoChange())
		if got, want := m.Doc.Root.Object[0].Value.String, "Ada"; got != want {
			t.Fatalf("string after undo editor msg = %q, want %q", got, want)
		}
	})

	t.Run("jq blocks input until completion", func(t *testing.T) {
		m := testModel(t)
		runner := newBlockingJQRunner([]byte(`{"name":"Ada","items":[1,2],"active":true}`), nil, nil)
		m.JQRunner = runner
		beforeSelection := m.Session.SelectedID

		cmd := m.handleCommand(`jq . + {"active":true}`)
		if cmd == nil {
			t.Fatal("handleCommand(jq) returned nil cmd")
		}
		if got, want := m.Session.Mode, session.ModeBusy; got != want {
			t.Fatalf("Mode after jq start = %q, want %q", got, want)
		}
		if got, want := m.Session.Status, "running jq..."; got != want {
			t.Fatalf("Status after jq start = %q, want %q", got, want)
		}

		msgCh := make(chan tea.Msg, 1)
		go func() {
			msgCh <- cmd()
		}()
		waitForChannelSignal(t, runner.started, "jq runner start")

		updated, next := m.Update(key("j"))
		if next != nil {
			t.Fatal("busy Update(j) returned unexpected cmd")
		}
		m = updated.(*Model)
		if got, want := m.Session.SelectedID, beforeSelection; got != want {
			t.Fatalf("SelectedID while jq busy = %d, want %d", got, want)
		}
		if got, want := m.Session.Mode, session.ModeBusy; got != want {
			t.Fatalf("Mode while jq busy = %q, want %q", got, want)
		}

		close(runner.release)
		applyMsg(t, m, waitForChannelSignal(t, msgCh, "jq completion"))

		if got, want := len(m.Doc.Root.Object), 3; got != want {
			t.Fatalf("root object length after jq msg = %d, want %d", got, want)
		}
		if got, want := m.Session.Mode, session.ModeNormal; got != want {
			t.Fatalf("Mode after jq completion = %q, want %q", got, want)
		}
		if got, want := m.Session.Status, "applied jq transform"; got != want {
			t.Fatalf("Status after jq completion = %q, want %q", got, want)
		}

		runCmd(t, m, m.undoChange())
		if got, want := len(m.Doc.Root.Object), 2; got != want {
			t.Fatalf("root object length after undo jq msg = %d, want %d", got, want)
		}
	})
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

func TestSaveAndQuitAndPrintDirectHelpers(t *testing.T) {
	t.Run("stdin writes exit output", func(t *testing.T) {
		doc, err := document.Parse([]byte(`{"name":"Ada"}`))
		if err != nil {
			t.Fatal(err)
		}
		m := NewModel(doc, source.Input{Kind: source.KindStdin}, ModelOptions{
			Clipboard: &stubClipboard{},
		})

		cmd := m.saveAndQuit("")
		if cmd == nil || cmd() == nil {
			t.Fatal("saveAndQuit() = nil, want quit command")
		}
		if got := string(m.ExitOutput); !strings.Contains(got, `"name": "Ada"`) {
			t.Fatalf("ExitOutput = %q, want formatted JSON", got)
		}
	})

	t.Run("explicit path saves file", func(t *testing.T) {
		m := testModel(t)
		path := filepath.Join(t.TempDir(), "saved.json")

		cmd := m.saveAndQuit(path)
		if cmd == nil || cmd() == nil {
			t.Fatal("saveAndQuit(path) = nil, want quit command")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", path, err)
		}
		if got := string(data); !strings.Contains(got, `"name": "Ada"`) {
			t.Fatalf("saved file = %q, want JSON content", got)
		}
		if got, want := m.Session.SourcePath, path; got != want {
			t.Fatalf("SourcePath = %q, want %q", got, want)
		}
	})

	t.Run("print writes exit output", func(t *testing.T) {
		m := testModel(t)
		cmd := m.printAndQuit()
		if cmd == nil || cmd() == nil {
			t.Fatal("printAndQuit() = nil, want quit command")
		}
		if got := string(m.ExitOutput); !strings.Contains(got, `"items": [`) {
			t.Fatalf("ExitOutput = %q, want pretty JSON", got)
		}
	})
}

func TestHandleCommandUnknownWritePathAndForceQuit(t *testing.T) {
	m := testModel(t)

	runCmd(t, m, m.handleCommand("unknown"))
	if got, want := m.Session.Error, "unknown command: :unknown"; got != want {
		t.Fatalf("Error = %q, want %q", got, want)
	}

	path := filepath.Join(t.TempDir(), "saved.json")
	runCmd(t, m, m.handleCommand("w "+path))
	if got, want := m.Session.Status, "saved"; got != want {
		t.Fatalf("Status = %q, want %q", got, want)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("saved file stat error = %v", err)
	}

	if cmd := m.handleCommand("q!"); cmd == nil || cmd() == nil {
		t.Fatal("handleCommand(q!) = nil, want quit command")
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
	if got, want := len(m.Session.SearchHits), 3; got != want {
		t.Fatalf("SearchHits = %d, want %d", got, want)
	}
}

func TestSearchRevealsCollapsedMatchOnSubmitAndNavigate(t *testing.T) {
	m := testModel(t)

	m.openPrompt(promptSearch, "/ ", "")
	m.prompt.SetValue("[0]")
	runCmd(t, m, m.submitPrompt())

	firstItemID := m.Doc.Root.Object[1].Value.Array[0].ID
	if got, want := m.Session.SelectedID, firstItemID; got != want {
		t.Fatalf("SelectedID = %d, want %d", got, want)
	}
	if !m.Session.Expanded[m.Doc.Root.Object[1].Value.ID] {
		t.Fatal("items container is not expanded after search submit")
	}

	updated, _ := m.Update(key("N"))
	m = updated.(*Model)
	if got, want := m.Session.SelectedID, firstItemID; got != want {
		t.Fatalf("SelectedID = %d, want %d after N", got, want)
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

func TestApplyJQAndEditExternalValidationErrors(t *testing.T) {
	t.Run("empty jq expression", func(t *testing.T) {
		m := testModel(t)
		runCmd(t, m, m.applyJQ("", false))
		if got, want := m.Session.Error, "jq expression cannot be empty"; got != want {
			t.Fatalf("Error = %q, want %q", got, want)
		}
	})

	t.Run("batch row cannot run subtree jq", func(t *testing.T) {
		doc := testLongArrayDoc(t, 101)
		m := NewModel(doc, source.Input{Kind: source.KindFile, Path: "sample.json"}, ModelOptions{
			Clipboard: &stubClipboard{},
		})
		items := doc.Root.Object[0].Value
		selectFirstBatch(t, m, items.ID)

		runCmd(t, m, m.applyJQ(". + 1", true))
		if got, want := m.Session.Error, "batch rows are navigation only"; got != want {
			t.Fatalf("Error = %q, want %q", got, want)
		}
	})

	t.Run("external edit requires editor", func(t *testing.T) {
		m := testModel(t)
		m.Session.SelectNode(m.Doc.Root.Object[0].Value.ID)
		t.Setenv("EDITOR", "")

		runCmd(t, m, m.editExternal())
		if got, want := m.Session.Error, "$EDITOR is not set"; got != want {
			t.Fatalf("Error = %q, want %q", got, want)
		}
	})
}

func TestSettingsModalPreviewsWrapAndIndentWithoutSaving(t *testing.T) {
	m := testModel(t)

	updated, _ := m.Update(key("S"))
	m = updated.(*Model)
	updated, _ = m.Update(specialKey(tea.KeyDown))
	m = updated.(*Model)
	updated, _ = m.Update(specialKey(tea.KeyDown))
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

	m.Session.SelectedID = doc.Root.Object[1].Value.ID
	updated, cmd = m.Update(key("z"))
	m = updated.(*Model)
	runCmd(t, m, cmd)
	updated, cmd = m.Update(key("a"))
	m = updated.(*Model)
	runCmd(t, m, cmd)
	if got, want := len(m.Session.Rows), 8; got != want {
		t.Fatalf("rows = %d, want %d after za", got, want)
	}
	if got, want := m.Session.Status, "expanded array elements one level"; got != want {
		t.Fatalf("Status = %q, want %q after za", got, want)
	}
	m.Session.SelectedID = doc.Root.Object[1].Value.Array[0].Object[0].Value.ID
	updated, cmd = m.Update(key("z"))
	m = updated.(*Model)
	runCmd(t, m, cmd)
	updated, cmd = m.Update(key("A"))
	m = updated.(*Model)
	runCmd(t, m, cmd)
	if got, want := len(m.Session.Rows), 6; got != want {
		t.Fatalf("rows = %d, want %d after zA", got, want)
	}
	if got, want := m.Session.SelectedID, doc.Root.Object[1].Value.Array[0].ID; got != want {
		t.Fatalf("SelectedID = %d, want %d after zA", got, want)
	}
	if got, want := m.Session.Status, "collapsed array elements"; got != want {
		t.Fatalf("Status = %q, want %q after zA", got, want)
	}

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

func TestExpandArrayElementsOneLevelRequiresArrayContext(t *testing.T) {
	m := testModel(t)

	runCmd(t, m, m.expandArrayElementsOneLevel())

	if got, want := m.Session.Error, "no array target available"; got != want {
		t.Fatalf("Error = %q, want %q", got, want)
	}
}

func TestExpandArrayElementsOneLevelDefaultsToFirstLongArrayBatch(t *testing.T) {
	doc := testLongObjectArrayDoc(t, 150)
	m := NewModel(doc, source.Input{Kind: source.KindFile, Path: "sample.json"}, ModelOptions{
		Clipboard: &stubClipboard{},
	})
	items := doc.Root.Object[0].Value
	m.Session.SelectedID = items.ID

	runCmd(t, m, m.expandArrayElementsOneLevel())

	firstBatch := session.BatchRowID(items.ID, 0)
	secondBatch := session.BatchRowID(items.ID, 100)
	if !m.Session.ExpandedBatches[firstBatch] {
		t.Fatalf("ExpandedBatches[%v] = false, want true", firstBatch)
	}
	if m.Session.ExpandedBatches[secondBatch] {
		t.Fatalf("ExpandedBatches[%v] = true, want false", secondBatch)
	}

	firstTitleID := items.Array[0].Object[0].Value.ID
	lastTitleID := items.Array[149].Object[0].Value.ID
	if _, ok := m.Session.RowIndex[firstTitleID]; !ok {
		t.Fatal("first nested title row is not visible after za")
	}
	if _, ok := m.Session.RowIndex[lastTitleID]; ok {
		t.Fatal("last nested title row is visible, want hidden outside first batch")
	}
	if got, want := m.Session.Status, "expanded array elements one level"; got != want {
		t.Fatalf("Status = %q, want %q after za on long array", got, want)
	}
}

func TestExpandArrayElementsOneLevelUsesSelectedLongArrayBatch(t *testing.T) {
	doc := testLongObjectArrayDoc(t, 150)
	m := NewModel(doc, source.Input{Kind: source.KindFile, Path: "sample.json"}, ModelOptions{
		Clipboard: &stubClipboard{},
	})
	items := doc.Root.Object[0].Value
	m.Session.Expanded[items.ID] = true
	m.refresh()
	m.Session.SelectRowID(session.BatchRowID(items.ID, 100))
	m.refresh()

	runCmd(t, m, m.expandArrayElementsOneLevel())

	firstBatch := session.BatchRowID(items.ID, 0)
	secondBatch := session.BatchRowID(items.ID, 100)
	if m.Session.ExpandedBatches[firstBatch] {
		t.Fatalf("ExpandedBatches[%v] = true, want false", firstBatch)
	}
	if !m.Session.ExpandedBatches[secondBatch] {
		t.Fatalf("ExpandedBatches[%v] = false, want true", secondBatch)
	}

	firstTitleID := items.Array[0].Object[0].Value.ID
	lastTitleID := items.Array[149].Object[0].Value.ID
	if _, ok := m.Session.RowIndex[firstTitleID]; ok {
		t.Fatal("first nested title row is visible, want hidden outside selected batch")
	}
	if _, ok := m.Session.RowIndex[lastTitleID]; !ok {
		t.Fatal("last nested title row is not visible after za on selected batch")
	}
}

func TestCollapseArrayElementsRequiresArrayContext(t *testing.T) {
	m := testModel(t)

	runCmd(t, m, m.collapseArrayElements())

	if got, want := m.Session.Error, "no array target available"; got != want {
		t.Fatalf("Error = %q, want %q", got, want)
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

func TestBatchRowsRejectNodeOnlyCommands(t *testing.T) {
	doc := testLongArrayDoc(t, 101)
	m := NewModel(doc, source.Input{Kind: source.KindFile, Path: "sample.json"}, ModelOptions{
		Clipboard: &stubClipboard{},
	})
	items := doc.Root.Object[0].Value

	selectFirstBatch(t, m, items.ID)

	runCmd(t, m, m.handleCommand("copy-path"))
	if got, want := m.Session.Error, "batch rows do not have a JSON path"; got != want {
		t.Fatalf("Error = %q, want %q", got, want)
	}

	runCmd(t, m, m.deleteSelected())
	if got, want := m.Session.Error, "batch rows are navigation only"; got != want {
		t.Fatalf("Error = %q, want %q", got, want)
	}

	runCmd(t, m, m.startScalarEdit())
	if got, want := m.Session.Error, "batch rows are navigation only"; got != want {
		t.Fatalf("Error = %q, want %q", got, want)
	}

	runCmd(t, m, m.startRename())
	if got, want := m.Session.Error, "batch rows are navigation only"; got != want {
		t.Fatalf("Error = %q, want %q", got, want)
	}

	runCmd(t, m, m.copyValue())
	if got, want := m.Session.Error, "batch rows are navigation only"; got != want {
		t.Fatalf("Error = %q, want %q", got, want)
	}

	runCmd(t, m, m.copySubtree())
	if got, want := m.Session.Error, "batch rows are navigation only"; got != want {
		t.Fatalf("Error = %q, want %q", got, want)
	}

	runCmd(t, m, m.editExternal())
	if got, want := m.Session.Error, "batch rows are navigation only"; got != want {
		t.Fatalf("Error = %q, want %q", got, want)
	}
}

func TestAddArrayItemFromBatchRowTargetsParentArray(t *testing.T) {
	doc := testLongArrayDoc(t, 101)
	m := NewModel(doc, source.Input{Kind: source.KindFile, Path: "sample.json"}, ModelOptions{
		Clipboard: &stubClipboard{},
	})
	items := doc.Root.Object[0].Value

	selectFirstBatch(t, m, items.ID)

	runCmd(t, m, m.startAdd())
	if got, want := m.promptKind, promptAddArray; got != want {
		t.Fatalf("promptKind = %q, want %q", got, want)
	}

	m.prompt.SetValue("999")
	runCmd(t, m, m.submitPrompt())

	if got, want := len(items.Array), 102; got != want {
		t.Fatalf("len(items.Array) = %d, want %d", got, want)
	}
	if got, want := items.Array[len(items.Array)-1].Number, "999"; got != want {
		t.Fatalf("last item = %q, want %q", got, want)
	}
	if got, want := m.Session.Status, "added array item"; got != want {
		t.Fatalf("Status = %q, want %q", got, want)
	}

	row, ok := m.Session.CurrentRow()
	if !ok {
		t.Fatal("CurrentRow() = false after appending from batch row")
	}
	if row.IsBatch() || row.ArrayIndex != 101 {
		t.Fatalf("CurrentRow() = %#v, want new array item row", row)
	}

	secondBatch := session.BatchRowID(items.ID, 100)
	if !m.Session.ExpandedBatches[secondBatch] {
		t.Fatalf("ExpandedBatches[%v] = false, want true for second batch", secondBatch)
	}
}

func TestUndoRedoBatchRowAppendRestoresSelection(t *testing.T) {
	doc := testLongArrayDoc(t, 101)
	m := NewModel(doc, source.Input{Kind: source.KindFile, Path: "sample.json"}, ModelOptions{
		Clipboard: &stubClipboard{},
	})
	items := doc.Root.Object[0].Value

	selectFirstBatch(t, m, items.ID)
	beforeRow, ok := m.Session.CurrentRow()
	if !ok {
		t.Fatal("CurrentRow() = false before batch append")
	}

	runCmd(t, m, m.startAdd())
	m.prompt.SetValue("999")
	runCmd(t, m, m.submitPrompt())

	if got, want := len(items.Array), 102; got != want {
		t.Fatalf("len(items.Array) after add = %d, want %d", got, want)
	}

	runCmd(t, m, m.undoChange())
	if got, want := len(items.Array), 101; got != want {
		t.Fatalf("len(items.Array) after undo = %d, want %d", got, want)
	}

	row, ok := m.Session.CurrentRow()
	if !ok {
		t.Fatal("CurrentRow() = false after undo batch append")
	}
	if !row.IsBatch() || row.BatchLabel() != beforeRow.BatchLabel() {
		t.Fatalf("CurrentRow() after undo = %#v, want original batch row", row)
	}

	runCmd(t, m, m.redoChange())
	if got, want := len(items.Array), 102; got != want {
		t.Fatalf("len(items.Array) after redo = %d, want %d", got, want)
	}

	row, ok = m.Session.CurrentRow()
	if !ok {
		t.Fatal("CurrentRow() = false after redo batch append")
	}
	if row.IsBatch() || row.ArrayIndex != 101 {
		t.Fatalf("CurrentRow() after redo = %#v, want restored appended row", row)
	}
}
