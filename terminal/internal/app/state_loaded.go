package app

import "github.com/connorb645/lazyreview/terminal/internal/server"

type stateLoaded struct {
	state server.State
	err   error
}
