package types

import "github.com/connorb645/lazyreview/terminal/internal/commands/server"

// Checked is where the server says a comment would go, before it is
// written, or why it cannot go anywhere.
type Checked struct {
	E     Editing
	Where server.Where
	Err   error
}
