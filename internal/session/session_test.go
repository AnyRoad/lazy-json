package session

import (
	"fmt"
	"strings"
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

func testLongScalarArrayDoc(t *testing.T, count int) *document.Document {
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
	if got := s.Rows[2].LineNumber; got != 3 {
		t.Fatalf("LineNumber = %d, want 3", got)
	}
	s.Expanded[doc.Root.Object[1].Value.ID] = true
	s.Refresh(doc)
	if got := s.Rows[3].Path; got != "$.items[0]" {
		t.Fatalf("nested path = %q", got)
	}
	if got := s.Rows[3].LineNumber; got != 4 {
		t.Fatalf("nested LineNumber = %d, want 4", got)
	}
}

func TestBuildRowsPreservesGlobalLineNumberGapsWhenCollapsed(t *testing.T) {
	doc := testDocWithSiblingBranches(t)
	s := New(doc, source.Input{Kind: source.KindFile, Path: "sample.json"}, "")
	itemsID := doc.Root.Object[1].Value.ID
	metaID := doc.Root.Object[2].Value.ID
	tailID := doc.Root.Object[3].Value.ID

	s.Expanded[itemsID] = true
	s.Expanded[doc.Root.Object[1].Value.Array[0].ID] = true
	s.Expanded[doc.Root.Object[1].Value.Array[1].ID] = true
	s.Expanded[metaID] = true
	s.Refresh(doc)

	if got, want := s.Rows[s.RowIndex[tailID]].LineNumber, 10; got != want {
		t.Fatalf("expanded tail LineNumber = %d, want %d", got, want)
	}

	delete(s.Expanded, itemsID)
	delete(s.Expanded, metaID)
	s.Refresh(doc)

	if got, want := s.Rows[s.RowIndex[itemsID]].LineNumber, 3; got != want {
		t.Fatalf("collapsed items LineNumber = %d, want %d", got, want)
	}
	if got, want := s.Rows[s.RowIndex[metaID]].LineNumber, 8; got != want {
		t.Fatalf("collapsed meta LineNumber = %d, want %d", got, want)
	}
	if got, want := s.Rows[s.RowIndex[tailID]].LineNumber, 10; got != want {
		t.Fatalf("collapsed tail LineNumber = %d, want %d", got, want)
	}
}

func TestBuildRowsBatchesArraysLongerThanThreshold(t *testing.T) {
	shortDoc := testLongScalarArrayDoc(t, 100)
	shortSession := New(shortDoc, source.Input{Kind: source.KindFile, Path: "sample.json"}, "")
	shortItems := shortDoc.Root.Object[0].Value
	shortSession.Expanded[shortItems.ID] = true
	shortSession.Refresh(shortDoc)

	if got, want := len(shortSession.Rows), 103; got != want {
		t.Fatalf("rows = %d, want %d for 100-item array", got, want)
	}
	if shortSession.Rows[2].IsBatch() {
		t.Fatalf("Rows[2] = %#v, want direct item row for 100-item array", shortSession.Rows[2])
	}

	longDoc := testLongScalarArrayDoc(t, 101)
	longSession := New(longDoc, source.Input{Kind: source.KindFile, Path: "sample.json"}, "")
	longItems := longDoc.Root.Object[0].Value
	longSession.Expanded[longItems.ID] = true
	longSession.Refresh(longDoc)

	if got, want := len(longSession.Rows), 5; got != want {
		t.Fatalf("rows = %d, want %d for 101-item array with batches", got, want)
	}
	if got, want := longSession.Rows[2].BatchLabel(), "[0-99]"; !longSession.Rows[2].IsBatch() || got != want {
		t.Fatalf("first batch = %#v, want [0-99]", longSession.Rows[2])
	}
	if got, want := longSession.Rows[3].BatchLabel(), "[100-100]"; !longSession.Rows[3].IsBatch() || got != want {
		t.Fatalf("second batch = %#v, want [100-100]", longSession.Rows[3])
	}

	largerDoc := testLongScalarArrayDoc(t, 250)
	largerSession := New(largerDoc, source.Input{Kind: source.KindFile, Path: "sample.json"}, "")
	largerItems := largerDoc.Root.Object[0].Value
	largerSession.Expanded[largerItems.ID] = true
	largerSession.Refresh(largerDoc)

	if got, want := len(largerSession.Rows), 6; got != want {
		t.Fatalf("rows = %d, want %d for 250-item array with three batches", got, want)
	}
	if got, want := largerSession.Rows[2].BatchLabel(), "[0-99]"; got != want {
		t.Fatalf("Rows[2].BatchLabel() = %q, want %q", got, want)
	}
	if got, want := largerSession.Rows[3].BatchLabel(), "[100-199]"; got != want {
		t.Fatalf("Rows[3].BatchLabel() = %q, want %q", got, want)
	}
	if got, want := largerSession.Rows[4].BatchLabel(), "[200-249]"; got != want {
		t.Fatalf("Rows[4].BatchLabel() = %q, want %q", got, want)
	}
}

func TestMoveIntoAndCollapseLongArrayBatches(t *testing.T) {
	doc := testLongScalarArrayDoc(t, 101)
	s := New(doc, source.Input{Kind: source.KindFile, Path: "sample.json"}, "")
	items := doc.Root.Object[0].Value

	s.SelectNode(items.ID)
	s.MoveInto(doc)
	s.Refresh(doc)
	if !s.Expanded[items.ID] {
		t.Fatal("items array is not expanded")
	}

	s.MoveInto(doc)
	row, ok := s.CurrentRow()
	if !ok {
		t.Fatal("CurrentRow() = false after entering first batch")
	}
	if !row.IsBatch() || row.BatchLabel() != "[0-99]" {
		t.Fatalf("CurrentRow() = %#v, want first batch row", row)
	}

	s.MoveInto(doc)
	s.Refresh(doc)
	row, ok = s.CurrentRow()
	if !ok {
		t.Fatal("CurrentRow() = false after expanding batch")
	}
	if !row.IsBatch() || !row.Expanded {
		t.Fatalf("CurrentRow() = %#v, want expanded batch row", row)
	}

	s.MoveInto(doc)
	row, ok = s.CurrentRow()
	if !ok {
		t.Fatal("CurrentRow() = false after entering first batch item")
	}
	if row.IsBatch() || row.ArrayIndex != 0 {
		t.Fatalf("CurrentRow() = %#v, want first array item row", row)
	}

	s.CollapseSelected(doc)
	s.Refresh(doc)
	row, ok = s.CurrentRow()
	if !ok {
		t.Fatal("CurrentRow() = false after collapsing to batch")
	}
	if !row.IsBatch() || row.BatchLabel() != "[0-99]" {
		t.Fatalf("CurrentRow() = %#v, want first batch row after collapse", row)
	}

	s.CollapseSelected(doc)
	s.Refresh(doc)
	row, ok = s.CurrentRow()
	if !ok {
		t.Fatal("CurrentRow() = false after closing batch")
	}
	if !row.IsBatch() || row.Expanded {
		t.Fatalf("CurrentRow() = %#v, want collapsed batch row", row)
	}

	s.CollapseSelected(doc)
	s.Refresh(doc)
	row, ok = s.CurrentRow()
	if !ok {
		t.Fatal("CurrentRow() = false after returning to array")
	}
	if row.IsBatch() || row.NodeID != items.ID {
		t.Fatalf("CurrentRow() = %#v, want items array row", row)
	}
}

func TestSearchRevealExpandsMatchingLongArrayBatch(t *testing.T) {
	doc := testLongObjectArrayDoc(t, 150)
	s := New(doc, source.Input{Kind: source.KindFile, Path: "sample.json"}, "")
	items := doc.Root.Object[0].Value
	targetID := items.Array[137].Object[0].Value.ID

	s.Search.Query = "item-137"
	s.UpdateSearchHits(doc)
	if got, want := len(s.SearchHits), 1; got != want {
		t.Fatalf("SearchHits = %d, want %d", got, want)
	}

	s.SelectNode(doc.Root.ID)
	if !s.NextSearchHit(doc, false) {
		t.Fatal("NextSearchHit() = false, want true")
	}
	if got, want := s.SelectedID, targetID; got != want {
		t.Fatalf("SelectedID = %d, want %d", got, want)
	}
	if !s.Expanded[items.ID] {
		t.Fatal("items array is not expanded after search reveal")
	}

	batchID := BatchRowID(items.ID, 100)
	if !s.ExpandedBatches[batchID] {
		t.Fatalf("ExpandedBatches[%v] = false, want true for second batch", batchID)
	}
	if !s.Expanded[items.Array[137].ID] {
		t.Fatal("matching array element container is not expanded after search reveal")
	}

	row, ok := s.CurrentRow()
	if !ok {
		t.Fatal("CurrentRow() = false after search reveal")
	}
	if row.IsBatch() || row.NodeID != targetID {
		t.Fatalf("CurrentRow() = %#v, want matching title node row", row)
	}
}

func TestRefreshReselectsVisibleParentWhenBatchRowDisappearsAfterEdit(t *testing.T) {
	doc := testLongScalarArrayDoc(t, 101)
	s := New(doc, source.Input{Kind: source.KindFile, Path: "sample.json"}, "")
	items := doc.Root.Object[0].Value

	s.Expanded[items.ID] = true
	s.SelectRowID(BatchRowID(items.ID, 100))
	s.Refresh(doc)

	if err := doc.Delete(items.Array[100].ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	s.Refresh(doc)

	row, ok := s.CurrentRow()
	if !ok {
		t.Fatal("CurrentRow() = false after batch row disappears")
	}
	if row.IsBatch() || row.NodeID != items.ID {
		t.Fatalf("CurrentRow() = %#v, want items array row after batch disappears", row)
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

func TestSearchHitLookupSetBuildsAndClears(t *testing.T) {
	doc := testDoc(t)
	s := New(doc, source.Input{Kind: source.KindFile}, "")
	targetID := doc.Root.Object[1].Value.Array[0].Object[0].Value.ID

	s.Search.Query = "alpha"
	s.UpdateSearchHits(doc)
	if !s.HasSearchHit(targetID) {
		t.Fatalf("HasSearchHit(%d) = false, want true", targetID)
	}
	if got := len(s.SearchHitSet); got == 0 {
		t.Fatal("SearchHitSet is empty, want populated lookup")
	}

	s.Search.Query = "missing"
	s.UpdateSearchHits(doc)
	if got := len(s.SearchHits); got != 0 {
		t.Fatalf("SearchHits = %d, want 0 after missing query", got)
	}
	if got := len(s.SearchHitSet); got != 0 {
		t.Fatalf("SearchHitSet size = %d, want 0 after missing query", got)
	}
	if s.HasSearchHit(targetID) {
		t.Fatalf("HasSearchHit(%d) = true, want false after clearing query results", targetID)
	}

	s.Search.Query = ""
	s.UpdateSearchHits(doc)
	if got := len(s.SearchHitSet); got != 0 {
		t.Fatalf("SearchHitSet size = %d, want 0 after empty query", got)
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
