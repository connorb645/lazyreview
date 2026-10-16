package app

import "review-desk/terminal/internal/server"

// liveNotice is a change notice from the server for a review.
type liveNotice struct {
	id     string
	notice server.Notice
}
