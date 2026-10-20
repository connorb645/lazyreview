package gui

import (
	stdcontext "context"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
)

// Options are what the app needs from outside: the reviews it reads and
// changes, the review to
// open, or none for the home screen, how to open an address in the
// browser, the clock and the time zone dates are
// shown in, the user's home folder, which a path under it is shown from as
// ~, the editor a comment is written in, as a command line the shell runs
// with the file after it, how a program such as the editor is given the
// terminal, how the app hears a review's changes: as messages, until the
// context ends, and how it sends itself a message after a wait.
type Options struct {
	Reviews  types.Reviews
	Review   string
	Open     func(address string) error
	Now      func() time.Time
	Location *time.Location
	Home     string
	Editor   string
	Exec     func(*exec.Cmd, tea.ExecCallback) tea.Cmd
	Listen   func(ctx stdcontext.Context, c types.Reviews, id string) <-chan tea.Msg
	Tick     func(time.Duration, func(time.Time) tea.Msg) tea.Cmd
}
