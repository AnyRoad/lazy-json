package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/anyroad/lazy-json/internal/config"
	"github.com/anyroad/lazy-json/internal/session"
)

const (
	settingsRowTheme = iota
	settingsRowShowLineNumbers
	settingsRowShowJSONPath
	settingsRowWrapLongStrings
	settingsRowSaveIndent
	settingsRowCount
)

var settingsIndentOptions = []config.SaveIndent{
	{Kind: config.IndentKindSpaces, Size: 2},
	{Kind: config.IndentKindSpaces, Size: 3},
	{Kind: config.IndentKindSpaces, Size: 4},
	{Kind: config.IndentKindTabs},
}

func (m *Model) settingsOpen() bool {
	return m.Session != nil && m.Session.Mode == session.ModeSettings
}

func (m *Model) openSettings() {
	m.Session.Mode = session.ModeSettings
	m.clearPendingPrefix()
	m.settingsRow = settingsRowTheme
}

func (m *Model) closeSettings() {
	m.Session.Mode = session.ModeNormal

	if m.currentSettings() == m.Settings.WithDefaults() {
		m.Session.SetStatus("closed settings")
		return
	}

	m.Session.SetStatus("kept preview settings for this session")
}

func (m *Model) updateSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.closeSettings()
	case "up", "k":
		m.moveSettingsRow(-1)
	case "down", "j":
		m.moveSettingsRow(1)
	case "h", "left":
		m.adjustSettingsRow(-1)
	case "l", "right":
		m.adjustSettingsRow(1)
	case "enter", " ":
		m.activateSettingsRow()
	case "s":
		m.saveSettings()
	}
	return m, nil
}

func (m *Model) previewSettingsTheme(step int) {
	themes := m.ThemeRegistry.Themes()
	if len(themes) == 0 {
		return
	}

	index := m.settingsThemeIndex(themes, m.Session.ThemeName)
	index = (index + step + len(themes)) % len(themes)
	m.Session.ThemeName = themes[index].Name

	if strings.EqualFold(m.Session.ThemeName, m.Settings.WithDefaults().Theme) {
		m.Session.SetStatus(fmt.Sprintf("previewing saved theme %q", m.Session.ThemeName))
		return
	}

	m.Session.SetStatus(fmt.Sprintf("previewing theme %q; press s to save", m.Session.ThemeName))
}

func (m *Model) currentSettings() config.Settings {
	settings := m.ActiveSettings.WithDefaults()
	settings.Theme = m.ThemeRegistry.ThemeByName(m.Session.ThemeName).Name
	return settings
}

func (m *Model) saveSettings() {
	if !m.settingsSaveAvailable() {
		m.Session.SetError("settings path unavailable; could not resolve user config dir")
		return
	}

	settings := m.currentSettings()
	if err := config.SaveSettings(m.SettingsPath, settings); err != nil {
		m.Session.SetError(err.Error())
		return
	}

	m.Settings = settings
	m.ActiveSettings = settings
	m.SettingsFilePresent = true
	m.SettingsPersisted = true
	m.Session.SetStatus("saved settings")
}

func (m *Model) saveThemeSettings() {
	m.saveSettings()
}

func (m *Model) settingsSaveAvailable() bool {
	return strings.TrimSpace(m.SettingsPath) != ""
}

func (m *Model) settingsDialogHint() string {
	if m.settingsSaveAvailable() {
		return "up/down row  left/right change  s save  esc close"
	}
	return "up/down row  left/right change  s unavailable  esc close"
}

func (m *Model) settingsFooterHint() string {
	if m.settingsSaveAvailable() {
		return "up/down row  left/right change  s save settings.json  esc close"
	}
	return "up/down row  left/right change  s unavailable  esc close"
}

func (m *Model) moveSettingsRow(step int) {
	m.settingsRow = (m.settingsRow + step + settingsRowCount) % settingsRowCount
}

func (m *Model) activateSettingsRow() {
	switch m.settingsRow {
	case settingsRowTheme:
		m.previewSettingsTheme(1)
	case settingsRowShowLineNumbers:
		m.toggleShowLineNumbers()
	case settingsRowShowJSONPath:
		m.toggleShowJSONPath()
	case settingsRowWrapLongStrings:
		m.toggleWrapLongStrings()
	case settingsRowSaveIndent:
		m.cycleSaveIndent(1)
	}
}

func (m *Model) adjustSettingsRow(step int) {
	switch m.settingsRow {
	case settingsRowTheme:
		m.previewSettingsTheme(step)
	case settingsRowShowLineNumbers:
		m.toggleShowLineNumbers()
	case settingsRowShowJSONPath:
		m.toggleShowJSONPath()
	case settingsRowWrapLongStrings:
		m.toggleWrapLongStrings()
	case settingsRowSaveIndent:
		m.cycleSaveIndent(step)
	}
}

func (m *Model) toggleWrapLongStrings() {
	settings := m.ActiveSettings.WithDefaults()
	settings.WrapLongStrings = !settings.WrapLongStrings
	m.ActiveSettings = settings
	state := "off"
	if settings.WrapLongStrings {
		state = "on"
	}
	m.Session.SetStatus(fmt.Sprintf("previewing long string wrapping %s; press s to save", state))
}

func (m *Model) toggleShowLineNumbers() {
	settings := m.ActiveSettings.WithDefaults()
	settings.ShowLineNumbers = !settings.ShowLineNumbers
	m.ActiveSettings = settings
	state := "off"
	if settings.ShowLineNumbers {
		state = "on"
	}
	m.Session.SetStatus(fmt.Sprintf("previewing line numbers %s; press s to save", state))
}

func (m *Model) toggleShowJSONPath() {
	settings := m.ActiveSettings.WithDefaults()
	settings.ShowJSONPath = !settings.ShowJSONPath
	m.ActiveSettings = settings
	state := "off"
	if settings.ShowJSONPath {
		state = "on"
	}
	m.Session.SetStatus(fmt.Sprintf("previewing JSON path display %s; press s to save", state))
}

func (m *Model) cycleSaveIndent(step int) {
	current := m.ActiveSettings.WithDefaults().SaveIndent
	index := 0
	for optionIndex, option := range settingsIndentOptions {
		if option.WithDefaults() == current {
			index = optionIndex
			break
		}
	}
	index = (index + step + len(settingsIndentOptions)) % len(settingsIndentOptions)
	settings := m.ActiveSettings.WithDefaults()
	settings.SaveIndent = settingsIndentOptions[index].WithDefaults()
	m.ActiveSettings = settings
	m.Session.SetStatus(fmt.Sprintf("previewing save indent %s; press s to save", settings.SaveIndent.Label()))
}

func (m *Model) settingsThemeIndex(themes []Theme, current string) int {
	current = normalizeThemeName(m.ThemeRegistry.ThemeByName(current).Name)
	for index, theme := range themes {
		if normalizeThemeName(theme.Name) == current {
			return index
		}
	}
	return 0
}

func (m *Model) settingsDialogView(theme Theme, width int) string {
	themes := m.ThemeRegistry.Themes()
	currentIndex := m.settingsThemeIndex(themes, m.Session.ThemeName)
	currentSettings := m.currentSettings()
	currentTheme := currentSettings.Theme
	savedSettings := m.Settings.WithDefaults()
	savedTheme := savedSettings.Theme

	position := "0/0"
	if len(themes) > 0 {
		position = fmt.Sprintf("%d/%d", currentIndex+1, len(themes))
	}

	stateLabel := theme.Muted.Render("saved")
	savedLabel := "Saved theme: " + savedTheme
	switch {
	case !m.settingsSaveAvailable():
		stateLabel = theme.Error.Render("unavailable")
		savedLabel = "Settings file unavailable; using " + savedTheme
	case !m.SettingsPersisted && !m.SettingsFilePresent:
		stateLabel = theme.Muted.Render("not saved")
		savedLabel = "Saved theme: none yet (using " + savedTheme + ")"
	case !m.SettingsPersisted:
		stateLabel = theme.Status.Render("fallback")
		savedLabel = "Saved theme unavailable; using " + savedTheme
	}
	if currentSettings != savedSettings {
		stateLabel = theme.Status.Render("preview only")
	}

	saveHint := theme.Muted.Render("Save writes settings.json on demand.")
	if !m.settingsSaveAvailable() {
		saveHint = theme.Error.Render("Save unavailable: config path could not be resolved.")
	}

	rows := []string{
		m.renderSettingsRow(theme, settingsRowTheme, "Theme", theme.Selected.Render(" "+currentTheme+" "), theme.Muted.Render(position)),
	}
	rows = append(rows,
		m.renderSettingsRow(theme, settingsRowShowLineNumbers, "Line numbers", theme.Selected.Render(" "+settingsBoolLabel(currentSettings.ShowLineNumbers)+" "), ""),
		m.renderSettingsRow(theme, settingsRowShowJSONPath, "JSON path", theme.Selected.Render(" "+settingsBoolLabel(currentSettings.ShowJSONPath)+" "), ""),
		m.renderSettingsRow(theme, settingsRowWrapLongStrings, "Long strings", theme.Selected.Render(" "+settingsBoolLabel(currentSettings.WrapLongStrings)+" "), ""),
		m.renderSettingsRow(theme, settingsRowSaveIndent, "Save indent", theme.Selected.Render(" "+currentSettings.SaveIndent.Label()+" "), ""),
	)

	lines := []string{
		theme.Status.Render("Settings"),
		"",
		rows[0],
	}
	lines = append(lines, rows[1:]...)
	lines = append(lines,
		theme.Muted.Render(savedLabel),
		theme.Muted.Render("Saved line numbers: "+settingsBoolLabel(savedSettings.ShowLineNumbers)+"  Saved JSON path: "+settingsBoolLabel(savedSettings.ShowJSONPath)),
		theme.Muted.Render("Saved long strings: "+settingsBoolLabel(savedSettings.WrapLongStrings)),
		theme.Muted.Render("Saved indent: "+savedSettings.SaveIndent.Label()),
		stateLabel,
		m.renderedStatusMessage(theme),
		theme.Help.Render(m.settingsDialogHint()),
		saveHint,
		theme.Muted.Render("Built-ins + config themes/*.json appear here."),
	)

	modalWidth := 68
	if width > 0 && width-4 < modalWidth {
		modalWidth = width - 4
	}
	if modalWidth < 24 {
		modalWidth = 24
	}

	return lipgloss.NewStyle().
		Width(modalWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Border.GetForeground()).
		Padding(0, 1).
		Render(strings.Join(lines, "\n"))
}

func (m *Model) renderSettingsRow(theme Theme, row int, label, value, extra string) string {
	line := theme.Key.Render(label) + " " + value
	if extra != "" {
		line += " " + extra
	}
	if row == m.settingsRow {
		return theme.Selected.Render(line)
	}
	return line
}

func settingsBoolLabel(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func (m *Model) renderSettingsOverlay(base string, theme Theme, width, height int) string {
	baseLines := strings.Split(base, "\n")
	if height > len(baseLines) {
		baseLines = append(baseLines, make([]string, height-len(baseLines))...)
	}

	modalLines := strings.Split(m.settingsDialogView(theme, width), "\n")
	top := 1
	if height > len(modalLines)+2 {
		top = (height - len(modalLines)) / 2
	}

	for index, line := range modalLines {
		row := top + index
		if row >= len(baseLines) {
			baseLines = append(baseLines, "")
		}
		padding := 0
		if width > lipgloss.Width(line) {
			padding = (width - lipgloss.Width(line)) / 2
		}
		baseLines[row] = trimWidth(strings.Repeat(" ", padding)+line, width)
	}

	return strings.Join(baseLines, "\n")
}
