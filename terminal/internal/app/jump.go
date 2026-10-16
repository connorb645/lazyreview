package app

import "github.com/connorb645/lazyreview/terminal/internal/server"

type jump struct {
	screen    screen
	item      int
	component string
	place     *server.Lines
	box       *diagramPlace
}
