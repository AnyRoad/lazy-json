package document

import (
	"fmt"
	"strconv"
	"strings"
)

type PathResolution struct {
	NodeID      NodeID
	MatchedPath string
	Ancestors   []NodeID
	Exact       bool
}

type pathSegment struct {
	key     string
	index   int
	isIndex bool
}

func (d *Document) ResolvePath(path string) (PathResolution, error) {
	if d == nil || d.Root == nil {
		return PathResolution{}, fmt.Errorf("document is empty")
	}

	segments, err := parsePath(strings.TrimSpace(path))
	if err != nil {
		return PathResolution{}, err
	}

	current := d.Root
	currentPath := "$"
	ancestors := make([]NodeID, 0, len(segments))

	for _, segment := range segments {
		next, nextPath, ok := resolvePathSegment(current, currentPath, segment)
		if !ok {
			return PathResolution{
				NodeID:      current.ID,
				MatchedPath: currentPath,
				Ancestors:   append([]NodeID(nil), ancestors...),
				Exact:       false,
			}, nil
		}

		if current.IsContainer() {
			ancestors = append(ancestors, current.ID)
		}
		current = next
		currentPath = nextPath
	}

	return PathResolution{
		NodeID:      current.ID,
		MatchedPath: currentPath,
		Ancestors:   append([]NodeID(nil), ancestors...),
		Exact:       true,
	}, nil
}

func resolvePathSegment(current *Node, currentPath string, segment pathSegment) (*Node, string, bool) {
	if current == nil {
		return nil, "", false
	}

	if segment.isIndex {
		if current.Kind != KindArray || segment.index < 0 || segment.index >= len(current.Array) {
			return nil, "", false
		}
		return current.Array[segment.index], fmt.Sprintf("%s[%d]", currentPath, segment.index), true
	}

	if current.Kind != KindObject {
		return nil, "", false
	}
	for _, entry := range current.Object {
		if entry.Key == segment.key {
			return entry.Value, appendPathKey(currentPath, segment.key), true
		}
	}
	return nil, "", false
}

func parsePath(path string) ([]pathSegment, error) {
	if path == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}
	if path[0] != '$' {
		return nil, fmt.Errorf("path must start with $")
	}

	segments := make([]pathSegment, 0)
	for index := 1; index < len(path); {
		switch path[index] {
		case '.':
			key, next, err := parsePathIdentifier(path, index+1)
			if err != nil {
				return nil, err
			}
			segments = append(segments, pathSegment{key: key})
			index = next
		case '[':
			segment, next, err := parsePathBracket(path, index+1)
			if err != nil {
				return nil, err
			}
			segments = append(segments, segment)
			index = next
		default:
			return nil, fmt.Errorf("unexpected %q at position %d", path[index], index)
		}
	}

	return segments, nil
}

func parsePathIdentifier(path string, index int) (string, int, error) {
	if index >= len(path) {
		return "", 0, fmt.Errorf("expected identifier after .")
	}
	if !isPathIdentifierStart(path[index]) {
		return "", 0, fmt.Errorf("expected identifier after .")
	}

	start := index
	index++
	for index < len(path) && isPathIdentifierContinue(path[index]) {
		index++
	}
	return path[start:index], index, nil
}

func parsePathBracket(path string, index int) (pathSegment, int, error) {
	if index >= len(path) {
		return pathSegment{}, 0, fmt.Errorf("expected array index or quoted key after [")
	}

	if path[index] == '"' {
		key, next, err := parseQuotedKey(path, index)
		if err != nil {
			return pathSegment{}, 0, err
		}
		if next >= len(path) || path[next] != ']' {
			return pathSegment{}, 0, fmt.Errorf("expected ] after quoted key")
		}
		return pathSegment{key: key}, next + 1, nil
	}

	if path[index] < '0' || path[index] > '9' {
		return pathSegment{}, 0, fmt.Errorf("expected array index or quoted key after [")
	}

	start := index
	for index < len(path) && path[index] >= '0' && path[index] <= '9' {
		index++
	}
	if index >= len(path) || path[index] != ']' {
		return pathSegment{}, 0, fmt.Errorf("expected ] after array index")
	}

	value, err := strconv.Atoi(path[start:index])
	if err != nil {
		return pathSegment{}, 0, fmt.Errorf("parse array index: %w", err)
	}

	return pathSegment{index: value, isIndex: true}, index + 1, nil
}

func parseQuotedKey(path string, index int) (string, int, error) {
	start := index
	index++
	for index < len(path) {
		switch path[index] {
		case '\\':
			index += 2
		case '"':
			quoted := path[start : index+1]
			key, err := strconv.Unquote(quoted)
			if err != nil {
				return "", 0, fmt.Errorf("parse quoted key: %w", err)
			}
			return key, index + 1, nil
		default:
			index++
		}
	}
	return "", 0, fmt.Errorf("unterminated quoted key")
}

func appendPathKey(parentPath, key string) string {
	if pathSimpleIdentifier(key) {
		return parentPath + "." + key
	}
	return fmt.Sprintf("%s[%q]", parentPath, key)
}

func pathSimpleIdentifier(key string) bool {
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

func isPathIdentifierStart(value byte) bool {
	return (value >= 'a' && value <= 'z') || (value >= 'A' && value <= 'Z') || value == '_'
}

func isPathIdentifierContinue(value byte) bool {
	return isPathIdentifierStart(value) || (value >= '0' && value <= '9')
}
