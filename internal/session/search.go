package session

import (
	"strings"

	"github.com/andrei/lazy-json/internal/document"
)

type SearchState struct {
	Query string
}

func (s *Session) UpdateSearchHits() {
	s.SearchHits = nil
	query := strings.TrimSpace(strings.ToLower(s.Search.Query))
	if query == "" {
		return
	}
	for _, row := range s.Rows {
		if strings.Contains(SearchableText(row), query) {
			s.SearchHits = append(s.SearchHits, row.NodeID)
		}
	}
}

func (s *Session) HasSearchHit(id document.NodeID) bool {
	for _, hit := range s.SearchHits {
		if hit == id {
			return true
		}
	}
	return false
}
