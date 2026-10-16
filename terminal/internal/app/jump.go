package app

import "review-desk/terminal/internal/server"

// jump is a place the app was before a jump: its screen, the item the
// Review view's cursor was on, and the component and line of the diff.
type jump struct {
	screen    screen
	item      int
	component string
	place     *server.Lines
}
