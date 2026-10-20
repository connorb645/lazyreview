package presentation

import (
	"github.com/connorb645/lazyreview/internal/gui/types"
	"github.com/connorb645/lazyreview/internal/models"
)

// Item is what a row stands for, so a key on it knows what to open: a
// section, a stack's pull requests all together, a pull request, a
// component, a file, a queued comment, a conversation, a thread, a link, a
// box or a choice.
type Item struct {
	Kind      string
	Title     string
	Section   string
	Component string
	Pull      int
	Comment   int
	Thread    int
	URL       string
	Place     *models.Lines
	Box       *types.DiagramPlace
}
