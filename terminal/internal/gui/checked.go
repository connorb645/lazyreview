package gui

import "github.com/connorb645/lazyreview/terminal/internal/commands/server"

type checked struct {
	e     editing
	where server.Where
	err   error
}
