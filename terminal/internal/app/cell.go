package app

// cell is one character of the diagram's canvas: its rune, the name of its
// style, and for a line the directions it runs in.
type cell struct {
	r     rune
	style string
	dirs  int
}
