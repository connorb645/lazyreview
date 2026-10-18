package controllers

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/controllers/helpers"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
)

// MenuController binds the keys of a menu: the keys popup, a list of a
// box's links or properties, a choice of pull requests, or what GitHub
// answered.
type MenuController struct {
	baseController
	c *helpers.HelperCommon
}

// NewMenuController binds the keys of a menu.
func NewMenuController(c *helpers.HelperCommon) *MenuController {
	return &MenuController{c: c}
}

// Context is the menu's.
func (self *MenuController) Context() context.ContextKey {
	return context.MenuKey
}

// Bindings are a menu's keys: moving by rows, to its ends, by half a page
// or a page, opening a row, and closing it.
func (self *MenuController) Bindings() []Binding {
	tr, h, kb := self.c.Tr, self.c.Helpers, self.c.Config.Keybinding
	return []Binding{
		{Action: "down", Keys: kb.Menu.Down, Handler: menuMoved(h, 1), Description: tr.Down},
		{Action: "up", Keys: kb.Menu.Up, Handler: menuMoved(h, -1), Description: tr.Up},
		{Action: "top", Keys: kb.Menu.Top, Handler: menuEnd(h, false), Description: tr.Top},
		{Action: "bottom", Keys: kb.Menu.Bottom, Handler: menuEnd(h, true), Description: tr.Bottom},
		{Action: "half down", Keys: kb.Menu.HalfDown, Handler: menuPaged(h, true, false), Description: tr.HalfPageDown},
		{Action: "half up", Keys: kb.Menu.HalfUp, Handler: menuPaged(h, true, true), Description: tr.HalfPageUp},
		{Action: "page down", Keys: kb.Menu.PageDown, Handler: menuPaged(h, false, false), Description: tr.PageDown},
		{Action: "page up", Keys: kb.Menu.PageUp, Handler: menuPaged(h, false, true), Description: tr.PageUp},
		{Action: "open", Keys: kb.Menu.Open, Handler: uncounted(h.Menu.OpenRow), Description: tr.Go},
		{Action: "close", Keys: kb.Menu.Close, Handler: stateOnly(context.ContextTree.WithoutMenu), Description: tr.Close},
	}
}

// Unbound closes the menu, and passes the key on.
func (self *MenuController) Unbound(s context.ContextTree, _ string, _ tea.KeyMsg) (context.ContextTree, tea.Cmd, bool) {
	return s.WithoutMenu(), nil, true
}

// ReadsSequences is true: a menu reads gg and a count before a move, as a
// place does.
func (self *MenuController) ReadsSequences() bool {
	return true
}

// Hints are what the bar shows of a menu's keys: moving when its rows are
// there to choose from, and opening when they open places.
func (self *MenuController) Hints(s context.ContextTree, labels presentation.KeyLabels) [][2]string {
	tr := self.c.Tr
	move, open, closing := [2]string{labels.Of("down", "up"), tr.Move}, [2]string{labels.Of("open"), tr.Go}, [2]string{labels.Of("close"), tr.Close}
	switch {
	case s.Menu.Opens():
		return [][2]string{move, open, closing}
	case s.Menu.Choosing():
		return [][2]string{move, closing}
	}
	return [][2]string{closing}
}

func menuMoved(h *helpers.Helpers, by int) Handler {
	return func(s context.ContextTree, count Count) (context.ContextTree, tea.Cmd) {
		return h.Menu.Moved(s, by*count.N), nil
	}
}

func menuEnd(h *helpers.Helpers, last bool) Handler {
	return func(s context.ContextTree, count Count) (context.ContextTree, tea.Cmd) {
		switch {
		case count.Typed:
			return h.Menu.MovedTo(s, count.N), nil
		case last:
			return h.Menu.MovedTo(s, len(s.Menu.Rows)), nil
		}
		return h.Menu.MovedTo(s, 1), nil
	}
}

func menuPaged(h *helpers.Helpers, half, up bool) Handler {
	return func(s context.ContextTree, count Count) (context.ContextTree, tea.Cmd) {
		by := h.Window.MenuRows(s)
		if half {
			by = max(1, by/2)
		}
		if up {
			by = -by
		}
		return h.Menu.Moved(s, by*count.N), nil
	}
}
