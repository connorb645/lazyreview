package queue

import "github.com/connorb645/lazyreview/internal/models"

// Absorbed is another review whose every live pull request a load names,
// whose queue moves into the review the load writes.
type Absorbed struct {
	Ref    models.ReviewRef
	Review models.Review
}
