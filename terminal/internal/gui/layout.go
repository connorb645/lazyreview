package gui

import (
	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/views"
)

const (
	narrowest = 60
	shortest  = 10
)

type layout struct {
	sides [context.SideCount + 1]views.Rect
	main  views.Rect
	bar   views.Rect
}

func lay(width, height, focus, driving int, mode context.ScreenMode, wants [context.SideCount + 1]int) layout {
	l := layout{bar: views.Rect{Y: height - 1, W: width, H: 1}}
	h := height - 1
	if mode == context.FullScreen || width < narrowest || h < shortest {
		whole := views.Rect{W: width, H: h}
		if focus == context.MainPanel {
			l.main = whole
		} else {
			l.sides[focus] = whole
		}
		return l
	}
	side := max(width/3, 24)
	switch {
	case mode == context.HalfScreen && focus == context.MainPanel:
		side = max(width/5, 16)
	case mode == context.HalfScreen:
		side = width / 2
	}
	l.main = views.Rect{X: side, W: width - side, H: h}
	first := min(wants[context.ReviewPanel]+2, max(3, h/4))
	l.sides[context.ReviewPanel] = views.Rect{W: side, H: first}
	heights := shared(h-first, driving-context.FilesPanel, []int{wants[context.FilesPanel] + 2, wants[context.NotesPanel] + 2, wants[context.DiagramPanel] + 2}, 6)
	y := first
	for i, n := range heights {
		l.sides[context.FilesPanel+i] = views.Rect{Y: y, W: side, H: n}
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
