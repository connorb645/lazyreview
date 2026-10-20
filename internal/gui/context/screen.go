package context

// Screen is what the screen shows: the home screen, or a review with its
// Review view, a diff or the diagram in the main panel.
type Screen int

// The screens.
const (
	ScreenHome Screen = iota
	ScreenReview
	ScreenDiff
	ScreenDiagram
)
