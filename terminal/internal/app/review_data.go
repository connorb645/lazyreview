package app

import "review-desk/terminal/internal/server"

type reviewData struct {
	id    string
	state server.State
	page  server.Page
}
