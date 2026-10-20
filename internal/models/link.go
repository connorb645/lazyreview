package models

// Link is a file:line a written review names, resolved to the one file of
// the review whose path ends with it.
type Link struct {
	Text string `json:"text"`
	Path string `json:"path"`
	Line int    `json:"line"`
}
