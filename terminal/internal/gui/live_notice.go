package gui

import "github.com/connorb645/lazyreview/terminal/internal/commands/server"

type liveNotice struct {
	id     string
	notice server.Notice
}
