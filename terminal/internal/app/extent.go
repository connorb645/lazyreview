package app

type extent struct {
	lines int
	top   int
}

func (e extent) thumb(page int) (start, size int) {
	if page < 2 || page >= e.lines {
		return 0, 0
	}
	last, area := e.lines-page, page-1
	size = page * area / e.lines
	if top := clamp(e.top, 0, last); top < last {
		return (top*(area-size-1) + last - 1) / last, size + 1
	}
	return area - size, size + 1
}
