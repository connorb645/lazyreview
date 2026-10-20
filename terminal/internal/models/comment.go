package models

// Comment is a comment queued for a pull request, as review.json keeps it:
// where it goes in that pull request's own diff, its text, the code it
// quotes, and on a stack where the combined diff showed it. Stale says why
// no button sends it, when its line may mean something else now. Version is
// never kept: the queue gives it, a short hash of the comment as it stands,
// so a change names the comment exactly as its window showed it.
type Comment struct {
	ID        int    `json:"id"`
	Path      string `json:"path"`
	Line      int    `json:"line"`
	Side      string `json:"side"`
	Body      string `json:"body"`
	Code      string `json:"code"`
	StartLine int    `json:"start_line,omitempty"`
	StartSide string `json:"start_side,omitempty"`
	Shown     *Lines `json:"shown,omitempty"`
	Stale     string `json:"stale,omitempty"`
	Version   string `json:"version,omitempty"`
}
