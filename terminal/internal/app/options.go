package app

import (
	"time"

	"review-desk/terminal/internal/server"
)

// Options are what the app needs from outside: the server, the review to
// open, or none for the list of reviews, whether the terminal is dark, how
// to open an address in the browser, the clock and the time zone dates are
// shown in.
type Options struct {
	Server   *server.Client
	Review   string
	Dark     bool
	Open     func(address string) error
	Now      func() time.Time
	Location *time.Location
}
