package controllers

import (
	"slices"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/controllers/helpers"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/keys"
)

// Controllers are every context's controller and its bindings, built once,
// and what they give together: the bindings of the context with the keys,
// the labels of those keys, the keys popup, and the bar's hints.
type Controllers struct {
	c        *helpers.HelperCommon
	byKey    map[context.ContextKey]Controller
	own      map[context.ContextKey][]Binding
	anywhere map[string][]string
}

// NewControllers builds every context's controller on what the helpers
// share.
func NewControllers(c *helpers.HelperCommon) *Controllers {
	self := &Controllers{c: c, byKey: map[context.ContextKey]Controller{}, own: map[context.ContextKey][]Binding{}, anywhere: map[string][]string{}}
	diagram := NewDiagramController(c)
	list := []Controller{
		NewGlobalController(c, self.KeysMenu),
		NewHomeController(c),
		NewReviewPanelController(c),
		NewFilesPanelController(c),
		NewNotesPanelController(c),
		NewDiagramPanelController(c, diagram),
		NewReviewViewController(c),
		NewDiffController(c),
		diagram,
		NewMenuController(c),
		NewFinderController(c),
		NewLabelsController(c),
		NewPostingController(c),
		NewPostingConfirmationController(c),
		NewConfirmationController(c),
		NewPromptController(c),
	}
	for _, controller := range list {
		self.byKey[controller.Context()] = controller
		self.own[controller.Context()] = controller.Bindings()
		for _, b := range self.own[controller.Context()] {
			if _, ok := self.anywhere[b.Action]; !ok {
				self.anywhere[b.Action] = b.Keys
			}
		}
	}
	return self
}

// For is a context's controller.
func (self *Controllers) For(key context.ContextKey) Controller {
	return self.byKey[key]
}

// Place is the controller of the context that reads keys in sequences:
// the one at the top of the focus stack under any popups.
func (self *Controllers) Place(s context.ContextTree) Controller {
	stack := s.Stack()
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i].Kind() != context.PopupKind {
			return self.For(stack[i])
		}
	}
	return self.For(context.GlobalKey)
}

// Bindings are what a context takes: a popup only its own, any other the
// global bindings and then its own, its own winning a key both bind.
func (self *Controllers) Bindings(controller Controller) []Binding {
	key := controller.Context()
	if key.Kind() == context.PopupKind || key == context.GlobalKey {
		return self.own[key]
	}
	return join(self.own[context.GlobalKey], self.own[key])
}

// Sequences are bindings as keys.Reader reads them: each key sequence and
// the action it does, a later binding winning a sequence an earlier one
// binds.
func Sequences(bindings []Binding) keys.Bindings {
	out := keys.Bindings{}
	for _, b := range bindings {
		for _, sequence := range b.Keys {
			out[sequence] = b.Action
		}
	}
	return out
}

// Labels label the keys of a context's actions, or, for an action it does
// not bind, the keys the first controller binding it gives it.
func (self *Controllers) Labels(controller Controller) presentation.KeyLabels {
	bindings := self.Bindings(controller)
	won := Sequences(bindings)
	place := map[string][]string{}
	for _, b := range bindings {
		for _, sequence := range b.Keys {
			if won[sequence] == b.Action && !slices.Contains(place[b.Action], sequence) {
				place[b.Action] = append(place[b.Action], sequence)
			}
		}
	}
	return presentation.NewKeyLabels(place, self.anywhere)
}

// KeysMenu is the keys popup of the place with the keys: its sections, each
// a heading and rows of keys and what they do.
func (self *Controllers) KeysMenu(s context.ContextTree) context.MenuContext {
	place := self.Place(s)
	pn := context.MenuContext{Menu: presentation.Menu{Title: self.c.Tr.KeysTitle}}
	for _, section := range place.Help(self.Labels(place)) {
		pn.Rows = append(pn.Rows, presentation.MenuRow{Heading: section.Heading})
		for _, r := range section.Rows {
			pn.Rows = append(pn.Rows, presentation.MenuRow{Left: r.Keys, Right: r.Words})
		}
	}
	return pn.First()
}

// Hints are the keys the bar shows, each with what it does: the top
// popup's that has some, or after Space the keys that can follow it, or
// the place's.
func (self *Controllers) Hints(s context.ContextTree) [][2]string {
	stack := s.Stack()
	for i := len(stack) - 1; i >= 0 && stack[i].Kind() == context.PopupKind; i-- {
		popup := self.For(stack[i])
		if hints := popup.Hints(s, self.Labels(popup)); hints != nil {
			return hints
		}
	}
	place := self.Place(s)
	if s.Keys.Pending() == "space" {
		return self.leaderHints(place)
	}
	return place.Hints(s, self.Labels(place))
}

func (self *Controllers) leaderHints(place Controller) [][2]string {
	bindings := self.Bindings(place)
	words := map[string]string{}
	for _, b := range bindings {
		words[b.Action] = b.Description
	}
	var hints [][2]string
	for _, k := range Sequences(bindings).After("space") {
		hints = append(hints, [2]string{keys.Label("space " + k[0]), words[k[1]]})
	}
	return hints
}

// Bound is the binding a key sequence does among bindings, the last one
// binding it.
func Bound(bindings []Binding, sequence string) (Binding, bool) {
	for i := len(bindings) - 1; i >= 0; i-- {
		if slices.Contains(bindings[i].Keys, sequence) {
			return bindings[i], true
		}
	}
	return Binding{}, false
}

// Named is the binding of an action among bindings, the last one doing it.
func Named(bindings []Binding, action string) (Binding, bool) {
	for i := len(bindings) - 1; i >= 0; i-- {
		if action != "" && bindings[i].Action == action {
			return bindings[i], true
		}
	}
	return Binding{}, false
}
