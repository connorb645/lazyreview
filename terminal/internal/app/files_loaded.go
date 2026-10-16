package app

import "github.com/connorb645/lazyreview/terminal/internal/server"

type filesLoaded struct {
	component string
	files     map[string]server.File
	place     *server.Lines
	err       error
}
