package app

type rect struct {
	x, y, w, h int
}

func (r rect) holds(x, y int) bool {
	return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h
}

func (r rect) inner() rect {
	return rect{x: r.x + 1, y: r.y + 1, w: max(0, r.w-2), h: max(0, r.h-2)}
}

func (r rect) shown() bool {
	return r.w > 2 && r.h > 2
}
