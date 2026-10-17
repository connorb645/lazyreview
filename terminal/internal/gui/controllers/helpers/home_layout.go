package helpers

import "github.com/connorb645/lazyreview/terminal/internal/gui/views"

// HomeLayout is where the home screen's windows sit: a panel for each
// repository, the main panel, and the bar. A window with no room is empty.
type HomeLayout struct {
	Panels []views.Rect
	Main   views.Rect
	Bar    views.Rect
}
