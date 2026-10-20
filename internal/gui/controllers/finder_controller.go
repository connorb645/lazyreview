package controllers

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/internal/gui/context"
	"github.com/connorb645/lazyreview/internal/gui/controllers/helpers"
	"github.com/connorb645/lazyreview/internal/gui/presentation"
)

// FinderController binds the keys of the finder, which narrows the
// review's components, types and files to those matching what is typed.
type FinderController struct {
	baseController
	c *helpers.HelperCommon
}

// NewFinderController binds the keys of the finder.
func NewFinderController(c *helpers.HelperCommon) *FinderController {
	return &FinderController{c: c}
}

// Context is the finder's.
func (self *FinderController) Context() context.ContextKey {
	return context.FinderKey
}

// Bindings are the finder's keys: moving, opening a match, clearing what
// is typed, and closing it. Every other key types.
func (self *FinderController) Bindings() []Binding {
	tr, h, kb := self.c.Tr, self.c.Helpers, self.c.Config.Keybinding
	return []Binding{
		{Action: "down", Keys: kb.Finder.Down, Handler: menuMoved(h, 1), Description: tr.Down},
		{Action: "up", Keys: kb.Finder.Up, Handler: menuMoved(h, -1), Description: tr.Up},
		{Action: "open", Keys: kb.Finder.Open, Handler: uncounted(h.Menu.OpenRow), Description: tr.Go},
		{Action: "close", Keys: kb.Finder.Close, Handler: stateOnly(context.ContextTree.WithoutMenu), Description: tr.Close},
		{Action: "clear", Keys: kb.Finder.Clear, Handler: stateOnly(self.clear), Description: tr.Clear},
	}
}

func (self *FinderController) clear(s context.ContextTree) context.ContextTree {
	return s.WithMenu(s.Menu.Narrowed(""))
}

// Unbound types a letter or a space toward the matches, or takes one back
// with Backspace. No key passes on.
func (self *FinderController) Unbound(s context.ContextTree, _ string, msg tea.KeyMsg) (context.ContextTree, tea.Cmd, bool) {
	pn := s.Menu
	switch msg.Type {
	case tea.KeyBackspace:
		if typed := []rune(pn.Query); len(typed) > 0 {
			pn = pn.Narrowed(string(typed[:len(typed)-1]))
		}
	case tea.KeyRunes, tea.KeySpace:
		pn = pn.Narrowed(pn.Query + string(msg.Runes))
	}
	return s.WithMenu(pn), nil, false
}

// Hints are what the bar shows of the finder's keys.
func (self *FinderController) Hints(s context.ContextTree, labels presentation.KeyLabels) [][2]string {
	tr := self.c.Tr
	return [][2]string{{tr.TypeKey, tr.Narrow}, {labels.Of("down", "up"), tr.Move}, {labels.Of("open"), tr.Go}, {labels.Of("close"), tr.Close}}
}
