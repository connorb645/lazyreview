package controllers

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/controllers/helpers"
)

// GlobalController binds the keys every place takes, the home screen's
// and a review's alike.
type GlobalController struct {
	baseController
	c        *helpers.HelperCommon
	keysMenu func(s context.ContextTree) context.MenuContext
}

// NewGlobalController binds the keys every place takes; keysMenu is the
// keys popup of the place with the keys.
func NewGlobalController(c *helpers.HelperCommon, keysMenu func(s context.ContextTree) context.MenuContext) *GlobalController {
	return &GlobalController{c: c, keysMenu: keysMenu}
}

// Context is the global context.
func (self *GlobalController) Context() context.ContextKey {
	return context.GlobalKey
}

// Bindings are the keys every place takes.
func (self *GlobalController) Bindings() []Binding {
	tr, h := self.c.Tr, self.c.Helpers
	return []Binding{
		{Action: "command", Keys: []string{":"}, Handler: stateOnly(self.command), Description: tr.CommandLine},
		{Action: "keys", Keys: []string{"space ?"}, Handler: stateOnly(self.keys), Description: tr.Keys},
		{Action: "review", Keys: []string{"space v"}, Handler: stateOnly(h.Review.Show), Description: tr.ReviewView},
		{Action: "github", Keys: []string{"g x"}, Handler: uncounted(h.Browser.OpenGitHub), Description: tr.GitHub},
		{Action: "web", Keys: []string{"g X"}, Handler: uncounted(h.Browser.OpenWeb), Description: tr.WebApp},
		{Action: "close", Keys: []string{"q", "esc"}, Handler: stateOnly(self.close), Description: tr.Close},
		{Action: "quit", Keys: []string{"Z Z", "ctrl+c"}, Handler: self.quit, Description: tr.Quit},
		{Action: "refresh", Keys: []string{"R"}, Handler: uncounted(h.Review.Refresh), Description: tr.Refresh},
		{Action: "bigger", Keys: []string{"+"}, Handler: stateOnly(self.bigger), Description: tr.Bigger},
		{Action: "smaller", Keys: []string{"_"}, Handler: stateOnly(self.smaller), Description: tr.Smaller},
	}
}

func (self *GlobalController) command(s context.ContextTree) context.ContextTree {
	return self.c.Helpers.Prompt.Ask(s, ":")
}

func (self *GlobalController) keys(s context.ContextTree) context.ContextTree {
	return s.WithMenu(self.keysMenu(s))
}

func (self *GlobalController) close(s context.ContextTree) context.ContextTree {
	h := self.c.Helpers
	switch {
	case s.MenuOpen():
		s = s.WithoutMenu()
	case s.Screen() == context.ScreenHome:
		return s.Said(self.c.Tr.HowToQuit, false)
	case s.Focus() == context.ReviewPanel && len(s.Filter) > 0:
		return h.Filter.SetFilter(s, nil)
	case s.Screen() == context.ScreenDiff && s.Diff.Selecting:
		s.Diff.Selecting = false
	case s.Focus() == context.MainPanel && s.Review != nil:
		return h.Refresh.Relaid(s.Focused(s.Driving(), s.Driving()))
	default:
		return s.Said(self.c.Tr.HowToQuit, false)
	}
	return s
}

func (self *GlobalController) quit(s context.ContextTree, _ Count) (context.ContextTree, tea.Cmd) {
	return s, tea.Quit
}

func (self *GlobalController) bigger(s context.ContextTree) context.ContextTree {
	return self.resized(s, 1)
}

func (self *GlobalController) smaller(s context.ContextTree) context.ContextTree {
	return self.resized(s, int(context.ScreenModes)-1)
}

func (self *GlobalController) resized(s context.ContextTree, by int) context.ContextTree {
	s.Mode = (s.Mode + context.ScreenMode(by)) % context.ScreenModes
	return self.c.Helpers.Refresh.Relaid(s)
}
