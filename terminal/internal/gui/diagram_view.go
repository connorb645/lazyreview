package gui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	model "github.com/connorb645/lazyreview/terminal/internal/diagram"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation/diagram"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/gui/views"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

type diagramView struct {
	diagram.Scene
	left  int
	top   int
	query string
}

const (
	cardGap  = 16
	rowLines = model.CardHeight + 2
)

func newDiagramView(tr *i18n.TranslationSet, level model.Level, focusID string) diagramView {
	v := diagramView{Scene: diagram.Scene{Level: level, Grid: model.Arrange(tr, level.Cards, cardGap)}}
	if i := level.Index(focusID); i >= 0 {
		v.Focus = i
	} else if order := v.Grid.Order(); len(order) > 0 {
		v.Focus = order[0]
	}
	return v
}

func levelAt(tr *i18n.TranslationSet, page server.Page, at types.DiagramPlace) model.Level {
	switch at.Level {
	case "types":
		return model.Types(tr, page, at.Component, at.Package)
	case "packages":
		l, _ := model.Packages(tr, page, at.Component)
		return l
	}
	return model.Components(tr, page)
}

func (v diagramView) place() types.DiagramPlace {
	at := types.DiagramPlace{Level: v.Level.Kind, Component: v.Level.Component, Package: v.Level.Package}
	if card, ok := v.Card(); ok {
		at.Box = card.ID
	}
	return at
}

func (v diagramView) moved(by int) diagramView {
	order := v.Grid.Order()
	for i, at := range order {
		if at == v.Focus {
			v.Focus = order[utils.Clamp(i+by, 0, len(order)-1)]
			return v
		}
	}
	return v
}

func (v diagramView) lines(tr *i18n.TranslationSet, d types.ReviewData, width, height int, filter types.PullFilter, styles map[string]lipgloss.Style) ([]string, views.Extent) {
	c, _, _ := diagram.Draw(tr, d, v.Scene, filter)
	return c.Lines(v.left, v.top, width, height, styles), views.Extent{Lines: c.H, Top: v.top}
}

func (v diagramView) boxAt(tr *i18n.TranslationSet, d types.ReviewData, x, y int, filter types.PullFilter) int {
	_, tops, _ := diagram.Draw(tr, d, v.Scene, filter)
	x, y = x+v.left, y+v.top
	for i, s := range v.Grid.Spots {
		if x >= s.Col && x < s.Col+s.Width && y >= tops[s.Row] && y < tops[s.Row]+model.CardHeight {
			return i
		}
	}
	return -1
}

func (v diagramView) scrolled(tr *i18n.TranslationSet, d types.ReviewData, across, by, width, height int, filter types.PullFilter) diagramView {
	c, _, _ := diagram.Draw(tr, d, v.Scene, filter)
	v.left = utils.Clamp(v.left+across, 0, max(0, c.W-width))
	v.top = utils.Clamp(v.top+by, 0, max(0, c.H-height))
	return v
}

func (v diagramView) shownIn(tr *i18n.TranslationSet, d types.ReviewData, width, height int, filter types.PullFilter) diagramView {
	if _, ok := v.Card(); !ok {
		return v
	}
	c, tops, below := diagram.Draw(tr, d, v.Scene, filter)
	s := v.Grid.Spots[v.Focus]
	x0, x1 := s.Col-2, s.Col+s.Width+2
	y0, y1 := tops[s.Row]-1, tops[s.Row]+model.CardHeight+below.Lines+1
	if s.Row == len(tops)-1 {
		y1 = c.H
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
