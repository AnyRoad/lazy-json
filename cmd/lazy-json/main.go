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

func run(args []string) error {
	input, err := source.Load(args, os.Stdin)
	if err != nil {
		if errors.Is(err, source.ErrNoInput) {
			return fmt.Errorf("usage: lazy-json <file.json> or cat file.json | lazy-json")
		}
		return err
	}

	doc, err := document.Parse(input.Data)
	if err != nil {
		return fmt.Errorf("parse json: %w", err)
	}

	model := tui.NewModel(doc, input, loadModelOptions())
	opts := []tea.ProgramOption{}
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
		ThemeRegistry: tui.BuiltinThemeRegistry(),
		Settings:      config.DefaultSettings(),
		SettingsPath:  paths.SettingsFile,
	}

	settings, warnings := config.LoadSettings(paths.SettingsFile)
	options.Settings = settings
	options.Warnings = append(options.Warnings, warnings...)

	discovered, warnings := config.DiscoverThemes(paths.ThemesDir)
	options.Warnings = append(options.Warnings, warnings...)

	registry, warnings := tui.NewThemeRegistry(discovered)
	options.ThemeRegistry = registry
	options.Warnings = append(options.Warnings, warnings...)

	options.Settings, warnings = resolveStartupSettings(options.Settings, options.ThemeRegistry)
	options.Warnings = append(options.Warnings, warnings...)

	return options
}

func resolveStartupSettings(settings config.Settings, registry tui.ThemeRegistry) (config.Settings, []string) {
	requested := strings.TrimSpace(settings.Theme)
	settings = settings.WithDefaults()

	theme, ok := registry.Lookup(settings.Theme)
	if ok {
		settings.Theme = theme.Name
		return settings, nil
	}

	settings = config.DefaultSettings()
	if requested == "" || strings.EqualFold(requested, config.DefaultThemeName) {
		return settings, nil
	}

	return settings, []string{
		fmt.Sprintf("configured theme %q not found; using %q", requested, config.DefaultThemeName),
	}
}
