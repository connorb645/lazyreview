package helpers

import (
	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/views"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

const (
	narrowest = 60
	shortest  = 10
	listWidth = 50
)

// WindowArrangementHelper arranges the windows on the screen, as lazygit's
// does: the panels on the left, the main panel and the bar, by the screen's
// size and mode and what each panel holds.
type WindowArrangementHelper struct {
	c *HelperCommon
}

// Layout is where a review's windows sit.
func (self *WindowArrangementHelper) Layout(s context.ContextTree) Layout {
	var wants [context.SideCount + 1]int
	for n := context.ReviewPanel; n <= context.SideCount; n++ {
		wants[n] = max(1, len(s.Side(n).Rows))
	}
	return lay(s.Width, s.Height, s.Focus(), s.Driving(), s.Mode, wants)
}

// HomeLayout is where the home screen's windows sit.
func (self *WindowArrangementHelper) HomeLayout(s context.ContextTree) HomeLayout {
	return layHome(s.Width, s.Height, s.Home.Focus, s.Mode, s.Home.Wants())
}

// MainRect is the inside of the main panel, as it is when the main panel
// has the keys, if it has no room otherwise.
func (self *WindowArrangementHelper) MainRect(s context.ContextTree) views.Rect {
	if r := self.Layout(s).Main; r.Shown() {
		return r.Inner()
	}
	s = s.Focused(context.MainPanel, s.Driving())
	return self.Layout(s).Main.Inner()
}

// MainHeight is how many lines the main panel shows, at least one.
func (self *WindowArrangementHelper) MainHeight(s context.ContextTree) int {
	return max(1, self.MainRect(s).H)
}

// MainWidth is how many columns the main panel shows, at least one.
func (self *WindowArrangementHelper) MainWidth(s context.ContextTree) int {
	return max(1, self.MainRect(s).W)
}

// SideHeight is how many rows a panel on the left shows, at least one.
func (self *WindowArrangementHelper) SideHeight(s context.ContextTree, n int) int {
	return max(1, self.Layout(s).Sides[n].Inner().H)
}

// MenuRows is how many rows a menu shows at most.
func (self *WindowArrangementHelper) MenuRows(s context.ContextTree) int {
	return max(1, min(s.Height*2/3, 24)-2)
}

// PopupWidth is how wide posting's popup is.
func (self *WindowArrangementHelper) PopupWidth(s context.ContextTree) int {
	return min(s.Width-4, max(s.Width*3/5, 60))
}

func lay(width, height, focus, driving int, mode context.ScreenMode, wants [context.SideCount + 1]int) Layout {
	l := Layout{Bar: views.Rect{Y: height - 1, W: width, H: 1}}
	h := height - 1
	if mode == context.FullScreen || width < narrowest || h < shortest {
		whole := views.Rect{W: width, H: h}
		if focus == context.MainPanel {
			l.Main = whole
		} else {
			l.Sides[focus] = whole
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
	l.Main = views.Rect{X: side, W: width - side, H: h}
	first := min(wants[context.ReviewPanel]+2, max(3, (h+3)/4))
	l.Sides[context.ReviewPanel] = views.Rect{W: side, H: first}
	heights := shared(h-first, driving-context.FilesPanel, []int{wants[context.FilesPanel] + 2, wants[context.NotesPanel] + 2, wants[context.DiagramPanel] + 2}, 6)
	y := first
	for i, n := range heights {
		l.Sides[context.FilesPanel+i] = views.Rect{Y: y, W: side, H: n}
		y += n
	}
	return l
}

func layHome(width, height, focus int, mode context.ScreenMode, wants []int) HomeLayout {
	l := HomeLayout{Bar: views.Rect{Y: height - 1, W: width, H: 1}, Panels: make([]views.Rect, len(wants))}
	h := height - 1
	side := width
	if mode != context.FullScreen && width >= narrowest && h >= shortest {
		side = max(width/3, min(width/2, listWidth))
		if mode == context.HalfScreen {
			side = width * 2 / 3
		}
		l.Main = views.Rect{X: side, W: width - side, H: h}
	}
	first, shown := 0, len(wants)
	if fits := max(1, h-2); shown > fits {
		shown = fits
		first = utils.Clamp(focus-shown/2, 0, len(wants)-shown)
	}
	y := 0
	for i, n := range shared(h, focus-first, wants[first:first+shown], 3) {
		l.Panels[first+i] = views.Rect{Y: y, W: side, H: n}
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
