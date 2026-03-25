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
	s.SearchHitSet = nil
	query := strings.TrimSpace(strings.ToLower(s.Search.Query))
	if query == "" || doc == nil || doc.Root == nil {
		return
	}

	collectSearchHits(doc.Root, "$", "$", "", "", -1, query, &s.SearchHits)
	if len(s.SearchHits) == 0 {
		return
	}

	s.SearchHitSet = make(map[document.NodeID]struct{}, len(s.SearchHits))
	for _, hit := range s.SearchHits {
		s.SearchHitSet[hit] = struct{}{}
	}
}

func (s *Session) HasSearchHit(id document.NodeID) bool {
	if len(s.SearchHitSet) == 0 {
		return false
	}
	_, ok := s.SearchHitSet[id]
	return ok
}

func collectSearchHits(node *document.Node, path, pathLower, key, keyLower string, arrayIndex int, query string, hits *[]document.NodeID) {
	if node == nil {
		return
	}

	if searchMatchesNode(node, pathLower, keyLower, query) {
		*hits = append(*hits, node.ID)
	}

	switch node.Kind {
	case document.KindObject:
		for _, entry := range node.Object {
			childPath := appendRowPath(path, entry.Key, -1)
			lowerKey := strings.ToLower(entry.Key)
			collectSearchHits(entry.Value, childPath, appendRowPath(pathLower, lowerKey, -1), entry.Key, lowerKey, -1, query, hits)
		}
	case document.KindArray:
		for index, child := range node.Array {
			childPath := appendRowPath(path, "", index)
			collectSearchHits(child, childPath, childPath, "", "", index, query, hits)
		}
	}
}

func searchMatchesNode(node *document.Node, pathLower, keyLower, query string) bool {
	if strings.Contains(pathLower, query) {
		return true
	}
	if keyLower != "" && strings.Contains(keyLower, query) {
		return true
	}
	return strings.Contains(strings.ToLower(node.Summary()), query)
}
