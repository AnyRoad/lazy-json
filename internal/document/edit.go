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
	d.AssignIDs(node)
	loc.Node.Object = append(loc.Node.Object, ObjectEntry{Key: key, Value: node})
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
	d.AssignIDs(node)
	loc.Node.Array = append(loc.Node.Array, node)
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
