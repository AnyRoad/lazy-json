package perftest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anyroad/lazy-json/internal/document"
)

const (
	FixtureHugeArray = "huge-array"
	FixtureHugeJSON  = "huge-json"

	EnvHugeArray = "LAZY_JSON_BENCH_HUGE_ARRAY"
	EnvHugeJSON  = "LAZY_JSON_BENCH_HUGE_JSON"
)

var ErrFixtureUnavailable = errors.New("benchmark fixture unavailable")

type FixtureSpec struct {
	Name        string
	EnvKey      string
	DefaultPath string
}

var fixtureSpecs = map[string]FixtureSpec{
	FixtureHugeArray: {
		Name:        FixtureHugeArray,
		EnvKey:      EnvHugeArray,
		DefaultPath: "~/PROJECTS/react-json-view-lite-benchmark/src/hugeArray.json",
	},
	FixtureHugeJSON: {
		Name:        FixtureHugeJSON,
		EnvKey:      EnvHugeJSON,
		DefaultPath: "~/PROJECTS/react-json-view-lite-benchmark/src/hugeJson.json",
	},
}

func ResolveFixturePath(name string) (string, error) {
	spec, ok := fixtureSpecs[name]
	if !ok {
		return "", fmt.Errorf("unknown benchmark fixture %q", name)
	}

	path := strings.TrimSpace(os.Getenv(spec.EnvKey))
	if path == "" {
		path = spec.DefaultPath
	}
	return expandHome(path)
}

func LoadFixture(name string) ([]byte, string, error) {
	path, err := ResolveFixturePath(name)
	if err != nil {
		return nil, "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, path, fmt.Errorf("%w: %s", ErrFixtureUnavailable, path)
		}
		return nil, path, err
	}
	return data, path, nil
}

func LoadDocument(name string) (*document.Document, string, error) {
	data, path, err := LoadFixture(name)
	if err != nil {
		return nil, path, err
	}

	doc, err := document.Parse(data)
	if err != nil {
		return nil, path, fmt.Errorf("parse fixture %s: %w", path, err)
	}
	return doc, path, nil
}

func expandHome(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("fixture path cannot be empty")
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		if path == "~" {
			return homeDir, nil
		}
		return filepath.Join(homeDir, path[2:]), nil
	}
	return path, nil
}
