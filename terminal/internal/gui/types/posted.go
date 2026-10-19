package types

import "github.com/connorb645/lazyreview/terminal/internal/models"

// Posted is the server's answer to posting to GitHub: its words and the
// state after, or why nothing was posted.
type Posted struct {
	Message string
	State   models.State
	Err     error
}
