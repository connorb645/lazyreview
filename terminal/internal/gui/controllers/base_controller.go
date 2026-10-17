package controllers

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
)

type baseController struct{}

func (self *baseController) Bindings() []Binding {
	return nil
}

func (self *baseController) Help(labels presentation.KeyLabels) []HelpSection {
	return nil
}

func (self *baseController) Hints(s context.ContextTree, labels presentation.KeyLabels) [][2]string {
	return nil
}

func (self *baseController) Unbound(s context.ContextTree, name string, msg tea.KeyMsg) (context.ContextTree, tea.Cmd, bool) {
	return s, nil, true
}
