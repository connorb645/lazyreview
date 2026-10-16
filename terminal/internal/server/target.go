package server

// Target is the pull request a comment goes on, at its own path and lines.
type Target struct {
	Pull      int    `json:"pull"`
	Path      string `json:"path"`
	Side      string `json:"side"`
	Line      int    `json:"line"`
	StartLine int    `json:"start_line"`
	StartSide string `json:"start_side"`
}
