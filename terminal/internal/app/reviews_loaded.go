package app

import "review-desk/terminal/internal/server"

type reviewsLoaded struct {
	reviews []server.Summary
	err     error
}
