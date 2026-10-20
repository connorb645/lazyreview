package presentation

import (
	"github.com/connorb645/lazyreview/internal/diagram"
	"github.com/connorb645/lazyreview/internal/gui/types"
)

// Dimmed says whether a box of the diagram is dimmed because the filter
// shows none of the pull requests that changed it, or anything it holds.
// Unchanged code is context for every pull request, and never dims.
func Dimmed(card diagram.Card, filter types.PullFilter) bool {
	if card.Context || card.State == "context" {
		return false
	}
	return !filter.ShowsAny(card.Pulls)
}
