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

	if got, want := themeNames(registry.Themes()), []string{"forest", "harbor", "paper", "ember"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Themes() = %v, want %v", got, want)
	}

	harbor, ok := registry.Lookup("HARBOR")
	if !ok {
		t.Fatal("Lookup(HARBOR) = false, want true")
	}
	if harbor.Name != "harbor" {
		t.Fatalf("Lookup(HARBOR).Name = %q, want harbor", harbor.Name)
	}

	paper, ok := registry.Lookup("paper")
	if !ok {
		t.Fatal("Lookup(paper) = false, want true")
	}
	if got, want := paper.Selected.GetBackground(), lipgloss.Color("#D9E8F5"); got != want {
		t.Fatalf("paper selected background = %#v, want %#v", got, want)
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

	if got, want := registry.NextTheme("forest").Name, "harbor"; got != want {
		t.Fatalf("NextTheme(forest).Name = %q, want %q", got, want)
	}
	if got, want := registry.NextTheme("ember").Name, "mist"; got != want {
		t.Fatalf("NextTheme(ember).Name = %q, want %q", got, want)
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

func TestNewThemeRegistryPreservesStableOrderingAndWarnsOnDuplicateNames(t *testing.T) {
	discovered := []config.DiscoveredTheme{
		{Path: "10-aurora.json", Spec: config.ThemeSpec{Name: "aurora"}},
		{Path: "20-paper.json", Spec: config.ThemeSpec{Name: "paper"}},
		{Path: "30-zenith.json", Spec: config.ThemeSpec{Name: "zenith"}},
	}

	registry, warnings := NewThemeRegistry(discovered)

	if got, want := themeNames(registry.Themes()), []string{"forest", "harbor", "paper", "ember", "aurora", "zenith"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Themes() = %v, want %v", got, want)
	}
	if len(warnings) != 1 {
		t.Fatalf("warnings = %v, want 1 warning", warnings)
	}
	if !strings.Contains(warnings[0], "20-paper.json") || !strings.Contains(warnings[0], "duplicate theme name") {
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
