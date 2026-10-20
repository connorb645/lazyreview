package store

import "github.com/connorb645/lazyreview/terminal/internal/models"

// KeptReview is a review the data home keeps, of a pull request or of a
// branch, with the ref its folder has.
type KeptReview struct {
	Ref    models.ReviewRef
	Review models.Review
}
