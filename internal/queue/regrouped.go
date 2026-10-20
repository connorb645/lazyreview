package queue

import "github.com/connorb645/lazyreview/internal/models"

// Regrouped is a review's members and earlier members after a load, each
// absorbed review emptied and pointing at it, and notes saying what moved
// and what went stale.
type Regrouped struct {
	Members []models.ReviewMember
	Earlier []models.ReviewMember
	Moved   []Absorbed
	Notes   []string
}
