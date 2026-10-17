package gui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/views"
)

func (a App) mouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if a.Screen() == context.ScreenHome {
		return a.homeMouse(msg)
	}
	if msg.Action != tea.MouseActionPress || a.Review == nil {
		return a, nil
	}
	l := a.layout()
	n, at := -1, views.Rect{}
	for i := context.ReviewPanel; i <= context.SideCount; i++ {
		if r := l.sides[i]; r.Shown() && r.Holds(msg.X, msg.Y) {
			n, at = i, r.Inner()
		}
	}
	if r := l.main; r.Shown() && r.Holds(msg.X, msg.Y) {
		n, at = context.MainPanel, r.Inner()
	}
	if n < 0 {
		return a, nil
	}
	x, y := msg.X-at.X, msg.Y-at.Y
	switch msg.Button {
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
		by := 3
		if msg.Button == tea.MouseButtonWheelUp {
			by = -3
		}
		if n != context.MainPanel {
			a.ContextTree = a.WithSide(n, a.Side(n).Scroll(by, at.H))
			return a, nil
		}
		switch a.Screen() {
		case context.ScreenReview:
			a.ReviewView.View = a.ReviewView.View.Scroll(by, at.H)
		case context.ScreenDiff:
			a.Diff = a.Diff.Scroll(by, at.H)
		case context.ScreenDiagram:
			a.Diagram = a.Diagram.Scrolled(a.c.Tr, *a.Review, 0, by, at.W, at.H, a.Filter)
		}
		return a, nil
	case tea.MouseButtonWheelLeft, tea.MouseButtonWheelRight:
		if n == context.MainPanel && a.Screen() == context.ScreenDiagram {
			by := 6
			if msg.Button == tea.MouseButtonWheelLeft {
				by = -6
			}
			a.Diagram = a.Diagram.Scrolled(a.c.Tr, *a.Review, by, 0, at.W, at.H, a.Filter)
		}
		return a, nil
	case tea.MouseButtonLeft:
		if y < 0 || y >= at.H || x < 0 || x >= at.W {
			return a, nil
		}
		now := a.opts.Now()
		double := msg.Y == a.ClickedY && now.Sub(a.ClickedAt) < doubleClick
		a.ClickedAt, a.ClickedY = now, msg.Y
		if n != context.MainPanel {
			return a.sideClicked(n, y, double)
		}
		a.ContextTree = a.Focused(context.MainPanel, a.Driving())
		return a.mainClicked(x, y, double)
	}
	return a, nil
}

func (a App) sideClicked(n, y int, double bool) (tea.Model, tea.Cmd) {
	i := a.Side(n).At(y)
	a.ContextTree = a.Focused(n, n)
	if i < 0 {
		return a.relaid(), nil
	}
	a.ContextTree = a.WithSide(n, a.Side(n).To(i, a.sideHeight(n)))
	next, cmd := a.relaid().followed()
	if !double {
		return next, cmd
	}
	after, more := next.(App).sideAction("open", 1)
	return after, tea.Batch(cmd, more)
}

func (a App) mainClicked(x, y int, double bool) (tea.Model, tea.Cmd) {
	h := a.mainHeight()
	switch a.Screen() {
	case context.ScreenDiagram:
		if i := a.Diagram.BoxAt(a.c.Tr, *a.Review, x, y, a.Filter); i >= 0 {
			a.Diagram.Focus = i
			a = a.diagramShown()
			if double {
				return a.stepIn()
			}
		}
		return a.mirrored(), nil
	case context.ScreenDiff:
		if i := a.Diff.RowAt(y); i >= 0 {
			a.Diff = a.Diff.To(i, h)
			if double {
				return a.diffAction("open", 1, false, h)
			}
		}
		return a.mirrored(), nil
	}
	if i := a.ReviewView.View.At(y); i >= 0 {
		a.ReviewView.View = a.ReviewView.View.To(i, h)
		if double {
			return a.reviewAction("open", 1, h)
		}
	}
	return a.mirrored(), nil
}
