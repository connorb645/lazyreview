// Command review-desk-terminal is Review Desk in a terminal: the same reviews
// as the web app, with Vim keys, as a second client of the same server.
// review-desk terminal builds it and runs it.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"review-desk/terminal/internal/app"
	"review-desk/terminal/internal/server"
)

func main() {
	serverFile := flag.String("server-file", "", "the Review Desk server's own record, server.json, holding its address and token")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: review-desk-terminal --server-file <server.json> [<review id>]")
	}
	flag.Parse()
	if *serverFile == "" || flag.NArg() > 1 {
		flag.Usage()
		os.Exit(2)
	}
	opts := app.Options{Server: server.NewClient(*serverFile), Review: flag.Arg(0), Dark: lipgloss.HasDarkBackground()}
	program := tea.NewProgram(app.New(opts), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "review-desk terminal:", err)
		os.Exit(1)
	}
}
