package controllers

import (
	"slices"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/internal/gui/context"
	"github.com/connorb645/lazyreview/internal/gui/controllers/helpers"
	"github.com/connorb645/lazyreview/internal/gui/presentation"
)

// NotesPanelController binds the keys of the Notes panel, which lists the
// queued comments, the conversations with Claude and the threads on GitHub.
type NotesPanelController struct {
	baseController
	c *helpers.HelperCommon
}

// NewNotesPanelController binds the keys of the Notes panel.
func NewNotesPanelController(c *helpers.HelperCommon) *NotesPanelController {
	return &NotesPanelController{c: c}
}

// Context is the Notes panel's.
func (self *NotesPanelController) Context() context.ContextKey {
	return context.NotesPanelKey
}

// Bindings are the keys of every place of a review, of every panel on the
// left, and the Notes panel's own: edit, drop, fold, post and stop.
func (self *NotesPanelController) Bindings() []Binding {
	tr, h, kb := self.c.Tr, self.c.Helpers, self.c.Config.Keybinding
	own := withReviewAll([]Binding{
		{Action: "edit", Keys: kb.NotesPanel.Edit, Handler: uncounted(h.Editing.EditQueued), Description: tr.Edit},
		{Action: "drop", Keys: kb.NotesPanel.Drop, Handler: stateOnly(h.Editing.DropAtCursor), Description: tr.Drop},
		{Action: "toggle fold", Keys: kb.NotesPanel.ToggleFold, Handler: selectedRow(h.Panels.NoteFolded), Description: tr.Fold},
		{Action: "post", Keys: kb.NotesPanel.Post, Handler: selectedRow(func(s context.ContextTree, row presentation.SideRow) (context.ContextTree, tea.Cmd) {
			return h.Posting.Open(s, row.Item.Pull), nil
		}), Description: tr.Post},
		{Action: "stop", Keys: kb.NotesPanel.Stop, Handler: uncounted(h.Editing.Stop), Description: tr.StopClaude},
	})
	return join(reviewScreenBindings(self.c), sidePanelBindings(self.c), focusPanelBindings(self.c), own)
}

// Help is the Notes panel's keys, then those of every panel.
func (self *NotesPanelController) Help(s context.ContextTree, labels presentation.KeyLabels) []HelpSection {
	tr := self.c.Tr
	rows := slices.Concat([]presentation.HelpRow{
		{Keys: sideMoveKeys(labels), Words: tr.HelpNotesMove},
		{Keys: labels.Of("half down", "half up"), Words: tr.HelpHalfPages},
		{Keys: labels.Of("page down", "page up"), Words: tr.HelpPages},
		{Keys: labels.Of("previous panel", "next panel"), Words: tr.HelpPanels},
		{Keys: labels.Of("open"), Words: tr.HelpOpenNote},
	}, only(takesComments(s), presentation.HelpRow{Keys: labels.Of("edit"), Words: tr.HelpEditAtCursor}), []presentation.HelpRow{
		{Keys: labels.Of("drop"), Words: dropWords(self.c, s, labels)},
		{Keys: labels.Of("toggle fold"), Words: tr.HelpFoldNote},
	}, only(takesComments(s), presentation.HelpRow{Keys: labels.Of("post"), Words: tr.HelpPostComment}), []presentation.HelpRow{
		{Keys: labels.Of("stop"), Words: tr.HelpStop},
	})
	return reviewScreenHelp(self.c, s, presentation.PanelNames(tr)[context.NotesPanel], rows, labels)
}

// Hints are what the bar shows of the Notes panel's keys, by the row
// selected: a conversation's, a queued comment's, folding on a note that
// starts folded, and posting on a row of a pull request, only where there
// is a queue to post.
func (self *NotesPanelController) Hints(s context.ContextTree, labels presentation.KeyLabels) [][2]string {
	return reviewScreenHints(self.c, s, labels, func() [][2]string {
		tr := self.c.Tr
		open, fold, post, panels := [2]string{labels.Of("open"), tr.Open}, [2]string{labels.Of("toggle fold"), tr.Fold}, [2]string{labels.Of("post"), tr.Post}, panelsHint(self.c, labels)
		row, _ := s.Side(context.NotesPanel).Selected()
		switch c, ok := self.c.Helpers.Editing.ConversationAt(s); {
		case ok && presentation.Waiting(c):
			return [][2]string{open, {labels.Of("stop"), tr.StopClaude}, {labels.Of("drop"), tr.Delete}, fold, panels}
		case ok:
			return [][2]string{open, {labels.Of("drop"), tr.Delete}, fold, panels}
		case row.Item.Kind == "thread" && canPost(s):
			return [][2]string{open, fold, post, panels}
		case row.Item.Kind == "thread":
			return [][2]string{open, fold, panels}
		case !canPost(s):
			return [][2]string{open, panels}
		case row.Item.Kind == "comment":
			return [][2]string{open, {labels.Of("edit"), tr.Edit}, {labels.Of("drop"), tr.Drop}, post, panels}
		case row.Item.Pull != 0:
			return [][2]string{open, post, panels}
		}
		return [][2]string{open, panels}
	})
}
