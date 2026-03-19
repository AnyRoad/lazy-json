package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type StyleSpec struct {
	Foreground string `json:"foreground,omitempty"`
	Background string `json:"background,omitempty"`
	Bold       *bool  `json:"bold,omitempty"`
}

type ThemeSpec struct {
	Name      string     `json:"name"`
	Key       *StyleSpec `json:"key,omitempty"`
	String    *StyleSpec `json:"string,omitempty"`
	Number    *StyleSpec `json:"number,omitempty"`
	Bool      *StyleSpec `json:"bool,omitempty"`
	Null      *StyleSpec `json:"null,omitempty"`
	Muted     *StyleSpec `json:"muted,omitempty"`
	Selected  *StyleSpec `json:"selected,omitempty"`
	SearchHit *StyleSpec `json:"search_hit,omitempty"`
	Status    *StyleSpec `json:"status,omitempty"`
	Error     *StyleSpec `json:"error,omitempty"`
	Border    *StyleSpec `json:"border,omitempty"`
	Help      *StyleSpec `json:"help,omitempty"`
	Prompt    *StyleSpec `json:"prompt,omitempty"`
}

type DiscoveredTheme struct {
	Path string
	Spec ThemeSpec
}

func DiscoverThemes(dir string) ([]DiscoveredTheme, []string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, []string{
			fmt.Sprintf("read themes dir %s: %v", dir, err),
		}
	}

	slices.SortFunc(entries, func(a, b os.DirEntry) int {
		return strings.Compare(a.Name(), b.Name())
	})

	discovered := make([]DiscoveredTheme, 0, len(entries))
	warnings := make([]string, 0)
	seenNames := make(map[string]string, len(entries))

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("skip theme %s: %v", entry.Name(), err))
			continue
		}

		var spec ThemeSpec
		if err := json.Unmarshal(data, &spec); err != nil {
			warnings = append(warnings, fmt.Sprintf("skip theme %s: %v", entry.Name(), err))
			continue
		}

		spec.Name = strings.TrimSpace(spec.Name)
		if spec.Name == "" {
			warnings = append(warnings, fmt.Sprintf("skip theme %s: missing name", entry.Name()))
			continue
		}

		duplicateKey := strings.ToLower(spec.Name)
		if previousFile, exists := seenNames[duplicateKey]; exists {
			warnings = append(
				warnings,
				fmt.Sprintf("skip theme %s: duplicate theme name %q already defined in %s", entry.Name(), spec.Name, previousFile),
			)
			continue
		}
		seenNames[duplicateKey] = entry.Name()

		discovered = append(discovered, DiscoveredTheme{
			Path: path,
			Spec: spec,
		})
	}

	return discovered, warnings
}
