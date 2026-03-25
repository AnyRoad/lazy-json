package session

import (
	"slices"
	"strings"

	"github.com/anyroad/lazy-json/internal/config"
	"github.com/anyroad/lazy-json/internal/document"
	"github.com/anyroad/lazy-json/internal/source"
)

type Mode string

const (
	ModeNormal   Mode = "normal"
	ModePrompt   Mode = "prompt"
	ModeCommand  Mode = "command"
	ModeSearch   Mode = "search"
	ModeBusy     Mode = "busy"
	ModeSettings Mode = "settings"
)

type Session struct {
	SelectedID      document.NodeID
	SelectedRowID   RowID
	Expanded        map[document.NodeID]bool
	ExpandedBatches map[RowID]bool
	Rows            []Row
	RowIndex        map[document.NodeID]int
	RowKeyIndex     map[RowID]int
	Dirty           bool
	SourceKind      source.Kind
	SourcePath      string
	Mode            Mode
	ThemeName       string
	Help            bool
	Search          SearchState
	SearchHits      []document.NodeID
	SearchHitSet    map[document.NodeID]struct{}
	Status          string
	Error           string
}

func New(doc *document.Document, src source.Input, themeName string) *Session {
	expanded := map[document.NodeID]bool{}
	if doc != nil && doc.Root != nil {
		expanded[doc.Root.ID] = true
	}
	themeName = strings.TrimSpace(themeName)
	if themeName == "" {
		themeName = config.DefaultThemeName
	}
	s := &Session{
		Expanded:        expanded,
		ExpandedBatches: map[RowID]bool{},
		SourceKind:      src.Kind,
		SourcePath:      src.Path,
		Mode:            ModeNormal,
		ThemeName:       themeName,
	}
	s.Refresh(doc)
	return s
}

func (s *Session) SelectedRow() RowID {
	return s.selectedRowID()
}

func (s *Session) SelectNode(nodeID document.NodeID) {
	s.setSelectedRowID(NodeRowID(nodeID))
}

func (s *Session) SelectRowID(rowID RowID) {
	s.setSelectedRowID(rowID)
}

func (s *Session) Refresh(doc *document.Document) {
	if s.ExpandedBatches == nil {
		s.ExpandedBatches = make(map[RowID]bool)
	}

	s.Rows = BuildRows(doc, s.Expanded, s.ExpandedBatches)
	s.RowIndex = make(map[document.NodeID]int, len(s.Rows))
	s.RowKeyIndex = make(map[RowID]int, len(s.Rows))
	for idx, row := range s.Rows {
		s.RowKeyIndex[row.ID] = idx
		if !row.IsBatch() {
			s.RowIndex[row.NodeID] = idx
		}
	}

	selected := s.selectedRowID()
	if len(s.Rows) == 0 {
		s.SelectedID = 0
		s.SelectedRowID = RowID{}
		return
	}
	if selected.IsZero() {
		s.setSelectedRowID(s.Rows[0].ID)
		return
	}
	if _, ok := s.RowKeyIndex[selected]; ok {
		s.setSelectedRowID(selected)
		return
	}
	s.reselectVisible(doc, selected)
}

func (s *Session) reselectVisible(doc *document.Document, selected RowID) {
	if doc == nil || doc.Root == nil {
		s.SelectedID = 0
		s.SelectedRowID = RowID{}
		return
	}

	current := selected
	for !current.IsZero() {
		parent, ok := logicalParentRowID(doc, current)
		if !ok {
			break
		}
		if _, visible := s.RowKeyIndex[parent]; visible {
			s.setSelectedRowID(parent)
			return
		}
		current = parent
	}
	s.setSelectedRowID(NodeRowID(doc.Root.ID))
}

func (s *Session) CurrentRow() (Row, bool) {
	idx, ok := s.RowKeyIndex[s.selectedRowID()]
	if !ok {
		return Row{}, false
	}
	return s.Rows[idx], true
}

func (s *Session) Move(delta int) {
	row, ok := s.CurrentRow()
	if !ok {
		return
	}
	next := row.Index + delta
	if next < 0 {
		next = 0
	}
	if next >= len(s.Rows) {
		next = len(s.Rows) - 1
	}
	s.setSelectedRowID(s.Rows[next].ID)
}

func (s *Session) MoveToTop() {
	if len(s.Rows) == 0 {
		return
	}
	s.setSelectedRowID(s.Rows[0].ID)
}

func (s *Session) MoveToBottom() {
	if len(s.Rows) == 0 {
		return
	}
	s.setSelectedRowID(s.Rows[len(s.Rows)-1].ID)
}

func (s *Session) ExpandSelected() {
	row, ok := s.CurrentRow()
	if !ok || !row.IsContainer {
		return
	}
	s.expandRow(row)
}

func (s *Session) CollapseSelected(doc *document.Document) {
	row, ok := s.CurrentRow()
	if !ok {
		return
	}
	if row.IsContainer && row.Expanded {
		s.collapseRow(row)
		return
	}
	if !row.ParentRowID.IsZero() {
		s.setSelectedRowID(row.ParentRowID)
	}
}

func (s *Session) MoveInto(doc *document.Document) {
	row, ok := s.CurrentRow()
	if !ok {
		return
	}
	if row.IsContainer && !row.Expanded {
		s.expandRow(row)
		return
	}
	for _, candidate := range s.Rows {
		if candidate.ParentRowID == row.ID {
			s.setSelectedRowID(candidate.ID)
			return
		}
	}
}

func (s *Session) ExpandAll(doc *document.Document) {
	s.Expanded = make(map[document.NodeID]bool)
	s.ExpandedBatches = make(map[RowID]bool)
	if doc == nil || doc.Root == nil {
		return
	}
	expandAllContainers(doc.Root, s.Expanded)
}

func (s *Session) ExpandNearestArrayOneLevel(doc *document.Document) bool {
	if doc == nil || doc.Root == nil {
		return false
	}

	target, ok := s.nearestArrayTarget(doc)
	if !ok {
		return false
	}
	if s.Expanded == nil {
		s.Expanded = make(map[document.NodeID]bool)
	}
	if s.ExpandedBatches == nil {
		s.ExpandedBatches = make(map[RowID]bool)
	}
	s.Expanded[target.ID] = true
	children := target.Array
	if shouldBatchArray(target) {
		batchID, ok := s.currentBatchRowIDForArray(doc, target.ID)
		if !ok {
			batchID = BatchRowID(target.ID, 0)
		}
		s.ExpandedBatches[batchID] = true
		start := batchID.BatchStart()
		end := start + longArrayBatchSize
		if end > len(children) {
			end = len(children)
		}
		children = children[start:end]
	}
	for _, child := range children {
		if child != nil && child.IsContainer() {
			s.Expanded[child.ID] = true
		}
	}
	return true
}

func (s *Session) CollapseNearestArrayElements(doc *document.Document) bool {
	if doc == nil || doc.Root == nil {
		return false
	}

	target, ok := s.nearestArrayTarget(doc)
	if !ok {
		return false
	}
	if s.Expanded == nil {
		s.Expanded = make(map[document.NodeID]bool)
	}
	s.Expanded[target.ID] = true
	for _, child := range target.Array {
		collapseExpandedSubtree(child, s.Expanded)
	}
	s.deleteBatchExpansionsForArray(target.ID)
	return true
}

func expandAllContainers(node *document.Node, expanded map[document.NodeID]bool) {
	if node == nil || !node.IsContainer() {
		return
	}
	expanded[node.ID] = true
	switch node.Kind {
	case document.KindObject:
		for _, entry := range node.Object {
			expandAllContainers(entry.Value, expanded)
		}
	case document.KindArray:
		for _, child := range node.Array {
			expandAllContainers(child, expanded)
		}
	}
}

func collapseExpandedSubtree(node *document.Node, expanded map[document.NodeID]bool) {
	if node == nil || !node.IsContainer() {
		return
	}
	delete(expanded, node.ID)
	switch node.Kind {
	case document.KindObject:
		for _, entry := range node.Object {
			collapseExpandedSubtree(entry.Value, expanded)
		}
	case document.KindArray:
		for _, child := range node.Array {
			collapseExpandedSubtree(child, expanded)
		}
	}
}

func (s *Session) CollapseAll(doc *document.Document) {
	s.Expanded = make(map[document.NodeID]bool)
	s.ExpandedBatches = make(map[RowID]bool)
	if doc == nil || doc.Root == nil {
		return
	}
	if doc.Root.IsContainer() {
		s.Expanded[doc.Root.ID] = true
	}
}

func (s *Session) RevealSelection(doc *document.Document, nodeID document.NodeID, ancestors []document.NodeID) {
	if doc == nil || doc.Root == nil || nodeID == 0 {
		return
	}
	if s.Expanded == nil {
		s.Expanded = make(map[document.NodeID]bool)
	}
	if s.ExpandedBatches == nil {
		s.ExpandedBatches = make(map[RowID]bool)
	}
	if doc.Root.IsContainer() {
		s.Expanded[doc.Root.ID] = true
	}
	for _, ancestor := range ancestors {
		s.Expanded[ancestor] = true
	}
	for _, batchID := range batchRowAncestors(doc, nodeID) {
		s.ExpandedBatches[batchID] = true
	}
	s.setSelectedRowID(NodeRowID(nodeID))
	s.Refresh(doc)
}

func (s *Session) RevealNode(doc *document.Document, nodeID document.NodeID) bool {
	ancestors, ok := nodeAncestors(doc, nodeID)
	if !ok {
		return false
	}
	s.RevealSelection(doc, nodeID, ancestors)
	return true
}

func (s *Session) NextParentSibling(doc *document.Document) bool {
	row, ok := s.CurrentRow()
	if !ok {
		return false
	}

	current := row
	for !current.ParentRowID.IsZero() {
		parentIndex, ok := s.RowKeyIndex[current.ParentRowID]
		if !ok {
			return false
		}
		parent := s.Rows[parentIndex]
		for idx := parentIndex + 1; idx < len(s.Rows); idx++ {
			candidate := s.Rows[idx]
			if candidate.ParentRowID == parent.ParentRowID {
				s.setSelectedRowID(candidate.ID)
				return true
			}
		}
		current = parent
	}
	return false
}

func (s *Session) SetStatus(message string) {
	s.Status = message
	s.Error = ""
}

func (s *Session) SetError(message string) {
	s.Error = message
	s.Status = ""
}

func (s *Session) ClearMessages() {
	s.Status = ""
	s.Error = ""
}

func (s *Session) NextSearchHit(doc *document.Document, reverse bool) bool {
	if len(s.SearchHits) == 0 {
		return false
	}

	currentID := document.NodeID(0)
	if row, ok := s.CurrentRow(); ok && !row.IsBatch() {
		currentID = row.NodeID
	}

	current := slices.Index(s.SearchHits, currentID)
	targetID := document.NodeID(0)
	if current == -1 {
		if reverse {
			targetID = s.SearchHits[len(s.SearchHits)-1]
		} else {
			targetID = s.SearchHits[0]
		}
		return s.RevealNode(doc, targetID)
	}
	if reverse {
		current--
		if current < 0 {
			current = len(s.SearchHits) - 1
		}
	} else {
		current++
		if current >= len(s.SearchHits) {
			current = 0
		}
	}
	targetID = s.SearchHits[current]
	return s.RevealNode(doc, targetID)
}

func nodeAncestors(doc *document.Document, nodeID document.NodeID) ([]document.NodeID, bool) {
	if doc == nil || doc.Root == nil || nodeID == 0 {
		return nil, false
	}
	ancestors := make([]document.NodeID, 0, 4)
	currentID := nodeID
	for currentID != 0 {
		loc, ok := doc.Find(currentID)
		if !ok {
			return nil, false
		}
		if loc.Parent == nil {
			break
		}
		ancestors = append(ancestors, loc.Parent.ID)
		currentID = loc.Parent.ID
	}
	slices.Reverse(ancestors)
	return ancestors, true
}

func batchRowAncestors(doc *document.Document, nodeID document.NodeID) []RowID {
	if doc == nil || doc.Root == nil || nodeID == 0 {
		return nil
	}

	batches := make([]RowID, 0, 2)
	currentID := nodeID
	for currentID != 0 {
		loc, ok := doc.Find(currentID)
		if !ok || loc.Parent == nil {
			break
		}
		if loc.ParentKind == document.KindArray && shouldBatchArray(loc.Parent) {
			batches = append(batches, BatchRowID(loc.Parent.ID, batchStartForIndex(loc.Index)))
		}
		currentID = loc.Parent.ID
	}
	return batches
}

func logicalParentRowID(doc *document.Document, rowID RowID) (RowID, bool) {
	if doc == nil || doc.Root == nil || rowID.IsZero() {
		return RowID{}, false
	}
	if rowID.IsBatch() {
		return NodeRowID(rowID.NodeID()), true
	}

	loc, ok := doc.Find(rowID.NodeID())
	if !ok || loc.Parent == nil {
		return RowID{}, false
	}
	if loc.ParentKind == document.KindArray && shouldBatchArray(loc.Parent) {
		return BatchRowID(loc.Parent.ID, batchStartForIndex(loc.Index)), true
	}
	return NodeRowID(loc.Parent.ID), true
}

func (s *Session) selectedRowID() RowID {
	if s.SelectedID != 0 {
		if !s.SelectedRowID.IsZero() && s.SelectedRowID.NodeID() == s.SelectedID {
			return s.SelectedRowID
		}
		return NodeRowID(s.SelectedID)
	}
	return s.SelectedRowID
}

func (s *Session) setSelectedRowID(rowID RowID) {
	s.SelectedRowID = rowID
	s.SelectedID = rowID.NodeID()
}

func (s *Session) expandRow(row Row) {
	if row.IsBatch() {
		if s.ExpandedBatches == nil {
			s.ExpandedBatches = make(map[RowID]bool)
		}
		s.ExpandedBatches[row.ID] = true
		return
	}
	if s.Expanded == nil {
		s.Expanded = make(map[document.NodeID]bool)
	}
	s.Expanded[row.NodeID] = true
}

func (s *Session) collapseRow(row Row) {
	if row.IsBatch() {
		delete(s.ExpandedBatches, row.ID)
		return
	}
	delete(s.Expanded, row.NodeID)
}

func (s *Session) nearestArrayTarget(doc *document.Document) (*document.Node, bool) {
	row, ok := s.CurrentRow()
	if !ok {
		return nil, false
	}

	currentID := row.NodeID
	for currentID != 0 {
		loc, ok := doc.Find(currentID)
		if !ok || loc.Node == nil {
			return nil, false
		}
		if loc.Node.Kind == document.KindArray {
			return loc.Node, true
		}
		if loc.Parent == nil {
			return nil, false
		}
		currentID = loc.Parent.ID
	}
	return nil, false
}

func (s *Session) currentBatchRowIDForArray(doc *document.Document, arrayID document.NodeID) (RowID, bool) {
	row, ok := s.CurrentRow()
	if !ok {
		return RowID{}, false
	}
	if row.IsBatch() && row.NodeID == arrayID {
		return row.ID, true
	}
	if row.NodeID == 0 || row.NodeID == arrayID {
		return RowID{}, false
	}
	for _, batchID := range batchRowAncestors(doc, row.NodeID) {
		if batchID.NodeID() == arrayID {
			return batchID, true
		}
	}
	return RowID{}, false
}

func (s *Session) deleteBatchExpansionsForArray(arrayID document.NodeID) {
	for rowID := range s.ExpandedBatches {
		if rowID.IsBatch() && rowID.NodeID() == arrayID {
			delete(s.ExpandedBatches, rowID)
		}
	}
}
