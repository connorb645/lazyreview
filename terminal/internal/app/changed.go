package app

import "github.com/connorb645/lazyreview/terminal/internal/server"

type changed struct {
	e            editing
	state        *server.State
	conversation *server.Conversation
	err          error
}
