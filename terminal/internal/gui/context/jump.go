package context

import (
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/models"
)

// Jump is a place jumped from: what the screen showed, the Review view's
// item, the Files panel's rows the diff showed and its place, or the
// diagram's box.
type Jump struct {
	Screen Screen
	Item   int
	Chosen []string
	Place  *models.Lines
	Box    *types.DiagramPlace
}
