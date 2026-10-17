package controllers

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
)

// Controller is what a context does with keys: its bindings, the keys
// popup's sections while it has the keys, and the bar's hints, each key in
// them labelled from the bindings. A popup takes the keys it does not bind
// with Unbound, given the key's name as keys.Reader names it and the key
// itself, and says whether the key passes on to what is under the popup.
type Controller interface {
	Context() context.ContextKey
	Bindings() []Binding
	Help(labels presentation.KeyLabels) []HelpSection
	Hints(s context.ContextTree, labels presentation.KeyLabels) [][2]string
	Unbound(s context.ContextTree, name string, msg tea.KeyMsg) (context.ContextTree, tea.Cmd, bool)
}
