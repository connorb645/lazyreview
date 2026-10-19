package context

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

// DiagramContext is the diagram in the main panel: the level drawn, the
// place in its row that moving down and up keeps, how far it is scrolled,
// and what was last searched for in it.
type DiagramContext struct {
	diagram.Scene
	Item  int
	Left  int
	Top   int
	Query string
}

const cardGap = 16

// NewDiagramContext is a level of the diagram arranged, focused on a box,
// or on the first in reading order.
func NewDiagramContext(tr *i18n.TranslationSet, level model.Level, focusID string) DiagramContext {
	v := DiagramContext{Scene: diagram.Scene{Level: level, Grid: model.Arrange(tr, level.Cards, cardGap)}}
	if i := level.Index(focusID); i >= 0 {
		v = v.FocusedOn(i)
	} else if order := v.Grid.Order(); len(order) > 0 {
		v = v.FocusedOn(order[0])
	}
	return v
}

// LevelAt is the level of the diagram a place is on.
func LevelAt(tr *i18n.TranslationSet, page server.Page, at types.DiagramPlace) model.Level {
	switch at.Level {
	case "types":
		return model.Types(tr, page, at.Component, at.Package)
	case "packages":
		l, _ := model.Packages(tr, page, at.Component)
		return l
	}
	return model.Components(tr, page)
}

// Place is the box in focus, and the level it is on.
func (v DiagramContext) Place() types.DiagramPlace {
	at := types.DiagramPlace{Level: v.Level.Kind, Component: v.Level.Component, Package: v.Level.Package}
	if card, ok := v.Card(); ok {
		at.Box = card.ID
	}
	return at
}

// FocusedOn is the diagram with the focus on box i. Its place in its row is
// the one moving down and up then keeps.
func (v DiagramContext) FocusedOn(i int) DiagramContext {
	v.Focus, v.Item = i, v.Grid.Item(i)
	return v
}

// Moved is the diagram with the focus moved by boxes in reading order.
func (v DiagramContext) Moved(by int) DiagramContext {
	order := v.Grid.Order()
	for i, at := range order {
		if at == v.Focus {
			return v.FocusedOn(order[utils.Clamp(i+by, 0, len(order)-1)])
		}
	}
	return v
}

// Along is the diagram with the focus moved by boxes along its row: to the
// right, or to the left when by is negative, stopping at the row's ends.
func (v DiagramContext) Along(by int) DiagramContext {
	if _, ok := v.Card(); !ok {
		return v
	}
	return v.FocusedOn(v.Grid.Along(v.Focus, by))
}

// Down is the diagram with the focus moved by rows down, or up when by is
// negative, stopping at the top and bottom rows. It lands on the box at the
// place in the row it keeps, or on the last box of a shorter row, and keeps
// that place for the next row.
func (v DiagramContext) Down(by int) DiagramContext {
	if _, ok := v.Card(); !ok {
		return v
	}
	v.Focus = v.Grid.Down(v.Focus, by, v.Item)
	return v
}

// Lines are the lines of the diagram the main panel shows, and their
// extent.
func (v DiagramContext) Lines(tr *i18n.TranslationSet, d types.ReviewData, width, height int, filter types.PullFilter, styles map[string]lipgloss.Style) ([]string, views.Extent) {
	c, _, _ := diagram.Draw(tr, d, v.Scene, filter)
	return c.Lines(v.Left, v.Top, width, height, styles), views.Extent{Lines: c.H, Top: v.Top}
}

// BoxAt is the box at a cell of the main panel, or -1.
func (v DiagramContext) BoxAt(tr *i18n.TranslationSet, d types.ReviewData, x, y int, filter types.PullFilter) int {
	_, tops, _ := diagram.Draw(tr, d, v.Scene, filter)
	x, y = x+v.Left, y+v.Top
	for i, s := range v.Grid.Spots {
		if x >= s.Col && x < s.Col+s.Width && y >= tops[s.Row] && y < tops[s.Row]+model.CardHeight {
			return i
		}
	}
	return -1
}

// Scrolled is the diagram scrolled across and down in a width and height.
func (v DiagramContext) Scrolled(tr *i18n.TranslationSet, d types.ReviewData, across, by, width, height int, filter types.PullFilter) DiagramContext {
	c, _, _ := diagram.Draw(tr, d, v.Scene, filter)
	v.Left = utils.Clamp(v.Left+across, 0, max(0, c.W-width))
	v.Top = utils.Clamp(v.Top+by, 0, max(0, c.H-height))
	return v
}

// ShownIn is the diagram scrolled to show the box in focus, and its links
// below it, in a width and height.
func (v DiagramContext) ShownIn(tr *i18n.TranslationSet, d types.ReviewData, width, height int, filter types.PullFilter) DiagramContext {
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
	if x1-v.Left > width {
		v.Left = x1 - width
	}
	if x0 < v.Left {
		v.Left = max(0, x0)
	}
	if y1-v.Top > height {
		v.Top = y1 - height
	}
	if y0 < v.Top {
		v.Top = max(0, y0)
	}
	return v
}
