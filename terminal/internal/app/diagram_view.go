package app

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/connorb645/lazyreview/terminal/internal/diagram"
	"github.com/connorb645/lazyreview/terminal/internal/server"
)

type diagramView struct {
	level  diagram.Level
	grid   diagram.Grid
	focus  int
	left   int
	top    int
	all    bool
	query  string
	labels map[string]int
	typed  string
}

const (
	cardGap  = 16
	rowLines = diagram.CardHeight + 2
)

func newDiagramView(level diagram.Level, focusID string) diagramView {
	v := diagramView{level: level, grid: diagram.Arrange(level.Cards, cardGap)}
	if i := level.Index(focusID); i >= 0 {
		v.focus = i
	} else if order := v.grid.Order(); len(order) > 0 {
		v.focus = order[0]
	}
	return v
}

func levelAt(page server.Page, at diagramPlace) diagram.Level {
	switch at.level {
	case "types":
		return diagram.Types(page, at.component, at.pkg)
	case "packages":
		l, _ := diagram.Packages(page, at.component)
		return l
	}
	return diagram.Components(page)
}

func (v diagramView) card() (diagram.Card, bool) {
	if v.focus < 0 || v.focus >= len(v.level.Cards) {
		return diagram.Card{}, false
	}
	return v.level.Cards[v.focus], true
}

func (v diagramView) place() diagramPlace {
	at := diagramPlace{level: v.level.Kind, component: v.level.Component, pkg: v.level.Package}
	if card, ok := v.card(); ok {
		at.box = card.ID
	}
	return at
}

func (v diagramView) moved(by int) diagramView {
	order := v.grid.Order()
	for i, at := range order {
		if at == v.focus {
			v.focus = order[clamp(i+by, 0, len(order)-1)]
			return v
		}
	}
	return v
}

func (v diagramView) lines(d reviewData, width, height, filter int, styles map[string]lipgloss.Style) ([]string, extent) {
	c, _, _ := drawDiagram(d, v, filter)
	return c.lines(v.left, v.top, width, height, styles), extent{lines: c.h, top: v.top}
}

func (v diagramView) boxAt(d reviewData, x, y, filter int) int {
	_, tops, _ := drawDiagram(d, v, filter)
	x, y = x+v.left, y+v.top
	for i, s := range v.grid.Spots {
		if x >= s.Col && x < s.Col+s.Width && y >= tops[s.Row] && y < tops[s.Row]+diagram.CardHeight {
			return i
		}
	}
	return -1
}

func (v diagramView) scrolled(d reviewData, across, by, width, height, filter int) diagramView {
	c, _, _ := drawDiagram(d, v, filter)
	v.left = clamp(v.left+across, 0, max(0, c.w-width))
	v.top = clamp(v.top+by, 0, max(0, c.h-height))
	return v
}

func (v diagramView) shownIn(d reviewData, width, height, filter int) diagramView {
	if _, ok := v.card(); !ok {
		return v
	}
	c, tops, below := drawDiagram(d, v, filter)
	s := v.grid.Spots[v.focus]
	x0, x1 := s.Col-2, s.Col+s.Width+2
	y0, y1 := tops[s.Row]-1, tops[s.Row]+diagram.CardHeight+below.lines+1
	if s.Row == len(tops)-1 {
		y1 = c.h
	}
	if x1-v.left > width {
		v.left = x1 - width
	}
	if x0 < v.left {
		v.left = max(0, x0)
	}
	if y1-v.top > height {
		v.top = y1 - height
	}
	if y0 < v.top {
		v.top = max(0, y0)
	}
	return v
}
