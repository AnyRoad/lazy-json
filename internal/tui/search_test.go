package tui

import "testing"

func TestSearchNextAndPrevious(t *testing.T) {
	m := testModel(t)
	m.Session.Search.Query = "items"
	m.Session.UpdateSearchHits()
	if len(m.Session.SearchHits) != 1 {
		t.Fatalf("SearchHits = %d", len(m.Session.SearchHits))
	}
	m.Session.SelectedID = m.Doc.Root.ID
	m.Session.NextSearchHit(false)
	if m.Session.SelectedID != m.Session.SearchHits[0] {
		t.Fatalf("SelectedID = %d, want %d", m.Session.SelectedID, m.Session.SearchHits[0])
	}
	m.Session.NextSearchHit(true)
	if m.Session.SelectedID != m.Session.SearchHits[0] {
		t.Fatalf("SelectedID = %d, want %d", m.Session.SelectedID, m.Session.SearchHits[0])
	}
}
