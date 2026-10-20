package store

import "github.com/connorb645/lazyreview/terminal/internal/models"

// HeldReview is a pull request review with the ref its folder has.
type HeldReview struct {
	Ref    models.PullRef
	Review models.Review
}
