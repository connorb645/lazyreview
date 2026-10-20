package views

import "github.com/connorb645/lazyreview/internal/utils"

// Extent is how many lines a view holds, and the first one shown.
type Extent struct {
	Lines int
	Top   int
}

// Thumb is where the scroll thumb sits on a frame's side of page lines, and
// how long it is, as lazygit sizes and places it; none when all fits.
func (e Extent) Thumb(page int) (start, size int) {
	if page < 2 || page >= e.Lines {
		return 0, 0
	}
	last, area := e.Lines-page, page-1
	size = page * area / e.Lines
	if top := utils.Clamp(e.Top, 0, last); top < last {
		return (top*(area-size-1) + last - 1) / last, size + 1
	}
	return area - size, size + 1
}
