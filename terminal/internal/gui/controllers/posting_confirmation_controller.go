package controllers

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/controllers/helpers"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// PostingConfirmationController binds the keys of what posting sends,
// shown before it is sent.
type PostingConfirmationController struct {
	baseController
	c *helpers.HelperCommon
}

// NewPostingConfirmationController binds the keys of posting's
// confirmation.
func NewPostingConfirmationController(c *helpers.HelperCommon) *PostingConfirmationController {
	return &PostingConfirmationController{c: c}
}

// Context is posting's confirmation's.
func (self *PostingConfirmationController) Context() context.ContextKey {
	return context.PostingConfirmationKey
}

// Bindings are the confirmation's keys: send it, or go back to the choice.
func (self *PostingConfirmationController) Bindings() []Binding {
	tr := self.c.Tr
	return []Binding{
		{Action: "post", Keys: []string{"y"}, Handler: uncounted(self.c.Helpers.Posting.Post), Description: tr.Post},
		{Action: "back", Keys: []string{"n", "esc", "q"}, Handler: stateOnly(self.back), Description: tr.Cancel},
	}
}

func (self *PostingConfirmationController) back(s context.ContextTree) context.ContextTree {
	return s.Closed(context.PostingConfirmationKey)
}

// Unbound takes every other key and does nothing with it.
func (self *PostingConfirmationController) Unbound(s context.ContextTree, _ string, _ tea.KeyMsg) (context.ContextTree, tea.Cmd, bool) {
	return s, nil, false
}

// Hints are what the bar shows of the confirmation's keys, naming the
// action it sends.
func (self *PostingConfirmationController) Hints(s context.ContextTree, labels presentation.KeyLabels) [][2]string {
	tr, pn := self.c.Tr, s.Posting
	yes := utils.ResolvePlaceholderString(tr.YesAction, map[string]string{"action": strings.ToLower(presentation.PostLabels(tr)[pn.Actions[pn.Choice]])})
	return [][2]string{{labels.Of("post"), yes}, {labels.Of("back"), tr.Cancel}}
}
