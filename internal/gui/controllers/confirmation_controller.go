package controllers

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/internal/gui/context"
	"github.com/connorb645/lazyreview/internal/gui/controllers/helpers"
	"github.com/connorb645/lazyreview/internal/gui/presentation"
)

// ConfirmationController binds the keys of the question whether to drop a
// queued comment, or to delete a conversation with Claude.
type ConfirmationController struct {
	baseController
	c *helpers.HelperCommon
}

// NewConfirmationController binds the keys of the drop and delete
// confirmation.
func NewConfirmationController(c *helpers.HelperCommon) *ConfirmationController {
	return &ConfirmationController{c: c}
}

// Context is the confirmation's.
func (self *ConfirmationController) Context() context.ContextKey {
	return context.ConfirmationKey
}

// Bindings are the confirmation's one key: drop the comment, or delete
// the conversation.
func (self *ConfirmationController) Bindings() []Binding {
	kb := self.c.Config.Keybinding
	return []Binding{
		{Action: "confirm", Keys: kb.Confirmation.Confirm, Handler: uncounted(self.c.Helpers.Editing.Dropped), Description: self.c.Tr.Drop},
	}
}

// Unbound keeps the comment, or the conversation, whatever the key.
func (self *ConfirmationController) Unbound(s context.ContextTree, _ string, _ tea.KeyMsg) (context.ContextTree, tea.Cmd, bool) {
	return self.c.Helpers.Editing.Kept(s), nil, false
}

// Hints are what the bar shows of the confirmation's keys.
func (self *ConfirmationController) Hints(s context.ContextTree, labels presentation.KeyLabels) [][2]string {
	tr := self.c.Tr
	if s.Confirmation.Conversation != nil {
		return [][2]string{{labels.Of("confirm"), tr.Delete}, {tr.AnyOtherKey, tr.Keep}}
	}
	return [][2]string{{labels.Of("confirm"), tr.Drop}, {tr.AnyOtherKey, tr.Keep}}
}
