package types

import "slices"

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
