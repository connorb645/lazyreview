package helpers

import (
	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/views"
)

// Layout is where a review's windows sit: the four panels on the left, the
// main panel, and the bar. A window with no room is empty.
type Layout struct {
	Sides [context.SideCount + 1]views.Rect
	Main  views.Rect
	Bar   views.Rect
}
