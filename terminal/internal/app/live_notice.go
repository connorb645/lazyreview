package app

import "github.com/connorb645/lazyreview/terminal/internal/server"

type liveNotice struct {
	id     string
	notice server.Notice
}
