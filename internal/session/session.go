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

func (s *Session) NextSearchHit(reverse bool) {
	if len(s.SearchHits) == 0 {
		return
	}
	current := slices.Index(s.SearchHits, s.SelectedID)
	if current == -1 {
		if reverse {
			s.SelectedID = s.SearchHits[len(s.SearchHits)-1]
		} else {
			s.SelectedID = s.SearchHits[0]
		}
		return
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
	s.SelectedID = s.SearchHits[current]
}
