package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anyroad/lazy-json/internal/document"
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

func TestSerializeNodeToTempRejectsUnsupportedNode(t *testing.T) {
	if _, err := SerializeNodeToTemp(&document.Node{Kind: document.Kind("unknown")}); err == nil {
		t.Fatal("SerializeNodeToTemp() error = nil, want error")
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

func TestReadEditedNodeErrors(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		if _, err := ReadEditedNode(filepath.Join(t.TempDir(), "missing.json")); err == nil {
			t.Fatal("ReadEditedNode() error = nil, want error")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "broken.json")
		if err := os.WriteFile(path, []byte(`{"name":`), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadEditedNode(path); err == nil {
			t.Fatal("ReadEditedNode() error = nil, want error")
		}
	})
}
