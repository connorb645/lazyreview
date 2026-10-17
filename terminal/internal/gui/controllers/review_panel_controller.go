package controllers

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/controllers/helpers"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// ReviewPanelController binds the keys of the Review panel, which lists
// the review's pull requests.
type ReviewPanelController struct {
	baseController
	c *helpers.HelperCommon
}

// NewReviewPanelController binds the keys of the Review panel.
func NewReviewPanelController(c *helpers.HelperCommon) *ReviewPanelController {
	return &ReviewPanelController{c: c}
}

// Context is the Review panel's.
func (self *ReviewPanelController) Context() context.ContextKey {
	return context.ReviewPanelKey
}

// Bindings are the keys of every place of a review, of every panel on the
// left, and the Review panel's own: post and highlight.
func (self *ReviewPanelController) Bindings() []Binding {
	tr, h, kb := self.c.Tr, self.c.Helpers, self.c.Config.Keybinding
	own := withReviewAll([]Binding{
		{Action: "post", Keys: kb.ReviewPanel.Post, Handler: selectedRow(func(s context.ContextTree, row presentation.SideRow) (context.ContextTree, tea.Cmd) {
			return h.Posting.Open(s, row.Item.Pull), nil
		}), Description: tr.Post},
		{Action: "highlight", Keys: kb.ReviewPanel.Highlight, Handler: selectedRow(func(s context.ContextTree, row presentation.SideRow) (context.ContextTree, tea.Cmd) {
			return h.Filter.Highlight(s, row), nil
		}), Description: tr.Highlight},
	})
	return join(reviewScreenBindings(self.c), sidePanelBindings(self.c), focusPanelBindings(self.c), own)
}

// Help is the Review panel's keys, then those of every panel.
func (self *ReviewPanelController) Help(labels presentation.KeyLabels) []HelpSection {
	tr := self.c.Tr
	rows := []presentation.HelpRow{
		{Keys: sideMoveKeys(labels), Words: tr.HelpPullsMove},
		{Keys: labels.Of("previous panel", "next panel"), Words: utils.ResolvePlaceholderString(tr.HelpPanelsRound, map[string]string{"previousPanel": labels.Of("previous panel")})},
		{Keys: labels.Of("open"), Words: tr.HelpOpenReviewView},
		{Keys: labels.Of("highlight"), Words: utils.ResolvePlaceholderString(tr.HelpHighlight, map[string]string{"move": labels.Of("down", "up")})},
		{Keys: labels.Nth("close", 1), Words: tr.HelpClearHighlight},
		{Keys: labels.Of("post"), Words: tr.HelpPostPull},
	}
	return reviewScreenHelp(self.c, presentation.PanelNames(tr)[context.ReviewPanel], rows, labels)
}

// Hints are what the bar shows of the Review panel's keys: growing the
// highlight while it grows, and on a stack's All row only what works there.
func (self *ReviewPanelController) Hints(s context.ContextTree, labels presentation.KeyLabels) [][2]string {
	return reviewScreenHints(self.c, s, labels, func() [][2]string {
		tr := self.c.Tr
		panels := panelsHint(self.c, labels)
		open, all, post, github := [2]string{labels.Of("open"), tr.Review}, [2]string{labels.Nth("close", 1), tr.All}, [2]string{labels.Of("post"), tr.Post}, [2]string{labels.Of("github"), tr.GitHub}
		highlight := [2]string{labels.Of("highlight"), tr.Highlight}
		switch {
		case s.GrowFrom != 0:
			return [][2]string{{labels.Of("down", "up"), tr.GrowShrink}, all, panels}
		case s.Review.Page.Stack == nil:
			return [][2]string{open, post, github, panels}
		case len(s.Filter) > 0:
			return [][2]string{open, highlight, all, post, github, panels}
		}
		return [][2]string{open, panels}
	})
}
