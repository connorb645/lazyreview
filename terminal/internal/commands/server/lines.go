package server

// Lines are the lines a note is on: one line, or a range from StartLine,
// which may start on the other side from its last line.
type Lines struct {
	Path      string `json:"path"`
	Side      string `json:"side"`
	Line      int    `json:"line"`
	StartLine int    `json:"start_line,omitempty"`
	StartSide string `json:"start_side,omitempty"`
}
