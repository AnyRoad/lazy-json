package tui

import "github.com/charmbracelet/lipgloss"

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

var themes = []Theme{
	{
		Name:      "forest",
		Key:       lipgloss.NewStyle().Foreground(lipgloss.Color("#9CCF5D")),
		String:    lipgloss.NewStyle().Foreground(lipgloss.Color("#F6BD60")),
		Number:    lipgloss.NewStyle().Foreground(lipgloss.Color("#84A59D")),
		Bool:      lipgloss.NewStyle().Foreground(lipgloss.Color("#F28482")),
		Null:      lipgloss.NewStyle().Foreground(lipgloss.Color("#CDB4DB")),
		Muted:     lipgloss.NewStyle().Foreground(lipgloss.Color("#7A7A7A")),
		Selected:  lipgloss.NewStyle().Background(lipgloss.Color("#283618")).Foreground(lipgloss.Color("#FEFAE0")).Bold(true),
		SearchHit: lipgloss.NewStyle().Background(lipgloss.Color("#3A5A40")).Foreground(lipgloss.Color("#FEFAE0")),
		Status:    lipgloss.NewStyle().Foreground(lipgloss.Color("#D4A373")),
		Error:     lipgloss.NewStyle().Foreground(lipgloss.Color("#E63946")).Bold(true),
		Border:    lipgloss.NewStyle().Foreground(lipgloss.Color("#606C38")),
		Help:      lipgloss.NewStyle().Foreground(lipgloss.Color("#DDA15E")),
		Prompt:    lipgloss.NewStyle().Foreground(lipgloss.Color("#FEFAE0")),
	},
	{
		Name:      "harbor",
		Key:       lipgloss.NewStyle().Foreground(lipgloss.Color("#7BDFF2")),
		String:    lipgloss.NewStyle().Foreground(lipgloss.Color("#F7A072")),
		Number:    lipgloss.NewStyle().Foreground(lipgloss.Color("#B2F7EF")),
		Bool:      lipgloss.NewStyle().Foreground(lipgloss.Color("#F2B5D4")),
		Null:      lipgloss.NewStyle().Foreground(lipgloss.Color("#CDB4DB")),
		Muted:     lipgloss.NewStyle().Foreground(lipgloss.Color("#5C677D")),
		Selected:  lipgloss.NewStyle().Background(lipgloss.Color("#0B3954")).Foreground(lipgloss.Color("#FFF3B0")).Bold(true),
		SearchHit: lipgloss.NewStyle().Background(lipgloss.Color("#087E8B")).Foreground(lipgloss.Color("#FFF3B0")),
		Status:    lipgloss.NewStyle().Foreground(lipgloss.Color("#F7A072")),
		Error:     lipgloss.NewStyle().Foreground(lipgloss.Color("#D7263D")).Bold(true),
		Border:    lipgloss.NewStyle().Foreground(lipgloss.Color("#247BA0")),
		Help:      lipgloss.NewStyle().Foreground(lipgloss.Color("#FFF3B0")),
		Prompt:    lipgloss.NewStyle().Foreground(lipgloss.Color("#E0FBFC")),
	},
}

func ThemeByName(name string) Theme {
	for _, theme := range themes {
		if theme.Name == name {
			return theme
		}
	}
	return themes[0]
}

func NextTheme(name string) Theme {
	for idx, theme := range themes {
		if theme.Name == name {
			return themes[(idx+1)%len(themes)]
		}
	}
	return themes[0]
}
