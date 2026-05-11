// internal/ui/styles/styles.go
package styles

import "github.com/charmbracelet/lipgloss"

var (
	Breadcrumb = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")).
			Bold(true)

	Selected = lipgloss.NewStyle().
			Background(lipgloss.Color("237")).
			Foreground(lipgloss.Color("15"))

	Dim = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15"))

	Subtitle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	Error = lipgloss.NewStyle().
		Foreground(lipgloss.Color("9")).
		Bold(true)

	StatusBar = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	Overlay = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	Label = lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Bold(true)
)
