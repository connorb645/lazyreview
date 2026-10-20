package models

// File is one file of a review ready to draw: its diff hunk by hunk, the
// runs of unchanged lines between the hunks, its whole text at the head
// when the page kept it, and for Markdown its Rendered view.
type File struct {
	Path     string    `json:"path"`
	Hunks    []Hunk    `json:"hunks"`
	Gaps     []Gap     `json:"gaps"`
	Whole    []string  `json:"whole"`
	Rendered *Rendered `json:"rendered"`
}
