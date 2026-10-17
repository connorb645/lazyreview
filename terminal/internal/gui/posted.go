package gui

import "github.com/connorb645/lazyreview/terminal/internal/commands/server"

type posted struct {
	message string
	state   server.State
	err     error
}
