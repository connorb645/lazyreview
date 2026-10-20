package types

import (
	"slices"

	"github.com/connorb645/lazyreview/internal/models"
)

// PullFilter is the pull requests of a stack the screen shows, a run of
// them bottom first, or every one when it is empty.
type PullFilter []int

// Shows says whether the filter shows a pull request.
func (f PullFilter) Shows(pull int) bool {
	return len(f) == 0 || slices.Contains(f, pull)
}

// ShowsAny says whether the filter shows any of the pull requests that
// changed something, or shows everything.
func (f PullFilter) ShowsAny(pulls []int) bool {
	for _, pull := range pulls {
		if slices.Contains(f, pull) {
			return true
		}
	}
	return len(f) == 0
}

// Within is the filter less the pull requests the stack no longer holds.
func (f PullFilter) Within(stack []int) PullFilter {
	var kept PullFilter
	for _, pull := range f {
		if slices.Contains(stack, pull) {
			kept = append(kept, pull)
		}
	}
	return kept
}

// Lines are the lines of a file the filter shows: all it adds and removes,
// or those its pull requests added and removed.
func (f PullFilter) Lines(e models.FileEntry) (adds, dels int) {
	if len(f) == 0 {
		return e.Adds, e.Dels
	}
	for _, pull := range f {
		own := e.ByPull[pull]
		adds, dels = adds+own[0], dels+own[1]
	}
	return adds, dels
}

// Over is the filter over a stack's pull requests: none, the whole stack,
// when it holds every one of them.
func (f PullFilter) Over(stack []int) PullFilter {
	for _, pull := range stack {
		if !slices.Contains(f, pull) {
			return f
		}
	}
	return nil
}
