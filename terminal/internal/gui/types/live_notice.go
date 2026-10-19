package types

import "github.com/connorb645/lazyreview/terminal/internal/models"

// LiveNotice is a review's change notice, heard from the server.
type LiveNotice struct {
	ID     string
	Notice models.Notice
}
