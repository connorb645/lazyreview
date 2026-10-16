package app

import "github.com/connorb645/lazyreview/terminal/internal/server"

type posted struct {
	message string
	state   server.State
	err     error
}
