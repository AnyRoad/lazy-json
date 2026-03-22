package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anyroad/lazy-json/internal/document"
	"github.com/anyroad/lazy-json/internal/integration"
	"github.com/anyroad/lazy-json/internal/source"
)

type editorFinishedMsg struct {
	Node *document.Node
	Err  error
}

type jqFinishedMsg struct {
	Node     *document.Node
	TargetID document.NodeID
	Err      error
}

func (m *Model) SelectPath(selectPath string) (document.PathResolution, error) {
	selectPath = strings.TrimSpace(selectPath)
	if selectPath == "" {
		return document.PathResolution{}, fmt.Errorf("select path cannot be empty")
	}

	resolution, err := m.Doc.ResolvePath(selectPath)
	if err != nil {
		return document.PathResolution{}, fmt.Errorf("invalid select path: %w", err)
	}

	m.Session.RevealSelection(m.Doc, resolution.NodeID, resolution.Ancestors)
	return resolution, nil
}

func (m *Model) selectedNode() (*document.Node, error) {
	loc, ok := m.Doc.Find(m.Session.SelectedID)
	if !ok {
		return nil, fmt.Errorf("selected node not found")
	}
	return loc.Node, nil
}

func (m *Model) currentContainerTarget() (*document.Node, error) {
	loc, ok := m.Doc.Find(m.Session.SelectedID)
	if !ok {
		return nil, fmt.Errorf("selected node not found")
	}
	if loc.Node.IsContainer() {
		return loc.Node, nil
	}
	if loc.Parent != nil && loc.Parent.IsContainer() {
		return loc.Parent, nil
	}
	return nil, fmt.Errorf("no object or array target available")
}

func (m *Model) openPrompt(kind promptKind, placeholder, initial string) {
	m.clearPendingPrefix()
	m.promptKind = kind
	m.prompt = newPrompt()
	m.prompt.Prompt = placeholder
	m.prompt.SetValue(initial)
	m.prompt.Focus()
	switch kind {
	case promptCommand:
		m.Session.Mode = "command"
	case promptSearch:
		m.Session.Mode = "search"
	default:
		m.Session.Mode = "prompt"
	}
}

func (m *Model) closePrompt() {
	m.clearPendingPrefix()
	m.promptKind = promptNone
	m.prompt.Blur()
	m.Session.Mode = "normal"
}

func (m *Model) submitPrompt() tea.Cmd {
	value := strings.TrimSpace(m.prompt.Value())
	kind := m.promptKind
	m.closePrompt()
	switch kind {
	case promptCommand:
		return m.handleCommand(value)
	case promptSearch:
		m.Session.Search.Query = value
		m.Session.UpdateSearchHits(m.Doc)
		if len(m.Session.SearchHits) == 0 {
			m.Session.SetError("no search results")
			return nil
		}
		if !m.Session.RevealNode(m.Doc, m.Session.SearchHits[0]) {
			m.Session.SetError("search result not found")
			return nil
		}
		m.Session.SetStatus(fmt.Sprintf("%d matches", len(m.Session.SearchHits)))
		return nil
	case promptEditScalar:
		return m.replaceSelected(value, true)
	case promptRenameKey:
		if value == "" {
			m.Session.SetError("key cannot be empty")
			return nil
		}
		if err := m.Doc.RenameKey(m.Session.SelectedID, value); err != nil {
			m.Session.SetError(err.Error())
			return nil
		}
		m.Session.Dirty = true
		m.refresh()
		m.Session.SetStatus("renamed key")
		return nil
	case promptAddObject:
		return m.addObjectEntry(value)
	case promptAddArray:
		return m.addArrayItem(value)
	default:
		return nil
	}
}

func (m *Model) replaceSelected(input string, scalarOnly bool) tea.Cmd {
	node, err := document.ParseNode([]byte(input))
	if err != nil {
		m.Session.SetError(err.Error())
		return nil
	}
	if scalarOnly && node.IsContainer() {
		m.Session.SetError("expected a scalar JSON value")
		return nil
	}
	if err := m.Doc.Replace(m.Session.SelectedID, node); err != nil {
		m.Session.SetError(err.Error())
		return nil
	}
	m.Session.Dirty = true
	m.refresh()
	m.Session.SetStatus("updated node")
	return nil
}

func (m *Model) addObjectEntry(input string) tea.Cmd {
	container, err := m.currentContainerTarget()
	if err != nil {
		m.Session.SetError(err.Error())
		return nil
	}
	if container.Kind != document.KindObject {
		m.Session.SetError("target is not an object")
		return nil
	}
	key, raw, ok := strings.Cut(input, "=")
	if !ok {
		m.Session.SetError("expected key=<json>")
		return nil
	}
	key = strings.TrimSpace(key)
	raw = strings.TrimSpace(raw)
	if key == "" {
		m.Session.SetError("key cannot be empty")
		return nil
	}
	node, err := document.ParseNode([]byte(raw))
	if err != nil {
		m.Session.SetError(err.Error())
		return nil
	}
	if err := m.Doc.AddObjectEntry(container.ID, key, node); err != nil {
		m.Session.SetError(err.Error())
		return nil
	}
	m.Session.Dirty = true
	m.Session.Expanded[container.ID] = true
	m.refresh()
	m.Session.SelectedID = node.ID
	m.Session.SetStatus("added object field")
	return nil
}

func (m *Model) addArrayItem(input string) tea.Cmd {
	container, err := m.currentContainerTarget()
	if err != nil {
		m.Session.SetError(err.Error())
		return nil
	}
	if container.Kind != document.KindArray {
		m.Session.SetError("target is not an array")
		return nil
	}
	node, err := document.ParseNode([]byte(input))
	if err != nil {
		m.Session.SetError(err.Error())
		return nil
	}
	if err := m.Doc.AddArrayItem(container.ID, node); err != nil {
		m.Session.SetError(err.Error())
		return nil
	}
	m.Session.Dirty = true
	m.Session.Expanded[container.ID] = true
	m.refresh()
	m.Session.SelectedID = node.ID
	m.Session.SetStatus("added array item")
	return nil
}

func (m *Model) save(path string) error {
	data, err := m.Doc.MarshalIndentWith(m.ActiveSettings.WithDefaults().SaveIndent.String())
	if err != nil {
		return err
	}
	if path == "" {
		path = m.Session.SourcePath
	}
	if path == "" {
		return fmt.Errorf("no file path available; use :w <path> or :print")
	}
	if err := source.WriteAtomic(path, data); err != nil {
		return err
	}
	m.Session.SourceKind = source.KindFile
	m.Session.SourcePath = path
	m.Session.Dirty = false
	return nil
}

func trimClipboardJSON(data []byte) string {
	return strings.TrimSuffix(string(data), "\n")
}

func (m *Model) copyClipboardText(label, text string) tea.Cmd {
	if err := m.Clipboard.WriteAll(text); err != nil {
		m.Session.SetError("could not copy to clipboard: " + err.Error())
		return nil
	}
	m.Session.SetStatus("copied " + label + " to clipboard")
	return nil
}

func (m *Model) copyPath() tea.Cmd {
	row, ok := m.Session.CurrentRow()
	if !ok {
		m.Session.SetError("selected row not found")
		return nil
	}
	return m.copyClipboardText("path", row.Path)
}

func (m *Model) copyKey() tea.Cmd {
	loc, ok := m.Doc.Find(m.Session.SelectedID)
	if !ok {
		m.Session.SetError("selected node not found")
		return nil
	}
	if loc.Parent == nil || loc.ParentKind != document.KindObject {
		m.Session.SetError("selected node does not have an object key")
		return nil
	}
	return m.copyClipboardText("key", loc.Key)
}

func (m *Model) copyValue() tea.Cmd {
	node, err := m.selectedNode()
	if err != nil {
		m.Session.SetError(err.Error())
		return nil
	}
	data, err := document.MarshalCompactNode(node)
	if err != nil {
		m.Session.SetError(err.Error())
		return nil
	}
	return m.copyClipboardText("value", string(data))
}

func (m *Model) copySubtree() tea.Cmd {
	node, err := m.selectedNode()
	if err != nil {
		m.Session.SetError(err.Error())
		return nil
	}
	data, err := document.MarshalIndentNodeWithIndent(node, m.ActiveSettings.WithDefaults().SaveIndent.String())
	if err != nil {
		m.Session.SetError(err.Error())
		return nil
	}
	return m.copyClipboardText("subtree", trimClipboardJSON(data))
}

func (m *Model) copyDocument() tea.Cmd {
	data, err := m.Doc.MarshalIndentWith(m.ActiveSettings.WithDefaults().SaveIndent.String())
	if err != nil {
		m.Session.SetError(err.Error())
		return nil
	}
	return m.copyClipboardText("document", trimClipboardJSON(data))
}

func (m *Model) expandAll() tea.Cmd {
	m.Session.ExpandAll(m.Doc)
	m.refresh()
	m.Session.SetStatus("expanded all nodes")
	return nil
}

func (m *Model) expandArrayElementsOneLevel() tea.Cmd {
	if !m.Session.ExpandNearestArrayOneLevel(m.Doc) {
		m.Session.SetError("no array target available")
		return nil
	}
	m.refresh()
	m.Session.SetStatus("expanded array elements one level")
	return nil
}

func (m *Model) collapseArrayElements() tea.Cmd {
	if !m.Session.CollapseNearestArrayElements(m.Doc) {
		m.Session.SetError("no array target available")
		return nil
	}
	m.refresh()
	m.Session.SetStatus("collapsed array elements")
	return nil
}

func (m *Model) collapseAll() tea.Cmd {
	m.Session.CollapseAll(m.Doc)
	m.refresh()
	m.Session.SetStatus("collapsed all nodes")
	return nil
}

func (m *Model) nextParentSibling() tea.Cmd {
	if !m.Session.NextParentSibling(m.Doc) {
		m.Session.SetError("no next parent sibling")
		return nil
	}
	m.Session.SetStatus("moved to next parent sibling")
	return nil
}

func (m *Model) selectPathCommand(selectPath string) tea.Cmd {
	selectPath = strings.TrimSpace(selectPath)
	if selectPath == "" {
		m.Session.SetError("select-path requires a JSON path")
		return nil
	}

	resolution, err := m.SelectPath(selectPath)
	if err != nil {
		m.Session.SetError(err.Error())
		return nil
	}
	if resolution.Exact {
		m.Session.SetStatus("selected path " + resolution.MatchedPath)
		return nil
	}
	if resolution.MatchedPath == "$" {
		m.Session.SetError(fmt.Sprintf("select path not found: %s; opened root $", selectPath))
		return nil
	}
	m.Session.SetStatus(fmt.Sprintf("select path not found: %s; opened nearest existing ancestor %s", selectPath, resolution.MatchedPath))
	return nil
}

func (m *Model) saveAndQuit(path string) tea.Cmd {
	if path == "" && m.Session.SourceKind == source.KindStdin && m.Session.SourcePath == "" {
		data, err := m.Doc.MarshalIndentWith(m.ActiveSettings.WithDefaults().SaveIndent.String())
		if err != nil {
			m.Session.SetError(err.Error())
			return nil
		}
		m.ExitOutput = data
		return tea.Quit
	}
	if err := m.save(path); err != nil {
		m.Session.SetError(err.Error())
		return nil
	}
	return tea.Quit
}

func (m *Model) printAndQuit() tea.Cmd {
	data, err := m.Doc.MarshalIndentWith(m.ActiveSettings.WithDefaults().SaveIndent.String())
	if err != nil {
		m.Session.SetError(err.Error())
		return nil
	}
	m.ExitOutput = data
	return tea.Quit
}

func (m *Model) handleCommand(command string) tea.Cmd {
	command = strings.TrimSpace(command)
	if command == "" {
		return nil
	}
	switch {
	case command == "w":
		if err := m.save(""); err != nil {
			m.Session.SetError(err.Error())
		} else {
			m.Session.SetStatus("saved")
		}
		return nil
	case strings.HasPrefix(command, "w "):
		path := strings.TrimSpace(strings.TrimPrefix(command, "w "))
		if err := m.save(path); err != nil {
			m.Session.SetError(err.Error())
		} else {
			m.Session.SetStatus("saved")
		}
		return nil
	case command == "x":
		return m.saveAndQuit("")
	case strings.HasPrefix(command, "x "):
		return m.saveAndQuit(strings.TrimSpace(strings.TrimPrefix(command, "x ")))
	case command == "print":
		return m.printAndQuit()
	case command == "q":
		if m.Session.Dirty {
			m.Session.SetError("unsaved changes; use :x, :w, or :q!")
			return nil
		}
		return tea.Quit
	case command == "q!":
		return tea.Quit
	case command == "theme":
		m.cycleTheme()
		return nil
	case command == "settings":
		m.openSettings()
		return nil
	case command == "select-path":
		return m.selectPathCommand("")
	case strings.HasPrefix(command, "select-path "):
		return m.selectPathCommand(strings.TrimSpace(strings.TrimPrefix(command, "select-path ")))
	case command == "copy-path":
		return m.copyPath()
	case command == "copy-key":
		return m.copyKey()
	case command == "copy-value":
		return m.copyValue()
	case command == "copy-subtree":
		return m.copySubtree()
	case command == "copy-json":
		return m.copyDocument()
	case command == "expand-all":
		return m.expandAll()
	case command == "collapse-all":
		return m.collapseAll()
	case command == "next-parent-sibling":
		return m.nextParentSibling()
	case strings.HasPrefix(command, "jq! "):
		expr := strings.TrimSpace(strings.TrimPrefix(command, "jq! "))
		return m.applyJQ(expr, true)
	case strings.HasPrefix(command, "jq "):
		expr := strings.TrimSpace(strings.TrimPrefix(command, "jq "))
		return m.applyJQ(expr, false)
	case command == "edit-external":
		return m.editExternal()
	default:
		m.Session.SetError("unknown command: :" + command)
		return nil
	}
}

func (m *Model) applyJQ(expr string, subtree bool) tea.Cmd {
	if expr == "" {
		m.Session.SetError("jq expression cannot be empty")
		return nil
	}
	targetID := m.Doc.Root.ID
	if subtree {
		targetID = m.Session.SelectedID
	}
	loc, ok := m.Doc.Find(targetID)
	if !ok {
		m.Session.SetError("target node not found")
		return nil
	}
	target := loc.Node
	runner := m.JQRunner
	if runner == nil {
		runner = integration.ExecJQRunner{}
	}
	return func() tea.Msg {
		node, err := integration.ApplyJQ(runner, expr, target)
		return jqFinishedMsg{Node: node, TargetID: targetID, Err: err}
	}
}

func (m *Model) editExternal() tea.Cmd {
	node, err := m.selectedNode()
	if err != nil {
		m.Session.SetError(err.Error())
		return nil
	}
	path, err := integration.SerializeNodeToTemp(node)
	if err != nil {
		m.Session.SetError(err.Error())
		return nil
	}
	cmd, err := integration.EditorCommand(os.Getenv("EDITOR"), path)
	if err != nil {
		os.Remove(path)
		m.Session.SetError(err.Error())
		return nil
	}
	return tea.ExecProcess(cmd, func(execErr error) tea.Msg {
		defer os.Remove(path)
		if execErr != nil {
			return editorFinishedMsg{Err: execErr}
		}
		edited, err := integration.ReadEditedNode(path)
		return editorFinishedMsg{Node: edited, Err: err}
	})
}
