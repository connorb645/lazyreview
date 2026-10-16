package app

import "review-desk/terminal/internal/server"

type changed struct {
	e            editing
	state        *server.State
	conversation *server.Conversation
	err          error
}
