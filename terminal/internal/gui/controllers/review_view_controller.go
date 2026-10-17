package controllers

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/controllers/helpers"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// ReviewViewController binds the keys of the Review view in the main
// panel: the review's sections, queued comments, conversations and links.
type ReviewViewController struct {
	baseController
	c *helpers.HelperCommon
}

// NewReviewViewController binds the keys of the Review view.
func NewReviewViewController(c *helpers.HelperCommon) *ReviewViewController {
	return &ReviewViewController{c: c}
}

// Context is the Review view's.
func (self *ReviewViewController) Context() context.ContextKey {
	return context.ReviewViewKey
}

// Bindings are the keys of every place of a review, and the Review view's
// own.
func (self *ReviewViewController) Bindings() []Binding {
	tr, h := self.c.Tr, self.c.Helpers
	own := []Binding{
		{Action: "down", Keys: []string{"j", "down"}, Handler: self.moved(func(v context.Scroller, count Count, height int) context.Scroller {
			return v.Move(count.N, height)
		}), Description: tr.Down},
		{Action: "up", Keys: []string{"k", "up"}, Handler: self.moved(func(v context.Scroller, count Count, height int) context.Scroller {
			return v.Move(-count.N, height)
		}), Description: tr.Up},
		{Action: "top", Keys: []string{"g g"}, Handler: self.moved(func(v context.Scroller, _ Count, height int) context.Scroller {
			return v.To(0, height)
		}), Description: tr.Top},
		{Action: "bottom", Keys: []string{"G"}, Handler: self.moved(func(v context.Scroller, _ Count, height int) context.Scroller {
			return v.To(len(v.Items)-1, height)
		}), Description: tr.Bottom},
		{Action: "next section", Keys: []string{"] ]"}, Handler: self.section(false), Description: tr.NextSection},
		{Action: "previous section", Keys: []string{"[ ["}, Handler: self.section(true), Description: tr.PreviousSection},
		{Action: "fold", Keys: []string{"z a"}, Handler: stateOnly(self.fold), Description: tr.Fold},
		{Action: "open", Keys: []string{"enter"}, Handler: uncounted(self.open), Description: tr.Open},
		{Action: "edit", Keys: []string{"e"}, Handler: uncounted(h.Editing.EditQueued), Description: tr.Edit},
		{Action: "drop", Keys: []string{"d d"}, Handler: stateOnly(h.Editing.DropQueued), Description: tr.Drop},
		{Action: "post", Keys: []string{"P"}, Handler: stateOnly(self.post), Description: tr.Post},
	}
	return join(reviewScreenBindings(self.c), withReviewAll(own))
}

func (self *ReviewViewController) moved(move func(v context.Scroller, count Count, height int) context.Scroller) Handler {
	return func(s context.ContextTree, count Count) (context.ContextTree, tea.Cmd) {
		s.ReviewView.View = move(s.ReviewView.View, count, self.c.Helpers.Window.MainHeight(s))
		return s, nil
	}
}

func (self *ReviewViewController) section(back bool) Handler {
	return func(s context.ContextTree, count Count) (context.ContextTree, tea.Cmd) {
		s.ReviewView = s.ReviewView.Section(back, count.N, self.c.Helpers.Window.MainHeight(s))
		return s, nil
	}
}

func (self *ReviewViewController) open(s context.ContextTree) (context.ContextTree, tea.Cmd) {
	h, v := self.c.Helpers, s.ReviewView.View
	if v.Cursor >= len(v.Items) {
		return s, nil
	}
	it := v.Items[v.Cursor]
	switch {
	case it.Kind == "section":
		return self.fold(s), nil
	case it.Kind == "component":
		s = h.Jump.Leave(s)
		s = s.Focused(s.Focus(), context.FilesPanel)
		return h.Diff.OpenPlace(s, it.Component, nil)
	case it.Place != nil:
		s = h.Jump.Leave(s)
		s = s.Focused(s.Focus(), context.FilesPanel)
		if it.Kind == "comment" || it.Kind == "thread" || it.Kind == "conversation" {
			s = s.Focused(s.Focus(), context.NotesPanel)
		}
		return h.Diff.OpenPlace(s, h.Diff.ComponentOf(s, it.Place.Path), it.Place)
	}
	return s, nil
}

func (self *ReviewViewController) fold(s context.ContextTree) context.ContextTree {
	id := s.ReviewView.SectionAt()
	if id == "" {
		return s
	}
	folded := utils.CopyBools(s.ReviewView.Folded)
	folded[id] = !folded[id]
	s.ReviewView.Folded = folded
	for i, it := range s.ReviewView.View.Items {
		if it.Kind == "section" && it.Section == id {
			s.ReviewView.View.Cursor = i
		}
	}
	return self.c.Helpers.Refresh.Relaid(s)
}

func (self *ReviewViewController) post(s context.ContextTree) context.ContextTree {
	pull := 0
	if v := s.ReviewView.View; v.Cursor < len(v.Items) {
		pull = v.Items[v.Cursor].Pull
	}
	return self.c.Helpers.Posting.Open(s, pull)
}

// Help is the Review view's keys, then those of every panel.
func (self *ReviewViewController) Help(labels presentation.KeyLabels) []HelpSection {
	tr := self.c.Tr
	rows := []presentation.HelpRow{
		{Keys: labels.Of("down", "up"), Words: tr.HelpItemsMove},
		{Keys: labels.Of("next section", "previous section"), Words: tr.HelpSections},
		{Keys: labels.Of("top", "bottom"), Words: tr.HelpItemsEnds},
		{Keys: labels.Of("open"), Words: tr.HelpOpenItem},
		{Keys: labels.Of("fold"), Words: tr.HelpFoldSection},
		{Keys: labels.Of("edit", "drop"), Words: tr.HelpEditDrop},
		{Keys: labels.Of("post"), Words: tr.HelpPostAtCursor},
	}
	return reviewScreenHelp(self.c, tr.ReviewTitle, rows, labels)
}

// Hints are what the bar shows of the Review view's keys: posting and the
// queued comment's keys on a pull request's item, folding on others.
func (self *ReviewViewController) Hints(s context.ContextTree, labels presentation.KeyLabels) [][2]string {
	return reviewScreenHints(self.c, s, labels, func() [][2]string {
		tr := self.c.Tr
		move, open, sections, list := [2]string{labels.Of("down", "up"), tr.Move}, [2]string{labels.Of("open"), tr.Open}, [2]string{labels.Of("next section", "previous section"), tr.Sections}, [2]string{labels.Nth("close", 1), tr.List}
		if v := s.ReviewView.View; v.Cursor < len(v.Items) && v.Items[v.Cursor].Pull != 0 {
			return [][2]string{move, open, {labels.Of("post"), tr.Post}, {labels.Of("edit", "drop"), tr.EditDrop}, sections, list}
		}
		return [][2]string{move, open, sections, {labels.Of("fold"), tr.Fold}, list}
	})
}
