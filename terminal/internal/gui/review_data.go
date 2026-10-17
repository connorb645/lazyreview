package gui

import "github.com/connorb645/lazyreview/terminal/internal/commands/server"

type reviewData struct {
	id    string
	state server.State
	page  server.Page
}
