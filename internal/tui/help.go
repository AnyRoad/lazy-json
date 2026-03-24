package tui

import "strings"

func (m *Model) helpView() string {
	theme := m.theme()
	_, height := m.viewportSize()
	lines := strings.Split(theme.Help.Render(helpText), "\n")
	return strings.Join(fitLinesToHeight(lines, height), "\n")
}
