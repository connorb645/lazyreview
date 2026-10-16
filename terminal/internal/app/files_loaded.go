package app

import "review-desk/terminal/internal/server"

// filesLoaded is a component's files the server gave, or why it gave none,
// and the place in them to open on.
type filesLoaded struct {
	component string
	files     map[string]server.File
	place     *server.Lines
	err       error
}
