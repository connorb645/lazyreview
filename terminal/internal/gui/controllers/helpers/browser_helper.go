package helpers

import (
	"net/url"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// BrowserHelper opens the pull request or the review in the browser.
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

// OpenWeb is the state once the browser opens the web app on the review
// open, or on its home page.
func (self *BrowserHelper) OpenWeb(s context.ContextTree) (context.ContextTree, tea.Cmd) {
	base, err := self.c.Server.Address()
	if err != nil {
		return s.Said(err.Error(), true), nil
	}
	if s.Screen() != context.ScreenReview || s.Review == nil {
		return self.opened(s, base+"/"), nil
	}
	parts := strings.Split(s.Review.ID, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return self.opened(s, base+"/review/"+strings.Join(parts, "/")+"#review"), nil
}

func (self *BrowserHelper) opened(s context.ContextTree, address string) context.ContextTree {
	if err := self.c.Open(address); err != nil {
		return s.Said(utils.ResolvePlaceholderString(self.c.Tr.CouldNotOpen, map[string]string{"name": address})+": "+err.Error(), true)
	}
	return s.Said(utils.ResolvePlaceholderString(self.c.Tr.Opened, map[string]string{"address": address}), false)
}
