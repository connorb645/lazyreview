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

// Bindings are a menu's keys: moving, opening a row, and closing it.
func (self *MenuController) Bindings() []Binding {
	tr, h, kb := self.c.Tr, self.c.Helpers, self.c.Config.Keybinding
	return []Binding{
		{Action: "down", Keys: kb.Menu.Down, Handler: menuMoved(h, 1), Description: tr.Down},
		{Action: "up", Keys: kb.Menu.Up, Handler: menuMoved(h, -1), Description: tr.Up},
		{Action: "open", Keys: kb.Menu.Open, Handler: uncounted(h.Menu.OpenRow), Description: tr.Go},
		{Action: "close", Keys: kb.Menu.Close, Handler: stateOnly(context.ContextTree.WithoutMenu), Description: tr.Close},
	}
}

// Unbound closes the menu, and passes the key on.
func (self *MenuController) Unbound(s context.ContextTree, _ string, _ tea.KeyMsg) (context.ContextTree, tea.Cmd, bool) {
	return s.WithoutMenu(), nil, true
}

// Hints are what the bar shows of a menu's keys: moving and opening when
// its rows open places.
func (self *MenuController) Hints(s context.ContextTree, labels presentation.KeyLabels) [][2]string {
	tr := self.c.Tr
	if s.Menu.Choosing() {
		return [][2]string{{labels.Of("down", "up"), tr.Move}, {labels.Of("open"), tr.Go}, {labels.Of("close"), tr.Close}}
	}
	return [][2]string{{labels.Of("close"), tr.Close}}
}

func menuMoved(h *helpers.Helpers, by int) Handler {
	return func(s context.ContextTree, _ Count) (context.ContextTree, tea.Cmd) {
		return h.Menu.Moved(s, by), nil
	}
}
