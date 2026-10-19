package controllers

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/controllers/helpers"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// HomeController binds the keys of the home screen, which lists the open
// pull requests of each repository.
type HomeController struct {
	baseController
	c *helpers.HelperCommon
}

// NewHomeController binds the keys of the home screen.
func NewHomeController(c *helpers.HelperCommon) *HomeController {
	return &HomeController{c: c}
}

// Context is the home screen's.
func (self *HomeController) Context() context.ContextKey {
	return context.HomeKey
}

// Bindings are the home screen's keys: moving through the pull requests
// and the repositories, opening one's review, and hiding a repository or
// showing the hidden ones.
func (self *HomeController) Bindings() []Binding {
	tr, kb := self.c.Tr, self.c.Config.Keybinding
	bindings := []Binding{
		{Action: "down", Keys: kb.Home.Down, Handler: self.moved(func(h context.HomeContext, count Count) context.HomeContext { return h.Moved(count.N) }), Description: tr.Down},
		{Action: "up", Keys: kb.Home.Up, Handler: self.moved(func(h context.HomeContext, count Count) context.HomeContext { return h.Moved(-count.N) }), Description: tr.Up},
		{Action: "top", Keys: kb.Home.Top, Handler: self.ends(false), Description: tr.Top},
		{Action: "bottom", Keys: kb.Home.Bottom, Handler: self.ends(true), Description: tr.Bottom},
		{Action: "half down", Keys: kb.Home.HalfDown, Handler: self.paged(true, false), Description: tr.HalfPageDown},
		{Action: "half up", Keys: kb.Home.HalfUp, Handler: self.paged(true, true), Description: tr.HalfPageUp},
		{Action: "page down", Keys: kb.Home.PageDown, Handler: self.paged(false, false), Description: tr.PageDown},
		{Action: "page up", Keys: kb.Home.PageUp, Handler: self.paged(false, true), Description: tr.PageUp},
		{Action: "open", Keys: kb.Home.Open, Handler: uncounted(self.c.Helpers.Home.OpenSelected), Description: tr.Open},
		{Action: "previous panel", Keys: kb.Home.PreviousPanel, Handler: self.moved(func(h context.HomeContext, _ Count) context.HomeContext { return h.PanelBy(-1) }), Description: tr.PreviousPanel},
		{Action: "next panel", Keys: kb.Home.NextPanel, Handler: self.moved(func(h context.HomeContext, _ Count) context.HomeContext { return h.PanelBy(1) }), Description: tr.NextPanel},
		{Action: "hide", Keys: kb.Home.HideRepository, Handler: uncounted(self.c.Helpers.Home.HideOrShow), Description: tr.HideRepository},
		{Action: "show hidden", Keys: kb.Home.ShowHidden, Handler: stateOnly(self.c.Helpers.Home.ToggleHidden), Description: tr.ShowHidden},
	}
	for n := 1; n <= 9; n++ {
		bindings = append(bindings, Binding{
			Action:      "repository " + utils.Itoa(n),
			Keys:        numbered(kb.Home.JumpToRepository, n-1),
			Handler:     self.moved(func(h context.HomeContext, _ Count) context.HomeContext { return h.ToRepo(n) }),
			Description: utils.ResolvePlaceholderString(tr.RepositoryNumber, map[string]string{"number": utils.Itoa(n)}),
		})
	}
	return bindings
}

func (self *HomeController) moved(move func(h context.HomeContext, count Count) context.HomeContext) Handler {
	return func(s context.ContextTree, count Count) (context.ContextTree, tea.Cmd) {
		s.Home = move(s.Home, count)
		return self.c.Helpers.Refresh.HomeRelaid(s), nil
	}
}

func (self *HomeController) ends(last bool) Handler {
	return self.moved(func(h context.HomeContext, count Count) context.HomeContext {
		if count.Typed {
			return h.Nth(count.N)
		}
		return h.Ends(last)
	})
}

func (self *HomeController) paged(half, up bool) Handler {
	return func(s context.ContextTree, count Count) (context.ContextTree, tea.Cmd) {
		height := 1
		if panels := self.c.Helpers.Window.HomeLayout(s).Panels; s.Home.Focus < len(panels) {
			height = max(1, panels[s.Home.Focus].Inner().H)
		}
		s.Home = s.Home.Moved(pageBy(height, half, up) * count.N)
		return self.c.Helpers.Refresh.HomeRelaid(s), nil
	}
}

// Help is the home screen's keys, then what its marks mean.
func (self *HomeController) Help(labels presentation.KeyLabels) []HelpSection {
	tr := self.c.Tr
	command := map[string]string{"command": labels.Of("command")}
	rows := []presentation.HelpRow{
		{Keys: labels.Of("down", "up"), Words: tr.HelpHomeMove},
		{Keys: labels.Of("top", "bottom"), Words: tr.HelpHomeEnds},
		{Keys: labels.Of("half down", "half up"), Words: tr.HelpHalfPages},
		{Keys: labels.Of("page down", "page up"), Words: tr.HelpPages},
		{Keys: labels.Of("previous panel", "next panel"), Words: tr.HelpRepositories},
		{Keys: utils.ResolvePlaceholderString(tr.HelpFirstNineKeys, map[string]string{"first": labels.Of("repository 1"), "last": labels.Of("repository 9")}), Words: tr.HelpFirstNine},
		{Keys: labels.Of("open"), Words: tr.HelpOpenPull},
		{Keys: labels.Of("hide"), Words: tr.HelpHide},
		{Keys: labels.Of("show hidden"), Words: tr.HelpShowHidden},
		{Keys: refreshKeys(self.c, labels), Words: tr.HelpReadAgain},
		{Keys: labels.Of("github"), Words: tr.HelpGitHubPull},
		{Keys: labels.Of("web"), Words: tr.HelpWebList},
		{Keys: labels.Of("review"), Words: tr.HelpReviewOpen},
		{Keys: labels.Of("bigger", "smaller"), Words: utils.ResolvePlaceholderString(tr.HelpListMode, map[string]string{"smaller": labels.Of("smaller")})},
		{Keys: labels.Of("command"), Words: utils.ResolvePlaceholderString(tr.HelpCommandQuit, command)},
		{Keys: labels.Of("quit"), Words: utils.ResolvePlaceholderString(tr.HelpQuit, command)},
	}
	marks := []presentation.HelpRow{
		{Keys: "●", Words: tr.MarkWaiting},
		{Keys: "◆", Words: tr.MarkLoaded},
		{Keys: "┌ └", Words: tr.MarkStack},
		{Keys: tr.Dim, Words: tr.MarkDraft},
		{Keys: tr.HiddenTag, Words: hiddenMark(tr, labels.Of("show hidden"))},
	}
	return []HelpSection{{Heading: tr.HomeTitle, Rows: rows}, {Heading: tr.Marks, Rows: marks}}
}

func hiddenMark(tr *i18n.TranslationSet, show string) string {
	if show == "" {
		return tr.MarkHidden
	}
	return utils.ResolvePlaceholderString(tr.MarkHiddenShownBy, map[string]string{"show": show})
}

// Hints are what the bar shows of the home screen's keys: opening the pull
// request selected, or reading the list again.
func (self *HomeController) Hints(s context.ContextTree, labels presentation.KeyLabels) [][2]string {
	tr := self.c.Tr
	keys, quit := [2]string{labels.Of("keys"), tr.Keys}, [2]string{quitKeys(self.c, labels), tr.Quit}
	if _, u, _, ok := s.Home.Pull(); ok {
		open := [2]string{labels.Of("open"), tr.Open}
		if u.Review == "" {
			open = [2]string{labels.Of("open"), tr.LoadAndOpen}
		}
		hints := [][2]string{open}
		if len(s.Home.Panels) > 1 {
			hints = append(hints, [2]string{labels.Of("previous panel", "next panel"), tr.Repositories})
		}
		return append(hints, [2]string{labels.Of("github"), tr.GitHub}, [2]string{labels.Of("refresh"), tr.Refresh}, keys)
	}
	if s.Home.List == nil && s.Home.Reading {
		return [][2]string{quit, keys}
	}
	return [][2]string{{labels.Of("refresh"), tr.ReadAgain}, quit, keys}
}
