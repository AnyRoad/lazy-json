package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andrei/lazy-json/internal/document"
)

func TestSerializeAndReadEditedNode(t *testing.T) {
	node, err := document.ParseNode([]byte(`{"name":"Ada"}`))
	if err != nil {
		t.Fatal(err)
	}
	path, err := SerializeNodeToTemp(node)
	if err != nil {
		t.Fatalf("SerializeNodeToTemp() error = %v", err)
	}
	t.Cleanup(func() { os.Remove(path) })

	readNode, err := ReadEditedNode(path)
	if err != nil {
		t.Fatalf("ReadEditedNode() error = %v", err)
	}
	if got := readNode.Object[0].Key; got != "name" {
		t.Fatalf("key = %q", got)
	}
}

func TestEditorCommandRunsScript(t *testing.T) {
	script := filepath.Join(t.TempDir(), "editor.sh")
	content := "#!/bin/sh\nprintf '{\"edited\":true}\\n' > \"$1\"\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	node, err := document.ParseNode([]byte(`{"name":"Ada"}`))
	if err != nil {
		t.Fatal(err)
	}
	path, err := SerializeNodeToTemp(node)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(path) })

	cmd, err := EditorCommand(script, path)
	if err != nil {
		t.Fatalf("EditorCommand() error = %v", err)
	}
	if err := cmd.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	edited, err := ReadEditedNode(path)
	if err != nil {
		t.Fatalf("ReadEditedNode() error = %v", err)
	}
	if got := edited.Object[0].Key; got != "edited" {
		t.Fatalf("edited key = %q", got)
	}
}

func TestEditorCommandRejectsEmptyEditor(t *testing.T) {
	if _, err := EditorCommand("", "sample.json"); err == nil {
		t.Fatal("EditorCommand() error = nil, want error")
	}
}
