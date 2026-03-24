package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anyroad/lazy-json/internal/config"
	"github.com/anyroad/lazy-json/internal/document"
	"github.com/anyroad/lazy-json/internal/source"
	"github.com/anyroad/lazy-json/internal/tui"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type startupArgs struct {
	SelectPath string
	InputArgs  []string
}

func run(args []string) error {
	startup, err := parseStartupArgs(args)
	if err != nil {
		return err
	}

	input, err := source.Load(startup.InputArgs, os.Stdin)
	if err != nil {
		if errors.Is(err, source.ErrNoInput) {
			return fmt.Errorf("usage: lazy-json [--select <path>] <file.json> or cat file.json | lazy-json [--select <path>]")
		}
		return err
	}

	doc, err := document.Parse(input.Data)
	if err != nil {
		return fmt.Errorf("parse json: %w", err)
	}

	model := tui.NewModel(doc, input, loadModelOptions())
	if err := applyStartupSelection(model, startup.SelectPath); err != nil {
		return err
	}
	opts := []tea.ProgramOption{tea.WithAltScreen()}
	if input.Kind == source.KindStdin {
		opts = append(opts, tea.WithInputTTY())
	}
	program := tea.NewProgram(model, opts...)
	finalModel, err := program.Run()
	if err != nil {
		return err
	}
	typed, ok := finalModel.(*tui.Model)
	if !ok {
		return nil
	}
	if len(typed.ExitOutput) > 0 {
		if _, err := os.Stdout.Write(typed.ExitOutput); err != nil {
			return fmt.Errorf("write stdout: %w", err)
		}
	}
	return nil
}

func parseStartupArgs(args []string) (startupArgs, error) {
	parsed := startupArgs{InputArgs: make([]string, 0, len(args))}

	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch {
		case arg == "--":
			parsed.InputArgs = append(parsed.InputArgs, args[index+1:]...)
			return parsed, nil
		case arg == "--select":
			if parsed.SelectPath != "" {
				return startupArgs{}, fmt.Errorf("--select may only be provided once")
			}
			if index+1 >= len(args) {
				return startupArgs{}, fmt.Errorf("--select requires a JSON path")
			}
			nextArg := args[index+1]
			switch {
			case nextArg == "--select" || strings.HasPrefix(nextArg, "--select="):
				return startupArgs{}, fmt.Errorf("--select may only be provided once")
			case strings.HasPrefix(nextArg, "--"):
				return startupArgs{}, fmt.Errorf("--select requires a JSON path")
			}
			value := strings.TrimSpace(nextArg)
			if value == "" {
				return startupArgs{}, fmt.Errorf("--select requires a JSON path")
			}
			parsed.SelectPath = value
			index++
		case strings.HasPrefix(arg, "--select="):
			if parsed.SelectPath != "" {
				return startupArgs{}, fmt.Errorf("--select may only be provided once")
			}
			value := strings.TrimSpace(strings.TrimPrefix(arg, "--select="))
			if value == "" {
				return startupArgs{}, fmt.Errorf("--select requires a JSON path")
			}
			parsed.SelectPath = value
		default:
			parsed.InputArgs = append(parsed.InputArgs, arg)
		}
	}

	return parsed, nil
}

func applyStartupSelection(model *tui.Model, selectPath string) error {
	selectPath = strings.TrimSpace(selectPath)
	if selectPath == "" {
		return nil
	}

	resolution, err := model.SelectPath(selectPath)
	if err != nil {
		return err
	}
	if resolution.Exact {
		return nil
	}
	if resolution.MatchedPath == "$" {
		model.Session.SetError(fmt.Sprintf("select path not found: %s; opened root $", selectPath))
		return nil
	}
	model.Session.SetStatus(fmt.Sprintf("select path not found: %s; opened nearest existing ancestor %s", selectPath, resolution.MatchedPath))
	return nil
}

func loadModelOptions() tui.ModelOptions {
	return loadModelOptionsWithResolver(config.ResolvePaths)
}

func loadModelOptionsWithResolver(resolvePaths func() (config.Paths, error)) tui.ModelOptions {
	paths, err := resolvePaths()
	if err != nil {
		return tui.ModelOptions{
			Warnings: []string{fmt.Sprintf("resolve config paths: %v", err)},
		}
	}
	return loadModelOptionsFromPaths(paths)
}

func loadModelOptionsFromPaths(paths config.Paths) tui.ModelOptions {
	options := tui.ModelOptions{
		ThemeRegistry:       tui.BuiltinThemeRegistry(),
		Settings:            config.DefaultSettings(),
		SettingsPath:        paths.SettingsFile,
		SettingsFilePresent: settingsFilePresent(paths.SettingsFile),
		SettingsPersisted:   false,
	}

	settings, persisted, warnings := config.LoadSettingsState(paths.SettingsFile)
	options.Settings = settings
	options.SettingsPersisted = persisted
	options.Warnings = append(options.Warnings, warnings...)

	discovered, warnings := config.DiscoverThemes(paths.ThemesDir)
	options.Warnings = append(options.Warnings, warnings...)

	registry, warnings := tui.NewThemeRegistry(discovered)
	options.ThemeRegistry = registry
	options.Warnings = append(options.Warnings, warnings...)

	options.Settings, warnings = resolveStartupSettings(options.Settings, options.ThemeRegistry)
	options.Warnings = append(options.Warnings, warnings...)
	if len(warnings) != 0 {
		options.SettingsPersisted = false
	}

	return options
}

func settingsFilePresent(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := os.Lstat(path)
	return err == nil || !errors.Is(err, os.ErrNotExist)
}

func resolveStartupSettings(settings config.Settings, registry tui.ThemeRegistry) (config.Settings, []string) {
	requested := strings.TrimSpace(settings.Theme)
	settings = settings.WithDefaults()

	theme, ok := registry.Lookup(settings.Theme)
	if ok {
		settings.Theme = theme.Name
		return settings, nil
	}

	settings.Theme = config.DefaultThemeName
	if requested == "" || strings.EqualFold(requested, config.DefaultThemeName) {
		return settings, nil
	}

	return settings, []string{
		fmt.Sprintf("configured theme %q not found; using %q", requested, config.DefaultThemeName),
	}
}
