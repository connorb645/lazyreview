package context

// ScreenMode is how much of the screen the panel with the keys takes: its
// share, half, or all of it.
type ScreenMode int

// The screen modes, and how many there are.
const (
	NormalScreen ScreenMode = iota
	HalfScreen
	FullScreen
	ScreenModes
)
