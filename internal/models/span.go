package models

// Span is a run of source lines, from its first to its last.
type Span struct {
	StartLine int `json:"start_line"`
	Line      int `json:"line"`
}
