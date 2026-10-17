package helpers

import (
	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
)

// RefreshHelper builds the rows of what the screen shows again, after a
// change to the state, as lazygit's refreshes its views.
type RefreshHelper struct {
	c *HelperCommon
}

// Relaid is the state with the home screen's panels, or the review's side
// panels, main panel and Review view, built again for the screen's size,
// the panels on the left following the main panel.
func (self *RefreshHelper) Relaid(s context.ContextTree) context.ContextTree {
	if s.Screen() == context.ScreenHome {
		return self.HomeRelaid(s)
	}
	if s.Width == 0 || s.Review == nil {
		return s
	}
	window := self.c.Helpers.Window
	s = self.SidesBuilt(s)
	h := window.MainHeight(s)
	switch {
	case s.Screen() == context.ScreenDiagram:
		s = self.c.Helpers.Diagram.Shown(s)
	case s.Screen() == context.ScreenDiff && s.Diff.Files != nil:
		s.Diff.Width = window.MainWidth(s)
		s.Diff = s.Diff.Rebuilt(self.c.Tr, self.c.Markdown, self.c.Location, *s.Review, s.Filter, h)
	}
	s.ReviewView = s.ReviewView.Rebuilt(self.c.Tr, self.c.Theme, self.c.Markdown, self.c.Location, *s.Review, s.Filter, window.MainWidth(s), h)
	return self.Mirrored(s)
}

// SidesBuilt is the state with the rows of the four panels on the left
// built again, each keeping its selection, shown in its height.
func (self *RefreshHelper) SidesBuilt(s context.ContextTree) context.ContextTree {
	d, tr, th := *s.Review, self.c.Tr, self.c.Theme
	built := [context.SideCount + 1][]presentation.SideRow{
		context.ReviewPanel:  presentation.ReviewRows(th, d, s.Filter),
		context.FilesPanel:   presentation.FileRows(tr, th, d, s.FilesPanel.Folded, s.Filter),
		context.NotesPanel:   presentation.NoteListRows(tr, th, d, s.Filter),
		context.DiagramPanel: presentation.BoxRows(tr, th, d, s.Diagram.Level, s.Diagram.Grid, s.Filter),
	}
	for n := context.ReviewPanel; n <= context.SideCount; n++ {
		s = s.WithSide(n, s.Side(n).Reset(built[n]))
	}
	for n := context.ReviewPanel; n <= context.SideCount; n++ {
		s = s.WithSide(n, s.Side(n).Shown(self.c.Helpers.Window.SideHeight(s, n)))
	}
	return s
}

// Mirrored is the state with the panels on the left selecting what the
// main panel's cursor is on: its pull request, its file or its note, or the
// diagram's box. The panel with the keys keeps its own selection.
func (self *RefreshHelper) Mirrored(s context.ContextTree) context.ContextTree {
	if s.Review == nil || s.Width == 0 {
		return s
	}
	pick := func(n int, key string) {
		if (n == s.Focus() && n != context.DiagramPanel) || key == "" {
			return
		}
		side := s.Side(n)
		if i := side.Index(key); i >= 0 {
			s = s.WithSide(n, side.To(i, self.c.Helpers.Window.SideHeight(s, n)))
		}
	}
	switch s.Screen() {
	case context.ScreenReview:
		if v := s.ReviewView.View; v.Cursor < len(v.Items) && v.Items[v.Cursor].Pull != 0 {
			pick(context.ReviewPanel, presentation.PullKey(v.Items[v.Cursor].Pull))
		}
	case context.ScreenDiff:
		if path := s.Diff.File(); path != "" {
			pick(context.FilesPanel, self.shownFile(s, path))
		}
		if v := s.Diff; v.Cursor < len(v.Rows) && (v.Rows[v.Cursor].Kind == "notehead" || v.Rows[v.Cursor].Kind == "notetext") {
			pick(context.NotesPanel, v.Rows[v.Cursor].Key)
		}
	case context.ScreenDiagram:
		s = self.SidesBuilt(s)
		if card, ok := s.Diagram.Card(); ok {
			pick(context.DiagramPanel, presentation.BoxKey(card.ID))
		}
	}
	return s
}

// HomeRelaid is the state with the home screen's panels built, if they
// are not yet, each shown in its height.
func (self *RefreshHelper) HomeRelaid(s context.ContextTree) context.ContextTree {
	if len(s.Home.Panels) == 0 {
		s.Home = s.Home.Rebuilt(self.c.Tr, self.c.Theme)
	}
	if s.Width == 0 {
		return s
	}
	l := self.c.Helpers.Window.HomeLayout(s)
	heights := make([]int, len(l.Panels))
	for i, r := range l.Panels {
		heights[i] = r.Inner().H
	}
	s.Home = s.Home.Shown(heights)
	return s
}

// WithState is the state with the review's state as the server now gives
// it, everything built again from it.
func (self *RefreshHelper) WithState(s context.ContextTree, st server.State) context.ContextTree {
	d := *s.Review
	d.State = st
	s.Review = &d
	return self.Relaid(s)
}

func (self *RefreshHelper) shownFile(s context.ContextTree, path string) string {
	if s.Side(context.FilesPanel).Index(presentation.FileKey(path)) >= 0 {
		return presentation.FileKey(path)
	}
	component := self.c.Helpers.Diff.ComponentOf(s, path)
	if dir := presentation.DirKey(component, path); s.Side(context.FilesPanel).Index(dir) >= 0 {
		return dir
	}
	return presentation.ComponentKey(component)
}
