package gui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/internal/gui/context"
	"github.com/connorb645/lazyreview/internal/gui/controllers"
	"github.com/connorb645/lazyreview/internal/keys"
)

func (a App) key(s context.ContextTree, msg tea.KeyMsg) (context.ContextTree, tea.Cmd) {
	name := keyName(msg)
	stack := s.Stack()
	for i := len(stack) - 1; i >= 0 && stack[i].Kind() == context.PopupKind; i-- {
		popup := a.controllers.For(stack[i])
		if popup.ReadsSequences() {
			return a.readPopup(s, popup, name, msg)
		}
		if b, ok := controllers.Bound(a.controllers.Bindings(popup), name); ok {
			return b.Handler(s, controllers.Count{N: 1})
		}
		var cmd tea.Cmd
		var pass bool
		if s, cmd, pass = popup.Unbound(s, name, msg); !pass {
			return s, cmd
		}
	}
	return a.read(s, name)
}

func (a App) readPopup(s context.ContextTree, popup controllers.Controller, name string, msg tea.KeyMsg) (context.ContextTree, tea.Cmd) {
	bindings := a.controllers.Bindings(popup)
	typed := s.Keys.Keys()
	var res keys.Result
	s.Keys, res = s.Keys.Read(name, controllers.Sequences(bindings))
	if res.Waiting {
		return s, nil
	}
	if b, ok := controllers.Named(bindings, res.Action); ok {
		return b.Handler(s, controllers.Count{N: res.Count, Typed: res.Counted})
	}
	s, cmd, pass := popup.Unbound(s, name, msg)
	if !pass {
		return s, cmd
	}
	cmds := []tea.Cmd{cmd}
	for _, k := range append(typed, name) {
		var more tea.Cmd
		s, more = a.read(s, k)
		cmds = append(cmds, more)
	}
	return s, tea.Batch(cmds...)
}

func (a App) read(s context.ContextTree, name string) (context.ContextTree, tea.Cmd) {
	bindings := a.controllers.Bindings(a.controllers.Place(s))
	var res keys.Result
	s.Keys, res = s.Keys.Read(name, controllers.Sequences(bindings))
	if !res.Waiting {
		s = s.Said("", false)
	}
	b, ok := controllers.Named(bindings, res.Action)
	if !ok {
		return s, nil
	}
	s, cmd := b.Handler(s, controllers.Count{N: res.Count, Typed: res.Counted})
	s = a.helpers.Refresh.Mirrored(s)
	if where := a.helpers.Editing.WhereNow(s); where != nil {
		return s, tea.Batch(cmd, where)
	}
	return s, cmd
}

func (a App) opened(s context.ContextTree) (context.ContextTree, tea.Cmd) {
	b, ok := controllers.Named(a.controllers.Bindings(a.controllers.Place(s)), "open")
	if !ok {
		return s, nil
	}
	return b.Handler(s, controllers.Count{N: 1})
}

func keyName(msg tea.KeyMsg) string {
	switch msg.Type {
	case tea.KeySpace:
		return "space"
	case tea.KeyEnter:
		return "enter"
	case tea.KeyEsc:
		return "esc"
	case tea.KeyTab:
		return "tab"
	case tea.KeyUp:
		return "up"
	case tea.KeyDown:
		return "down"
	}
	return msg.String()
}
