package gui

import tea "github.com/charmbracelet/bubbletea"

func (a App) mouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if a.screen == screenHome {
		return a.homeMouse(msg)
	}
	if msg.Action != tea.MouseActionPress || a.review == nil {
		return a, nil
	}
	l := a.layout()
	n, at := -1, rect{}
	for i := reviewPanel; i <= sideCount; i++ {
		if r := l.sides[i]; r.shown() && r.holds(msg.X, msg.Y) {
			n, at = i, r.inner()
		}
	}
	if r := l.main; r.shown() && r.holds(msg.X, msg.Y) {
		n, at = mainPanel, r.inner()
	}
	if n < 0 {
		return a, nil
	}
	x, y := msg.X-at.x, msg.Y-at.y
	switch msg.Button {
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
		by := 3
		if msg.Button == tea.MouseButtonWheelUp {
			by = -3
		}
		if n != mainPanel {
			a.sides[n] = a.sides[n].scroll(by, at.h)
			return a, nil
		}
		switch a.screen {
		case screenReview:
			a.rv.view = a.rv.view.scroll(by, at.h)
		case screenDiff:
			a.diff = a.diff.scroll(by, at.h)
		case screenDiagram:
			a.dg = a.dg.scrolled(a.c.Tr, *a.review, 0, by, at.w, at.h, a.filter)
		}
		return a, nil
	case tea.MouseButtonWheelLeft, tea.MouseButtonWheelRight:
		if n == mainPanel && a.screen == screenDiagram {
			by := 6
			if msg.Button == tea.MouseButtonWheelLeft {
				by = -6
			}
			a.dg = a.dg.scrolled(a.c.Tr, *a.review, by, 0, at.w, at.h, a.filter)
		}
		return a, nil
	case tea.MouseButtonLeft:
		if y < 0 || y >= at.h || x < 0 || x >= at.w {
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
			a.dg.focus = i
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
