package helpers

import (
	stdcontext "context"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/common"
	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
)

// HelperCommon is what every helper is given: what every layer shares, the
// server, how to open an address in the browser, the clock and the time
// zone dates are shown in, the user's home folder, the editor and how a
// program is given the terminal, how a review's changes are heard, the
// Markdown renderer, every helper, so one can call another, and the labels
// of each context's keys, as its bindings have them, to name a key in
// words.
type HelperCommon struct {
	*common.Common
	Server    *server.Client
	Open      func(address string) error
	Now       func() time.Time
	Location  *time.Location
	Home      string
	Editor    string
	Exec      func(*exec.Cmd, tea.ExecCallback) tea.Cmd
	Listen    func(ctx stdcontext.Context, c *server.Client, id string) <-chan tea.Msg
	Markdown  *presentation.Markdown
	Helpers   *Helpers
	KeyLabels func(key context.ContextKey) presentation.KeyLabels
}
