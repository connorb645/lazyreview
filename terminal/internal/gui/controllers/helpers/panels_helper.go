package helpers

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// PanelsHelper moves the keys between the panels, and has the main panel
// follow the one on the left that drives it.
type PanelsHelper struct {
	c *HelperCommon
}

// FocusOn is the state with the keys in a panel, by number, the main panel
// following a panel on the left.
func (self *PanelsHelper) FocusOn(s context.ContextTree, n int) (context.ContextTree, tea.Cmd) {
	if s.Review == nil {
		return s, nil
	}
	if n == context.MainPanel {
		s = s.Focused(context.MainPanel, s.Driving())
		return self.c.Helpers.Refresh.Relaid(s), nil
	}
	s = self.c.Helpers.Refresh.Relaid(s.Focused(n, n))
	return self.Followed(s)
}

// FocusBy is the state with the keys in the panel on the left by panels on
// from the one with them, going round at the ends.
func (self *PanelsHelper) FocusBy(s context.ContextTree, by int) (context.ContextTree, tea.Cmd) {
	return self.FocusOn(s, context.ReviewPanel+utils.Around(s.Focus()-context.ReviewPanel+by, context.SideCount))
}

// Followed is the state with the main panel showing what the selected row
// of the panel with the keys stands for.
func (self *PanelsHelper) Followed(s context.ContextTree) (context.ContextTree, tea.Cmd) {
	h := self.c.Helpers
	row, ok := s.Side(s.Focus()).Selected()
	if !ok {
		return s, nil
	}
	switch s.Focus() {
	case context.ReviewPanel:
		return h.Refresh.Relaid(s.Showing(context.ScreenReview)), nil
	case context.FilesPanel:
		place := row.Item.Place
		if row.Item.Kind == "component" {
			place = nil
		}
		next, cmd := h.Diff.OpenPlace(s, row.Item.Component, place)
		if place != nil && cmd == nil {
			next.Diff = next.Diff.Find(*place).AtTop(h.Window.MainHeight(next))
			return h.Refresh.Mirrored(next), nil
		}
		return h.Refresh.Mirrored(next), cmd
	case context.NotesPanel:
		return self.toNote(s, row)
	case context.DiagramPanel:
		if i := s.Diagram.Level.Index(row.Item.Box.Box); i >= 0 {
			s.Diagram = s.Diagram.FocusedOn(i)
		}
		s = s.Showing(context.ScreenDiagram)
		return h.Refresh.Mirrored(h.Diagram.Shown(s)), nil
	}
	return s, nil
}

// TreeFolded is the state with a component or a folder of the Files panel
// folded or unfolded.
func (self *PanelsHelper) TreeFolded(s context.ContextTree, row presentation.SideRow) (context.ContextTree, tea.Cmd) {
	s.FilesPanel.Folded = utils.CopyBools(s.FilesPanel.Folded)
	s.FilesPanel.Folded[row.Key] = !s.FilesPanel.Folded[row.Key]
	return self.c.Helpers.Refresh.Relaid(s), nil
}

// Name is the name of the place with the keys, as the keys popup heads it.
func (self *PanelsHelper) Name(s context.ContextTree) string {
	tr := self.c.Tr
	if s.Focus() != context.MainPanel {
		return presentation.PanelNames(tr)[s.Focus()]
	}
	return map[context.Screen]string{context.ScreenReview: tr.ReviewTitle, context.ScreenDiff: tr.DiffTitle, context.ScreenDiagram: tr.DiagramTitle}[s.Screen()]
}

func (self *PanelsHelper) toNote(s context.ContextTree, row presentation.SideRow) (context.ContextTree, tea.Cmd) {
	h := self.c.Helpers
	place := row.Item.Place
	component := h.Diff.ComponentOf(s, place.Path)
	if component == "" {
		return s.Said(utils.ResolvePlaceholderString(self.c.Tr.NotInDiff, map[string]string{"path": place.Path}), true), nil
	}
	s, cmd := h.Diff.OpenPlace(s, component, place)
	if cmd != nil {
		return s, cmd
	}
	s.Diff = s.Diff.Copied()
	s.Diff.Notes[row.Key] = true
	s.Diff = s.Diff.Rebuilt(self.c.Tr, self.c.Markdown, self.c.Location, *s.Review, s.ShownPulls(), h.Window.MainHeight(s))
	if i := s.Diff.First(func(r presentation.DiffRow) bool { return r.Kind == "notehead" && r.Key == row.Key }); i >= 0 {
		s.Diff = s.Diff.To(i, h.Window.MainHeight(s)).Placed(h.Window.MainHeight(s))
	}
	return h.Refresh.Mirrored(s), nil
}
