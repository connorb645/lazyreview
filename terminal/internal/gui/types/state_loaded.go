package types

import "github.com/connorb645/lazyreview/terminal/internal/commands/server"

// StateLoaded is a review's state read again, or why not.
type StateLoaded struct {
	State server.State
	Err   error
}
