// Package loader loads reviews: a pull request, or a stack of them, from
// GitHub, and a branch from a repository on this machine. Every file of a
// load is written or none.
package loader

import (
	"github.com/connorb645/lazyreview/terminal/internal/build/page"
	"github.com/connorb645/lazyreview/terminal/internal/models"
)

// Loaded is a review as a load left it, with what it drew and the notes
// worth telling whoever loaded it.
type Loaded struct {
	Ref    models.ReviewRef
	Review models.Review
	Counts page.Counts
	Notes  []string
}
