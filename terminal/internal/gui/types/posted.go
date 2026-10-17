package types

import "github.com/connorb645/lazyreview/terminal/internal/commands/server"

// Posted is the server's answer to posting to GitHub: its words and the
// state after, or why nothing was posted.
type Posted struct {
	Message string
	State   server.State
	Err     error
}
