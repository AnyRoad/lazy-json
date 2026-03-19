package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/anyroad/lazy-json/internal/config"
	"github.com/anyroad/lazy-json/internal/session"
)

func (m *Model) settingsOpen() bool {
	return m.Session != nil && m.Session.Mode == session.ModeSettings
}

func (m *Model) openSettings() {
	m.Session.Mode = session.ModeSettings
	m.lastKey = ""
}

func (m *Model) closeSettings() {
	m.Session.Mode = session.ModeNormal

	if strings.EqualFold(m.Session.ThemeName, m.Settings.Theme) {
		m.Session.SetStatus("closed settings")
		return
	}

	m.Session.SetStatus(fmt.Sprintf("kept preview theme %q for this session", m.Session.ThemeName))
}

func (m *Model) updateSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.closeSettings()
	case "h", "left":
		m.previewSettingsTheme(-1)
	case "l", "right":
		m.previewSettingsTheme(1)
	case "s":
		m.saveThemeSettings()
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

	if strings.EqualFold(m.Session.ThemeName, m.Settings.Theme) {
		m.Session.SetStatus(fmt.Sprintf("previewing saved theme %q", m.Session.ThemeName))
		return
	}

	m.Session.SetStatus(fmt.Sprintf("previewing theme %q; press s to save", m.Session.ThemeName))
}

func (m *Model) saveThemeSettings() {
	if strings.TrimSpace(m.SettingsPath) == "" {
		m.Session.SetError("settings path unavailable; could not resolve user config dir")
		return
	}

	settings := m.Settings.WithDefaults()
	settings.Theme = m.ThemeRegistry.ThemeByName(m.Session.ThemeName).Name
	if err := config.SaveSettings(m.SettingsPath, settings); err != nil {
		m.Session.SetError(err.Error())
		return
	}

	m.Settings = settings
	m.Session.SetStatus(fmt.Sprintf("saved theme %q", m.Settings.Theme))
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
	currentTheme := m.ThemeRegistry.ThemeByName(m.Session.ThemeName).Name
	savedTheme := m.Settings.WithDefaults().Theme

	position := "0/0"
	if len(themes) > 0 {
		position = fmt.Sprintf("%d/%d", currentIndex+1, len(themes))
	}

	stateLabel := theme.Muted.Render("saved")
	if !strings.EqualFold(currentTheme, savedTheme) {
		stateLabel = theme.Status.Render("preview only")
	}

	saveHint := theme.Muted.Render("Save writes settings.json on demand.")
	if strings.TrimSpace(m.SettingsPath) == "" {
		saveHint = theme.Error.Render("Save unavailable: config path could not be resolved.")
	}

	lines := []string{
		theme.Status.Render("Theme Settings"),
		"",
		theme.Key.Render("Theme") + " " + theme.Selected.Render(" "+currentTheme+" ") + " " + theme.Muted.Render(position) + " " + stateLabel,
		theme.Muted.Render("Saved theme: " + savedTheme),
		theme.Help.Render("h/left prev  l/right next  s save  esc close"),
		saveHint,
	}

	modalWidth := 60
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
