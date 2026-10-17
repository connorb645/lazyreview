package presentation

import (
	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
)

// Item is what a row stands for, so a key on it knows what to open: a
// section, a pull request, a component, a file, a queued comment, a
// conversation, a thread, a link, a box or a choice.
type Item struct {
	Kind      string
	Title     string
	Section   string
	Component string
	Pull      int
	Comment   int
	URL       string
	Place     *server.Lines
	Box       *types.DiagramPlace
}
