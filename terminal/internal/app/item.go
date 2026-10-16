package app

import "review-desk/terminal/internal/server"

// item is a thing in a view the cursor stops on: a section, a link, a
// component, a queued comment, a thread on GitHub or a conversation, or a
// row of a panel. Place is the line it opens in the diff, and box the box
// it shows on the diagram, when it has one.
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
