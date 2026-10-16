package app

import "github.com/connorb645/lazyreview/terminal/internal/server"

type checked struct {
	e     editing
	where server.Where
	err   error
}
