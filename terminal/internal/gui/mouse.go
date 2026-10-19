package gui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/views"
)

const doubleClick = 400 * time.Millisecond

func (a App) mouse(s context.ContextTree, msg tea.MouseMsg) (context.ContextTree, tea.Cmd) {
	if s.Screen() == context.ScreenHome {
		return a.homeMouse(s, msg)
	}
	if msg.Action != tea.MouseActionPress || s.Review == nil {
		return s, nil
	}
	l := a.helpers.Window.Layout(s)
	n, at := -1, views.Rect{}
	for i := context.ReviewPanel; i <= context.SideCount; i++ {
		if r := l.Sides[i]; r.Shown() && r.Holds(msg.X, msg.Y) {
			n, at = i, r.Inner()
		}
	}
	if r := l.Main; r.Shown() && r.Holds(msg.X, msg.Y) {
		n, at = context.MainPanel, r.Inner()
	}
	if n < 0 {
		return s, nil
	}
	x, y := msg.X-at.X, msg.Y-at.Y
	switch msg.Button {
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
		by := 3
		if msg.Button == tea.MouseButtonWheelUp {
			by = -3
		}
		if msg.Shift && n == context.MainPanel && s.Screen() == context.ScreenDiff {
			return a.helpers.Diff.Sideways(s, 2*by), nil
		}
		if n != context.MainPanel {
			return s.WithSide(n, s.Side(n).Scroll(by, at.H)), nil
		}
		switch s.Screen() {
		case context.ScreenReview:
			s.ReviewView.View = s.ReviewView.View.Scroll(by, at.H)
		case context.ScreenDiff:
			s.Diff = s.Diff.Scroll(by, at.H)
		case context.ScreenDiagram:
			s.Diagram = s.Diagram.Scrolled(a.c.Tr, *s.Review, 0, by, at.W, at.H, s.ShownPulls())
		}
		return s, nil
	case tea.MouseButtonWheelLeft, tea.MouseButtonWheelRight:
		by := 6
		if msg.Button == tea.MouseButtonWheelLeft {
			by = -6
		}
		switch {
		case n == context.MainPanel && s.Screen() == context.ScreenDiagram:
			s.Diagram = s.Diagram.Scrolled(a.c.Tr, *s.Review, by, 0, at.W, at.H, s.ShownPulls())
		case n == context.MainPanel && s.Screen() == context.ScreenDiff:
			s = a.helpers.Diff.Sideways(s, by)
		}
		return s, nil
	case tea.MouseButtonLeft:
		if y < 0 || y >= at.H || x < 0 || x >= at.W {
			return s, nil
		}
		now := a.c.Now()
		double := msg.Y == s.ClickedY && now.Sub(s.ClickedAt) < doubleClick
		s.ClickedAt, s.ClickedY = now, msg.Y
		if n != context.MainPanel {
			return a.sideClicked(s, n, y, double)
		}
		s = s.Focused(context.MainPanel, s.Driving())
		return a.mainClicked(s, x, y, double)
	}
	return s, nil
}

func (a App) sideClicked(s context.ContextTree, n, y int, double bool) (context.ContextTree, tea.Cmd) {
	h := a.helpers
	i := s.Side(n).At(y)
	s = s.Focused(n, n)
	if i < 0 {
		return h.Refresh.Relaid(s), nil
	}
	s = s.WithSide(n, s.Side(n).To(i, h.Window.SideHeight(s, n)))
	s, cmd := h.Panels.Followed(h.Refresh.Relaid(h.Files.FromCursor(h.Filter.FromCursor(s))))
	if !double {
		return s, cmd
	}
	s, more := a.opened(s)
	return s, tea.Batch(cmd, more)
}

func (a App) mainClicked(s context.ContextTree, x, y int, double bool) (context.ContextTree, tea.Cmd) {
	h := a.helpers
	height := h.Window.MainHeight(s)
	switch s.Screen() {
	case context.ScreenDiagram:
		if i := s.Diagram.BoxAt(a.c.Tr, *s.Review, x, y, s.ShownPulls()); i >= 0 {
			s.Diagram = s.Diagram.FocusedOn(i)
			s = h.Diagram.Shown(s)
			if double {
				return a.opened(s)
			}
		}
		return h.Refresh.Mirrored(s), nil
	case context.ScreenDiff:
		if i := s.Diff.RowAt(y); i >= 0 {
			s.Diff = s.Diff.To(i, height)
			if double {
				return a.opened(s)
			}
		}
		return h.Refresh.Mirrored(s), nil
	}
	if i := s.ReviewView.View.At(y); i >= 0 {
		s.ReviewView.View = s.ReviewView.View.To(i, height)
		if double {
			return a.opened(s)
		}
	}
	return h.Refresh.Mirrored(s), nil
}

func (a App) homeMouse(s context.ContextTree, msg tea.MouseMsg) (context.ContextTree, tea.Cmd) {
	if msg.Action != tea.MouseActionPress {
		return s, nil
	}
	h := a.helpers
	for i, r := range h.Window.HomeLayout(s).Panels {
		if r.W == 0 || !r.Holds(msg.X, msg.Y) {
			continue
		}
		in := r.Inner()
		switch msg.Button {
		case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
			by := 3
			if msg.Button == tea.MouseButtonWheelUp {
				by = -3
			}
			panels := append([]context.HomePanel{}, s.Home.Panels...)
			panels[i].Rows = panels[i].Rows.Scroll(by, max(1, in.H))
			s.Home.Panels = panels
			return s, nil
		case tea.MouseButtonLeft:
			now := a.c.Now()
			double := msg.Y == s.ClickedY && now.Sub(s.ClickedAt) < doubleClick
			s.ClickedAt, s.ClickedY = now, msg.Y
			row := s.Home.Panels[i].Rows.At(msg.Y - in.Y)
			if row < 0 {
				s.Home.Focus = i
				return h.Refresh.HomeRelaid(s), nil
			}
			s.Home = s.Home.At(i, row)
			if double {
				return h.Home.OpenSelected(h.Refresh.HomeRelaid(s))
			}
			return h.Refresh.HomeRelaid(s), nil
		}
	}
	return s, nil
}
