package controllers

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/controllers/helpers"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/keys"
)

// LabelsController takes the keys typed toward a box's label on the
// diagram. It binds none: each key is a letter of a label.
type LabelsController struct {
	baseController
	c *helpers.HelperCommon
}

// NewLabelsController takes the keys typed toward a label.
func NewLabelsController(c *helpers.HelperCommon) *LabelsController {
	return &LabelsController{c: c}
}

// Context is the labels'.
func (self *LabelsController) Context() context.ContextKey {
	return context.LabelsKey
}

// Unbound types a key toward a label while the diagram shows, and passes
// keys on while it does not.
func (self *LabelsController) Unbound(s context.ContextTree, name string, _ tea.KeyMsg) (context.ContextTree, tea.Cmd, bool) {
	if s.Screen() != context.ScreenDiagram {
		return s, nil, true
	}
	h := self.c.Helpers
	return h.Refresh.Mirrored(h.Diagram.TypeLabel(s, name)), nil, false
}

// Hints are what the bar shows while the labels are up on the diagram.
func (self *LabelsController) Hints(s context.ContextTree, labels presentation.KeyLabels) [][2]string {
	if s.Screen() != context.ScreenDiagram {
		return nil
	}
	tr := self.c.Tr
	return [][2]string{{tr.ALabelKey, tr.JumpToBox}, {keys.Label("esc"), tr.Stop}}
}
