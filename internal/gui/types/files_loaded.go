package types

import "github.com/connorb645/lazyreview/internal/models"

// FilesLoaded is files loaded to show their diff at a place, or why they
// could not be.
type FilesLoaded struct {
	Files map[string]models.File
	Place *models.Lines
	Err   error
}
