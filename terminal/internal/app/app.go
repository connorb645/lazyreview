// Package app starts the terminal app: it builds what every layer shares,
// the server client and the gui, then runs Bubble Tea until the app quits.
package app

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/common"
	"github.com/connorb645/lazyreview/terminal/internal/gui"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/theme"
)

// Run opens the app on the review id names, or with none on the home
// screen, as a client of the server whose record is at serverFile, and
// returns once the app quits.
func Run(serverFile, review string) error {
	cmn := &common.Common{Tr: i18n.EnglishTranslationSet(), Theme: terminalTheme()}
	opts := gui.Options{Server: server.NewClient(cmn, serverFile), Review: review, Editor: editor()}
	program := tea.NewProgram(gui.New(cmn, opts), tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := program.Run()
	return err
}

func terminalTheme() theme.Theme {
	if lipgloss.HasDarkBackground() {
		return theme.New(theme.DarkColours(), true, lipgloss.ColorProfile())
	}
	return theme.New(theme.LightColours(), false, lipgloss.ColorProfile())
}

func editor() string {
	for _, name := range []string{"LAZYREVIEW_EDITOR", "VISUAL", "EDITOR"} {
		if command := os.Getenv(name); command != "" {
			return command
		}
	}
	return "vi"
}
