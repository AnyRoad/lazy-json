package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDiscoverThemesMissingDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "themes")

	discovered, warnings := DiscoverThemes(dir)

	if len(discovered) != 0 {
		t.Fatalf("discovered = %d, want 0", len(discovered))
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
}

func TestDiscoverThemesReadDirFailureReturnsWarning(t *testing.T) {
	path := filepath.Join(t.TempDir(), "themes")
	writeFile(t, path, []byte("not a directory"))

	discovered, warnings := DiscoverThemes(path)

	if len(discovered) != 0 {
		t.Fatalf("discovered = %d, want 0", len(discovered))
	}
	if len(warnings) != 1 {
		t.Fatalf("warnings = %v, want 1 warning", warnings)
	}
	if !strings.Contains(warnings[0], "read themes dir") {
		t.Fatalf("warning = %q, want read themes dir warning", warnings[0])
	}
}

func TestDiscoverThemesUnreadableFileReturnsWarning(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission-based unreadable file test is not portable on Windows")
	}

	dir := filepath.Join(t.TempDir(), ThemesDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "10-secret.json")
	writeFile(t, path, []byte(`{"name":"secret"}`))
	if err := os.Chmod(path, 0); err != nil {
		t.Fatalf("Chmod(%q) error = %v", path, err)
	}

	discovered, warnings := DiscoverThemes(dir)

	if len(discovered) != 0 {
		t.Fatalf("discovered = %d, want 0", len(discovered))
	}
	if len(warnings) != 1 {
		t.Fatalf("warnings = %v, want 1 warning", warnings)
	}
	if !strings.Contains(warnings[0], "10-secret.json") {
		t.Fatalf("warning = %q, want unreadable file warning", warnings[0])
	}
}

func TestDiscoverThemesSortsAndSkipsInvalidEntries(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ThemesDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeFile(t, filepath.Join(dir, "20-zeta.json"), []byte(`{"name":"zeta","status":{"foreground":"#999999"}}`))
	writeFile(t, filepath.Join(dir, "10-alpha.json"), []byte(`{"name":"alpha","key":{"foreground":"#111111"},"selected":{"background":"#222222","bold":true}}`))
	writeFile(t, filepath.Join(dir, "15-blank.json"), []byte(`{"name":"   "}`))
	writeFile(t, filepath.Join(dir, "30-duplicate.json"), []byte(`{"name":"Alpha","prompt":{"foreground":"#333333"}}`))
	writeFile(t, filepath.Join(dir, "05-broken.json"), []byte(`{"name":`))
	writeFile(t, filepath.Join(dir, "notes.txt"), []byte(`ignored`))

	discovered, warnings := DiscoverThemes(dir)

	if len(discovered) != 2 {
		t.Fatalf("discovered = %d, want 2", len(discovered))
	}
	if got, want := discovered[0].Spec.Name, "alpha"; got != want {
		t.Fatalf("first theme = %q, want %q", got, want)
	}
	if got, want := discovered[1].Spec.Name, "zeta"; got != want {
		t.Fatalf("second theme = %q, want %q", got, want)
	}
	if discovered[0].Spec.Key == nil || discovered[0].Spec.Key.Foreground != "#111111" {
		t.Fatalf("alpha key style = %#v, want foreground", discovered[0].Spec.Key)
	}
	if discovered[0].Spec.String != nil {
		t.Fatalf("alpha string style = %#v, want nil for partial spec", discovered[0].Spec.String)
	}
	if discovered[0].Spec.Selected == nil || discovered[0].Spec.Selected.Background != "#222222" {
		t.Fatalf("alpha selected style = %#v, want background", discovered[0].Spec.Selected)
	}
	if discovered[0].Spec.Selected.Bold == nil || !*discovered[0].Spec.Selected.Bold {
		t.Fatalf("alpha selected bold = %#v, want true", discovered[0].Spec.Selected.Bold)
	}

	if len(warnings) != 3 {
		t.Fatalf("warnings = %v, want 3 warnings", warnings)
	}
	if !strings.Contains(warnings[0], "05-broken.json") {
		t.Fatalf("warning[0] = %q, want broken file", warnings[0])
	}
	if !strings.Contains(warnings[1], "15-blank.json") || !strings.Contains(warnings[1], "missing name") {
		t.Fatalf("warning[1] = %q, want missing-name warning", warnings[1])
	}
	if !strings.Contains(warnings[2], "30-duplicate.json") || !strings.Contains(warnings[2], "duplicate theme name") {
		t.Fatalf("warning[2] = %q, want duplicate warning", warnings[2])
	}
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return data
}
