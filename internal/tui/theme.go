package tui

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"

	"github.com/anyroad/lazy-json/internal/config"
)

type Theme struct {
	Name      string
	Key       lipgloss.Style
	String    lipgloss.Style
	Number    lipgloss.Style
	Bool      lipgloss.Style
	Null      lipgloss.Style
	Muted     lipgloss.Style
	Selected  lipgloss.Style
	SearchHit lipgloss.Style
	Status    lipgloss.Style
	Error     lipgloss.Style
	Border    lipgloss.Style
	Help      lipgloss.Style
	Prompt    lipgloss.Style
}

type ThemeRegistry struct {
	themes    []Theme
	index     map[string]int
	fallback  Theme
	hasThemes bool
}

func BuiltinThemeRegistry() ThemeRegistry {
	return newThemeRegistry(prioritizeTheme(builtinThemes(), config.DefaultThemeName))
}

func NewThemeRegistry(discovered []config.DiscoveredTheme) (ThemeRegistry, []string) {
	registry := BuiltinThemeRegistry()
	warnings := make([]string, 0)
	base := registry.ThemeByName(config.DefaultThemeName)

	for _, candidate := range discovered {
		name := strings.TrimSpace(candidate.Spec.Name)
		source := themeSource(candidate.Path)
		if name == "" {
			warnings = append(warnings, fmt.Sprintf("skip theme %s: missing name", source))
			continue
		}
		if _, exists := registry.Lookup(name); exists {
			warnings = append(warnings, fmt.Sprintf("skip theme %s: duplicate theme name %q", source, name))
			continue
		}
		theme, themeWarnings := themeFromSpec(candidate.Spec, base, source)
		warnings = append(warnings, themeWarnings...)
		registry.add(theme)
	}

	return registry, warnings
}

func (r ThemeRegistry) Themes() []Theme {
	return append([]Theme(nil), r.themes...)
}

func (r ThemeRegistry) Lookup(name string) (Theme, bool) {
	if !r.hasThemes {
		return Theme{}, false
	}
	index, ok := r.index[normalizeThemeName(name)]
	if !ok {
		return Theme{}, false
	}
	return r.themes[index], true
}

func (r ThemeRegistry) ThemeByName(name string) Theme {
	if theme, ok := r.Lookup(name); ok {
		return theme
	}
	return r.fallback
}

func (r ThemeRegistry) NextTheme(name string) Theme {
	if !r.hasThemes {
		return Theme{}
	}
	index, ok := r.index[normalizeThemeName(name)]
	if !ok {
		return r.fallback
	}
	return r.themes[(index+1)%len(r.themes)]
}

func (r *ThemeRegistry) add(theme Theme) {
	if !r.hasThemes {
		r.fallback = theme
		r.hasThemes = true
	}
	r.index[normalizeThemeName(theme.Name)] = len(r.themes)
	r.themes = append(r.themes, theme)
}

func newThemeRegistry(themes []Theme) ThemeRegistry {
	registry := ThemeRegistry{
		themes: make([]Theme, 0, len(themes)),
		index:  make(map[string]int, len(themes)),
	}
	for _, theme := range themes {
		registry.add(theme)
	}
	if theme, ok := registry.Lookup(config.DefaultThemeName); ok {
		registry.fallback = theme
	}
	return registry
}

func prioritizeTheme(themes []Theme, name string) []Theme {
	if len(themes) == 0 {
		return nil
	}
	index := -1
	target := normalizeThemeName(name)
	for i, theme := range themes {
		if normalizeThemeName(theme.Name) == target {
			index = i
			break
		}
	}
	if index <= 0 {
		return append([]Theme(nil), themes...)
	}
	prioritized := make([]Theme, 0, len(themes))
	prioritized = append(prioritized, themes[index:]...)
	prioritized = append(prioritized, themes[:index]...)
	return prioritized
}

func normalizeThemeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func themeSource(path string) string {
	if base := filepath.Base(path); base != "." && base != string(filepath.Separator) && base != "" {
		return base
	}
	return "theme"
}

func themeFromSpec(spec config.ThemeSpec, base Theme, source string) (Theme, []string) {
	theme := Theme{Name: strings.TrimSpace(spec.Name)}
	warnings := make([]string, 0)

	theme.Key, warnings = applyStyleSpec(base.Key, spec.Key, source, "key", warnings)
	theme.String, warnings = applyStyleSpec(base.String, spec.String, source, "string", warnings)
	theme.Number, warnings = applyStyleSpec(base.Number, spec.Number, source, "number", warnings)
	theme.Bool, warnings = applyStyleSpec(base.Bool, spec.Bool, source, "bool", warnings)
	theme.Null, warnings = applyStyleSpec(base.Null, spec.Null, source, "null", warnings)
	theme.Muted, warnings = applyStyleSpec(base.Muted, spec.Muted, source, "muted", warnings)
	theme.Selected, warnings = applyStyleSpec(base.Selected, spec.Selected, source, "selected", warnings)
	theme.SearchHit, warnings = applyStyleSpec(base.SearchHit, spec.SearchHit, source, "search_hit", warnings)
	theme.Status, warnings = applyStyleSpec(base.Status, spec.Status, source, "status", warnings)
	theme.Error, warnings = applyStyleSpec(base.Error, spec.Error, source, "error", warnings)
	theme.Border, warnings = applyStyleSpec(base.Border, spec.Border, source, "border", warnings)
	theme.Help, warnings = applyStyleSpec(base.Help, spec.Help, source, "help", warnings)
	theme.Prompt, warnings = applyStyleSpec(base.Prompt, spec.Prompt, source, "prompt", warnings)

	return theme, warnings
}

func applyStyleSpec(base lipgloss.Style, spec *config.StyleSpec, source, slot string, warnings []string) (lipgloss.Style, []string) {
	style := base
	if spec == nil {
		return style, warnings
	}
	if foreground := strings.TrimSpace(spec.Foreground); foreground != "" {
		if isValidThemeColor(foreground) {
			style = style.Foreground(lipgloss.Color(foreground))
		} else {
			warnings = append(warnings, invalidThemeColorWarning(source, slot, "foreground", foreground))
		}
	}
	if background := strings.TrimSpace(spec.Background); background != "" {
		if isValidThemeColor(background) {
			style = style.Background(lipgloss.Color(background))
		} else {
			warnings = append(warnings, invalidThemeColorWarning(source, slot, "background", background))
		}
	}
	if spec.Bold != nil {
		style = style.Bold(*spec.Bold)
	}
	return style, warnings
}

func isValidThemeColor(value string) bool {
	if strings.HasPrefix(value, "#") {
		_, err := colorful.Hex(value)
		return err == nil
	}

	number, err := strconv.Atoi(value)
	if err != nil {
		return false
	}
	return number >= 0 && number <= 255
}

func invalidThemeColorWarning(source, slot, field, value string) string {
	return fmt.Sprintf("theme %s %s.%s %q is invalid; using default", source, slot, field, value)
}

func themeStyle(foreground, background string, bold bool) lipgloss.Style {
	style := lipgloss.NewStyle()
	if foreground != "" {
		style = style.Foreground(lipgloss.Color(foreground))
	}
	if background != "" {
		style = style.Background(lipgloss.Color(background))
	}
	if bold {
		style = style.Bold(true)
	}
	return style
}
