package document

import "fmt"

type NodeID int64

type Kind string

const (
	KindObject Kind = "object"
	KindArray  Kind = "array"
	KindString Kind = "string"
	KindNumber Kind = "number"
	KindBool   Kind = "bool"
	KindNull   Kind = "null"
)

type ObjectEntry struct {
	Key   string
	Value *Node
}

type Node struct {
	ID      NodeID
	Kind    Kind
	Object  []ObjectEntry
	Array   []*Node
	String  string
	Number  string
	Boolean bool
}

type Document struct {
	Root   *Node
	nextID NodeID
}

type Location struct {
	Node       *Node
	Parent     *Node
	ParentKind Kind
	Index      int
	Key        string
}

func New(root *Node) *Document {
	doc := &Document{Root: root}
	doc.nextID = doc.maxID(root) + 1
	return doc
}

func (d *Document) maxID(node *Node) NodeID {
	if node == nil {
		return 0
	}
	max := node.ID
	switch node.Kind {
	case KindObject:
		for _, entry := range node.Object {
			if id := d.maxID(entry.Value); id > max {
				max = id
			}
		}
	case KindArray:
		for _, child := range node.Array {
			if id := d.maxID(child); id > max {
				max = id
			}
		}
	}
	return max
}

func (d *Document) newID() NodeID {
	id := d.nextID
	d.nextID++
	return id
}

func (n *Node) IsContainer() bool {
	return n != nil && (n.Kind == KindObject || n.Kind == KindArray)
}

func (n *Node) Len() int {
	switch n.Kind {
	case KindObject:
		return len(n.Object)
	case KindArray:
		return len(n.Array)
	default:
		return 0
	}
}

func (n *Node) Summary() string {
	switch n.Kind {
	case KindObject:
		return fmt.Sprintf("{%d}", len(n.Object))
	case KindArray:
		return fmt.Sprintf("[%d]", len(n.Array))
	case KindString:
		return n.String
	case KindNumber:
		return n.Number
	case KindBool:
		if n.Boolean {
			return "true"
		}
		return "false"
	case KindNull:
		return "null"
	default:
		return ""
	}
}

func (d *Document) Find(id NodeID) (Location, bool) {
	if d.Root == nil {
		return Location{}, false
	}
	if d.Root.ID == id {
		return Location{Node: d.Root}, true
	}
	return findLocation(d.Root, nil, id)
}

func findLocation(node, parent *Node, id NodeID) (Location, bool) {
	switch node.Kind {
	case KindObject:
		for idx, entry := range node.Object {
			if entry.Value.ID == id {
				return Location{
					Node:       entry.Value,
					Parent:     node,
					ParentKind: KindObject,
					Index:      idx,
					Key:        entry.Key,
				}, true
			}
			if loc, ok := findLocation(entry.Value, node, id); ok {
				return loc, true
			}
		}
	case KindArray:
		for idx, child := range node.Array {
			if child.ID == id {
				return Location{
					Node:       child,
					Parent:     node,
					ParentKind: KindArray,
					Index:      idx,
				}, true
			}
			if loc, ok := findLocation(child, node, id); ok {
				return loc, true
			}
		}
	}
	return Location{}, false
}

func (d *Document) AssignIDs(node *Node) {
	d.assignIDs(node, 0)
}

func (d *Document) assignIDs(node *Node, preserve NodeID) {
	if node == nil {
		return
	}
	if preserve != 0 {
		node.ID = preserve
	} else {
		node.ID = d.newID()
	}
	switch node.Kind {
	case KindObject:
		for _, entry := range node.Object {
			d.assignIDs(entry.Value, 0)
		}
	case KindArray:
		for _, child := range node.Array {
			d.assignIDs(child, 0)
		}
	}
}
