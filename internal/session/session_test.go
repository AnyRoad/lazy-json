package session

import (
	"testing"

	"github.com/andrei/lazy-json/internal/document"
	"github.com/andrei/lazy-json/internal/source"
)

func testDoc(t *testing.T) *document.Document {
	t.Helper()
	doc, err := document.Parse([]byte(`{"name":"Ada","items":[{"title":"alpha"},{"title":"beta"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestRefreshAndPaths(t *testing.T) {
	doc := testDoc(t)
	s := New(doc, source.Input{Kind: source.KindFile, Path: "sample.json"})
	if len(s.Rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(s.Rows))
	}
	if got := s.Rows[1].Path; got != "$.name" {
		t.Fatalf("path = %q", got)
	}
	if got := s.Rows[2].Path; got != "$.items" {
		t.Fatalf("path = %q", got)
	}
	s.Expanded[doc.Root.Object[1].Value.ID] = true
	s.Refresh(doc)
	if got := s.Rows[3].Path; got != "$.items[0]" {
		t.Fatalf("nested path = %q", got)
	}
}

func TestCollapseAndReselectVisible(t *testing.T) {
	doc := testDoc(t)
	s := New(doc, source.Input{Kind: source.KindStdin})
	itemsID := doc.Root.Object[1].Value.ID
	s.Expanded[itemsID] = true
	firstItemID := doc.Root.Object[1].Value.Array[0].ID
	s.SelectedID = firstItemID
	s.Refresh(doc)

	s.CollapseSelected(doc)
	s.Refresh(doc)

	if s.SelectedID != itemsID {
		t.Fatalf("SelectedID = %d, want %d", s.SelectedID, itemsID)
	}
}

func TestSearchHits(t *testing.T) {
	doc := testDoc(t)
	s := New(doc, source.Input{Kind: source.KindFile})
	itemsID := doc.Root.Object[1].Value.ID
	s.Expanded[itemsID] = true
	s.Expanded[doc.Root.Object[1].Value.Array[0].ID] = true
	s.Refresh(doc)
	s.Search.Query = "alpha"
	s.UpdateSearchHits()
	if len(s.SearchHits) != 1 {
		t.Fatalf("SearchHits = %d, want 1", len(s.SearchHits))
	}
	s.SelectedID = doc.Root.ID
	s.NextSearchHit(false)
	if s.SelectedID != s.SearchHits[0] {
		t.Fatalf("SelectedID = %d, want %d", s.SelectedID, s.SearchHits[0])
	}
}
