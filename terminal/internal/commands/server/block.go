package server

// Block is one block of a Markdown file's Rendered view: its source, ready
// to draw alone, whether it was added, removed or is the same, the side
// and source lines it was read from, an unchanged block's lines before the
// change, and for a partial view the hunk it comes from.
type Block struct {
	State     string `json:"state"`
	Markdown  string `json:"markdown"`
	Side      string `json:"side"`
	StartLine int    `json:"start_line"`
	Line      int    `json:"line"`
	Left      *Span  `json:"left"`
	Part      int    `json:"part"`
}
