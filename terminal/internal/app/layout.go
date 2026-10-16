package app

const (
	mainPanel = iota
	reviewPanel
	filesPanel
	notesPanel
	diagramPanel
	sideCount = diagramPanel
)

const (
	narrowest = 60
	shortest  = 10
)

type layout struct {
	sides [sideCount + 1]rect
	main  rect
	bar   rect
}

func lay(width, height, focus, driving int, mode screenMode, wants [sideCount + 1]int) layout {
	l := layout{bar: rect{y: height - 1, w: width, h: 1}}
	h := height - 1
	if mode == fullScreen || width < narrowest || h < shortest {
		whole := rect{w: width, h: h}
		if focus == mainPanel {
			l.main = whole
		} else {
			l.sides[focus] = whole
		}
		return l
	}
	side := width / 3
	switch {
	case mode == halfScreen && focus == mainPanel:
		side = width / 5
	case mode == halfScreen:
		side = width / 2
	}
	side = max(side, 24)
	l.main = rect{x: side, w: width - side, h: h}
	first := min(wants[reviewPanel]+2, max(3, h/4))
	l.sides[reviewPanel] = rect{w: side, h: first}
	heights := shared(h-first, driving-filesPanel, [3]int{wants[filesPanel] + 2, wants[notesPanel] + 2, wants[diagramPanel] + 2})
	y := first
	for i, n := range heights {
		l.sides[filesPanel+i] = rect{y: y, w: side, h: n}
		y += n
	}
	return l
}

func shared(rest, grow int, wants [3]int) [3]int {
	var out [3]int
	fair := rest / 3
	if fair < 6 {
		for i := range out {
			out[i] = 3
		}
		if grow < 0 || grow > 2 {
			grow = 0
		}
		out[grow] = rest - 6
		return out
	}
	left := rest
	for i := range out {
		out[i] = max(3, min(wants[i], fair))
		left -= out[i]
	}
	order := []int{0, 1, 2}
	if grow >= 0 && grow < 3 {
		order = append([]int{grow}, append(order[:grow:grow], order[grow+1:]...)...)
	}
	for _, i := range order {
		more := min(left, max(0, wants[i]-out[i]))
		out[i] += more
		left -= more
	}
	for i := range out {
		out[i] += left / 3
	}
	out[order[0]] += left % 3
	return out
}
