package app

import "review-desk/terminal/internal/server"

// stateLoaded is the review's state read again.
type stateLoaded struct {
	state server.State
	err   error
}
