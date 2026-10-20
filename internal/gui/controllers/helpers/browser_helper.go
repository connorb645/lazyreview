package helpers

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/internal/commands/oscommands"
	"github.com/connorb645/lazyreview/internal/gui/context"
	"github.com/connorb645/lazyreview/internal/gui/presentation"
	"github.com/connorb645/lazyreview/internal/utils"
)

// BrowserHelper opens a pull request on GitHub, or a written review's
// drawings, in the browser.
type BrowserHelper struct {
	c *HelperCommon
}

// OpenGitHub is the state once the browser opens on GitHub what the
// cursor is on, or the review's first pull request.
func (self *BrowserHelper) OpenGitHub(s context.ContextTree) (context.ContextTree, tea.Cmd) {
	address := ""
	switch {
	case s.Screen() == context.ScreenHome:
		if _, _, m, ok := s.Home.Pull(); ok {
			address = m.URL
		}
	case s.Review == nil:
	case s.Focus() != context.MainPanel:
		if row, ok := s.Side(s.Focus()).Selected(); ok {
			address = row.Item.URL
		}
	case s.Screen() == context.ScreenDiff:
		if n, _, ok := self.c.Helpers.Editing.NoteAt(s); ok && n.Kind == "thread" && len(n.Thread.Comments) > 0 {
			address = n.Thread.Comments[0].URL
		}
	case s.Screen() == context.ScreenReview:
		if v := s.ReviewView.View; v.Cursor < len(v.Items) {
			address = v.Items[v.Cursor].URL
		}
	}
	if address == "" && s.Screen() != context.ScreenHome && s.Review != nil && len(s.Review.State.Members) > 0 {
		address = s.Review.State.Members[0].URL
	}
	if address == "" {
		return s.Said(self.c.Tr.NoPullRequestOnGitHub, true), nil
	}
	return self.opened(s, address), nil
}

// OpenDrawings is the state once the browser opens the drawings of the
// review's written review, which a terminal cannot show, from a page written
// for them.
func (self *BrowserHelper) OpenDrawings(s context.ContextTree) (context.ContextTree, tea.Cmd) {
	if s.Review == nil || s.Review.State.Written == nil {
		return s.Said(self.c.Tr.NoDrawings, true), nil
	}
	written := *s.Review.State.Written
	title := written.Title
	if title == "" {
		title = s.Review.State.Title
	}
	page, ok := presentation.DrawingsPage(self.c.Tr, title, written)
	if !ok {
		return s.Said(self.c.Tr.NoDrawings, true), nil
	}
	path, err := oscommands.WritePage(strings.ReplaceAll(s.Review.ID, "/", "_")+"-drawings", page)
	if err != nil {
		return s.Said(self.c.Tr.CouldNotWriteDrawings+": "+err.Error(), true), nil
	}
	if err := self.c.Open(path); err != nil {
		return s.Said(utils.ResolvePlaceholderString(self.c.Tr.CouldNotOpen, map[string]string{"name": path})+": "+err.Error(), true), nil
	}
	return s.Said(self.c.Tr.DrawingsOpened, false), nil
}

func (self *BrowserHelper) opened(s context.ContextTree, address string) context.ContextTree {
	if err := self.c.Open(address); err != nil {
		return s.Said(utils.ResolvePlaceholderString(self.c.Tr.CouldNotOpen, map[string]string{"name": address})+": "+err.Error(), true)
	}
	return s.Said(utils.ResolvePlaceholderString(self.c.Tr.Opened, map[string]string{"address": address}), false)
}
