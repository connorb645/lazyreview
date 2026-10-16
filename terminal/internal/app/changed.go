package app

import "review-desk/terminal/internal/server"

// changed is the server's answer to a change: the state or the
// conversation as it then stands, or why it refused.
type changed struct {
	e            editing
	state        *server.State
	conversation *server.Conversation
	err          error
}
