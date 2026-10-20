package controllers

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/internal/gui/context"
	"github.com/connorb645/lazyreview/internal/gui/controllers/helpers"
)

// PromptController takes the keys typed in the bar, toward a command or a
// search. It binds none: Esc and Enter are the text field's own.
type PromptController struct {
	baseController
	c *helpers.HelperCommon
}

// NewPromptController takes the keys typed in the bar.
func NewPromptController(c *helpers.HelperCommon) *PromptController {
	return &PromptController{c: c}
}

// Context is the bar's prompt's.
func (self *PromptController) Context() context.ContextKey {
	return context.PromptKey
}

// Unbound closes the prompt on Esc, does what was typed on Enter, and
// types every other key. No key passes on.
func (self *PromptController) Unbound(s context.ContextTree, _ string, msg tea.KeyMsg) (context.ContextTree, tea.Cmd, bool) {
	h := self.c.Helpers
	switch msg.Type {
	case tea.KeyEsc:
		return s.Closed(context.PromptKey), nil, false
	case tea.KeyEnter:
		s, cmd := h.Prompt.Submitted(s)
		return s, cmd, false
	}
	s, cmd := h.Prompt.Typed(s, msg)
	return s, cmd, false
}
