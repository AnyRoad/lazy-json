package session

import (
	"fmt"
	"strings"

	"github.com/anyroad/lazy-json/internal/document"
)

const longArrayBatchSize = 100

type rowKind int

const (
	rowKindNode rowKind = iota
	rowKindBatch
)

type RowID struct {
	kind       rowKind
	nodeID     document.NodeID
	batchStart int
}

func NodeRowID(nodeID document.NodeID) RowID {
	return RowID{nodeID: nodeID}
}

func BatchRowID(arrayID document.NodeID, start int) RowID {
	return RowID{
		kind:       rowKindBatch,
		nodeID:     arrayID,
		batchStart: start,
	}
}

func (id RowID) IsZero() bool {
	return id.nodeID == 0
}

func (id RowID) IsBatch() bool {
	return id.kind == rowKindBatch && id.nodeID != 0
}

func (id RowID) NodeID() document.NodeID {
	return id.nodeID
}

func (id RowID) BatchStart() int {
	return id.batchStart
}

type Row struct {
	Index       int
	LineNumber  int
	ID          RowID
	NodeID      document.NodeID
	ParentID    document.NodeID
	ParentRowID RowID
	Depth       int
	Key         string
	ArrayIndex  int
	Path        string
	Kind        document.Kind
	IsContainer bool
	Expanded    bool
	Summary     string
	StringValue string
	BatchEnd    int
}

func (r Row) IsBatch() bool {
	return r.ID.IsBatch()
}

func (r Row) BatchStart() int {
	return r.ID.BatchStart()
}

func (r Row) BatchLabel() string {
	return fmt.Sprintf("[%d-%d]", r.BatchStart(), r.BatchEnd)
}

func BuildRows(doc *document.Document, expanded map[document.NodeID]bool, expandedBatches map[RowID]bool) []Row {
	if doc == nil || doc.Root == nil {
		return nil
	}
	rows := []Row{}
	lineNumber := 0
	buildRows(doc.Root, 0, "", -1, 0, RowID{}, "", expanded, expandedBatches, true, &lineNumber, &rows)
	for idx := range rows {
		rows[idx].Index = idx
	}
	return rows
}

func buildRows(
	node *document.Node,
	depth int,
	key string,
	arrayIndex int,
	parentID document.NodeID,
	parentRowID RowID,
	parentPath string,
	expanded map[document.NodeID]bool,
	expandedBatches map[RowID]bool,
	visible bool,
	lineNumber *int,
	rows *[]Row,
) {
	if !visible {
		countHiddenRows(node, lineNumber)
		return
	}

	path := "$"
	if parentID != 0 {
		path = appendRowPath(parentPath, key, arrayIndex)
	}
	*lineNumber++

	row := Row{
		LineNumber:  *lineNumber,
		ID:          NodeRowID(node.ID),
		NodeID:      node.ID,
		ParentID:    parentID,
		ParentRowID: parentRowID,
		Depth:       depth,
		Key:         key,
		ArrayIndex:  arrayIndex,
		Path:        path,
		Kind:        node.Kind,
		IsContainer: node.IsContainer(),
		Expanded:    expanded[node.ID],
		Summary:     node.Summary(),
		StringValue: node.String,
	}
	*rows = append(*rows, row)

	childVisible := row.IsContainer && row.Expanded
	switch node.Kind {
	case document.KindObject:
		for _, entry := range node.Object {
			buildRows(entry.Value, depth+1, entry.Key, -1, node.ID, row.ID, path, expanded, expandedBatches, childVisible, lineNumber, rows)
		}
	case document.KindArray:
		if shouldBatchArray(node) {
			buildArrayBatchRows(node, depth+1, path, row.ID, expanded, expandedBatches, childVisible, lineNumber, rows)
			return
		}
		for idx, child := range node.Array {
			buildRows(child, depth+1, "", idx, node.ID, row.ID, path, expanded, expandedBatches, childVisible, lineNumber, rows)
		}
	}
}

func countHiddenRows(node *document.Node, lineNumber *int) {
	if node == nil {
		return
	}

	*lineNumber++
	switch node.Kind {
	case document.KindObject:
		for _, entry := range node.Object {
			countHiddenRows(entry.Value, lineNumber)
		}
	case document.KindArray:
		if shouldBatchArray(node) {
			countHiddenBatchRows(node, lineNumber)
			return
		}
		for _, child := range node.Array {
			countHiddenRows(child, lineNumber)
		}
	}
}

func countHiddenBatchRows(node *document.Node, lineNumber *int) {
	for start := 0; start < len(node.Array); start += longArrayBatchSize {
		end := start + longArrayBatchSize
		if end > len(node.Array) {
			end = len(node.Array)
		}
		*lineNumber++
		for _, child := range node.Array[start:end] {
			countHiddenRows(child, lineNumber)
		}
	}
}

func buildArrayBatchRows(
	node *document.Node,
	depth int,
	path string,
	parentRowID RowID,
	expanded map[document.NodeID]bool,
	expandedBatches map[RowID]bool,
	visible bool,
	lineNumber *int,
	rows *[]Row,
) {
	for start := 0; start < len(node.Array); start += longArrayBatchSize {
		end := start + longArrayBatchSize - 1
		if end >= len(node.Array) {
			end = len(node.Array) - 1
		}

		batchID := BatchRowID(node.ID, start)
		*lineNumber++
		row := Row{
			LineNumber:  *lineNumber,
			ID:          batchID,
			NodeID:      node.ID,
			ParentID:    node.ID,
			ParentRowID: parentRowID,
			Depth:       depth,
			ArrayIndex:  -1,
			Path:        path,
			Kind:        document.KindArray,
			IsContainer: true,
			Expanded:    expandedBatches[batchID],
			Summary:     fmt.Sprintf("%d items", end-start+1),
			BatchEnd:    end,
		}
		if visible {
			*rows = append(*rows, row)
		}

		childVisible := visible && row.Expanded
		for idx := start; idx <= end; idx++ {
			buildRows(node.Array[idx], depth+1, "", idx, node.ID, batchID, path, expanded, expandedBatches, childVisible, lineNumber, rows)
		}
	}
}

func shouldBatchArray(node *document.Node) bool {
	return node != nil && node.Kind == document.KindArray && len(node.Array) > longArrayBatchSize
}

func batchStartForIndex(index int) int {
	if index < 0 {
		return 0
	}
	return (index / longArrayBatchSize) * longArrayBatchSize
}

func appendRowPath(parentPath, key string, arrayIndex int) string {
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
