package gui

import "github.com/connorb645/lazyreview/terminal/internal/commands/server"

type changed struct {
	e            editing
	state        *server.State
	conversation *server.Conversation
	err          error
}
