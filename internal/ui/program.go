package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/gusandrioli/tfp/internal/planmodel"
	"github.com/gusandrioli/tfp/internal/summary"
)

// Run launches the interactive TUI for root/rep and blocks until the
// user quits.
func Run(root *planmodel.Module, rep summary.Report) error {
	p := tea.NewProgram(New(root, rep), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
