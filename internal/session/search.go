package session

import (
	"strings"

	"github.com/anyroad/lazy-json/internal/document"
)

type SearchState struct {
	Query string
}

func (s *Session) UpdateSearchHits(doc *document.Document) {
	s.SearchHits = nil
	query := strings.TrimSpace(strings.ToLower(s.Search.Query))
	if query == "" || doc == nil || doc.Root == nil {
		return
	}
	collectSearchHits(doc.Root, "$", "", -1, query, &s.SearchHits)
}

func (s *Session) HasSearchHit(id document.NodeID) bool {
	for _, hit := range s.SearchHits {
		if hit == id {
			return true
		}
	}
	return false
}

func collectSearchHits(node *document.Node, path, key string, arrayIndex int, query string, hits *[]document.NodeID) {
	if node == nil {
		return
	}

	row := Row{
		NodeID:     node.ID,
		Key:        key,
		ArrayIndex: arrayIndex,
		Path:       path,
		Summary:    node.Summary(),
	}
	if strings.Contains(SearchableText(row), query) {
		*hits = append(*hits, node.ID)
	}

	switch node.Kind {
	case document.KindObject:
		for _, entry := range node.Object {
			collectSearchHits(entry.Value, appendRowPath(path, entry.Key, -1), entry.Key, -1, query, hits)
		}
	case document.KindArray:
		for index, child := range node.Array {
			collectSearchHits(child, appendRowPath(path, "", index), "", index, query, hits)
		}
	}
}
