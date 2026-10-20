package queue

import "github.com/connorb645/lazyreview/internal/models"

// Regrouping is what Regroup takes a review's queue to: Previous, the
// review kept at its folder, or nil for a new one; Fresh, each pull
// request the load names as GitHub has it now, bottom first; Leaving, the
// state and end of each live member of Previous the load no longer names,
// as GitHub has it now; Absorbed, the other reviews whose queues move in;
// Anchor, this review's id, for the moved reviews to point at; and Now,
// when.
type Regrouping struct {
	Previous *models.Review
	Fresh    []models.ReviewMember
	Leaving  map[int]models.ReviewMember
	Absorbed []Absorbed
	Anchor   string
	Now      string
}
