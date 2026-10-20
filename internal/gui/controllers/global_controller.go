package controllers

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/internal/gui/context"
	"github.com/connorb645/lazyreview/internal/gui/controllers/helpers"
	"github.com/connorb645/lazyreview/internal/utils"
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
	tr, h, kb := self.c.Tr, self.c.Helpers, self.c.Config.Keybinding
	return []Binding{
		{Action: "command", Keys: kb.Universal.Command, Handler: stateOnly(self.command), Description: tr.CommandLine},
		{Action: "keys", Keys: kb.Universal.Keys, Handler: stateOnly(self.keys), Description: tr.Keys},
		{Action: "review", Keys: kb.Universal.Review, Handler: stateOnly(h.Review.Show), Description: tr.ReviewView},
		{Action: "github", Keys: kb.Universal.GitHub, Handler: uncounted(h.Browser.OpenGitHub), Description: tr.GitHub},
		{Action: "close", Keys: kb.Universal.Close, Handler: uncounted(self.close), Description: tr.Close},
		{Action: "quit", Keys: kb.Universal.Quit, Handler: self.quit, Description: tr.Quit},
		{Action: "refresh", Keys: kb.Universal.Refresh, Handler: uncounted(h.Review.Refresh), Description: tr.Refresh},
		{Action: "bigger", Keys: kb.Universal.Bigger, Handler: stateOnly(self.bigger), Description: tr.Bigger},
		{Action: "smaller", Keys: kb.Universal.Smaller, Handler: stateOnly(self.smaller), Description: tr.Smaller},
	}
}

func (self *GlobalController) command(s context.ContextTree) context.ContextTree {
	return self.c.Helpers.Prompt.Ask(s, ":")
}

func (self *GlobalController) keys(s context.ContextTree) context.ContextTree {
	return s.WithMenu(self.keysMenu(s))
}

func (self *GlobalController) close(s context.ContextTree) (context.ContextTree, tea.Cmd) {
	h := self.c.Helpers
	switch {
	case s.MenuOpen():
		s = s.WithoutMenu()
	case s.Screen() == context.ScreenHome:
		return self.howToQuit(s), nil
	case s.Focus() == context.ReviewPanel && len(s.Filter) > 0:
		return h.Filter.SetFilter(s, nil), nil
	case s.Focus() == context.FilesPanel && s.Review != nil && s.FilesPanel.HasSelection():
		return h.Files.Stop(s)
	case s.Screen() == context.ScreenDiff && s.Diff.Selecting:
		s.Diff.Selecting = false
	case s.Focus() == context.MainPanel && s.Review != nil:
		return h.Refresh.Relaid(s.Focused(s.Driving(), s.Driving())), nil
	default:
		return self.howToQuit(s), nil
	}
	return s, nil
}

func (self *GlobalController) howToQuit(s context.ContextTree) context.ContextTree {
	labels := self.c.KeyLabels(s.Place())
	return s.Said(utils.ResolvePlaceholderString(self.c.Tr.HowToQuit, map[string]string{"quit": labels.Of("quit"), "command": labels.Of("command")}), false)
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
