package session

import (
	"fmt"
	"strings"

	"github.com/anyroad/lazy-json/internal/document"
)

type Row struct {
	Index       int
	NodeID      document.NodeID
	ParentID    document.NodeID
	Depth       int
	Key         string
	ArrayIndex  int
	Path        string
	Kind        document.Kind
	IsContainer bool
	Expanded    bool
	Summary     string
}

func BuildRows(doc *document.Document, expanded map[document.NodeID]bool) []Row {
	if doc == nil || doc.Root == nil {
		return nil
	}
	rows := []Row{}
	buildRows(doc.Root, 0, "", -1, 0, expanded, &rows)
	for idx := range rows {
		rows[idx].Index = idx
	}
	return rows
}

func buildRows(node *document.Node, depth int, key string, arrayIndex int, parentID document.NodeID, expanded map[document.NodeID]bool, rows *[]Row) {
	row := Row{
		NodeID:      node.ID,
		ParentID:    parentID,
		Depth:       depth,
		Key:         key,
		ArrayIndex:  arrayIndex,
		Path:        joinPath(parentID, key, arrayIndex, rows),
		Kind:        node.Kind,
		IsContainer: node.IsContainer(),
		Expanded:    expanded[node.ID],
		Summary:     node.Summary(),
	}
	*rows = append(*rows, row)
	if !row.IsContainer || !row.Expanded {
		return
	}
	switch node.Kind {
	case document.KindObject:
		for _, entry := range node.Object {
			buildRows(entry.Value, depth+1, entry.Key, -1, node.ID, expanded, rows)
		}
	case document.KindArray:
		for idx, child := range node.Array {
			buildRows(child, depth+1, "", idx, node.ID, expanded, rows)
		}
	}
}

func joinPath(parentID document.NodeID, key string, arrayIndex int, rows *[]Row) string {
	if parentID == 0 {
		return "$"
	}
	parentPath := "$"
	for i := len(*rows) - 1; i >= 0; i-- {
		if (*rows)[i].NodeID == parentID {
			parentPath = (*rows)[i].Path
			break
		}
	}
	if arrayIndex >= 0 {
		return fmt.Sprintf("%s[%d]", parentPath, arrayIndex)
	}
	if key == "" {
		return parentPath
	}
	if simpleIdentifier(key) {
		return parentPath + "." + key
	}
	return fmt.Sprintf("%s[%q]", parentPath, key)
}

func simpleIdentifier(key string) bool {
	if key == "" {
		return false
	}
	for idx, r := range key {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' || (idx > 0 && r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
}

func SearchableText(row Row) string {
	parts := []string{row.Path, row.Summary}
	if row.Key != "" {
		parts = append(parts, row.Key)
	}
	return strings.ToLower(strings.Join(parts, " "))
}
