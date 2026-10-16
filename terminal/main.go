// Command lazyreview-terminal is lazyreview in a terminal: the same reviews
// as the web app, with Vim keys, as a second client of the same server.
// lazyreview terminal builds it and runs it.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/connorb645/lazyreview/terminal/internal/app"
	"github.com/connorb645/lazyreview/terminal/internal/server"
)

func editor() string {
	for _, name := range []string{"LAZYREVIEW_EDITOR", "VISUAL", "EDITOR"} {
		if command := os.Getenv(name); command != "" {
			return command
		}
	}
	return "vi"
}

func main() {
	serverFile := flag.String("server-file", "", "the lazyreview server's own record, server.json, holding its address and token")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: lazyreview-terminal --server-file <server.json> [<review id>]")
	}
	flag.Parse()
	if *serverFile == "" || flag.NArg() > 1 {
		flag.Usage()
		os.Exit(2)
	}
	opts := app.Options{Server: server.NewClient(*serverFile), Review: flag.Arg(0), Dark: lipgloss.HasDarkBackground(), Editor: editor()}
	program := tea.NewProgram(app.New(opts), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "lazyreview terminal:", err)
		os.Exit(1)
	}
}
