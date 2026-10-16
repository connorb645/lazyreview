package app

import "github.com/connorb645/lazyreview/terminal/internal/server"

type item struct {
	kind      string
	title     string
	section   string
	component string
	pull      int
	comment   int
	url       string
	place     *server.Lines
	box       *diagramPlace
}
