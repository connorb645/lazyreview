package page

import "github.com/connorb645/lazyreview/internal/models"

// Built is a page built: what the terminal app draws, the commit each
// member's own diff starts from, bottom first, what it drew, and lines
// worth telling whoever loaded it, such as a file whose whole text was
// left out for its size.
type Built struct {
	Page   models.PageRecord
	Starts []string
	Counts Counts
	Notes  []string
}
