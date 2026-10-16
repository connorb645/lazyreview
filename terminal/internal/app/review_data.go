package app

import "review-desk/terminal/internal/server"

// reviewData is the review open: its state, and the page it was drawn from.
type reviewData struct {
	id    string
	state server.State
	page  server.Page
}
