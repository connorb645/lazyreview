package controllers

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
)

func uncounted(f func(s context.ContextTree) (context.ContextTree, tea.Cmd)) Handler {
	return func(s context.ContextTree, _ Count) (context.ContextTree, tea.Cmd) {
		return f(s)
	}
}

func stateOnly(f func(s context.ContextTree) context.ContextTree) Handler {
	return func(s context.ContextTree, _ Count) (context.ContextTree, tea.Cmd) {
		return f(s), nil
	}
}

func withReviewAll(bindings []Binding) []Binding {
	out := make([]Binding, len(bindings))
	for i, b := range bindings {
		handler := b.Handler
		b.Handler = func(s context.ContextTree, count Count) (context.ContextTree, tea.Cmd) {
			if s.Review == nil {
				return s, nil
			}
			return handler(s, count)
		}
		out[i] = b
	}
	return out
}

func join(sets ...[]Binding) []Binding {
	var out []Binding
	for _, set := range sets {
		out = append(out, set...)
	}
	return out
}
