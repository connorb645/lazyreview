package server

// Comment is a comment queued for a pull request, at that pull request's own
// lines. Shown is where the app showed it, on a stack's combined diff;
// Version names it exactly as it now stands.
type Comment struct {
	ID        int    `json:"id"`
	Path      string `json:"path"`
	Side      string `json:"side"`
	Line      int    `json:"line"`
	StartLine int    `json:"start_line"`
	StartSide string `json:"start_side"`
	Body      string `json:"body"`
	Code      string `json:"code"`
	Shown     *Lines `json:"shown"`
	Stale     string `json:"stale"`
	Version   string `json:"version"`
}
