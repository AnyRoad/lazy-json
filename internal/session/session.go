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
	ModeSettings Mode = "settings"
)

type Session struct {
	SelectedID document.NodeID
	Expanded   map[document.NodeID]bool
	Rows       []Row
	RowIndex   map[document.NodeID]int
	Dirty      bool
	SourceKind source.Kind
	SourcePath string
	Mode       Mode
	ThemeName  string
	Help       bool
	Search     SearchState
	SearchHits []document.NodeID
	Status     string
	Error      string
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
		Expanded:   expanded,
		SourceKind: src.Kind,
		SourcePath: src.Path,
		Mode:       ModeNormal,
		ThemeName:  themeName,
	}
	s.Refresh(doc)
	return s
}

func (s *Session) Refresh(doc *document.Document) {
	s.Rows = BuildRows(doc, s.Expanded)
	s.RowIndex = make(map[document.NodeID]int, len(s.Rows))
	for idx, row := range s.Rows {
		s.RowIndex[row.NodeID] = idx
	}
	if len(s.Rows) == 0 {
		s.SelectedID = 0
		return
	}
	if s.SelectedID == 0 {
		s.SelectedID = s.Rows[0].NodeID
		return
	}
	if _, ok := s.RowIndex[s.SelectedID]; ok {
		return
	}
	s.reselectVisible(doc)
}

func (s *Session) reselectVisible(doc *document.Document) {
	if doc == nil || doc.Root == nil {
		s.SelectedID = 0
		return
	}
	current := s.SelectedID
	for current != 0 {
		loc, ok := doc.Find(current)
		if !ok || loc.Parent == nil {
			break
		}
		if _, visible := s.RowIndex[loc.Parent.ID]; visible {
			s.SelectedID = loc.Parent.ID
			return
		}
		current = loc.Parent.ID
	}
	s.SelectedID = doc.Root.ID
}

func (s *Session) CurrentRow() (Row, bool) {
	idx, ok := s.RowIndex[s.SelectedID]
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
	s.SelectedID = s.Rows[next].NodeID
}

func (s *Session) MoveToTop() {
	if len(s.Rows) == 0 {
		return
	}
	s.SelectedID = s.Rows[0].NodeID
}

func (s *Session) MoveToBottom() {
	if len(s.Rows) == 0 {
		return
	}
	s.SelectedID = s.Rows[len(s.Rows)-1].NodeID
}

func (s *Session) ExpandSelected() {
	s.Expanded[s.SelectedID] = true
}

func (s *Session) CollapseSelected(doc *document.Document) {
	row, ok := s.CurrentRow()
	if !ok {
		return
	}
	if row.IsContainer && s.Expanded[s.SelectedID] {
		delete(s.Expanded, s.SelectedID)
		return
	}
	if row.ParentID != 0 {
		s.SelectedID = row.ParentID
	}
}

func (s *Session) MoveInto(doc *document.Document) {
	row, ok := s.CurrentRow()
	if !ok {
		return
	}
	if row.IsContainer && !s.Expanded[s.SelectedID] {
		s.Expanded[s.SelectedID] = true
		return
	}
	for _, candidate := range s.Rows {
		if candidate.ParentID == row.NodeID {
			s.SelectedID = candidate.NodeID
			return
		}
	}
}

func (s *Session) ExpandAll(doc *document.Document) {
	s.Expanded = make(map[document.NodeID]bool)
	if doc == nil || doc.Root == nil {
		return
	}
	expandAllContainers(doc.Root, s.Expanded)
}

func (s *Session) ExpandNearestArrayOneLevel(doc *document.Document) bool {
	if doc == nil || doc.Root == nil || s.SelectedID == 0 {
		return false
	}

	target, ok := nearestArrayTarget(doc, s.SelectedID)
	if !ok {
		return false
	}
	if s.Expanded == nil {
		s.Expanded = make(map[document.NodeID]bool)
	}
	s.Expanded[target.ID] = true
	for _, child := range target.Array {
		if child != nil && child.IsContainer() {
			s.Expanded[child.ID] = true
		}
	}
	return true
}

func (s *Session) CollapseNearestArrayElements(doc *document.Document) bool {
	if doc == nil || doc.Root == nil || s.SelectedID == 0 {
		return false
	}

	target, ok := nearestArrayTarget(doc, s.SelectedID)
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

func nearestArrayTarget(doc *document.Document, selectedID document.NodeID) (*document.Node, bool) {
	currentID := selectedID
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
	if doc.Root.IsContainer() {
		s.Expanded[doc.Root.ID] = true
	}
	for _, ancestor := range ancestors {
		s.Expanded[ancestor] = true
	}
	s.SelectedID = nodeID
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
	if doc == nil || doc.Root == nil || s.SelectedID == 0 {
		return false
	}
	loc, ok := doc.Find(s.SelectedID)
	if !ok || loc.Parent == nil {
		return false
	}

	currentID := loc.Parent.ID
	for currentID != 0 {
		current, ok := doc.Find(currentID)
		if !ok || current.Parent == nil {
			return false
		}
		switch current.ParentKind {
		case document.KindObject:
			if current.Index+1 < len(current.Parent.Object) {
				s.SelectedID = current.Parent.Object[current.Index+1].Value.ID
				return true
			}
		case document.KindArray:
			if current.Index+1 < len(current.Parent.Array) {
				s.SelectedID = current.Parent.Array[current.Index+1].ID
				return true
			}
		}
		currentID = current.Parent.ID
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
	current := slices.Index(s.SearchHits, s.SelectedID)
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
