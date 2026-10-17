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

func withReview(f Handler) Handler {
	return func(s context.ContextTree, count Count) (context.ContextTree, tea.Cmd) {
		if s.Review == nil {
			return s, nil
		}
		return f(s, count)
	}
}

func withReviewAll(bindings []Binding) []Binding {
	out := make([]Binding, len(bindings))
	for i, b := range bindings {
		b.Handler = withReview(b.Handler)
		out[i] = b
	}
	return out
}

func numbered(sequences []string, at int) []string {
	if at >= len(sequences) || sequences[at] == "" {
		return nil
	}
	return sequences[at : at+1]
}

func join(sets ...[]Binding) []Binding {
	var out []Binding
	for _, set := range sets {
		out = append(out, set...)
	}
	return out
}
