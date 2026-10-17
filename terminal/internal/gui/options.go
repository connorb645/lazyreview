package gui

import (
	"context"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
)

// Options are what the app needs from outside: the server, the review to
// open, or none for the home screen, whether the terminal is dark, how to
// open an address in the browser, the clock and the time zone dates are
// shown in, the user's home folder, which a path under it is shown from as
// ~, the editor a comment is written in, as a command line the shell runs
// with the file after it, how a program such as the editor is given the
// terminal, and how the app hears a review's changes: as messages, until
// the context ends.
type Options struct {
	Server   *server.Client
	Review   string
	Dark     bool
	Open     func(address string) error
	Now      func() time.Time
	Location *time.Location
	Home     string
	Editor   string
	Exec     func(*exec.Cmd, tea.ExecCallback) tea.Cmd
	Listen   func(ctx context.Context, c *server.Client, id string) <-chan tea.Msg
}
