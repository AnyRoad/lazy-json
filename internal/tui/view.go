package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/anyroad/lazy-json/internal/config"
	"github.com/anyroad/lazy-json/internal/document"
	"github.com/anyroad/lazy-json/internal/session"
	"github.com/anyroad/lazy-json/internal/source"
)

type documentRenderOptions struct {
	settings        config.Settings
	lineNumberWidth int
}

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
	width, height := m.viewportSize()
	view := m.documentView(theme, width, height)
	if m.settingsOpen() {
		return m.renderSettingsOverlay(view, theme, width, height)
	}
	return view
}

func (m *Model) viewportSize() (int, int) {
	width := m.Width
	if width <= 0 {
		width = 100
	}
	height := m.Height
	if height <= 0 {
		height = 24
	}
	return width, height
}

func (m *Model) documentView(theme Theme, width, height int) string {
	opts := m.documentRenderOptions()
	reservedRows := m.documentReservedRows()
	bodyHeight := m.documentBodyHeightForHeight(height)
	lines := m.visibleDocumentLines(theme, width, bodyHeight, opts)
	if height > reservedRows {
		lines = padLinesToHeight(lines, bodyHeight)
	}
	lines = append(lines, trimWidth(m.renderFooter(theme), width))
	if m.promptKind != promptNone {
		lines = append(lines, trimWidth(theme.Prompt.Render(m.prompt.View()), width))
	}
	return strings.Join(lines, "\n")
}

func (m *Model) documentReservedRows() int {
	if m.promptKind != promptNone {
		return 2
	}
	return 1
}

func (m *Model) documentBodyHeight() int {
	_, height := m.viewportSize()
	return m.documentBodyHeightForHeight(height)
}

func (m *Model) documentBodyHeightForHeight(height int) int {
	bodyHeight := height - m.documentReservedRows()
	if bodyHeight < 1 {
		return 1
	}
	return bodyHeight
}

func (m *Model) visibleDocumentLines(theme Theme, width, bodyHeight int, opts documentRenderOptions) []string {
	if len(m.Session.Rows) == 0 {
		return nil
	}
	if !opts.settings.WrapLongStrings {
		return m.visibleSingleLineRows(theme, width, bodyHeight, opts)
	}

	rowLines := make([][]string, 0, len(m.Session.Rows))
	totalLines := 0
	selectedStart := 0
	currentIndex, hasCurrent := m.Session.RowKeyIndex[m.Session.SelectedRow()]

	for index, row := range m.Session.Rows {
		rendered := m.renderRowLinesWithOptions(row, theme, width, opts)
		if len(rendered) == 0 {
			rendered = []string{""}
		}
		if hasCurrent && index == currentIndex {
			selectedStart = totalLines
		}
		totalLines += len(rendered)
		rowLines = append(rowLines, rendered)
	}

	if totalLines <= bodyHeight {
		return flattenLines(rowLines)
	}

	top := selectedStart - bodyHeight/2
	if top < 0 {
		top = 0
	}
	maxTop := totalLines - bodyHeight
	if maxTop < 0 {
		maxTop = 0
	}
	if top > maxTop {
		top = maxTop
	}

	return sliceVisibleLines(rowLines, top, top+bodyHeight)
}

func (m *Model) visibleSingleLineRows(theme Theme, width, bodyHeight int, opts documentRenderOptions) []string {
	rows := m.Session.Rows
	currentIndex, hasCurrent := m.Session.RowKeyIndex[m.Session.SelectedRow()]
	start := 0
	if hasCurrent && len(rows) > bodyHeight {
		start = currentIndex - bodyHeight/2
		if start < 0 {
			start = 0
		}
		maxStart := len(rows) - bodyHeight
		if maxStart < 0 {
			maxStart = 0
		}
		if start > maxStart {
			start = maxStart
		}
	}

	end := start + bodyHeight
	if end > len(rows) {
		end = len(rows)
	}

	lines := make([]string, 0, end-start)
	for _, row := range rows[start:end] {
		lines = append(lines, m.renderSingleRowLine(row, theme, width, opts))
	}
	return lines
}

func (m *Model) renderRow(row session.Row, theme Theme) string {
	lines := m.renderRowLinesWithOptions(row, theme, 0, m.documentRenderOptions())
	if len(lines) == 0 {
		return ""
	}
	return lines[0]
}

func (m *Model) renderRowLines(row session.Row, theme Theme, width int) []string {
	return m.renderRowLinesWithOptions(row, theme, width, m.documentRenderOptions())
}

func (m *Model) renderRowLinesWithOptions(row session.Row, theme Theme, width int, opts documentRenderOptions) []string {
	label := m.renderRowLabelWithWidth(row, theme, opts.lineNumberWidth)
	if row.IsBatch() {
		return []string{m.renderSingleRowLineWithLabel(row, theme, width, label, opts.settings.ShowJSONPath)}
	}
	if opts.settings.WrapLongStrings && row.Kind == document.KindString {
		return m.renderWrappedStringLines(row, theme, width, label, opts.settings.ShowJSONPath, opts.lineNumberWidth)
	}
	return []string{m.renderSingleRowLineWithLabel(row, theme, width, label, opts.settings.ShowJSONPath)}
}

func (m *Model) renderRowLabel(row session.Row, theme Theme) string {
	return m.renderRowLabelWithWidth(row, theme, m.lineNumberWidth())
}

func (m *Model) renderRowLabelWithWidth(row session.Row, theme Theme, lineNumberWidth int) string {
	label := m.renderLineNumberPrefixWithWidth(row, theme, lineNumberWidth)
	indent := strings.Repeat("  ", row.Depth)
	marker := " "
	if row.IsContainer {
		if row.Expanded {
			marker = "▾"
		} else {
			marker = "▸"
		}
	}
	label += indent + marker + " "
	if row.IsBatch() {
		return label + theme.Muted.Render(row.BatchLabel())
	}
	switch {
	case row.Key != "":
		label += theme.Key.Render(row.Key) + theme.Muted.Render(": ")
	case row.ArrayIndex >= 0:
		label += theme.Muted.Render(fmt.Sprintf("[%d]: ", row.ArrayIndex))
	}
	return label
}

func (m *Model) renderSingleRowLine(row session.Row, theme Theme, width int, opts documentRenderOptions) string {
	label := m.renderRowLabelWithWidth(row, theme, opts.lineNumberWidth)
	return m.renderSingleRowLineWithLabel(row, theme, width, label, opts.settings.ShowJSONPath)
}

func (m *Model) renderSingleRowLineWithLabel(row session.Row, theme Theme, width int, label string, showJSONPath bool) string {
	line := label
	if !row.IsBatch() {
		line += renderNodeValue(row, theme)
		if showJSONPath {
			line += theme.Muted.Render("  " + row.Path)
		}
	}
	if width > 0 {
		line = trimWidth(line, width)
	}
	return m.decorateRowLine(row, theme, line)
}

func (m *Model) documentRenderOptions() documentRenderOptions {
	settings := m.ActiveSettings.WithDefaults()
	lineNumberWidth := 0
	if settings.ShowLineNumbers && m.Session != nil && len(m.Session.Rows) > 0 {
		lineNumberWidth = len(strconv.Itoa(m.Session.Rows[len(m.Session.Rows)-1].LineNumber))
	}
	return documentRenderOptions{
		settings:        settings,
		lineNumberWidth: lineNumberWidth,
	}
}

func (m *Model) lineNumberWidth() int {
	return m.documentRenderOptions().lineNumberWidth
}

func (m *Model) renderLineNumberPrefix(row session.Row, theme Theme) string {
	return m.renderLineNumberPrefixWithWidth(row, theme, m.lineNumberWidth())
}

func (m *Model) renderLineNumberPrefixWithWidth(row session.Row, theme Theme, width int) string {
	if width == 0 {
		return ""
	}
	return theme.Muted.Render(fmt.Sprintf("%*d ", width, row.LineNumber))
}

func (m *Model) lineNumberBlankPrefix() string {
	return m.lineNumberBlankPrefixWithWidth(m.lineNumberWidth())
}

func (m *Model) lineNumberBlankPrefixWithWidth(width int) string {
	if width == 0 {
		return ""
	}
	return strings.Repeat(" ", width+1)
}

func (m *Model) renderWrappedStringLines(row session.Row, theme Theme, width int, label string, showJSONPath bool, lineNumberWidth int) []string {
	valueText := fmt.Sprintf("%q", row.StringValue)
	if !showJSONPath {
		if width <= 0 {
			line := label + theme.String.Render(valueText)
			return []string{m.decorateRowLine(row, theme, line)}
		}

		labelWidth := lipgloss.Width(label)
		wrapWidth := width - labelWidth
		if wrapWidth < 1 {
			wrapWidth = 1
		}
		parts := wrapText(valueText, wrapWidth)
		continuationPrefix := strings.Repeat(" ", labelWidth)
		lines := make([]string, 0, len(parts))
		for index, part := range parts {
			prefix := label
			if index > 0 {
				prefix = continuationPrefix
			}
			line := prefix + theme.String.Render(part)
			lines = append(lines, m.decorateWrappedRowLine(row, theme, trimWidth(line, width), index == 0))
		}
		return lines
	}

	if width <= 0 {
		line := label + theme.String.Render(valueText) + theme.Muted.Render("  "+row.Path)
		return []string{m.decorateRowLine(row, theme, line)}
	}

	labelWidth := lipgloss.Width(label)
	wrapWidth := width - labelWidth
	if wrapWidth < 1 {
		wrapWidth = 1
	}
	pathSuffix := "  " + row.Path
	pathWidth := lipgloss.Width(pathSuffix)
	parts := wrapText(valueText, wrapWidth)
	if pathWidth < wrapWidth {
		parts = wrapTextWithTrailingWidth(valueText, wrapWidth, wrapWidth-pathWidth)
	}
	continuationPrefix := strings.Repeat(" ", labelWidth)
	lines := make([]string, 0, len(parts))
	if len(parts) == 1 {
		line := label + theme.String.Render(parts[0]) + theme.Muted.Render(pathSuffix)
		return []string{m.decorateRowLine(row, theme, trimWidth(line, width))}
	}
	for index, part := range parts {
		prefix := label
		if index > 0 {
			prefix = continuationPrefix
		}
		line := prefix + theme.String.Render(part)
		if index == len(parts)-1 && pathWidth < wrapWidth {
			line += theme.Muted.Render(pathSuffix)
		}
		lines = append(lines, m.decorateWrappedRowLine(row, theme, trimWidth(line, width), index == 0))
	}
	if pathWidth >= wrapWidth {
		pathPrefix := m.lineNumberBlankPrefixWithWidth(lineNumberWidth) + strings.Repeat("  ", row.Depth) + "  "
		lines = append(lines, m.decorateWrappedRowLine(row, theme, trimWidth(pathPrefix+theme.Muted.Render(row.Path), width), false))
	}
	return lines
}

func (m *Model) decorateRowLine(row session.Row, theme Theme, line string) string {
	return m.decorateWrappedRowLine(row, theme, line, true)
}

func (m *Model) decorateWrappedRowLine(row session.Row, theme Theme, line string, allowSelection bool) string {
	if allowSelection && row.ID == m.Session.SelectedRow() {
		return theme.Selected.Render(line)
	}
	if !row.IsBatch() && m.Session.HasSearchHit(row.NodeID) {
		return theme.SearchHit.Render(line)
	}
	return line
}

func renderNodeValue(row session.Row, theme Theme) string {
	switch row.Kind {
	case document.KindObject, document.KindArray:
		return theme.Muted.Render(row.Summary)
	case document.KindString:
		return theme.String.Render(fmt.Sprintf("%q", row.StringValue))
	case document.KindNumber:
		return theme.Number.Render(row.Summary)
	case document.KindBool:
		if row.Summary == "true" {
			return theme.Bool.Render("true")
		}
		return theme.Bool.Render("false")
	case document.KindNull:
		return theme.Null.Render("null")
	default:
		return ""
	}
}

func (m *Model) renderedStatusMessage(theme Theme) string {
	message := m.Session.Status
	style := theme.Status
	if m.Session.Error != "" {
		message = m.Session.Error
		style = theme.Error
	} else if message == "" && m.StartupWarning != "" {
		message = m.StartupWarning
	}
	if message == "" {
		return ""
	}
	return style.Render(message)
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
	left := fmt.Sprintf("[%s%s] %s", mode, dirty, sourceLabel)
	rightParts := make([]string, 0, 2)
	if message := m.renderedStatusMessage(theme); message != "" {
		rightParts = append(rightParts, message)
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
	if m.busy() {
		return theme.Status.Render("input blocked while jq runs")
	}
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

func flattenLines(groups [][]string) []string {
	lines := make([]string, 0)
	for _, group := range groups {
		lines = append(lines, group...)
	}
	return lines
}

func fitLinesToHeight(lines []string, height int) []string {
	if height <= 0 {
		return lines
	}
	if len(lines) > height {
		return lines[:height]
	}
	return padLinesToHeight(lines, height)
}

func padLinesToHeight(lines []string, height int) []string {
	if height <= 0 || len(lines) >= height {
		return lines
	}
	padded := append([]string{}, lines...)
	for len(padded) < height {
		padded = append(padded, "")
	}
	return padded
}

func sliceVisibleLines(groups [][]string, start, end int) []string {
	lines := make([]string, 0, end-start)
	offset := 0
	for _, group := range groups {
		next := offset + len(group)
		if next <= start {
			offset = next
			continue
		}
		if offset >= end {
			break
		}
		from := 0
		if start > offset {
			from = start - offset
		}
		to := len(group)
		if end < next {
			to = end - offset
		}
		lines = append(lines, group[from:to]...)
		offset = next
	}
	return lines
}

func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	lines := make([]string, 0, 1)
	var current strings.Builder
	currentWidth := 0
	for _, r := range text {
		runeWidth := lipgloss.Width(string(r))
		if current.Len() > 0 && currentWidth+runeWidth > width {
			lines = append(lines, current.String())
			current.Reset()
			currentWidth = 0
		}
		current.WriteRune(r)
		currentWidth += runeWidth
	}
	if current.Len() > 0 || len(lines) == 0 {
		lines = append(lines, current.String())
	}
	return lines
}

func wrapTextWithTrailingWidth(text string, width, trailingWidth int) []string {
	if width <= 0 || trailingWidth <= 0 || trailingWidth >= width {
		return wrapText(text, width)
	}
	if lipgloss.Width(text) <= trailingWidth {
		return []string{text}
	}
	prefix, trailing := splitSuffixByWidth(text, trailingWidth)
	lines := wrapText(prefix, width)
	return append(lines, trailing)
}

func splitSuffixByWidth(text string, width int) (string, string) {
	if width <= 0 {
		return text, ""
	}
	runes := []rune(text)
	split := len(runes)
	currentWidth := 0
	for split > 0 {
		runeWidth := lipgloss.Width(string(runes[split-1]))
		if currentWidth+runeWidth > width {
			break
		}
		currentWidth += runeWidth
		split--
	}
	return string(runes[:split]), string(runes[split:])
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
