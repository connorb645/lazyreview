package views

// Rect is a rectangle of the screen, from its top left cell.
type Rect struct {
	X, Y, W, H int
}

// Holds says whether a cell is in the rectangle.
func (r Rect) Holds(x, y int) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

// Inner is the rectangle inside a frame drawn round this one.
func (r Rect) Inner() Rect {
	return Rect{X: r.X + 1, Y: r.Y + 1, W: max(0, r.W-2), H: max(0, r.H-2)}
}

// Shown says whether the rectangle has room for a frame and a line in it.
func (r Rect) Shown() bool {
	return r.W > 2 && r.H > 2
}
