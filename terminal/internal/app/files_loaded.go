package app

import "review-desk/terminal/internal/server"

type filesLoaded struct {
	component string
	files     map[string]server.File
	place     *server.Lines
	err       error
}
