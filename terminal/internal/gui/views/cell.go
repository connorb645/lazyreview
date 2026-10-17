package views

// Cell is one cell of a Canvas: its rune, the name of its style, and the
// directions a line through it goes.
type Cell struct {
	R     rune
	Style string
	Dirs  int
}
