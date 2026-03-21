package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/anyroad/lazy-json/internal/document"
	"github.com/anyroad/lazy-json/internal/session"
	"github.com/anyroad/lazy-json/internal/source"
)

func (m *Model) theme() Theme {
	return m.ThemeRegistry.ThemeByName(m.Session.ThemeName)
}

func (m *Model) View() string {
	if m.Doc == nil || m.Session == nil {
		return "loading..."
	}
	if m.Session.Help {
		return m.helpView()
	}

	theme := m.theme()
	width := m.Width
	if width <= 0 {
		width = 100
	}
	height := m.Height
	if height <= 0 {
		height = 24
	}
	view := m.documentView(theme, width, height)
	if m.settingsOpen() {
		return m.renderSettingsOverlay(view, theme, width, height)
	}
	return view
}

func (m *Model) documentView(theme Theme, width, height int) string {
	bodyHeight := height - 2
	if bodyHeight < 1 {
		bodyHeight = len(m.Session.Rows)
	}
	start, end := visibleWindow(m.Session, bodyHeight)
	lines := make([]string, 0, end-start+2)
	for _, row := range m.Session.Rows[start:end] {
		lines = append(lines, trimWidth(m.renderRow(row, theme), width))
	}
	lines = append(lines, trimWidth(m.renderFooter(theme), width))
	if m.promptKind != promptNone {
		lines = append(lines, trimWidth(theme.Prompt.Render(m.prompt.View()), width))
	}
	return strings.Join(lines, "\n")
}

func visibleWindow(s *session.Session, bodyHeight int) (int, int) {
	if len(s.Rows) <= bodyHeight {
		return 0, len(s.Rows)
	}
	current, ok := s.RowIndex[s.SelectedID]
	if !ok {
		return 0, bodyHeight
	}
	start := current - bodyHeight/2
	if start < 0 {
		start = 0
	}
	end := start + bodyHeight
	if end > len(s.Rows) {
		end = len(s.Rows)
		start = end - bodyHeight
	}
	return start, end
}

func (m *Model) renderRow(row session.Row, theme Theme) string {
	loc, _ := m.Doc.Find(row.NodeID)
	node := loc.Node
	indent := strings.Repeat("  ", row.Depth)
	marker := " "
	if row.IsContainer {
		if row.Expanded {
			marker = "▾"
		} else {
			marker = "▸"
		}
	}
	label := indent + marker + " "
	switch {
	case row.Key != "":
		label += theme.Key.Render(row.Key) + theme.Muted.Render(": ")
	case row.ArrayIndex >= 0:
		label += theme.Muted.Render(fmt.Sprintf("[%d]: ", row.ArrayIndex))
	}
	value := renderNodeValue(node, theme)
	line := label + value + theme.Muted.Render("  "+row.Path)
	if row.NodeID == m.Session.SelectedID {
		return theme.Selected.Render(line)
	}
	if m.Session.HasSearchHit(row.NodeID) {
		return theme.SearchHit.Render(line)
	}
	return line
}

func renderNodeValue(node *document.Node, theme Theme) string {
	switch node.Kind {
	case document.KindObject, document.KindArray:
		return theme.Muted.Render(node.Summary())
	case document.KindString:
		return theme.String.Render(fmt.Sprintf("%q", node.String))
	case document.KindNumber:
		return theme.Number.Render(node.Number)
	case document.KindBool:
		if node.Boolean {
			return theme.Bool.Render("true")
		}
		return theme.Bool.Render("false")
	case document.KindNull:
		return theme.Null.Render("null")
	default:
		return ""
	}
}

func (m *Model) renderFooter(theme Theme) string {
	sourceLabel := string(m.Session.SourceKind)
	if m.Session.SourceKind == source.KindFile && m.Session.SourcePath != "" {
		sourceLabel = m.Session.SourcePath
	}
	mode := string(m.Session.Mode)
	dirty := ""
	if m.Session.Dirty {
		dirty = " dirty"
	}
	message := m.Session.Status
	style := theme.Status
	if m.Session.Error != "" {
		message = m.Session.Error
		style = theme.Error
	} else if message == "" && m.StartupWarning != "" {
		message = m.StartupWarning
	}
	left := fmt.Sprintf("[%s%s] %s", mode, dirty, sourceLabel)
	rightParts := make([]string, 0, 2)
	if message != "" {
		rightParts = append(rightParts, style.Render(message))
	}
	if hint := m.footerHint(theme); hint != "" {
		rightParts = append(rightParts, hint)
	}
	right := strings.Join(rightParts, "  ")
	if right == "" {
		return theme.Border.Render(left)
	}
	return theme.Border.Render(left) + " " + right
}

func (m *Model) footerHint(theme Theme) string {
	if m.settingsOpen() {
		return theme.Status.Render(m.settingsFooterHint())
	}
	if m.promptKind != promptNone {
		return ""
	}
	if menu, ok := m.pendingPrefixMenu(); ok {
		return m.renderPrefixMenu(theme, menu)
	}
	return theme.Status.Render("S open settings  t quick preview  ? help")
}

func (m *Model) renderPrefixMenu(theme Theme, menu prefixMenu) string {
	items := make([]string, 0, len(menu.Items))
	for _, item := range menu.Items {
		items = append(items, item.Key+":"+item.Label)
	}
	return theme.Help.Render("["+menu.Tag+"]") + " " + theme.Status.Render(strings.Join(items, " "))
}

func trimWidth(s string, width int) string {
	if width <= 0 {
		return s
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	return lipgloss.NewStyle().MaxWidth(width).Render(s)
}
