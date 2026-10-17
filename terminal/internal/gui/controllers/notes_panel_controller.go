package controllers

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/controllers/helpers"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
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
// left, and the Notes panel's own: edit, drop and post.
func (self *NotesPanelController) Bindings() []Binding {
	tr, h, kb := self.c.Tr, self.c.Helpers, self.c.Config.Keybinding
	own := withReviewAll([]Binding{
		{Action: "edit", Keys: kb.NotesPanel.Edit, Handler: uncounted(h.Editing.EditQueued), Description: tr.Edit},
		{Action: "drop", Keys: kb.NotesPanel.Drop, Handler: stateOnly(h.Editing.DropQueued), Description: tr.Drop},
		{Action: "post", Keys: kb.NotesPanel.Post, Handler: selectedRow(func(s context.ContextTree, row presentation.SideRow) (context.ContextTree, tea.Cmd) {
			return h.Posting.Open(s, row.Item.Pull), nil
		}), Description: tr.Post},
	})
	return join(reviewScreenBindings(self.c), sidePanelBindings(self.c), focusPanelBindings(self.c), own)
}

// Help is the Notes panel's keys, then those of every panel.
func (self *NotesPanelController) Help(labels presentation.KeyLabels) []HelpSection {
	tr := self.c.Tr
	rows := []presentation.HelpRow{
		{Keys: sideMoveKeys(labels), Words: tr.HelpNotesMove},
		{Keys: labels.Of("previous panel", "next panel"), Words: tr.HelpPanels},
		{Keys: labels.Of("open"), Words: tr.HelpOpenNote},
		{Keys: labels.Of("edit", "drop"), Words: tr.HelpEditDrop},
		{Keys: labels.Of("post"), Words: tr.HelpPostComment},
	}
	return reviewScreenHelp(self.c, presentation.PanelNames(tr)[context.NotesPanel], rows, labels)
}

// Hints are what the bar shows of the Notes panel's keys.
func (self *NotesPanelController) Hints(s context.ContextTree, labels presentation.KeyLabels) [][2]string {
	return reviewScreenHints(self.c, s, labels, func() [][2]string {
		tr := self.c.Tr
		return [][2]string{{labels.Of("open"), tr.Open}, {labels.Of("edit"), tr.Edit}, {labels.Of("drop"), tr.Drop}, {labels.Of("post"), tr.Post}, panelsHint(self.c, labels)}
	})
}
