package app

import "review-desk/terminal/internal/server"

type posted struct {
	message string
	state   server.State
	err     error
}
