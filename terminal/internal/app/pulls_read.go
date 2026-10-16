package app

import "github.com/connorb645/lazyreview/terminal/internal/server"

type pullsRead struct {
	pulls server.Pulls
	err   error
}
