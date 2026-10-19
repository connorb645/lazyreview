package types

import "github.com/connorb645/lazyreview/terminal/internal/models"

// StateLoaded is a review's state read again, or why not.
type StateLoaded struct {
	State models.State
	Err   error
}
