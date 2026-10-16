package app

import "review-desk/terminal/internal/server"

type jump struct {
	screen    screen
	item      int
	component string
	place     *server.Lines
	box       *diagramPlace
}
