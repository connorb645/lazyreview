package types

import "github.com/connorb645/lazyreview/terminal/internal/models"

// PullsRead is the open pull requests read from GitHub, or why not.
type PullsRead struct {
	Pulls models.Pulls
	Err   error
}
