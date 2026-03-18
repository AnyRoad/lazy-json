package integration

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/anyroad/lazy-json/internal/document"
)

func SerializeNodeToTemp(node *document.Node) (string, error) {
	data, err := document.MarshalIndentNode(node)
	if err != nil {
		return "", err
	}
	file, err := os.CreateTemp("", "lazy-json-editor-*.json")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	path := file.Name()
	if _, err := file.Write(data); err != nil {
		file.Close()
		os.Remove(path)
		return "", fmt.Errorf("write temp file: %w", err)
	}
	if err := file.Close(); err != nil {
		os.Remove(path)
		return "", fmt.Errorf("close temp file: %w", err)
	}
	return path, nil
}

func EditorCommand(editor, path string) (*exec.Cmd, error) {
	editor = strings.TrimSpace(editor)
	if editor == "" {
		return nil, fmt.Errorf("$EDITOR is not set")
	}
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return nil, fmt.Errorf("$EDITOR is not set")
	}
	args := append(parts[1:], path)
	cmd := exec.Command(parts[0], args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd, nil
}

func ReadEditedNode(path string) (*document.Node, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read edited file: %w", err)
	}
	node, err := document.ParseNode(data)
	if err != nil {
		return nil, fmt.Errorf("parse edited json: %w", err)
	}
	return node, nil
}
