package document

import (
	"encoding/json"
	"fmt"
	"strings"
)

const defaultIndent = "  "

func (d *Document) MarshalIndent() ([]byte, error) {
	if d.Root == nil {
		return nil, fmt.Errorf("document is empty")
	}
	return MarshalIndentNodeWithIndent(d.Root, defaultIndent)
}

func (d *Document) MarshalIndentWith(indent string) ([]byte, error) {
	if d.Root == nil {
		return nil, fmt.Errorf("document is empty")
	}
	return MarshalIndentNodeWithIndent(d.Root, indent)
}

func MarshalIndentNode(node *Node) ([]byte, error) {
	return MarshalIndentNodeWithIndent(node, defaultIndent)
}

func MarshalIndentNodeWithIndent(node *Node, indent string) ([]byte, error) {
	var b strings.Builder
	if err := writeNode(&b, node, 0, normalizeIndent(indent)); err != nil {
		return nil, err
	}
	b.WriteByte('\n')
	return []byte(b.String()), nil
}

func MarshalCompactNode(node *Node) ([]byte, error) {
	var b strings.Builder
	if err := writeCompactNode(&b, node); err != nil {
		return nil, err
	}
	return []byte(b.String()), nil
}

func normalizeIndent(indent string) string {
	if indent == "" {
		return defaultIndent
	}
	return indent
}

func writeNode(b *strings.Builder, node *Node, depth int, indent string) error {
	switch node.Kind {
	case KindObject:
		if len(node.Object) == 0 {
			b.WriteString("{}")
			return nil
		}
		b.WriteString("{\n")
		for i, entry := range node.Object {
			b.WriteString(strings.Repeat(indent, depth+1))
			key, _ := json.Marshal(entry.Key)
			b.Write(key)
			b.WriteString(": ")
			if err := writeNode(b, entry.Value, depth+1, indent); err != nil {
				return err
			}
			if i < len(node.Object)-1 {
				b.WriteByte(',')
			}
			b.WriteByte('\n')
		}
		b.WriteString(strings.Repeat(indent, depth))
		b.WriteByte('}')
	case KindArray:
		if len(node.Array) == 0 {
			b.WriteString("[]")
			return nil
		}
		b.WriteString("[\n")
		for i, child := range node.Array {
			b.WriteString(strings.Repeat(indent, depth+1))
			if err := writeNode(b, child, depth+1, indent); err != nil {
				return err
			}
			if i < len(node.Array)-1 {
				b.WriteByte(',')
			}
			b.WriteByte('\n')
		}
		b.WriteString(strings.Repeat(indent, depth))
		b.WriteByte(']')
	case KindString:
		data, _ := json.Marshal(node.String)
		b.Write(data)
	case KindNumber:
		b.WriteString(node.Number)
	case KindBool:
		if node.Boolean {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case KindNull:
		b.WriteString("null")
	default:
		return fmt.Errorf("unsupported node kind %s", node.Kind)
	}
	return nil
}

func writeCompactNode(b *strings.Builder, node *Node) error {
	switch node.Kind {
	case KindObject:
		b.WriteByte('{')
		for i, entry := range node.Object {
			key, _ := json.Marshal(entry.Key)
			b.Write(key)
			b.WriteByte(':')
			if err := writeCompactNode(b, entry.Value); err != nil {
				return err
			}
			if i < len(node.Object)-1 {
				b.WriteByte(',')
			}
		}
		b.WriteByte('}')
	case KindArray:
		b.WriteByte('[')
		for i, child := range node.Array {
			if err := writeCompactNode(b, child); err != nil {
				return err
			}
			if i < len(node.Array)-1 {
				b.WriteByte(',')
			}
		}
		b.WriteByte(']')
	case KindString:
		data, _ := json.Marshal(node.String)
		b.Write(data)
	case KindNumber:
		b.WriteString(node.Number)
	case KindBool:
		if node.Boolean {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case KindNull:
		b.WriteString("null")
	default:
		return fmt.Errorf("unsupported node kind %s", node.Kind)
	}
	return nil
}
