package types

import "github.com/connorb645/lazyreview/terminal/internal/commands/server"

// LiveNotice is a review's change notice, heard from the server.
type LiveNotice struct {
	ID     string
	Notice server.Notice
}
