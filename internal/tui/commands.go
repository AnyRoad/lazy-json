package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/andrei/lazy-json/internal/document"
	"github.com/andrei/lazy-json/internal/integration"
	"github.com/andrei/lazy-json/internal/source"
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
		m.Session.UpdateSearchHits()
		if len(m.Session.SearchHits) == 0 {
			m.Session.SetError("no search results")
			return nil
		}
		m.Session.SelectedID = m.Session.SearchHits[0]
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
	data, err := m.Doc.MarshalIndent()
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

func (m *Model) saveAndQuit(path string) tea.Cmd {
	if path == "" && m.Session.SourceKind == source.KindStdin && m.Session.SourcePath == "" {
		data, err := m.Doc.MarshalIndent()
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
	data, err := m.Doc.MarshalIndent()
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
		next := NextTheme(m.Session.ThemeName)
		m.Session.ThemeName = next.Name
		m.Session.SetStatus("switched theme to " + next.Name)
		return nil
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
