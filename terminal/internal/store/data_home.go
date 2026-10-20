// Package store keeps lazyreview's files under its data home: each review's
// folder, its record, its page and its conversations, each file written
// whole, and the locks that let one process at a time change a review.
package store

import (
	"path/filepath"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/models"
)

// DataHome is the folder lazyreview keeps its files under:
//
//	reviews/<host>/<owner>/<repo>/<pull>/  one folder per pull request review
//	reviews/local/<repo key>/<branch>/      one folder per branch review
//
// It only names paths, and reads and writes nothing.
type DataHome struct {
	Root string
}

// Reviews is the folder holding every review.
func (h DataHome) Reviews() string {
	return filepath.Join(h.Root, "reviews")
}

// Review is the folder of the review a ref names.
func (h DataHome) Review(ref models.ReviewRef) ReviewFolder {
	return ReviewFolder{Path: filepath.Join(append([]string{h.Reviews()}, strings.Split(ref.ID(), "/")...)...)}
}
