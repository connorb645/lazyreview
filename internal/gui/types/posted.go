package types

import "github.com/connorb645/lazyreview/internal/models"

// Posted is the answer to posting to a pull request on GitHub: its words
// and the state after, or why nothing was posted.
type Posted struct {
	Pull    int
	Message string
	State   models.State
	Err     error
}
