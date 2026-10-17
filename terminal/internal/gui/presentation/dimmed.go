package presentation

import (
	"github.com/connorb645/lazyreview/terminal/internal/diagram"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
)

// Dimmed says whether a box of the diagram is dimmed because the filter
// shows none of the pull requests that changed it. A component or a
// package is never dimmed.
func Dimmed(card diagram.Card, filter types.PullFilter) bool {
	if card.Kind == "component" || card.Kind == "package" {
		return false
	}
	return !filter.ShowsAny(card.Pulls)
}
