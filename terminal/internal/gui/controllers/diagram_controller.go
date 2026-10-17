package controllers

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/controllers/helpers"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// DiagramController binds the keys of the diagram in the main panel: moving
// between its boxes, stepping in and out of them, and listing their links.
type DiagramController struct {
	baseController
	c *helpers.HelperCommon
}

// NewDiagramController binds the keys of the diagram.
func NewDiagramController(c *helpers.HelperCommon) *DiagramController {
	return &DiagramController{c: c}
}

// Context is the diagram's.
func (self *DiagramController) Context() context.ContextKey {
	return context.DiagramKey
}

// Bindings are the keys of every place of a review, and the diagram's own.
func (self *DiagramController) Bindings() []Binding {
	tr := self.c.Tr
	moves := []Binding{
		{Action: "left", Keys: []string{"h", "left"}, Handler: self.nearest("h"), Description: tr.Left},
		{Action: "down", Keys: []string{"j", "down"}, Handler: self.nearest("j"), Description: tr.Down},
		{Action: "up", Keys: []string{"k", "up"}, Handler: self.nearest("k"), Description: tr.Up},
		{Action: "right", Keys: []string{"l", "right"}, Handler: self.nearest("l"), Description: tr.Right},
		{Action: "next box", Keys: []string{"w"}, Handler: self.movedBy(1), Description: tr.NextBox},
		{Action: "previous box", Keys: []string{"b"}, Handler: self.movedBy(-1), Description: tr.PreviousBox},
		{Action: "top", Keys: []string{"g g"}, Handler: stateOnly(self.top), Description: tr.Top},
		{Action: "bottom", Keys: []string{"G"}, Handler: stateOnly(self.bottom), Description: tr.Bottom},
		{Action: "open", Keys: []string{"enter"}, Handler: uncounted(self.c.Helpers.Diagram.StepIn), Description: tr.In},
	}
	return join(reviewScreenBindings(self.c), withReviewAll(append(moves, self.boxBindings()...)))
}

func (self *DiagramController) boxBindings() []Binding {
	tr, h := self.c.Tr, self.c.Helpers
	return []Binding{
		{Action: "back", Keys: []string{"-"}, Handler: stateOnly(h.Diagram.StepOut), Description: tr.Out},
		{Action: "follow", Keys: []string{"g d"}, Handler: uncounted(h.Diagram.Follow), Description: tr.Follow},
		{Action: "references", Keys: []string{"g r"}, Handler: stateOnly(h.Diagram.References), Description: tr.References},
		{Action: "search", Keys: []string{"/"}, Handler: self.search(false), Description: tr.Search},
		{Action: "search back", Keys: []string{"?"}, Handler: self.search(true), Description: tr.SearchBack},
		{Action: "next match", Keys: []string{"n"}, Handler: self.match(false), Description: tr.NextMatch},
		{Action: "previous match", Keys: []string{"N"}, Handler: self.match(true), Description: tr.PreviousMatch},
		{Action: "labels", Keys: []string{"f"}, Handler: stateOnly(h.Diagram.Labelled), Description: tr.Labels},
		{Action: "links", Keys: []string{"space l"}, Handler: stateOnly(h.Diagram.Links), Description: tr.Links},
		{Action: "properties", Keys: []string{"space p"}, Handler: stateOnly(h.Diagram.Properties), Description: tr.Properties},
		{Action: "box diff", Keys: []string{"space d"}, Handler: uncounted(h.Diagram.BoxDiff), Description: tr.Diff},
		{Action: "all links", Keys: []string{"space a"}, Handler: stateOnly(h.Diagram.AllLinks), Description: tr.EveryLink},
	}
}

func (self *DiagramController) nearest(dir string) Handler {
	return func(s context.ContextTree, count Count) (context.ContextTree, tea.Cmd) {
		return self.c.Helpers.Diagram.Nearest(s, dir, count.N), nil
	}
}

func (self *DiagramController) movedBy(by int) Handler {
	return func(s context.ContextTree, count Count) (context.ContextTree, tea.Cmd) {
		return self.c.Helpers.Diagram.MovedBy(s, by*count.N), nil
	}
}

func (self *DiagramController) top(s context.ContextTree) context.ContextTree {
	return self.c.Helpers.Diagram.MovedBy(s, -len(s.Diagram.Level.Cards))
}

func (self *DiagramController) bottom(s context.ContextTree) context.ContextTree {
	return self.c.Helpers.Diagram.MovedBy(s, len(s.Diagram.Level.Cards))
}

func (self *DiagramController) search(back bool) Handler {
	return func(s context.ContextTree, _ Count) (context.ContextTree, tea.Cmd) {
		return self.c.Helpers.Prompt.Search(s, back), nil
	}
}

func (self *DiagramController) match(back bool) Handler {
	return func(s context.ContextTree, _ Count) (context.ContextTree, tea.Cmd) {
		return self.c.Helpers.Diagram.FindBox(s, back != s.SearchBack)
	}
}

// Help is the diagram's keys, then those of every panel.
func (self *DiagramController) Help(labels presentation.KeyLabels) []HelpSection {
	tr := self.c.Tr
	rows := []presentation.HelpRow{
		{Keys: labels.Of("left", "down", "up", "right"), Words: tr.HelpBoxesNearest},
		{Keys: labels.Of("next box", "previous box"), Words: tr.HelpBoxesOrder},
		{Keys: labels.Of("top", "bottom"), Words: tr.HelpBoxesEnds},
		{Keys: labels.Of("open"), Words: tr.HelpStepIn},
		{Keys: labels.Of("back"), Words: tr.HelpStepOut},
		{Keys: labels.Of("follow"), Words: tr.HelpFollow},
		{Keys: labels.Of("references"), Words: tr.HelpReferences},
		{Keys: labels.Of("search", "search back", "next match", "previous match"), Words: tr.HelpFindBox},
		{Keys: labelsKeys(self.c, labels), Words: tr.HelpLabels},
		{Keys: labels.Of("links"), Words: tr.HelpLinks},
		{Keys: labels.Of("properties"), Words: tr.HelpProperties},
		{Keys: labels.Of("box diff"), Words: tr.HelpBoxDiff},
		{Keys: labels.Of("all links"), Words: tr.HelpAllLinks},
	}
	return reviewScreenHelp(self.c, tr.DiagramTitle, rows, labels)
}

// Hints are what the bar shows of the diagram's keys, by the kind of box
// in focus.
func (self *DiagramController) Hints(s context.ContextTree, labels presentation.KeyLabels) [][2]string {
	return reviewScreenHints(self.c, s, labels, func() [][2]string {
		tr := self.c.Tr
		list := [2]string{labels.Nth("close", 1), tr.List}
		card, _ := s.Diagram.Card()
		switch card.Kind {
		case "type", "file":
			return [][2]string{{labels.Of("open"), tr.Diff}, {labels.Of("follow"), tr.Follow}, {labels.Of("links"), tr.Links}, {labels.Of("properties"), tr.Properties}, {labels.Of("back"), tr.Out}, list}
		case "group":
			return [][2]string{{labels.Of("open"), tr.Members}, {labels.Of("follow"), tr.Follow}, {labels.Of("links"), tr.Links}, {labels.Of("back"), tr.Out}, list}
		}
		out := tr.Out
		if s.Diagram.Level.Kind == "components" {
			out = tr.ReviewView
		}
		return [][2]string{{labels.Of("left", "down", "up", "right"), tr.Move}, {labels.Of("open"), tr.In}, {labels.Of("box diff"), tr.Diff}, {labels.Of("search"), tr.Find}, {labels.Of("labels"), tr.Labels}, {labels.Of("back"), out}, list}
	})
}

func labelsKeys(c *helpers.HelperCommon, labels presentation.KeyLabels) string {
	return utils.ResolvePlaceholderString(c.Tr.HelpLabelsKeys, map[string]string{"labels": labels.Of("labels")})
}
