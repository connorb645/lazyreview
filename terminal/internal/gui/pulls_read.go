package gui

import "github.com/connorb645/lazyreview/terminal/internal/commands/server"

type pullsRead struct {
	pulls server.Pulls
	err   error
}
