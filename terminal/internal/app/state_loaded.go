package app

import "review-desk/terminal/internal/server"

type stateLoaded struct {
	state server.State
	err   error
}
