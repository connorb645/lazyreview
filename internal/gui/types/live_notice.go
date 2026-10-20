package types

import "github.com/connorb645/lazyreview/internal/models"

// LiveNotice is a review's change notice, heard by watching its files.
type LiveNotice struct {
	ID     string
	Notice models.Notice
}
