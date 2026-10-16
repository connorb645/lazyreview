package app

// screen is the view the app shows.
type screen int

const (
	screenReviews screen = iota
	screenReview
	screenDiff
	screenDiagram
)
