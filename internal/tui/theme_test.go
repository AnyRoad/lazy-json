package tui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/anyroad/lazy-json/internal/config"
)

func TestBuiltinThemeRegistryLookupAndFallback(t *testing.T) {
	registry := BuiltinThemeRegistry()
	names := themeNames(registry.Themes())

	if len(names) == 0 {
		t.Fatal("Themes() returned no built-in themes")
	}

	defaultTheme, ok := registry.Lookup(strings.ToUpper(config.DefaultThemeName))
	if !ok {
		t.Fatalf("Lookup(%s) = false, want true", strings.ToUpper(config.DefaultThemeName))
	}
	if defaultTheme.Name != config.DefaultThemeName {
		t.Fatalf("Lookup(%s).Name = %q, want %q", strings.ToUpper(config.DefaultThemeName), defaultTheme.Name, config.DefaultThemeName)
	}
	if got, want := defaultTheme.Selected.GetBackground(), lipgloss.Color("#585858"); got != want {
		t.Fatalf("default selected background = %#v, want %#v", got, want)
	}

	githubDark, ok := registry.Lookup("GITHUB-DARK")
	if !ok {
		t.Fatal("Lookup(GITHUB-DARK) = false, want true")
	}
	if githubDark.Name != "github-dark" {
		t.Fatalf("Lookup(GITHUB-DARK).Name = %q, want github-dark", githubDark.Name)
	}
	if got, want := githubDark.Key.GetForeground(), lipgloss.Color("#79c0ff"); got != want {
		t.Fatalf("github-dark key foreground = %#v, want %#v", got, want)
	}

	nord, ok := registry.Lookup("NORD")
	if !ok {
		t.Fatal("Lookup(NORD) = false, want true")
	}
	if got, want := nord.Selected.GetBackground(), lipgloss.Color("#4d535e"); got != want {
		t.Fatalf("nord selected background = %#v, want %#v", got, want)
	}

	if got, want := registry.ThemeByName("missing").Name, config.DefaultThemeName; got != want {
		t.Fatalf("ThemeByName(missing).Name = %q, want %q", got, want)
	}
}

func TestNewThemeRegistryCyclesAcrossBuiltInAndExternalThemes(t *testing.T) {
	discovered := []config.DiscoveredTheme{
		{Path: "10-mist.json", Spec: config.ThemeSpec{Name: "mist"}},
		{Path: "20-aurora.json", Spec: config.ThemeSpec{Name: "aurora"}},
	}

	registry, warnings := NewThemeRegistry(discovered)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}

	if got, want := registry.NextTheme(config.DefaultThemeName).Name, nextThemeName(t, BuiltinThemeRegistry(), config.DefaultThemeName); got != want {
		t.Fatalf("NextTheme(%s).Name = %q, want %q", config.DefaultThemeName, got, want)
	}
	if got, want := registry.NextTheme(lastBuiltinThemeName(t)).Name, "mist"; got != want {
		t.Fatalf("NextTheme(lastBuiltin).Name = %q, want %q", got, want)
	}
	if got, want := registry.NextTheme("aurora").Name, config.DefaultThemeName; got != want {
		t.Fatalf("NextTheme(aurora).Name = %q, want %q", got, want)
	}
}

func TestNewThemeRegistryConvertsPartialExternalThemesWithFallbacks(t *testing.T) {
	bold := false
	discovered := []config.DiscoveredTheme{
		{
			Path: "20-frost.json",
			Spec: config.ThemeSpec{
				Name: "frost",
				Key: &config.StyleSpec{
					Foreground: "#112233",
				},
				Selected: &config.StyleSpec{
					Background: "#ddeeff",
					Bold:       &bold,
				},
			},
		},
	}

	registry, warnings := NewThemeRegistry(discovered)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}

	base := registry.ThemeByName(config.DefaultThemeName)
	frost := registry.ThemeByName("frost")

	if got, want := frost.Key.GetForeground(), lipgloss.Color("#112233"); got != want {
		t.Fatalf("frost key foreground = %#v, want %#v", got, want)
	}
	if got, want := frost.Key.GetBackground(), base.Key.GetBackground(); got != want {
		t.Fatalf("frost key background = %#v, want %#v", got, want)
	}
	if got, want := frost.String.GetForeground(), base.String.GetForeground(); got != want {
		t.Fatalf("frost string foreground = %#v, want %#v", got, want)
	}
	if got, want := frost.Selected.GetBackground(), lipgloss.Color("#ddeeff"); got != want {
		t.Fatalf("frost selected background = %#v, want %#v", got, want)
	}
	if got, want := frost.Selected.GetForeground(), base.Selected.GetForeground(); got != want {
		t.Fatalf("frost selected foreground = %#v, want %#v", got, want)
	}
	if frost.Selected.GetBold() {
		t.Fatal("frost selected bold = true, want false")
	}
}

func TestNewThemeRegistryWarnsOnInvalidColorsAndKeepsFallbackStyles(t *testing.T) {
	discovered := []config.DiscoveredTheme{
		{
			Path: "10-mist.json",
			Spec: config.ThemeSpec{
				Name: "mist",
				Key: &config.StyleSpec{
					Foreground: "not-a-color",
				},
				Status: &config.StyleSpec{
					Foreground: "#112233",
				},
			},
		},
	}

	registry, warnings := NewThemeRegistry(discovered)
	if len(warnings) != 1 {
		t.Fatalf("warnings = %v, want 1 warning", warnings)
	}
	if !strings.Contains(warnings[0], "10-mist.json") || !strings.Contains(warnings[0], "key.foreground") {
		t.Fatalf("warning = %q, want invalid color warning", warnings[0])
	}

	base := registry.ThemeByName(config.DefaultThemeName)
	mist := registry.ThemeByName("mist")
	if got, want := mist.Key.GetForeground(), base.Key.GetForeground(); got != want {
		t.Fatalf("mist key foreground = %#v, want %#v", got, want)
	}
	if got, want := mist.Status.GetForeground(), lipgloss.Color("#112233"); got != want {
		t.Fatalf("mist status foreground = %#v, want %#v", got, want)
	}
}

func TestNewThemeRegistryPreservesStableOrderingAndWarnsOnDuplicateNames(t *testing.T) {
	discovered := []config.DiscoveredTheme{
		{Path: "10-aurora.json", Spec: config.ThemeSpec{Name: "aurora"}},
		{Path: "20-default.json", Spec: config.ThemeSpec{Name: config.DefaultThemeName}},
		{Path: "30-zenith.json", Spec: config.ThemeSpec{Name: "zenith"}},
	}

	registry, warnings := NewThemeRegistry(discovered)
	builtinNames := themeNames(BuiltinThemeRegistry().Themes())
	registryNames := themeNames(registry.Themes())

	if got, want := registryNames[:len(builtinNames)], builtinNames; !reflect.DeepEqual(got, want) {
		t.Fatalf("builtin prefix = %v, want %v", got, want)
	}
	if got, want := registryNames[len(registryNames)-2:], []string{"aurora", "zenith"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("external suffix = %v, want %v", got, want)
	}
	if len(warnings) != 1 {
		t.Fatalf("warnings = %v, want 1 warning", warnings)
	}
	if !strings.Contains(warnings[0], "20-default.json") || !strings.Contains(warnings[0], "duplicate theme name") {
		t.Fatalf("warning = %q, want duplicate warning", warnings[0])
	}
}

func themeNames(themes []Theme) []string {
	names := make([]string, 0, len(themes))
	for _, theme := range themes {
		names = append(names, theme.Name)
	}
	return names
}
