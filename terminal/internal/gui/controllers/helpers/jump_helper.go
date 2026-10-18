package helpers

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
)

// JumpHelper keeps the jump list, as Vim's: where the main panel was before
// each jump, to go back and forward through.
type JumpHelper struct {
	c *HelperCommon
}

// Leave is the state with where the main panel is now kept as a jump, the
// jumps after the one gone back to forgotten.
func (self *JumpHelper) Leave(s context.ContextTree) context.ContextTree {
	s.Jumps = append(append([]context.Jump{}, s.Jumps[:s.JumpAt]...), here(s))
	s.JumpAt = len(s.Jumps)
	return s
}

// JumpBy is the state with the main panel back by jumps, or on by them,
// as far as there are, the keys in it.
func (self *JumpHelper) JumpBy(s context.ContextTree, by int) (context.ContextTree, tea.Cmd) {
	h := self.c.Helpers
	if by < 0 {
		if s.JumpAt == 0 {
			return s, nil
		}
		if s.JumpAt == len(s.Jumps) {
			s.Jumps = append(append([]context.Jump{}, s.Jumps...), here(s))
		}
		s.JumpAt = max(0, s.JumpAt+by)
	} else {
		if s.JumpAt+1 >= len(s.Jumps) {
			return s, nil
		}
		s.JumpAt = min(len(s.Jumps)-1, s.JumpAt+by)
	}
	j := s.Jumps[s.JumpAt]
	s = s.Focused(context.MainPanel, s.Driving())
	switch j.Screen {
	case context.ScreenDiff:
		s = s.Showing(context.ScreenDiff)
		if s.Driving() != context.NotesPanel {
			s = s.Focused(s.Focus(), context.FilesPanel)
		}
		return h.Diff.OpenPlace(s, j.Component, j.Place)
	case context.ScreenReview:
		s = s.Showing(context.ScreenReview).Focused(s.Focus(), context.ReviewPanel)
		s.ReviewView.View = s.ReviewView.View.To(j.Item, h.Window.MainHeight(s))
		return h.Refresh.Relaid(s), nil
	case context.ScreenDiagram:
		s = s.Focused(s.Focus(), context.DiagramPanel)
		return h.Diagram.ShowBox(s, *j.Box), nil
	}
	s = s.Showing(j.Screen)
	return h.Refresh.Relaid(s), nil
}

func here(s context.ContextTree) context.Jump {
	j := context.Jump{Screen: s.Screen(), Item: s.ReviewView.View.Cursor, Component: s.Diff.Component}
	switch s.Screen() {
	case context.ScreenDiff:
		j.Place = s.Diff.Place()
	case context.ScreenDiagram:
		at := s.Diagram.Place()
		j.Box = &at
	}
	return j
}
