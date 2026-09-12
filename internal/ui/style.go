package ui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/gusandrioli/tfp/internal/planmodel"
)

var (
	styleTitle = lipgloss.NewStyle().Bold(true)

	styleSelected = lipgloss.NewStyle().Reverse(true)

	styleModule = lipgloss.NewStyle().Bold(true)

	styleDimmed = lipgloss.NewStyle().Faint(true)

	styleCreate  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styleUpdate  = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styleDelete  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	styleReplace = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))

	styleStatusBar = lipgloss.NewStyle().Faint(true)

	stylePane = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			Padding(0, 1)

	stylePaneFocused = stylePane.
				BorderForeground(lipgloss.Color("6"))
)

func kindStyle(k planmodel.ChangeKind) lipgloss.Style {
	switch k.Symbol() {
	case "+":
		return styleCreate
	case "~":
		return styleUpdate
	case "-":
		return styleDelete
	case "-/+":
		return styleReplace
	default:
		return lipgloss.NewStyle()
	}
}
