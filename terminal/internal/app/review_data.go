package app

import "github.com/connorb645/lazyreview/terminal/internal/server"

type reviewData struct {
	id    string
	state server.State
	page  server.Page
}
