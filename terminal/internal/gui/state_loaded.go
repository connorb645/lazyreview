package gui

import "github.com/connorb645/lazyreview/terminal/internal/commands/server"

type stateLoaded struct {
	state server.State
	err   error
}
