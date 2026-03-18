package document

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

func Parse(data []byte) (*Document, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	first, err := dec.Token()
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("empty input")
		}
		return nil, fmt.Errorf("read first token: %w", err)
	}

	var nextID NodeID = 1
	root, err := parseToken(dec, first, &nextID)
	if err != nil {
		return nil, err
	}

	if tok, err := dec.Token(); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("unexpected trailing token %v", tok)
		}
		return nil, fmt.Errorf("read trailing token: %w", err)
	}

	return &Document{Root: root, nextID: nextID}, nil
}

func ParseNode(data []byte) (*Node, error) {
	doc, err := Parse(data)
	if err != nil {
		return nil, err
	}
	return doc.Root, nil
}

func parseToken(dec *json.Decoder, token json.Token, nextID *NodeID) (*Node, error) {
	switch tok := token.(type) {
	case json.Delim:
		switch tok {
		case '{':
			node := &Node{ID: *nextID, Kind: KindObject}
			*nextID++
			for dec.More() {
				keyTok, err := dec.Token()
				if err != nil {
					return nil, fmt.Errorf("read object key: %w", err)
				}
				key, ok := keyTok.(string)
				if !ok {
					return nil, fmt.Errorf("expected object key, got %T", keyTok)
				}
				valueTok, err := dec.Token()
				if err != nil {
					return nil, fmt.Errorf("read object value: %w", err)
				}
				child, err := parseToken(dec, valueTok, nextID)
				if err != nil {
					return nil, err
				}
				node.Object = append(node.Object, ObjectEntry{Key: key, Value: child})
			}
			end, err := dec.Token()
			if err != nil {
				return nil, fmt.Errorf("close object: %w", err)
			}
			if end != json.Delim('}') {
				return nil, fmt.Errorf("expected object close, got %v", end)
			}
			return node, nil
		case '[':
			node := &Node{ID: *nextID, Kind: KindArray}
			*nextID++
			for dec.More() {
				valueTok, err := dec.Token()
				if err != nil {
					return nil, fmt.Errorf("read array value: %w", err)
				}
				child, err := parseToken(dec, valueTok, nextID)
				if err != nil {
					return nil, err
				}
				node.Array = append(node.Array, child)
			}
			end, err := dec.Token()
			if err != nil {
				return nil, fmt.Errorf("close array: %w", err)
			}
			if end != json.Delim(']') {
				return nil, fmt.Errorf("expected array close, got %v", end)
			}
			return node, nil
		default:
			return nil, fmt.Errorf("unexpected delimiter %q", tok)
		}
	case string:
		node := &Node{ID: *nextID, Kind: KindString, String: tok}
		*nextID++
		return node, nil
	case json.Number:
		node := &Node{ID: *nextID, Kind: KindNumber, Number: tok.String()}
		*nextID++
		return node, nil
	case float64:
		node := &Node{ID: *nextID, Kind: KindNumber, Number: json.Number(fmt.Sprintf("%v", tok)).String()}
		*nextID++
		return node, nil
	case bool:
		node := &Node{ID: *nextID, Kind: KindBool, Boolean: tok}
		*nextID++
		return node, nil
	case nil:
		node := &Node{ID: *nextID, Kind: KindNull}
		*nextID++
		return node, nil
	default:
		return nil, fmt.Errorf("unsupported token type %T", token)
	}
}
