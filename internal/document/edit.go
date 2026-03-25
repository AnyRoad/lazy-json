package document

import (
	"errors"
	"fmt"
)

var ErrDeleteRoot = errors.New("cannot delete the root node")

func (d *Document) Replace(id NodeID, node *Node) error {
	loc, ok := d.Find(id)
	if !ok {
		return fmt.Errorf("node %d not found", id)
	}
	d.assignIDs(node, id)
	if loc.Parent == nil {
		d.Root = node
		return nil
	}
	switch loc.ParentKind {
	case KindObject:
		loc.Parent.Object[loc.Index].Value = node
	case KindArray:
		loc.Parent.Array[loc.Index] = node
	default:
		return fmt.Errorf("unsupported parent kind %s", loc.ParentKind)
	}
	return nil
}

func (d *Document) Restore(id NodeID, node *Node) error {
	loc, ok := d.Find(id)
	if !ok {
		return fmt.Errorf("node %d not found", id)
	}
	if node == nil {
		return fmt.Errorf("replacement node cannot be nil")
	}
	if node.ID != 0 && node.ID != id {
		return fmt.Errorf("replacement root id %d does not match target %d", node.ID, id)
	}
	if node.ID == 0 {
		node.ID = id
	}
	d.reserveIDs(node)
	if loc.Parent == nil {
		d.Root = node
		return nil
	}
	switch loc.ParentKind {
	case KindObject:
		loc.Parent.Object[loc.Index].Value = node
	case KindArray:
		loc.Parent.Array[loc.Index] = node
	default:
		return fmt.Errorf("unsupported parent kind %s", loc.ParentKind)
	}
	return nil
}

func (d *Document) Delete(id NodeID) error {
	loc, ok := d.Find(id)
	if !ok {
		return fmt.Errorf("node %d not found", id)
	}
	if loc.Parent == nil {
		return ErrDeleteRoot
	}
	switch loc.ParentKind {
	case KindObject:
		loc.Parent.Object = append(loc.Parent.Object[:loc.Index], loc.Parent.Object[loc.Index+1:]...)
	case KindArray:
		loc.Parent.Array = append(loc.Parent.Array[:loc.Index], loc.Parent.Array[loc.Index+1:]...)
	default:
		return fmt.Errorf("unsupported parent kind %s", loc.ParentKind)
	}
	return nil
}

func (d *Document) AddObjectEntry(parentID NodeID, key string, node *Node) error {
	loc, ok := d.Find(parentID)
	if !ok {
		return fmt.Errorf("node %d not found", parentID)
	}
	if loc.Node.Kind != KindObject {
		return fmt.Errorf("node %d is not an object", parentID)
	}
	if node == nil {
		return fmt.Errorf("insert node cannot be nil")
	}
	d.AssignIDs(node)
	return d.InsertObjectEntryAt(parentID, len(loc.Node.Object), key, node)
}

func (d *Document) InsertObjectEntryAt(parentID NodeID, index int, key string, node *Node) error {
	loc, ok := d.Find(parentID)
	if !ok {
		return fmt.Errorf("node %d not found", parentID)
	}
	if loc.Node.Kind != KindObject {
		return fmt.Errorf("node %d is not an object", parentID)
	}
	if index < 0 || index > len(loc.Node.Object) {
		return fmt.Errorf("object insert index %d out of range", index)
	}
	if node == nil {
		return fmt.Errorf("insert node cannot be nil")
	}
	if node.ID == 0 {
		d.AssignIDs(node)
	} else {
		d.reserveIDs(node)
	}
	loc.Node.Object = append(loc.Node.Object, ObjectEntry{})
	copy(loc.Node.Object[index+1:], loc.Node.Object[index:])
	loc.Node.Object[index] = ObjectEntry{Key: key, Value: node}
	return nil
}

func (d *Document) AddArrayItem(parentID NodeID, node *Node) error {
	loc, ok := d.Find(parentID)
	if !ok {
		return fmt.Errorf("node %d not found", parentID)
	}
	if loc.Node.Kind != KindArray {
		return fmt.Errorf("node %d is not an array", parentID)
	}
	if node == nil {
		return fmt.Errorf("insert node cannot be nil")
	}
	d.AssignIDs(node)
	return d.InsertArrayItemAt(parentID, len(loc.Node.Array), node)
}

func (d *Document) InsertArrayItemAt(parentID NodeID, index int, node *Node) error {
	loc, ok := d.Find(parentID)
	if !ok {
		return fmt.Errorf("node %d not found", parentID)
	}
	if loc.Node.Kind != KindArray {
		return fmt.Errorf("node %d is not an array", parentID)
	}
	if index < 0 || index > len(loc.Node.Array) {
		return fmt.Errorf("array insert index %d out of range", index)
	}
	if node == nil {
		return fmt.Errorf("insert node cannot be nil")
	}
	if node.ID == 0 {
		d.AssignIDs(node)
	} else {
		d.reserveIDs(node)
	}
	loc.Node.Array = append(loc.Node.Array, nil)
	copy(loc.Node.Array[index+1:], loc.Node.Array[index:])
	loc.Node.Array[index] = node
	return nil
}

func (d *Document) RenameKey(id NodeID, newKey string) error {
	loc, ok := d.Find(id)
	if !ok {
		return fmt.Errorf("node %d not found", id)
	}
	if loc.ParentKind != KindObject || loc.Parent == nil {
		return fmt.Errorf("node %d is not an object child", id)
	}
	loc.Parent.Object[loc.Index].Key = newKey
	return nil
}
