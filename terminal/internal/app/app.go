// Package app starts the terminal app: it builds the server client and the
// gui, then runs Bubble Tea until the app quits.
package app

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/gui"
)

// Run opens the app on the review id names, or with none on the home
// screen, as a client of the server whose record is at serverFile, and
// returns once the app quits.
func Run(serverFile, review string) error {
	opts := gui.Options{Server: server.NewClient(serverFile), Review: review, Dark: lipgloss.HasDarkBackground(), Editor: editor()}
	program := tea.NewProgram(gui.New(opts), tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := program.Run()
	return err
}

func editor() string {
	for _, name := range []string{"LAZYREVIEW_EDITOR", "VISUAL", "EDITOR"} {
		if command := os.Getenv(name); command != "" {
			return command
		}
	}
	return "vi"
}
