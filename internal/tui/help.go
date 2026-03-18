package tui

func (m *Model) helpView() string {
	theme := m.theme()
	return theme.Help.Render(helpText)
}
