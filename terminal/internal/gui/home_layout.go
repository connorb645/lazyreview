package gui

const listWidth = 50

type homeLayout struct {
	panels []rect
	main   rect
	bar    rect
}

func layHome(width, height, focus int, mode screenMode, wants []int) homeLayout {
	l := homeLayout{bar: rect{y: height - 1, w: width, h: 1}, panels: make([]rect, len(wants))}
	h := height - 1
	side := width
	if mode != fullScreen && width >= narrowest && h >= shortest {
		side = max(width/3, min(width/2, listWidth))
		if mode == halfScreen {
			side = width * 2 / 3
		}
		l.main = rect{x: side, w: width - side, h: h}
	}
	first, shown := 0, len(wants)
	if fits := max(1, h-2); shown > fits {
		shown = fits
		first = clamp(focus-shown/2, 0, len(wants)-shown)
	}
	y := 0
	for i, n := range shared(h, focus-first, wants[first:first+shown], 3) {
		l.panels[first+i] = rect{y: y, w: side, h: n}
		y += n
	}
	return l
}
