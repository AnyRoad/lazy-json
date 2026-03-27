package tui

import (
	"regexp"
	"strings"
	"testing"

	"github.com/anyroad/lazy-json/internal/config"
	"github.com/anyroad/lazy-json/internal/document"
	"github.com/anyroad/lazy-json/internal/session"
	"github.com/anyroad/lazy-json/internal/source"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

func splitViewLines(view string) []string {
	return strings.Split(view, "\n")
}

func TestViewContainsKeyAndFooter(t *testing.T) {
	m := testModel(t)
	m.Width = 120
	m.Height = 20
	view := m.View()
	if !strings.Contains(view, "name") {
		t.Fatalf("View() missing key: %q", view)
	}
	if !strings.Contains(view, "sample.json") {
		t.Fatalf("View() missing footer source: %q", view)
	}
	if !strings.Contains(view, "S open settings  t quick preview  ? help") {
		t.Fatalf("View() missing settings hint: %q", view)
	}
}

func TestViewShowsGlobalLineNumberGutter(t *testing.T) {
	doc, err := document.Parse([]byte(`{"name":"Ada","items":[{"title":"alpha"},{"title":"beta"}],"tail":true}`))
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(doc, source.Input{Kind: source.KindFile, Path: "sample.json"}, ModelOptions{
		Settings: config.DefaultSettings().WithShowLineNumbers(true),
	})
	items := doc.Root.Object[1].Value
	m.Session.Expanded[items.ID] = true
	m.Session.Refresh(doc)
	m.Width = 120
	m.Height = 20

	lines := splitViewLines(stripANSI(m.View()))
	if got := lines[0]; !regexp.MustCompile(`^1\s+▾ \{3\}`).MatchString(got) {
		t.Fatalf("first line = %q, want root line number gutter", got)
	}
	if got := lines[1]; !regexp.MustCompile(`^2\s+name: "Ada"`).MatchString(got) {
		t.Fatalf("second line = %q, want sequential visible gutter", got)
	}
	if got := lines[3]; !regexp.MustCompile(`^4\s+▸ \[0\]: \{1\}`).MatchString(got) {
		t.Fatalf("fourth line = %q, want child line number gutter", got)
	}
	if got := lines[5]; !regexp.MustCompile(`^\s*8\s+tail: true`).MatchString(got) {
		t.Fatalf("sixth line = %q, want later visible row number", got)
	}
}

func TestViewPreservesGlobalLineNumberGapsWhenCollapsed(t *testing.T) {
	doc, err := document.Parse([]byte(`{"name":"Ada","items":[{"title":"alpha"},{"title":"beta"}],"meta":{"count":2},"tail":true}`))
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(doc, source.Input{Kind: source.KindFile, Path: "sample.json"}, ModelOptions{
		Settings: config.DefaultSettings().WithShowLineNumbers(true),
	})
	items := doc.Root.Object[1].Value
	meta := doc.Root.Object[2].Value
	m.Session.Expanded[items.ID] = true
	m.Session.Expanded[items.Array[0].ID] = true
	m.Session.Expanded[items.Array[1].ID] = true
	m.Session.Expanded[meta.ID] = true
	m.Session.Refresh(doc)
	delete(m.Session.Expanded, items.ID)
	delete(m.Session.Expanded, meta.ID)
	m.Session.Refresh(doc)
	m.Width = 120
	m.Height = 20

	lines := splitViewLines(stripANSI(m.View()))
	if got := lines[2]; !regexp.MustCompile(`^\s*3\s+▸ items: \[2\]`).MatchString(got) {
		t.Fatalf("items line = %q, want global line number 3", got)
	}
	if got := lines[3]; !regexp.MustCompile(`^\s*8\s+▸ meta: \{1\}`).MatchString(got) {
		t.Fatalf("meta line = %q, want global line number gap", got)
	}
	if got := lines[4]; !regexp.MustCompile(`^\s*10\s+tail: true`).MatchString(got) {
		t.Fatalf("tail line = %q, want global line number gap", got)
	}
}

func TestViewRendersLongArrayBatchLabels(t *testing.T) {
	doc := testLongArrayDoc(t, 101)
	m := NewModel(doc, source.Input{Kind: source.KindFile, Path: "sample.json"}, ModelOptions{
		Clipboard: &stubClipboard{},
	})
	items := doc.Root.Object[0].Value
	m.Session.Expanded[items.ID] = true
	m.Session.Refresh(doc)
	m.Width = 120
	m.Height = 20

	lines := splitViewLines(stripANSI(m.View()))
	if got := lines[2]; !strings.Contains(got, "▸ [0-99]") {
		t.Fatalf("third line = %q, want first batch label", got)
	}
	if got := lines[3]; !strings.Contains(got, "▸ [100-100]") {
		t.Fatalf("fourth line = %q, want second batch label", got)
	}
	if got := lines[2]; strings.Contains(got, "[0]:") {
		t.Fatalf("third line = %q, want batch row instead of direct item row", got)
	}
}

func TestViewPadsToViewportHeightAndAnchorsFooter(t *testing.T) {
	m := testModel(t)
	m.Width = 120
	m.Height = 20

	lines := splitViewLines(stripANSI(m.View()))
	if got, want := len(lines), 20; got != want {
		t.Fatalf("len(lines) = %d, want %d; lines=%q", got, want, lines)
	}
	if got := lines[len(lines)-1]; !strings.Contains(got, "sample.json") {
		t.Fatalf("last line = %q, want footer on final row", got)
	}
}

func TestViewKeepsSelectedSingleLineRowVisible(t *testing.T) {
	doc := testLongArrayDoc(t, 150)
	m := NewModel(doc, source.Input{Kind: source.KindFile, Path: "sample.json"}, ModelOptions{
		Clipboard: &stubClipboard{},
	})
	items := doc.Root.Object[0].Value
	m.Session.Expanded[items.ID] = true
	m.Session.ExpandedBatches[session.BatchRowID(items.ID, 0)] = true
	m.Session.SelectNode(items.Array[75].ID)
	m.Session.Refresh(doc)
	m.Width = 120
	m.Height = 6

	view := stripANSI(m.View())
	if !strings.Contains(view, "[75]: 75") {
		t.Fatalf("View() = %q, want selected row visible in single-line viewport", view)
	}
	if !strings.Contains(view, "sample.json") {
		t.Fatalf("View() = %q, want footer preserved", view)
	}
}

func TestViewKeepsFooterAbovePromptOnLastRow(t *testing.T) {
	m := testModel(t)
	m.Width = 120
	m.Height = 20
	m.openPrompt(promptCommand, ": ", "q")

	lines := splitViewLines(stripANSI(m.View()))
	if got, want := len(lines), 20; got != want {
		t.Fatalf("len(lines) = %d, want %d; lines=%q", got, want, lines)
	}
	if got := lines[len(lines)-2]; !strings.Contains(got, "sample.json") {
		t.Fatalf("second-last line = %q, want footer above prompt", got)
	}
	if got := lines[len(lines)-1]; !strings.Contains(got, ": q") {
		t.Fatalf("last line = %q, want prompt on final row", got)
	}
}

func TestHelpViewFitsViewportHeight(t *testing.T) {
	m := testModel(t)
	m.Width = 120
	m.Height = 20
	m.Session.Help = true

	lines := splitViewLines(stripANSI(m.View()))
	if got, want := len(lines), 20; got != want {
		t.Fatalf("len(lines) = %d, want %d; lines=%q", got, want, lines)
	}
	if got := lines[0]; !strings.Contains(got, "Navigation") {
		t.Fatalf("first line = %q, want help content", got)
	}
}

func TestViewUsesRegistryThemeForRendering(t *testing.T) {
	profile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() {
		lipgloss.SetColorProfile(profile)
	})

	registry, warnings := NewThemeRegistry([]config.DiscoveredTheme{
		{
			Path: "10-mist.json",
			Spec: config.ThemeSpec{
				Name: "mist",
				Key: &config.StyleSpec{
					Foreground: "#112233",
				},
			},
		},
	})
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}

	m := testModelWithOptions(t, ModelOptions{
		ThemeRegistry: registry,
		Settings:      config.Settings{Theme: "mist"},
	})
	m.Width = 120
	m.Height = 20
	m.Session.SelectedID = m.Doc.Root.Object[0].Value.ID

	mistView := m.View()
	if want := registry.ThemeByName("mist").Key.Render("name"); !strings.Contains(mistView, want) {
		t.Fatalf("View() = %q, want mist key styling", mistView)
	}

	m.Session.ThemeName = config.DefaultThemeName
	defaultView := m.View()

	if mistView == defaultView {
		t.Fatalf("View() did not change after theme switch: %q", mistView)
	}
	if strings.Contains(defaultView, registry.ThemeByName("mist").Key.Render("name")) {
		t.Fatalf("View() = %q, unexpectedly contains mist key styling", defaultView)
	}
}

func TestViewShowsSettingsOverlayHints(t *testing.T) {
	paths := config.PathsFromUserConfigDir(t.TempDir())
	nextTheme := nextThemeName(t, BuiltinThemeRegistry(), config.DefaultThemeName)
	m := testModelWithOptions(t, ModelOptions{
		Settings:          config.Settings{Theme: config.DefaultThemeName},
		SettingsPath:      paths.SettingsFile,
		SettingsPersisted: true,
	})
	m.Width = 120
	m.Height = 20
	m.openSettings()

	initialView := m.View()
	if !strings.Contains(initialView, "Settings") {
		t.Fatalf("View() = %q, want settings title", initialView)
	}
	if !strings.Contains(initialView, "saved") {
		t.Fatalf("View() = %q, want saved state hint", initialView)
	}
	if !strings.Contains(initialView, "Saved theme: "+config.DefaultThemeName) {
		t.Fatalf("View() = %q, want saved theme label", initialView)
	}
	if !strings.Contains(initialView, "up/down row  left/right change") {
		t.Fatalf("View() = %q, want settings footer hint", initialView)
	}
	if strings.Contains(initialView, "enter open theme") {
		t.Fatalf("View() = %q, unexpectedly advertises theme popup controls", initialView)
	}
	if !strings.Contains(initialView, "Save writes settings.json on demand.") {
		t.Fatalf("View() = %q, want explicit save hint", initialView)
	}
	if !strings.Contains(initialView, "Built-ins + config themes/*.json appear here.") {
		t.Fatalf("View() = %q, want external theme hint", initialView)
	}

	m.previewSettingsTheme(1)
	previewView := m.View()
	if !strings.Contains(previewView, "preview only") {
		t.Fatalf("View() = %q, want preview-only state hint", previewView)
	}
	if !strings.Contains(previewView, "Saved theme: "+config.DefaultThemeName) {
		t.Fatalf("View() = %q, want original saved theme label during preview", previewView)
	}

	m.saveThemeSettings()
	savedView := m.View()
	if strings.Contains(savedView, "preview only") {
		t.Fatalf("View() = %q, want preview-only hint cleared after save", savedView)
	}
	if !strings.Contains(savedView, "Saved theme: "+nextTheme) {
		t.Fatalf("View() = %q, want updated saved theme label", savedView)
	}
}

func TestViewShowsThemePreviewAndPersistMessages(t *testing.T) {
	paths := config.PathsFromUserConfigDir(t.TempDir())
	nextTheme := nextThemeName(t, BuiltinThemeRegistry(), config.DefaultThemeName)
	m := testModelWithOptions(t, ModelOptions{
		Settings:     config.Settings{Theme: config.DefaultThemeName},
		SettingsPath: paths.SettingsFile,
	})
	m.Width = 120
	m.Height = 20

	updated, _ := m.Update(key("t"))
	m = updated.(*Model)

	previewView := m.View()
	if !strings.Contains(previewView, "previewing theme "+nextTheme) {
		t.Fatalf("View() = %q, want preview footer message", previewView)
	}
	if !strings.Contains(previewView, "S open settings  t quick preview  ? help") {
		t.Fatalf("View() = %q, want updated footer hint", previewView)
	}

	updated, _ = m.Update(key("S"))
	m = updated.(*Model)
	updated, _ = m.Update(key("s"))
	m = updated.(*Model)

	savedView := m.View()
	if got, want := m.Session.Status, "saved settings"; got != want {
		t.Fatalf("Status = %q, want %q", got, want)
	}
	if !strings.Contains(savedView, "Saved theme: "+nextTheme) {
		t.Fatalf("View() = %q, want saved theme label", savedView)
	}
}

func TestViewShowsPrefixMenus(t *testing.T) {
	testCases := []struct {
		name   string
		prefix string
		want   string
	}{
		{name: "go", prefix: "g", want: "[g:go] g:top"},
		{name: "copy", prefix: "y", want: "[y:copy] p:path k:key v:value s:subtree j:json"},
		{name: "fold", prefix: "z", want: "[z:fold] R:expand-all M:collapse-all a:array+1 A:array-1"},
		{name: "jump", prefix: "]", want: "[jump] p:parent-sibling"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			m := testModel(t)
			m.Width = 120
			m.Height = 20

			updated, _ := m.Update(key(testCase.prefix))
			m = updated.(*Model)

			view := stripANSI(m.View())
			if !strings.Contains(view, testCase.want) {
				t.Fatalf("View() = %q, want %q", view, testCase.want)
			}
			if strings.Contains(view, "S open settings  t quick preview  ? help") {
				t.Fatalf("View() = %q, unexpectedly shows default footer hint while prefix is pending", view)
			}
		})
	}
}

func TestRenderRowLinesWrapLongStrings(t *testing.T) {
	doc, err := document.Parse([]byte(`{"name":"abcdefghijklmnopqrstuvwxyz"}`))
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(doc, source.Input{Kind: source.KindFile, Path: "sample.json"}, ModelOptions{})
	row := m.Session.Rows[1]
	theme := m.theme()

	unwrapped := m.renderRowLines(row, theme, 20)
	if got, want := len(unwrapped), 1; got != want {
		t.Fatalf("len(unwrapped) = %d, want %d", got, want)
	}

	m.ActiveSettings.WrapLongStrings = true
	wrapped := m.renderRowLines(row, theme, 20)
	if len(wrapped) < 2 {
		t.Fatalf("len(wrapped) = %d, want at least 2", len(wrapped))
	}

	joined := stripANSI(strings.Join(wrapped, "\n"))
	lastLine := stripANSI(wrapped[len(wrapped)-1])
	if !strings.Contains(lastLine, "$.name") {
		t.Fatalf("wrapped lines = %q, want path on final wrapped line", joined)
	}
	if strings.TrimSpace(strings.ReplaceAll(lastLine, "$.name", "")) == "" {
		t.Fatalf("wrapped lines = %q, want path attached to the final value fragment", joined)
	}
	for _, line := range wrapped[1 : len(wrapped)-1] {
		if strings.Contains(stripANSI(line), "$.name") {
			t.Fatalf("wrapped middle continuation line = %q, want no repeated path", stripANSI(line))
		}
	}
	if !strings.Contains(joined, "abcdefghi") || !strings.Contains(joined, "jklmnopqrs") {
		t.Fatalf("wrapped lines = %q, want split string fragments", joined)
	}
}

func TestRenderRowLinesShowsLineNumberOnlyOnFirstWrappedLine(t *testing.T) {
	doc, err := document.Parse([]byte(`{"name":"abcdefghijklmnopqrstuvwxyz"}`))
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(doc, source.Input{Kind: source.KindFile, Path: "sample.json"}, ModelOptions{
		Settings: config.DefaultSettings().WithShowLineNumbers(true),
	})
	row := m.Session.Rows[1]
	theme := m.theme()
	m.ActiveSettings.WrapLongStrings = true

	lines := m.renderRowLines(row, theme, 20)
	if len(lines) < 2 {
		t.Fatalf("len(lines) = %d, want at least 2", len(lines))
	}

	first := stripANSI(lines[0])
	if !regexp.MustCompile(`^2\s+name: `).MatchString(first) {
		t.Fatalf("first line = %q, want line number gutter", first)
	}
	for _, line := range lines[1:] {
		stripped := stripANSI(line)
		if regexp.MustCompile(`^2\s+`).MatchString(stripped) {
			t.Fatalf("continuation line = %q, want blank gutter instead of repeated line number", stripped)
		}
	}
}

func TestRenderRowLinesAppliesSelectionOnlyToFirstWrappedLine(t *testing.T) {
	doc, err := document.Parse([]byte(`{"name":"abcdefghijklmnopqrstuvwxyz"}`))
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(doc, source.Input{Kind: source.KindFile, Path: "sample.json"}, ModelOptions{})
	row := m.Session.Rows[1]
	theme := m.theme()
	theme.Selected = lipgloss.NewStyle().PaddingLeft(1)
	m.ActiveSettings.WrapLongStrings = true
	m.Session.SelectNode(row.NodeID)

	selected := m.renderRowLines(row, theme, 20)
	if len(selected) < 2 {
		t.Fatalf("len(selected) = %d, want at least 2", len(selected))
	}

	m.Session.SelectNode(m.Doc.Root.ID)
	unselected := m.renderRowLines(row, theme, 20)
	if len(unselected) != len(selected) {
		t.Fatalf("len(unselected) = %d, want %d", len(unselected), len(selected))
	}
	if selected[0] == unselected[0] {
		t.Fatal("first wrapped line did not change under selection")
	}
	for index := 1; index < len(selected); index++ {
		if selected[index] != unselected[index] {
			t.Fatalf("continuation line %d changed under selection; got %q want %q", index, selected[index], unselected[index])
		}
	}
}

func TestRenderRowLinesHideJSONPath(t *testing.T) {
	doc, err := document.Parse([]byte(`{"name":"abcdefghijklmnopqrstuvwxyz"}`))
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(doc, source.Input{Kind: source.KindFile, Path: "sample.json"}, ModelOptions{
		Settings: config.DefaultSettings().WithShowJSONPath(false),
	})
	row := m.Session.Rows[1]
	theme := m.theme()

	lines := m.renderRowLines(row, theme, 20)
	for _, line := range lines {
		if strings.Contains(stripANSI(line), "$.name") {
			t.Fatalf("line = %q, want JSON path hidden", stripANSI(line))
		}
	}

	m.ActiveSettings.WrapLongStrings = true
	lines = m.renderRowLines(row, theme, 20)
	for _, line := range lines {
		if strings.Contains(stripANSI(line), "$.name") {
			t.Fatalf("wrapped line = %q, want JSON path hidden", stripANSI(line))
		}
	}
}

func TestViewShowsLineNumbersWithoutJSONPath(t *testing.T) {
	doc, err := document.Parse([]byte(`{"name":"Ada"}`))
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(doc, source.Input{Kind: source.KindFile, Path: "sample.json"}, ModelOptions{
		Settings: config.DefaultSettings().WithShowLineNumbers(true).WithShowJSONPath(false),
	})
	m.Width = 120
	m.Height = 20

	view := stripANSI(m.View())
	if !regexp.MustCompile(`(?m)^2\s+name: "Ada"$`).MatchString(view) {
		t.Fatalf("View() = %q, want line number without JSON path", view)
	}
	if strings.Contains(view, "$.name") {
		t.Fatalf("View() = %q, want JSON path hidden while line numbers remain visible", view)
	}
}

func TestViewHelperPrefixesAndSliceVisibleLines(t *testing.T) {
	m := testModelWithOptions(t, ModelOptions{
		Settings: config.DefaultSettings().WithShowLineNumbers(true),
	})
	theme := m.theme()
	row := m.Session.Rows[1]

	if got, want := m.lineNumberWidth(), 1; got != want {
		t.Fatalf("lineNumberWidth() = %d, want %d", got, want)
	}
	if got, want := stripANSI(m.renderLineNumberPrefix(row, theme)), "2 "; got != want {
		t.Fatalf("renderLineNumberPrefix() = %q, want %q", got, want)
	}
	if got, want := m.lineNumberBlankPrefix(), "  "; got != want {
		t.Fatalf("lineNumberBlankPrefix() = %q, want %q", got, want)
	}
	if got, want := sliceVisibleLines([][]string{{"a", "b"}, {"c"}, {"d", "e"}}, 1, 4), []string{"b", "c", "d"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("sliceVisibleLines() = %v, want %v", got, want)
	}
}

func TestRenderRowAndLabelHelpers(t *testing.T) {
	m := testModel(t)
	theme := m.theme()
	row := m.Session.Rows[1]

	label := stripANSI(m.renderRowLabel(row, theme))
	if !strings.Contains(label, `name: `) {
		t.Fatalf("renderRowLabel() = %q, want object-key label", label)
	}

	line := stripANSI(m.renderRow(row, theme))
	if !strings.Contains(line, `name: "Ada"`) {
		t.Fatalf("renderRow() = %q, want rendered value", line)
	}
	if !strings.Contains(line, `$.name`) {
		t.Fatalf("renderRow() = %q, want JSON path suffix", line)
	}
}
