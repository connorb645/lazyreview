package types

import "github.com/connorb645/lazyreview/terminal/internal/commands/server"

// PullsRead is the open pull requests read from GitHub, or why not.
type PullsRead struct {
	Pulls server.Pulls
	Err   error
}
