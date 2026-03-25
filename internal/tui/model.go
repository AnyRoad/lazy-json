package tui

import (
	"fmt"
	"strings"

	textinput "github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/anyroad/lazy-json/internal/config"
	"github.com/anyroad/lazy-json/internal/document"
	"github.com/anyroad/lazy-json/internal/integration"
	"github.com/anyroad/lazy-json/internal/session"
	"github.com/anyroad/lazy-json/internal/source"
)

type ModelOptions struct {
	ThemeRegistry       ThemeRegistry
	Settings            config.Settings
	SettingsPath        string
	SettingsFilePresent bool
	SettingsPersisted   bool
	Warnings            []string
	Clipboard           integration.Clipboard
}

type Model struct {
	Doc                 *document.Document
	Session             *session.Session
	ThemeRegistry       ThemeRegistry
	Settings            config.Settings
	ActiveSettings      config.Settings
	SettingsPath        string
	SettingsFilePresent bool
	SettingsPersisted   bool
	StartupWarning      string
	Width               int
	Height              int
	prompt              textinput.Model
	promptKind          promptKind
	pendingPrefix       string
	settingsRow         int
	undoStack           []historyStateEntry
	redoStack           []historyStateEntry
	nextRevision        historyRevisionID
	currentRevision     historyRevisionID
	savedRevision       historyRevisionID
	ExitOutput          []byte
	JQRunner            integration.JQRunner
	Clipboard           integration.Clipboard
}

type prefixMenu struct {
	Tag   string
	Items []prefixMenuItem
}

type prefixMenuItem struct {
	Key   string
	Label string
	Run   func(*Model) tea.Cmd
}

func (m prefixMenu) item(key string) (prefixMenuItem, bool) {
	for _, item := range m.Items {
		if item.Key == key {
			return item, true
		}
	}
	return prefixMenuItem{}, false
}

var prefixMenus = map[string]prefixMenu{
	"g": {
		Tag: "g:go",
		Items: []prefixMenuItem{
			{
				Key:   "g",
				Label: "top",
				Run: func(m *Model) tea.Cmd {
					m.Session.MoveToTop()
					return nil
				},
			},
		},
	},
	"y": {
		Tag: "y:copy",
		Items: []prefixMenuItem{
			{Key: "p", Label: "path", Run: func(m *Model) tea.Cmd { return m.copyPath() }},
			{Key: "k", Label: "key", Run: func(m *Model) tea.Cmd { return m.copyKey() }},
			{Key: "v", Label: "value", Run: func(m *Model) tea.Cmd { return m.copyValue() }},
			{Key: "s", Label: "subtree", Run: func(m *Model) tea.Cmd { return m.copySubtree() }},
			{Key: "j", Label: "json", Run: func(m *Model) tea.Cmd { return m.copyDocument() }},
		},
	},
	"z": {
		Tag: "z:fold",
		Items: []prefixMenuItem{
			{Key: "R", Label: "expand-all", Run: func(m *Model) tea.Cmd { return m.expandAll() }},
			{Key: "M", Label: "collapse-all", Run: func(m *Model) tea.Cmd { return m.collapseAll() }},
			{Key: "a", Label: "array+1", Run: func(m *Model) tea.Cmd { return m.expandArrayElementsOneLevel() }},
			{Key: "A", Label: "array-1", Run: func(m *Model) tea.Cmd { return m.collapseArrayElements() }},
		},
	},
	"]": {
		Tag: "jump",
		Items: []prefixMenuItem{
			{Key: "p", Label: "parent-sibling", Run: func(m *Model) tea.Cmd { return m.nextParentSibling() }},
		},
	},
}

func NewModel(doc *document.Document, src source.Input, options ModelOptions) *Model {
	opts := options.withDefaults()

	model := &Model{
		Doc:                 doc,
		Session:             session.New(doc, src, opts.Settings.Theme),
		ThemeRegistry:       opts.ThemeRegistry,
		Settings:            opts.Settings,
		ActiveSettings:      opts.Settings,
		SettingsPath:        opts.SettingsPath,
		SettingsFilePresent: opts.SettingsFilePresent,
		SettingsPersisted:   opts.SettingsPersisted,
		prompt:              newPrompt(),
		promptKind:          promptNone,
		Clipboard:           opts.Clipboard,
	}
	if message := startupWarningMessage(opts.Warnings); message != "" {
		model.StartupWarning = message
		model.Session.SetStatus(message)
	}
	model.syncDirty()
	return model
}

func (o ModelOptions) withDefaults() ModelOptions {
	if !o.ThemeRegistry.hasThemes {
		o.ThemeRegistry = BuiltinThemeRegistry()
	}
	o.Settings = o.Settings.WithDefaults()
	o.Settings.Theme = o.ThemeRegistry.ThemeByName(o.Settings.Theme).Name
	if o.Clipboard == nil {
		o.Clipboard = integration.SystemClipboard{}
	}
	return o
}

func startupWarningMessage(warnings []string) string {
	if len(warnings) == 0 {
		return ""
	}
	return "warning: " + strings.Join(warnings, "; ")
}

func (m *Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil
	case editorFinishedMsg:
		if msg.Err != nil {
			m.Session.SetError(msg.Err.Error())
			return m, nil
		}
		if err := m.Doc.Replace(msg.TargetID, msg.Node); err != nil {
			m.Session.SetError(err.Error())
			return m, nil
		}
		after, err := m.cloneNode(msg.TargetID)
		if err != nil {
			m.Session.SetError(err.Error())
			return m, nil
		}
		m.Session.SelectNode(msg.TargetID)
		m.refresh()
		m.pushHistory(newReplaceHistoryEntry(msg.TargetID, msg.Before, after, msg.BeforeSelection, m.Session.SelectedRow()))
		m.Session.SetStatus("updated node from external editor")
		return m, nil
	case jqFinishedMsg:
		m.finishBusy()
		if msg.Err != nil {
			m.Session.SetError(msg.Err.Error())
			return m, nil
		}
		if err := m.Doc.Replace(msg.TargetID, msg.Node); err != nil {
			m.Session.SetError(err.Error())
			return m, nil
		}
		after, err := m.cloneNode(msg.TargetID)
		if err != nil {
			m.Session.SetError(err.Error())
			return m, nil
		}
		m.Session.SelectNode(msg.TargetID)
		m.refresh()
		m.pushHistory(newReplaceHistoryEntry(msg.TargetID, msg.Before, after, msg.BeforeSelection, m.Session.SelectedRow()))
		m.Session.SetStatus("applied jq transform")
		return m, nil
	case tea.KeyMsg:
		if m.busy() {
			return m.updateBusy(msg)
		}
		if m.Session.Help {
			switch msg.String() {
			case "?", "esc", "q":
				m.Session.Help = false
			}
			return m, nil
		}
		if m.settingsOpen() {
			return m.updateSettings(msg)
		}
		switch m.promptKind {
		case promptCommand, promptSearch, promptEditScalar, promptRenameKey, promptAddObject, promptAddArray:
			return m.updatePrompt(msg)
		default:
			return m.updateNormal(msg)
		}
	}
	return m, nil
}

func (m *Model) updatePrompt(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.closePrompt()
		return m, nil
	case "enter":
		return m, m.submitPrompt()
	}
	var cmd tea.Cmd
	m.prompt, cmd = m.prompt.Update(msg)
	return m, cmd
}

func (m *Model) updateBusy(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *Model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.Session.ClearMessages()
	key := msg.String()
	if key == "ctrl+c" {
		m.clearPendingPrefix()
		if m.Session.Dirty {
			m.Session.SetError("unsaved changes; use :x, :w, or :q!")
			return m, nil
		}
		return m, tea.Quit
	}
	if key == "esc" && m.pendingPrefix != "" {
		m.clearPendingPrefix()
		return m, nil
	}
	if m.pendingPrefix != "" {
		return m.resolvePendingPrefix(key)
	}
	return m.handleNormalKey(key)
}

func (m *Model) handleNormalKey(key string) (tea.Model, tea.Cmd) {
	if _, ok := prefixMenuForKey(key); ok {
		m.pendingPrefix = key
		return m, nil
	}
	switch key {
	case "j", "down":
		m.Session.Move(1)
	case "k", "up":
		m.Session.Move(-1)
	case "h", "left":
		m.Session.CollapseSelected(m.Doc)
		m.refresh()
	case "l", "right":
		m.Session.MoveInto(m.Doc)
		m.refresh()
	case "G":
		m.Session.MoveToBottom()
	case "/":
		m.openPrompt(promptSearch, "/ ", m.Session.Search.Query)
	case ":":
		m.openPrompt(promptCommand, ": ", "")
	case "e":
		return m, m.startScalarEdit()
	case "E":
		return m, m.editExternal()
	case "a":
		return m, m.startAdd()
	case "r":
		return m, m.startRename()
	case "d":
		return m, m.deleteSelected()
	case "u":
		return m, m.undoChange()
	case "U", "ctrl+r":
		return m, m.redoChange()
	case "n":
		m.Session.NextSearchHit(m.Doc, false)
	case "N":
		m.Session.NextSearchHit(m.Doc, true)
	case "t":
		m.cycleTheme()
	case "S":
		m.openSettings()
	case "?":
		m.clearPendingPrefix()
		m.Session.Help = true
	case "q", "ctrl+c":
		if m.Session.Dirty {
			m.Session.SetError("unsaved changes; use :x, :w, or :q!")
			return m, nil
		}
		return m, tea.Quit
	}
	return m, nil
}

func (m *Model) busy() bool {
	return m.Session != nil && m.Session.Mode == session.ModeBusy
}

func (m *Model) startBusy(status string) {
	if m.Session == nil {
		return
	}
	m.clearPendingPrefix()
	m.Session.Mode = session.ModeBusy
	m.Session.SetStatus(status)
}

func (m *Model) finishBusy() {
	if m.Session == nil || m.Session.Mode != session.ModeBusy {
		return
	}
	m.Session.Mode = session.ModeNormal
}

func (m *Model) resolvePendingPrefix(key string) (tea.Model, tea.Cmd) {
	prefix := m.pendingPrefix
	m.clearPendingPrefix()

	if menu, ok := prefixMenuForKey(prefix); ok {
		if item, ok := menu.item(key); ok {
			return m, item.Run(m)
		}
	}

	m.Session.SetError("unknown shortcut: " + prefix + key)
	return m, nil
}

func prefixMenuForKey(key string) (prefixMenu, bool) {
	menu, ok := prefixMenus[key]
	return menu, ok
}

func (m *Model) pendingPrefixMenu() (prefixMenu, bool) {
	if m.pendingPrefix == "" {
		return prefixMenu{}, false
	}
	return prefixMenuForKey(m.pendingPrefix)
}

func (m *Model) clearPendingPrefix() {
	m.pendingPrefix = ""
}

func (m *Model) cycleTheme() {
	next := m.ThemeRegistry.NextTheme(m.Session.ThemeName)
	m.Session.ThemeName = next.Name
	m.Session.SetStatus("previewing theme " + next.Name)
}

func (m *Model) startScalarEdit() tea.Cmd {
	node, err := m.selectedNode()
	if err != nil {
		m.Session.SetError(err.Error())
		return nil
	}
	if node.IsContainer() {
		m.Session.SetError("selected node is not a scalar")
		return nil
	}
	data, err := document.MarshalCompactNode(node)
	if err != nil {
		m.Session.SetError(err.Error())
		return nil
	}
	m.openPrompt(promptEditScalar, "json> ", string(data))
	return nil
}

func (m *Model) startRename() tea.Cmd {
	loc, err := m.selectedLocation()
	if err != nil {
		m.Session.SetError(err.Error())
		return nil
	}
	if loc.Parent == nil || loc.ParentKind != document.KindObject {
		m.Session.SetError("selected node does not have an object key")
		return nil
	}
	m.openPrompt(promptRenameKey, "key> ", loc.Key)
	return nil
}

func (m *Model) startAdd() tea.Cmd {
	target, err := m.currentContainerTarget()
	if err != nil {
		m.Session.SetError(err.Error())
		return nil
	}
	switch target.Kind {
	case document.KindObject:
		m.openPrompt(promptAddObject, "key=<json> ", "")
	case document.KindArray:
		m.openPrompt(promptAddArray, "json> ", "")
	default:
		m.Session.SetError("target is not a container")
	}
	return nil
}

func (m *Model) deleteSelected() tea.Cmd {
	beforeSelection := m.Session.SelectedRow()
	nodeID, err := m.selectedNodeID()
	if err != nil {
		m.Session.SetError(err.Error())
		return nil
	}
	loc, ok := m.Doc.Find(nodeID)
	if !ok {
		m.Session.SetError("selected node not found")
		return nil
	}
	if loc.Parent == nil {
		m.Session.SetError("cannot delete the root node")
		return nil
	}
	deleted := document.CloneNode(loc.Node)
	nextSelection := loc.Parent.ID
	if loc.ParentKind == document.KindObject {
		if len(loc.Parent.Object) > 1 {
			if loc.Index > 0 {
				nextSelection = loc.Parent.Object[loc.Index-1].Value.ID
			} else {
				nextSelection = loc.Parent.Object[1].Value.ID
			}
		}
	}
	if loc.ParentKind == document.KindArray {
		if len(loc.Parent.Array) > 1 {
			if loc.Index > 0 {
				nextSelection = loc.Parent.Array[loc.Index-1].ID
			} else {
				nextSelection = loc.Parent.Array[1].ID
			}
		}
	}
	if err := m.Doc.Delete(nodeID); err != nil {
		m.Session.SetError(err.Error())
		return nil
	}
	m.Session.SelectNode(nextSelection)
	m.refresh()
	m.pushHistory(newDeleteHistoryEntry(loc.Parent.ID, loc.ParentKind, loc.Index, loc.Key, deleted, beforeSelection, m.Session.SelectedRow()))
	m.Session.SetStatus("deleted node")
	return nil
}

func (m *Model) refresh() {
	m.Session.Refresh(m.Doc)
	m.Session.UpdateSearchHits(m.Doc)
}

func (m *Model) Summary() string {
	return fmt.Sprintf("%d rows", len(m.Session.Rows))
}

func (m *Model) String() string {
	return strings.TrimSpace(m.View())
}
