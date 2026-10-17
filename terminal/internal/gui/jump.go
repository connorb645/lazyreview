package gui

import "github.com/connorb645/lazyreview/terminal/internal/commands/server"

type jump struct {
	screen    screen
	item      int
	component string
	place     *server.Lines
	box       *diagramPlace
}
