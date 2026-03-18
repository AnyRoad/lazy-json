package source

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.json")
	if err := os.WriteFile(path, []byte(`{"ok":true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	stdin, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stdin.Close() })

	input, err := Load([]string{path}, stdin)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if input.Kind != KindFile {
		t.Fatalf("Kind = %q, want %q", input.Kind, KindFile)
	}
	if input.Path != path {
		t.Fatalf("Path = %q, want %q", input.Path, path)
	}
	if string(input.Data) != `{"ok":true}` {
		t.Fatalf("Data = %q", input.Data)
	}
}

func TestLoadFromStdin(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "stdin-*.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString(`{"stdin":true}`); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { file.Close() })

	input, err := Load(nil, file)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if input.Kind != KindStdin {
		t.Fatalf("Kind = %q, want %q", input.Kind, KindStdin)
	}
	if string(input.Data) != `{"stdin":true}` {
		t.Fatalf("Data = %q", input.Data)
	}
}

func TestLoadAmbiguousInput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.json")
	if err := os.WriteFile(path, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	stdin, err := os.CreateTemp(t.TempDir(), "stdin-*.json")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stdin.Close() })

	_, err = Load([]string{path}, stdin)
	if !errors.Is(err, ErrAmbiguousInput) {
		t.Fatalf("Load() error = %v, want %v", err, ErrAmbiguousInput)
	}
}
