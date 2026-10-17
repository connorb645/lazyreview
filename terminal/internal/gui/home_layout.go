package gui

import (
	"github.com/connorb645/lazyreview/terminal/internal/gui/views"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

const listWidth = 50

type homeLayout struct {
	panels []views.Rect
	main   views.Rect
	bar    views.Rect
}

func layHome(width, height, focus int, mode screenMode, wants []int) homeLayout {
	l := homeLayout{bar: views.Rect{Y: height - 1, W: width, H: 1}, panels: make([]views.Rect, len(wants))}
	h := height - 1
	side := width
	if mode != fullScreen && width >= narrowest && h >= shortest {
		side = max(width/3, min(width/2, listWidth))
		if mode == halfScreen {
			side = width * 2 / 3
		}
		l.main = views.Rect{X: side, W: width - side, H: h}
	}
	first, shown := 0, len(wants)
	if fits := max(1, h-2); shown > fits {
		shown = fits
		first = utils.Clamp(focus-shown/2, 0, len(wants)-shown)
	}
	y := 0
	for i, n := range shared(h, focus-first, wants[first:first+shown], 3) {
		l.panels[first+i] = views.Rect{Y: y, W: side, H: n}
		y += n
	}
	return l
}
