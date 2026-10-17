package controllers

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/controllers/helpers"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
)

// DiagramPanelController binds the keys of the Diagram panel, which lists
// the boxes of the diagram's level. Its keys move through the diagram in
// the main panel, as the diagram's own keys do.
type DiagramPanelController struct {
	baseController
	c       *helpers.HelperCommon
	diagram *DiagramController
}

// NewDiagramPanelController binds the keys of the Diagram panel to what
// the diagram's do.
func NewDiagramPanelController(c *helpers.HelperCommon, diagram *DiagramController) *DiagramPanelController {
	return &DiagramPanelController{c: c, diagram: diagram}
}

// Context is the Diagram panel's.
func (self *DiagramPanelController) Context() context.ContextKey {
	return context.DiagramPanelKey
}

// Bindings are the keys of every place of a review, the keys to focus each
// panel, and the diagram's keys, its boxes in reading order going down
// and up the panel.
func (self *DiagramPanelController) Bindings() []Binding {
	tr, d := self.c.Tr, self.diagram
	moves := []Binding{
		{Action: "down", Keys: []string{"j", "down"}, Handler: d.movedBy(1), Description: tr.Down},
		{Action: "up", Keys: []string{"k", "up"}, Handler: d.movedBy(-1), Description: tr.Up},
		{Action: "top", Keys: []string{"g g"}, Handler: stateOnly(d.top), Description: tr.Top},
		{Action: "bottom", Keys: []string{"G"}, Handler: stateOnly(d.bottom), Description: tr.Bottom},
		{Action: "open", Keys: []string{"enter"}, Handler: uncounted(self.c.Helpers.Diagram.StepIn), Description: tr.In},
		{Action: "left", Keys: []string{"h"}, Handler: d.nearest("h"), Description: tr.Left},
		{Action: "right", Keys: []string{"l"}, Handler: d.nearest("l"), Description: tr.Right},
		{Action: "next box", Keys: []string{"w"}, Handler: d.movedBy(1), Description: tr.NextBox},
		{Action: "previous box", Keys: []string{"b"}, Handler: d.movedBy(-1), Description: tr.PreviousBox},
	}
	own := append(moves, d.boxBindings()...)
	for i := range own {
		own[i].Handler = self.onDiagram(own[i].Handler)
	}
	return join(reviewScreenBindings(self.c), focusPanelBindings(self.c), withReviewAll(own))
}

func (self *DiagramPanelController) onDiagram(f Handler) Handler {
	return func(s context.ContextTree, count Count) (context.ContextTree, tea.Cmd) {
		s, cmd := f(s.Showing(context.ScreenDiagram), count)
		if s.Screen() != context.ScreenDiagram {
			s = s.Focused(context.MainPanel, s.Driving())
		}
		return self.c.Helpers.Refresh.Relaid(s), cmd
	}
}

// Help is the Diagram panel's keys, then those of every panel.
func (self *DiagramPanelController) Help(labels presentation.KeyLabels) []HelpSection {
	tr := self.c.Tr
	rows := []presentation.HelpRow{
		{Keys: labels.Of("down", "up", "next box", "previous box", "top", "bottom"), Words: tr.HelpBoxesMove},
		{Keys: labels.Of("left", "right"), Words: tr.HelpBoxesSideways},
		{Keys: labels.Of("open"), Words: tr.HelpStepIn},
		{Keys: labels.Of("back"), Words: tr.HelpStepOut},
		{Keys: labels.Of("follow"), Words: tr.HelpFollowBox},
		{Keys: labels.Of("references"), Words: tr.HelpReferences},
		{Keys: labels.Of("search", "search back", "next match", "previous match"), Words: tr.HelpFindBox},
		{Keys: labelsKeys(self.c, labels), Words: tr.HelpLabels},
		{Keys: labels.Of("links"), Words: tr.HelpBoxLinks},
		{Keys: labels.Of("properties"), Words: tr.HelpProperties},
		{Keys: labels.Of("box diff"), Words: tr.HelpBoxDiff},
		{Keys: labels.Of("all links"), Words: tr.HelpAllLinks},
	}
	return reviewScreenHelp(self.c, presentation.PanelNames(tr)[context.DiagramPanel], rows, labels)
}

// Hints are what the bar shows of the Diagram panel's keys.
func (self *DiagramPanelController) Hints(s context.ContextTree, labels presentation.KeyLabels) [][2]string {
	return reviewScreenHints(self.c, s, labels, func() [][2]string {
		tr := self.c.Tr
		return [][2]string{
			{labels.Of("left", "down", "up", "right"), tr.Move}, {labels.Of("open"), tr.In}, {labels.Of("back"), tr.Out},
			{labels.Of("focus review", "focus files", "focus notes"), tr.Panels}, {labels.Of("follow"), tr.Follow},
			{labels.Of("links"), tr.Links}, {labels.Of("search"), tr.Find},
		}
	})
}
