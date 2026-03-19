package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	AppDirName       = "lazy-json"
	SettingsFileName = "settings.json"
	ThemesDirName    = "themes"
	DefaultThemeName = "forest"
)

type Paths struct {
	RootDir      string
	SettingsFile string
	ThemesDir    string
}

type Settings struct {
	Theme string `json:"theme"`
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

func DefaultSettings() Settings {
	return Settings{Theme: DefaultThemeName}
}

func (s Settings) WithDefaults() Settings {
	s.Theme = strings.TrimSpace(s.Theme)
	if s.Theme == "" {
		s.Theme = DefaultThemeName
	}
	return s
}

func LoadSettings(path string) (Settings, []string) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultSettings(), nil
		}
		return DefaultSettings(), []string{
			fmt.Sprintf("load settings %s: %v", path, err),
		}
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return DefaultSettings(), []string{
			fmt.Sprintf("parse settings %s: %v", path, err),
		}
	}

	return settings.WithDefaults(), nil
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

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}

	return nil
}
