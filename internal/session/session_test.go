package session

import (
	"testing"

	"github.com/anyroad/lazy-json/internal/document"
	"github.com/anyroad/lazy-json/internal/source"
)

func testDoc(t *testing.T) *document.Document {
	t.Helper()
	doc, err := document.Parse([]byte(`{"name":"Ada","items":[{"title":"alpha"},{"title":"beta"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func testDocWithSiblingBranches(t *testing.T) *document.Document {
	t.Helper()
	doc, err := document.Parse([]byte(`{"name":"Ada","items":[{"title":"alpha"},{"title":"beta"}],"meta":{"count":2},"tail":true}`))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestRefreshAndPaths(t *testing.T) {
	doc := testDoc(t)
	s := New(doc, source.Input{Kind: source.KindFile, Path: "sample.json"}, "")
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
	s := New(doc, source.Input{Kind: source.KindStdin}, "")
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
	s := New(doc, source.Input{Kind: source.KindFile}, "")
	s.Search.Query = "alpha"
	s.UpdateSearchHits(doc)
	if len(s.SearchHits) != 1 {
		t.Fatalf("SearchHits = %d, want 1", len(s.SearchHits))
	}
	s.SelectedID = doc.Root.ID
	if !s.NextSearchHit(doc, false) {
		t.Fatal("NextSearchHit() = false, want true")
	}
	if s.SelectedID != s.SearchHits[0] {
		t.Fatalf("SelectedID = %d, want %d", s.SelectedID, s.SearchHits[0])
	}
	if !s.Expanded[doc.Root.Object[1].Value.ID] {
		t.Fatal("items container is not expanded after search reveal")
	}
	if !s.Expanded[doc.Root.Object[1].Value.Array[0].ID] {
		t.Fatal("first array item is not expanded after search reveal")
	}
}

func TestExpandAllAndCollapseAll(t *testing.T) {
	doc := testDoc(t)
	s := New(doc, source.Input{Kind: source.KindFile}, "")

	s.ExpandAll(doc)
	s.Refresh(doc)
	if got, want := len(s.Rows), 7; got != want {
		t.Fatalf("rows = %d, want %d after ExpandAll", got, want)
	}

	titleID := doc.Root.Object[1].Value.Array[0].Object[0].Value.ID
	s.SelectedID = titleID
	s.CollapseAll(doc)
	s.Refresh(doc)

	if got, want := len(s.Rows), 3; got != want {
		t.Fatalf("rows = %d, want %d after CollapseAll", got, want)
	}
	if got, want := s.SelectedID, doc.Root.Object[1].Value.ID; got != want {
		t.Fatalf("SelectedID = %d, want %d after CollapseAll", got, want)
	}
}

func TestExpandNearestArrayOneLevel(t *testing.T) {
	doc := testDoc(t)
	s := New(doc, source.Input{Kind: source.KindFile}, "")
	items := doc.Root.Object[1].Value
	s.SelectedID = items.ID

	if !s.ExpandNearestArrayOneLevel(doc) {
		t.Fatal("ExpandNearestArrayOneLevel() = false, want true on selected array")
	}
	s.Refresh(doc)

	if !s.Expanded[items.ID] {
		t.Fatal("items array is not expanded")
	}
	if !s.Expanded[items.Array[0].ID] {
		t.Fatal("first array element is not expanded")
	}
	if !s.Expanded[items.Array[1].ID] {
		t.Fatal("second array element is not expanded")
	}
	if got, want := len(s.Rows), 7; got != want {
		t.Fatalf("rows = %d, want %d after one-level array expansion", got, want)
	}
}

func TestExpandNearestArrayOneLevelUsesNearestArrayAncestor(t *testing.T) {
	doc := testDoc(t)
	s := New(doc, source.Input{Kind: source.KindFile}, "")
	items := doc.Root.Object[1].Value
	s.Expanded[items.ID] = true
	s.Expanded[items.Array[0].ID] = true
	s.SelectedID = items.Array[0].Object[0].Value.ID
	s.Refresh(doc)

	if !s.ExpandNearestArrayOneLevel(doc) {
		t.Fatal("ExpandNearestArrayOneLevel() = false, want true inside array descendant")
	}
	s.Refresh(doc)

	if !s.Expanded[items.Array[1].ID] {
		t.Fatal("second array element is not expanded from nearest array ancestor")
	}
	if got, want := len(s.Rows), 7; got != want {
		t.Fatalf("rows = %d, want %d after nearest-array expansion", got, want)
	}
}

func TestCollapseNearestArrayElements(t *testing.T) {
	doc := testDoc(t)
	s := New(doc, source.Input{Kind: source.KindFile}, "")
	items := doc.Root.Object[1].Value
	s.SelectedID = items.ID
	s.ExpandNearestArrayOneLevel(doc)
	s.Refresh(doc)

	if !s.CollapseNearestArrayElements(doc) {
		t.Fatal("CollapseNearestArrayElements() = false, want true on selected array")
	}
	s.Refresh(doc)

	if !s.Expanded[items.ID] {
		t.Fatal("items array is not expanded")
	}
	if s.Expanded[items.Array[0].ID] {
		t.Fatal("first array element is expanded, want collapsed")
	}
	if s.Expanded[items.Array[1].ID] {
		t.Fatal("second array element is expanded, want collapsed")
	}
	if got, want := len(s.Rows), 5; got != want {
		t.Fatalf("rows = %d, want %d after collapsing array elements", got, want)
	}
}

func TestCollapseNearestArrayElementsUsesNearestArrayAncestor(t *testing.T) {
	doc := testDoc(t)
	s := New(doc, source.Input{Kind: source.KindFile}, "")
	items := doc.Root.Object[1].Value
	s.SelectedID = items.ID
	s.ExpandNearestArrayOneLevel(doc)
	s.SelectedID = items.Array[0].Object[0].Value.ID
	s.Refresh(doc)

	if !s.CollapseNearestArrayElements(doc) {
		t.Fatal("CollapseNearestArrayElements() = false, want true inside array descendant")
	}
	s.Refresh(doc)

	if got, want := s.SelectedID, items.Array[0].ID; got != want {
		t.Fatalf("SelectedID = %d, want %d after collapsing to visible ancestor", got, want)
	}
	if s.Expanded[items.Array[0].ID] || s.Expanded[items.Array[1].ID] {
		t.Fatal("array elements remain expanded, want collapsed")
	}
}

func TestCollapseNearestArrayElementsRejectsMissingArrayContext(t *testing.T) {
	doc := testDoc(t)
	s := New(doc, source.Input{Kind: source.KindFile}, "")
	s.SelectedID = doc.Root.Object[0].Value.ID

	if s.CollapseNearestArrayElements(doc) {
		t.Fatal("CollapseNearestArrayElements() = true, want false outside array context")
	}
}

func TestExpandNearestArrayOneLevelRejectsMissingArrayContext(t *testing.T) {
	doc := testDoc(t)
	s := New(doc, source.Input{Kind: source.KindFile}, "")
	s.SelectedID = doc.Root.Object[0].Value.ID

	if s.ExpandNearestArrayOneLevel(doc) {
		t.Fatal("ExpandNearestArrayOneLevel() = true, want false outside array context")
	}
}

func TestNextParentSiblingClimbsAncestors(t *testing.T) {
	doc := testDocWithSiblingBranches(t)
	s := New(doc, source.Input{Kind: source.KindFile}, "")
	s.ExpandAll(doc)
	s.Refresh(doc)

	titleID := doc.Root.Object[1].Value.Array[0].Object[0].Value.ID
	s.SelectedID = titleID
	if !s.NextParentSibling(doc) {
		t.Fatal("NextParentSibling() = false, want true for immediate parent sibling")
	}
	if got, want := s.SelectedID, doc.Root.Object[1].Value.Array[1].ID; got != want {
		t.Fatalf("SelectedID = %d, want %d", got, want)
	}

	lastTitleID := doc.Root.Object[1].Value.Array[1].Object[0].Value.ID
	s.SelectedID = lastTitleID
	if !s.NextParentSibling(doc) {
		t.Fatal("NextParentSibling() = false, want true for ancestor climb")
	}
	if got, want := s.SelectedID, doc.Root.Object[2].Value.ID; got != want {
		t.Fatalf("SelectedID = %d, want %d after climb", got, want)
	}

	s.SelectedID = doc.Root.Object[3].Value.ID
	if s.NextParentSibling(doc) {
		t.Fatal("NextParentSibling() = true, want false at end of document")
	}
}

func TestRevealSelectionExpandsAncestorsForDeepNode(t *testing.T) {
	doc := testDoc(t)
	s := New(doc, source.Input{Kind: source.KindFile}, "")
	targetID := doc.Root.Object[1].Value.Array[0].Object[0].Value.ID

	s.RevealSelection(doc, targetID, []document.NodeID{
		doc.Root.ID,
		doc.Root.Object[1].Value.ID,
		doc.Root.Object[1].Value.Array[0].ID,
	})

	if got, want := s.SelectedID, targetID; got != want {
		t.Fatalf("SelectedID = %d, want %d", got, want)
	}
	if !s.Expanded[doc.Root.Object[1].Value.ID] {
		t.Fatal("items container is not expanded")
	}
	if !s.Expanded[doc.Root.Object[1].Value.Array[0].ID] {
		t.Fatal("first array item is not expanded")
	}
	if got := s.Rows[4].Path; got != "$.items[0].title" {
		t.Fatalf("path = %q, want $.items[0].title", got)
	}
}

func TestRevealSelectionKeepsSelectedContainerCollapsed(t *testing.T) {
	doc := testDoc(t)
	s := New(doc, source.Input{Kind: source.KindFile}, "")
	targetID := doc.Root.Object[1].Value.ID

	s.RevealSelection(doc, targetID, []document.NodeID{doc.Root.ID})

	if got, want := s.SelectedID, targetID; got != want {
		t.Fatalf("SelectedID = %d, want %d", got, want)
	}
	if s.Expanded[targetID] {
		t.Fatal("selected container was expanded, want collapsed")
	}
	if got, want := len(s.Rows), 3; got != want {
		t.Fatalf("rows = %d, want %d with selected container collapsed", got, want)
	}
}
