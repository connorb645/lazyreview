package controllers

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/internal/gui/context"
	"github.com/connorb645/lazyreview/internal/gui/controllers/helpers"
	"github.com/connorb645/lazyreview/internal/gui/presentation"
	"github.com/connorb645/lazyreview/internal/utils"
)

// DiagramPanelController binds the keys of the Diagram panel, which lists
// the boxes of the diagram's level. Its moves go through the boxes in
// reading order, Enter gives the diagram in the main panel the keys, and
// the diagram's other keys do what they do there.
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

// Bindings are the keys of every place of a review, those of every panel
// on the left, the panel's moves going through the diagram's boxes in
// reading order, and the Diagram panel's own, which win a key both bind.
func (self *DiagramPanelController) Bindings() []Binding {
	tr, kb, d := self.c.Tr, self.c.Config.Keybinding.DiagramPanel, self.diagram
	boxMoves := map[string]Handler{
		"down": d.movedBy(1), "up": d.movedBy(-1), "top": d.end(false), "bottom": d.end(true),
		"half down": d.paged(true, false), "half up": d.paged(true, true), "page down": d.paged(false, false), "page up": d.paged(false, true),
	}
	panels := sidePanelBindings(self.c)
	for i, b := range panels {
		if move, ok := boxMoves[b.Action]; ok {
			panels[i].Handler = withReview(self.onDiagram(move))
		}
	}
	own := append([]Binding{
		{Action: "next box", Keys: kb.NextBox, Handler: d.movedBy(1), Description: tr.NextBox},
		{Action: "previous box", Keys: kb.PreviousBox, Handler: d.movedBy(-1), Description: tr.PreviousBox},
	}, d.boxBindings(boxKeys{
		back: kb.Back, follow: kb.Follow, references: kb.References, search: kb.Search, searchBack: kb.SearchBack, nextMatch: kb.NextMatch,
		previousMatch: kb.PreviousMatch, labels: kb.Labels, links: kb.Links, properties: kb.Properties, boxDiff: kb.BoxDiff, allLinks: kb.AllLinks,
	})...)
	for i := range own {
		own[i].Handler = self.onDiagram(own[i].Handler)
	}
	return join(reviewScreenBindings(self.c), panels, focusPanelBindings(self.c), withReviewAll(own))
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
		{Keys: labels.Of("half down", "half up"), Words: tr.HelpHalfPages},
		{Keys: labels.Of("page down", "page up"), Words: tr.HelpPages},
		{Keys: labels.Of("previous panel", "next panel"), Words: utils.ResolvePlaceholderString(tr.HelpPanelsToReview, map[string]string{"nextPanel": labels.Of("next panel")})},
		{Keys: labels.Of("open"), Words: tr.HelpOpenDiagram},
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
		out := tr.Out
		if s.Diagram.Level.Kind == "components" {
			out = tr.ReviewView
		}
		return [][2]string{
			{labels.Of("open"), tr.Diagram}, {labels.Of("back"), out}, panelsHint(self.c, labels), {labels.Of("follow"), tr.Follow},
			{labels.Of("links"), tr.Links}, {labels.Of("search"), tr.Find},
		}
	})
}
