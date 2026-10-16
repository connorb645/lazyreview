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
	side := max(width/3, 24)
	switch {
	case mode == halfScreen && focus == mainPanel:
		side = max(width/5, 16)
	case mode == halfScreen:
		side = width / 2
	}
	l.main = rect{x: side, w: width - side, h: h}
	first := min(wants[reviewPanel]+2, max(3, h/4))
	l.sides[reviewPanel] = rect{w: side, h: first}
	heights := shared(h-first, driving-filesPanel, []int{wants[filesPanel] + 2, wants[notesPanel] + 2, wants[diagramPanel] + 2}, 6)
	y := first
	for i, n := range heights {
		l.sides[filesPanel+i] = rect{y: y, w: side, h: n}
		y += n
	}
	return l
}

func shared(rest, grow int, wants []int, fold int) []int {
	out := make([]int, len(wants))
	if len(out) == 0 {
		return out
	}
	if grow < 0 || grow >= len(out) {
		grow = 0
	}
	order := append([]int{grow}, others(len(out), grow)...)
	total := 0
	for _, w := range wants {
		total += w
	}
	if total <= rest {
		copy(out, wants)
		spread := rest - total
		for i := range out {
			out[i] += spread / len(out)
		}
		out[len(out)-1] += spread % len(out)
		return out
	}
	fair := rest / len(out)
	if fair < fold {
		squashed := 3
		if rest-squashed*(len(out)-1) < squashed {
			squashed = 1
		}
		for i := range out {
			out[i] = squashed
		}
		out[grow] = rest - squashed*(len(out)-1)
		return out
	}
	left := rest
	for i := range out {
		out[i] = max(3, min(wants[i], fair))
		left -= out[i]
	}
	for _, i := range order {
		more := min(left, max(0, wants[i]-out[i]))
		out[i] += more
		left -= more
	}
	for i := range out {
		out[i] += left / len(out)
	}
	out[grow] += left % len(out)
	return out
}

func others(n, but int) []int {
	var out []int
	for i := 0; i < n; i++ {
		if i != but {
			out = append(out, i)
		}
	}
	return out
}
