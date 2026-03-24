package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anyroad/lazy-json/internal/source"
)

const (
	AppDirName       = "lazy-json"
	SettingsFileName = "settings.json"
	ThemesDirName    = "themes"
	DefaultThemeName = "default"
)

type IndentKind string

const (
	IndentKindSpaces IndentKind = "spaces"
	IndentKindTabs   IndentKind = "tabs"
)

type Paths struct {
	RootDir      string
	SettingsFile string
	ThemesDir    string
}

type SaveIndent struct {
	Kind IndentKind `json:"kind"`
	Size int        `json:"size,omitempty"`
}

type Settings struct {
	Theme           string     `json:"theme"`
	ShowLineNumbers bool       `json:"show_line_numbers"`
	WrapLongStrings bool       `json:"wrap_long_strings"`
	ShowJSONPath    bool       `json:"show_json_path"`
	SaveIndent      SaveIndent `json:"save_indent"`

	showJSONPathSet bool
}

func PathsFromUserConfigDir(userConfigDir string) Paths {
	rootDir := filepath.Join(userConfigDir, AppDirName)
	return Paths{
		RootDir:      rootDir,
		SettingsFile: filepath.Join(rootDir, SettingsFileName),
		ThemesDir:    filepath.Join(rootDir, ThemesDirName),
	}
}

func ResolvePaths() (Paths, error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, fmt.Errorf("resolve user config dir: %w", err)
	}
	return PathsFromUserConfigDir(userConfigDir), nil
}

func DefaultSaveIndent() SaveIndent {
	return SaveIndent{Kind: IndentKindSpaces, Size: 2}
}

func (i SaveIndent) WithDefaults() SaveIndent {
	switch IndentKind(strings.ToLower(strings.TrimSpace(string(i.Kind)))) {
	case "", IndentKindSpaces:
		i.Kind = IndentKindSpaces
		switch i.Size {
		case 2, 3, 4:
		default:
			i.Size = 2
		}
	case IndentKindTabs:
		i.Kind = IndentKindTabs
		i.Size = 0
	default:
		return DefaultSaveIndent()
	}
	return i
}

func (i SaveIndent) String() string {
	i = i.WithDefaults()
	if i.Kind == IndentKindTabs {
		return "\t"
	}
	return strings.Repeat(" ", i.Size)
}

func (i SaveIndent) Label() string {
	i = i.WithDefaults()
	if i.Kind == IndentKindTabs {
		return "tabs"
	}
	return fmt.Sprintf("spaces:%d", i.Size)
}

func DefaultSettings() Settings {
	return Settings{
		Theme:           DefaultThemeName,
		ShowLineNumbers: false,
		WrapLongStrings: false,
		ShowJSONPath:    true,
		SaveIndent:      DefaultSaveIndent(),
		showJSONPathSet: true,
	}
}

func (s Settings) WithDefaults() Settings {
	s.Theme = strings.TrimSpace(s.Theme)
	if s.Theme == "" {
		s.Theme = DefaultThemeName
	}
	s.SaveIndent = s.SaveIndent.WithDefaults()
	if !s.showJSONPathSet {
		s.ShowJSONPath = true
	}
	s.showJSONPathSet = true
	return s
}

func (s Settings) WithShowJSONPath(value bool) Settings {
	s.ShowJSONPath = value
	s.showJSONPathSet = true
	return s
}

func (s Settings) WithShowLineNumbers(value bool) Settings {
	s.ShowLineNumbers = value
	return s
}

func (s *Settings) UnmarshalJSON(data []byte) error {
	type rawSettings struct {
		Theme           string     `json:"theme"`
		ShowLineNumbers bool       `json:"show_line_numbers"`
		WrapLongStrings bool       `json:"wrap_long_strings"`
		ShowJSONPath    *bool      `json:"show_json_path"`
		SaveIndent      SaveIndent `json:"save_indent"`
	}

	var raw rawSettings
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	s.Theme = raw.Theme
	s.ShowLineNumbers = raw.ShowLineNumbers
	s.WrapLongStrings = raw.WrapLongStrings
	s.SaveIndent = raw.SaveIndent
	if raw.ShowJSONPath != nil {
		s.ShowJSONPath = *raw.ShowJSONPath
		s.showJSONPathSet = true
	} else {
		s.ShowJSONPath = false
		s.showJSONPathSet = false
	}

	return nil
}

func LoadSettings(path string) (Settings, []string) {
	settings, _, warnings := loadSettings(path)
	return settings, warnings
}

func LoadSettingsState(path string) (Settings, bool, []string) {
	return loadSettings(path)
}

func loadSettings(path string) (Settings, bool, []string) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultSettings(), false, nil
		}
		return DefaultSettings(), false, []string{
			fmt.Sprintf("load settings %s: %v", path, err),
		}
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return DefaultSettings(), false, []string{
			fmt.Sprintf("parse settings %s: %v", path, err),
		}
	}

	return settings.WithDefaults(), true, nil
}

func SaveSettings(path string, settings Settings) error {
	settings = settings.WithDefaults()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	data = append(data, '\n')

	if err := source.WriteAtomic(path, data); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}

	return nil
}
