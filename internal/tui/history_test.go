package tui

import (
	"testing"

	"github.com/anyroad/lazy-json/internal/document"
)

type noopHistoryEntry struct {
	historySelection
}

func (e noopHistoryEntry) undo(doc *document.Document) error {
	return nil
}

func (e noopHistoryEntry) redo(doc *document.Document) error {
	return nil
}

func TestPushHistoryClearsRedoAndTracksDirty(t *testing.T) {
	m := testModel(t)
	m.redoStack = []historyStateEntry{{revision: 99, change: noopHistoryEntry{}}}

	rootSelection := m.Session.SelectedRow()
	m.pushHistory(noopHistoryEntry{historySelection: historySelection{before: rootSelection, after: rootSelection}})

	if got, want := len(m.undoStack), 1; got != want {
		t.Fatalf("len(undoStack) = %d, want %d", got, want)
	}
	if got := len(m.redoStack); got != 0 {
		t.Fatalf("len(redoStack) = %d, want 0", got)
	}
	if got, want := m.currentRevision, historyRevisionID(1); got != want {
		t.Fatalf("currentRevision = %d, want %d", got, want)
	}
	if !m.Session.Dirty {
		t.Fatal("Dirty = false after pushHistory, want true")
	}

	m.markSavedRevision()
	if got, want := m.savedRevision, historyRevisionID(1); got != want {
		t.Fatalf("savedRevision = %d, want %d", got, want)
	}
	if m.Session.Dirty {
		t.Fatal("Dirty = true after markSavedRevision, want false")
	}
}

func TestUndoRedoMovesEntriesBetweenStacks(t *testing.T) {
	m := testModel(t)
	rootSelection := m.Session.SelectedRow()
	m.pushHistory(noopHistoryEntry{historySelection: historySelection{before: rootSelection, after: rootSelection}})

	runCmd(t, m, m.undoChange())
	if got, want := len(m.undoStack), 0; got != want {
		t.Fatalf("len(undoStack) after undo = %d, want %d", got, want)
	}
	if got, want := len(m.redoStack), 1; got != want {
		t.Fatalf("len(redoStack) after undo = %d, want %d", got, want)
	}
	if got, want := m.currentRevision, historyRevisionID(0); got != want {
		t.Fatalf("currentRevision after undo = %d, want %d", got, want)
	}
	if m.Session.Dirty {
		t.Fatal("Dirty = true after undo to revision 0, want false")
	}

	runCmd(t, m, m.redoChange())
	if got, want := len(m.undoStack), 1; got != want {
		t.Fatalf("len(undoStack) after redo = %d, want %d", got, want)
	}
	if got, want := len(m.redoStack), 0; got != want {
		t.Fatalf("len(redoStack) after redo = %d, want %d", got, want)
	}
	if got, want := m.currentRevision, historyRevisionID(1); got != want {
		t.Fatalf("currentRevision after redo = %d, want %d", got, want)
	}
	if !m.Session.Dirty {
		t.Fatal("Dirty = false after redo, want true")
	}
}

func TestSavedRevisionTracksRevisionIdentityAcrossBranches(t *testing.T) {
	m := testModel(t)
	rootSelection := m.Session.SelectedRow()

	m.pushHistory(noopHistoryEntry{historySelection: historySelection{before: rootSelection, after: rootSelection}})
	m.markSavedRevision()

	runCmd(t, m, m.undoChange())
	if !m.Session.Dirty {
		t.Fatal("Dirty = false after undo away from saved revision, want true")
	}

	m.pushHistory(noopHistoryEntry{historySelection: historySelection{before: rootSelection, after: rootSelection}})
	if !m.Session.Dirty {
		t.Fatal("Dirty = false after branching edit, want true")
	}
	if got, want := m.currentRevision, historyRevisionID(2); got != want {
		t.Fatalf("currentRevision = %d, want %d", got, want)
	}
	if got, want := m.savedRevision, historyRevisionID(1); got != want {
		t.Fatalf("savedRevision = %d, want %d", got, want)
	}
}
