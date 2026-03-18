package source

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAtomic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.json")
	if err := WriteAtomic(path, []byte(`{"saved":true}`)); err != nil {
		t.Fatalf("WriteAtomic() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"saved":true}` {
		t.Fatalf("data = %q", data)
	}
}
