package tui

import (
	"testing"

	"github.com/anyroad/lazy-json/internal/document"
	"github.com/anyroad/lazy-json/internal/session"
	"github.com/anyroad/lazy-json/internal/source"
)

func TestSearchNextAndPrevious(t *testing.T) {
	m := testModel(t)
	m.Session.Search.Query = "name"
	m.Session.UpdateSearchHits(m.Doc)
	if len(m.Session.SearchHits) != 1 {
		t.Fatalf("SearchHits = %d", len(m.Session.SearchHits))
	}
	m.Session.SelectedID = m.Doc.Root.ID
	m.Session.NextSearchHit(m.Doc, false)
	if m.Session.SelectedID != m.Session.SearchHits[0] {
		t.Fatalf("SelectedID = %d, want %d", m.Session.SelectedID, m.Session.SearchHits[0])
	}
	m.Session.NextSearchHit(m.Doc, true)
	if m.Session.SelectedID != m.Session.SearchHits[0] {
		t.Fatalf("SelectedID = %d, want %d", m.Session.SelectedID, m.Session.SearchHits[0])
	}
}

func TestSearchFindsCollapsedNodesAndRevealsOnNavigation(t *testing.T) {
	doc, err := document.Parse([]byte(`{"name":"Ada","items":[{"title":"alpha"},{"title":"beta"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(doc, source.Input{Kind: source.KindFile, Path: "sample.json"}, ModelOptions{
		Clipboard: &stubClipboard{},
	})
	m.Session.Search.Query = "title"
	m.Session.UpdateSearchHits(m.Doc)

	if got, want := len(m.Session.SearchHits), 2; got != want {
		t.Fatalf("SearchHits = %d, want %d", got, want)
	}

	m.Session.SelectedID = m.Doc.Root.ID
	if !m.Session.NextSearchHit(m.Doc, false) {
		t.Fatal("NextSearchHit() = false, want true")
	}
	if got, want := m.Session.SelectedID, m.Doc.Root.Object[1].Value.Array[0].Object[0].Value.ID; got != want {
		t.Fatalf("SelectedID = %d, want %d after first n", got, want)
	}
	if !m.Session.Expanded[m.Doc.Root.Object[1].Value.ID] || !m.Session.Expanded[m.Doc.Root.Object[1].Value.Array[0].ID] {
		t.Fatal("collapsed search match was not revealed")
	}

	if !m.Session.NextSearchHit(m.Doc, false) {
		t.Fatal("NextSearchHit() = false, want true on second match")
	}
	if got, want := m.Session.SelectedID, m.Doc.Root.Object[1].Value.Array[1].Object[0].Value.ID; got != want {
		t.Fatalf("SelectedID = %d, want %d after second n", got, want)
	}
	if !m.Session.Expanded[m.Doc.Root.Object[1].Value.Array[1].ID] {
		t.Fatal("second collapsed search match was not revealed")
	}
}

func TestSearchRevealsMatchingBatchInsideLongArray(t *testing.T) {
	doc := testLongObjectArrayDoc(t, 150)
	m := NewModel(doc, source.Input{Kind: source.KindFile, Path: "sample.json"}, ModelOptions{
		Clipboard: &stubClipboard{},
	})
	items := doc.Root.Object[0].Value
	targetID := items.Array[137].Object[0].Value.ID

	m.Session.Search.Query = "item-137"
	m.Session.UpdateSearchHits(m.Doc)
	if got, want := len(m.Session.SearchHits), 1; got != want {
		t.Fatalf("SearchHits = %d, want %d", got, want)
	}

	m.Session.SelectNode(m.Doc.Root.ID)
	if !m.Session.NextSearchHit(m.Doc, false) {
		t.Fatal("NextSearchHit() = false, want true")
	}
	if got, want := m.Session.SelectedID, targetID; got != want {
		t.Fatalf("SelectedID = %d, want %d", got, want)
	}
	if !m.Session.Expanded[items.ID] {
		t.Fatal("long array is not expanded after search reveal")
	}

	batchID := session.BatchRowID(items.ID, 100)
	if !m.Session.ExpandedBatches[batchID] {
		t.Fatalf("ExpandedBatches[%v] = false, want true for second batch", batchID)
	}
}
