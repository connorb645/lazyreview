package models

// Rendered is a Markdown file's Rendered view as blocks to draw one at a
// time. Partial says the page holds only the file's diff, so the blocks
// are only its changed parts.
type Rendered struct {
	Blocks  []Block `json:"blocks"`
	Partial bool    `json:"partial"`
}
