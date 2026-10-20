package types

import "github.com/connorb645/lazyreview/internal/models"

// Checked is where the queue says a comment would go, before it is
// written, or why it cannot go anywhere.
type Checked struct {
	E     Editing
	Where models.Where
	Err   error
}
