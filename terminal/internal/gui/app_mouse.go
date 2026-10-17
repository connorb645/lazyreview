package gui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/views"
)

func (a App) mouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if a.screen == screenHome {
		return a.homeMouse(msg)
	}
	if msg.Action != tea.MouseActionPress || a.review == nil {
		return a, nil
	}
	l := a.layout()
	n, at := -1, views.Rect{}
	for i := reviewPanel; i <= sideCount; i++ {
		if r := l.sides[i]; r.Shown() && r.Holds(msg.X, msg.Y) {
			n, at = i, r.Inner()
		}
	}
	if r := l.main; r.Shown() && r.Holds(msg.X, msg.Y) {
		n, at = mainPanel, r.Inner()
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
		if n != mainPanel {
			a.sides[n] = a.sides[n].scroll(by, at.H)
			return a, nil
		}
		switch a.screen {
		case screenReview:
			a.rv.view = a.rv.view.scroll(by, at.H)
		case screenDiff:
			a.diff = a.diff.scroll(by, at.H)
		case screenDiagram:
			a.dg = a.dg.scrolled(a.c.Tr, *a.review, 0, by, at.W, at.H, a.filter)
		}
		return a, nil
	case tea.MouseButtonWheelLeft, tea.MouseButtonWheelRight:
		if n == mainPanel && a.screen == screenDiagram {
			by := 6
			if msg.Button == tea.MouseButtonWheelLeft {
				by = -6
			}
			a.dg = a.dg.scrolled(a.c.Tr, *a.review, by, 0, at.W, at.H, a.filter)
		}
		return a, nil
	case tea.MouseButtonLeft:
		if y < 0 || y >= at.H || x < 0 || x >= at.W {
			return a, nil
		}
		now := a.opts.Now()
		double := msg.Y == a.clickedY && now.Sub(a.clickedAt) < doubleClick
		a.clickedAt, a.clickedY = now, msg.Y
		if n != mainPanel {
			return a.sideClicked(n, y, double)
		}
		a.focus = mainPanel
		return a.mainClicked(x, y, double)
	}
	return a, nil
}

func (a App) sideClicked(n, y int, double bool) (tea.Model, tea.Cmd) {
	i := a.sides[n].at(y)
	a.focus, a.driving = n, n
	if i < 0 {
		return a.relaid(), nil
	}
	a.sides[n] = a.sides[n].to(i, a.sideHeight(n))
	next, cmd := a.relaid().followed()
	if !double {
		return next, cmd
	}
	after, more := next.(App).sideAction("open", 1)
	return after, tea.Batch(cmd, more)
}

func (a App) mainClicked(x, y int, double bool) (tea.Model, tea.Cmd) {
	h := a.mainHeight()
	switch a.screen {
	case screenDiagram:
		if i := a.dg.boxAt(a.c.Tr, *a.review, x, y, a.filter); i >= 0 {
			a.dg.Focus = i
			a = a.diagramShown()
			if double {
				return a.stepIn()
			}
		}
		return a.mirrored(), nil
	case screenDiff:
		if i := a.diff.rowAt(y); i >= 0 {
			a.diff = a.diff.to(i, h)
			if double {
				return a.diffAction("open", 1, false, h)
			}
		}
		return a.mirrored(), nil
	}
	if i := a.rv.view.at(y); i >= 0 {
		a.rv.view = a.rv.view.to(i, h)
		if double {
			return a.reviewAction("open", 1, h)
		}
	}
	return a.mirrored(), nil
}
