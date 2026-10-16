package app

import "review-desk/terminal/internal/server"

// reviewsLoaded is the list of reviews the server gave, or why it gave none.
type reviewsLoaded struct {
	reviews []server.Summary
	err     error
}
