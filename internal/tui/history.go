package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anyroad/lazy-json/internal/document"
	"github.com/anyroad/lazy-json/internal/session"
)

type historyEntry interface {
	undo(doc *document.Document) error
	redo(doc *document.Document) error
	beforeSelection() session.RowID
	afterSelection() session.RowID
}

type historyRevisionID int

type historyStateEntry struct {
	revision historyRevisionID
	change   historyEntry
}

type historySelection struct {
	before session.RowID
	after  session.RowID
}

func (s historySelection) beforeSelection() session.RowID {
	return s.before
}

func (s historySelection) afterSelection() session.RowID {
	return s.after
}

type replaceHistoryEntry struct {
	targetID document.NodeID
	before   *document.Node
	after    *document.Node
	historySelection
}

func newReplaceHistoryEntry(targetID document.NodeID, before, after *document.Node, beforeSelection, afterSelection session.RowID) historyEntry {
	return replaceHistoryEntry{
		targetID: targetID,
		before:   document.CloneNode(before),
		after:    document.CloneNode(after),
		historySelection: historySelection{
			before: beforeSelection,
			after:  afterSelection,
		},
	}
}

func (e replaceHistoryEntry) undo(doc *document.Document) error {
	return doc.Restore(e.targetID, document.CloneNode(e.before))
}

func (e replaceHistoryEntry) redo(doc *document.Document) error {
	return doc.Restore(e.targetID, document.CloneNode(e.after))
}

type renameHistoryEntry struct {
	targetID document.NodeID
	oldKey   string
	newKey   string
	historySelection
}

func newRenameHistoryEntry(targetID document.NodeID, oldKey, newKey string, selection session.RowID) historyEntry {
	return renameHistoryEntry{
		targetID: targetID,
		oldKey:   oldKey,
		newKey:   newKey,
		historySelection: historySelection{
			before: selection,
			after:  selection,
		},
	}
}

func (e renameHistoryEntry) undo(doc *document.Document) error {
	return doc.RenameKey(e.targetID, e.oldKey)
}

func (e renameHistoryEntry) redo(doc *document.Document) error {
	return doc.RenameKey(e.targetID, e.newKey)
}

type objectInsertHistoryEntry struct {
	parentID document.NodeID
	index    int
	key      string
	node     *document.Node
	historySelection
}

func newObjectInsertHistoryEntry(parentID document.NodeID, index int, key string, node *document.Node, beforeSelection, afterSelection session.RowID) historyEntry {
	return objectInsertHistoryEntry{
		parentID: parentID,
		index:    index,
		key:      key,
		node:     document.CloneNode(node),
		historySelection: historySelection{
			before: beforeSelection,
			after:  afterSelection,
		},
	}
}

func (e objectInsertHistoryEntry) undo(doc *document.Document) error {
	return doc.Delete(e.node.ID)
}

func (e objectInsertHistoryEntry) redo(doc *document.Document) error {
	return doc.InsertObjectEntryAt(e.parentID, e.index, e.key, document.CloneNode(e.node))
}

type arrayInsertHistoryEntry struct {
	parentID document.NodeID
	index    int
	node     *document.Node
	historySelection
}

func newArrayInsertHistoryEntry(parentID document.NodeID, index int, node *document.Node, beforeSelection, afterSelection session.RowID) historyEntry {
	return arrayInsertHistoryEntry{
		parentID: parentID,
		index:    index,
		node:     document.CloneNode(node),
		historySelection: historySelection{
			before: beforeSelection,
			after:  afterSelection,
		},
	}
}

func (e arrayInsertHistoryEntry) undo(doc *document.Document) error {
	return doc.Delete(e.node.ID)
}

func (e arrayInsertHistoryEntry) redo(doc *document.Document) error {
	return doc.InsertArrayItemAt(e.parentID, e.index, document.CloneNode(e.node))
}

type deleteHistoryEntry struct {
	parentID   document.NodeID
	parentKind document.Kind
	index      int
	key        string
	node       *document.Node
	historySelection
}

func newDeleteHistoryEntry(parentID document.NodeID, parentKind document.Kind, index int, key string, node *document.Node, beforeSelection, afterSelection session.RowID) historyEntry {
	return deleteHistoryEntry{
		parentID:   parentID,
		parentKind: parentKind,
		index:      index,
		key:        key,
		node:       document.CloneNode(node),
		historySelection: historySelection{
			before: beforeSelection,
			after:  afterSelection,
		},
	}
}

func (e deleteHistoryEntry) undo(doc *document.Document) error {
	switch e.parentKind {
	case document.KindObject:
		return doc.InsertObjectEntryAt(e.parentID, e.index, e.key, document.CloneNode(e.node))
	case document.KindArray:
		return doc.InsertArrayItemAt(e.parentID, e.index, document.CloneNode(e.node))
	default:
		return fmt.Errorf("unsupported parent kind %s", e.parentKind)
	}
}

func (e deleteHistoryEntry) redo(doc *document.Document) error {
	return doc.Delete(e.node.ID)
}

func (m *Model) pushHistory(entry historyEntry) {
	if entry == nil {
		return
	}
	m.nextRevision++
	state := historyStateEntry{
		revision: m.nextRevision,
		change:   entry,
	}
	m.undoStack = append(m.undoStack, state)
	m.redoStack = nil
	m.currentRevision = state.revision
	m.syncDirty()
}

func (m *Model) markSavedRevision() {
	m.savedRevision = m.currentRevision
	m.syncDirty()
}

func (m *Model) syncDirty() {
	if m.Session == nil {
		return
	}
	m.Session.Dirty = m.currentRevision != m.savedRevision
}

func (m *Model) replayHistorySelection(selection session.RowID) {
	if m.Session == nil {
		return
	}
	if !selection.IsZero() {
		m.Session.SelectRowID(selection)
	}
	m.refresh()
}

func (m *Model) undoChange() tea.Cmd {
	if len(m.undoStack) == 0 {
		m.Session.SetError("nothing to undo")
		return nil
	}

	state := m.undoStack[len(m.undoStack)-1]
	if err := state.change.undo(m.Doc); err != nil {
		m.Session.SetError(err.Error())
		return nil
	}

	m.undoStack = m.undoStack[:len(m.undoStack)-1]
	m.redoStack = append(m.redoStack, state)
	if len(m.undoStack) == 0 {
		m.currentRevision = 0
	} else {
		m.currentRevision = m.undoStack[len(m.undoStack)-1].revision
	}
	m.replayHistorySelection(state.change.beforeSelection())
	m.syncDirty()
	m.Session.SetStatus("undid change")
	return nil
}

func (m *Model) redoChange() tea.Cmd {
	if len(m.redoStack) == 0 {
		m.Session.SetError("nothing to redo")
		return nil
	}

	state := m.redoStack[len(m.redoStack)-1]
	if err := state.change.redo(m.Doc); err != nil {
		m.Session.SetError(err.Error())
		return nil
	}

	m.redoStack = m.redoStack[:len(m.redoStack)-1]
	m.undoStack = append(m.undoStack, state)
	m.currentRevision = state.revision
	m.replayHistorySelection(state.change.afterSelection())
	m.syncDirty()
	m.Session.SetStatus("redid change")
	return nil
}
