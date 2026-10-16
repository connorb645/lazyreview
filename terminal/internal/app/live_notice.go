package app

import "review-desk/terminal/internal/server"

type liveNotice struct {
	id     string
	notice server.Notice
}
