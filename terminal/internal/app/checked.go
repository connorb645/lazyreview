package app

import "review-desk/terminal/internal/server"

// checked is where the server says a comment about to be written would go.
type checked struct {
	e     editing
	where server.Where
	err   error
}
