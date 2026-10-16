package app

import "review-desk/terminal/internal/server"

// posted is the server's answer to a post: its words and the state as it
// then stands, or why it refused.
type posted struct {
	message string
	state   server.State
	err     error
}
