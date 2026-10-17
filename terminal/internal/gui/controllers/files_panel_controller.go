package controllers

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/controllers/helpers"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
)

// FilesPanelController binds the keys of the Files panel, the tree of the
// review's components, folders and files.
type FilesPanelController struct {
	baseController
	c *helpers.HelperCommon
}

// NewFilesPanelController binds the keys of the Files panel.
func NewFilesPanelController(c *helpers.HelperCommon) *FilesPanelController {
	return &FilesPanelController{c: c}
}

// Context is the Files panel's.
func (self *FilesPanelController) Context() context.ContextKey {
	return context.FilesPanelKey
}

// Bindings are the keys of every place of a review, of every panel on the
// left, and the Files panel's own: folding the tree.
func (self *FilesPanelController) Bindings() []Binding {
	h := self.c.Helpers
	own := withReviewAll([]Binding{
		{Action: "toggle fold", Keys: []string{"z a"}, Handler: selectedRow(func(s context.ContextTree, row presentation.SideRow) (context.ContextTree, tea.Cmd) {
			if !folds(row) {
				return s, nil
			}
			return h.Panels.TreeFolded(s, row)
		}), Description: self.c.Tr.Fold},
	})
	return join(reviewScreenBindings(self.c), sidePanelBindings(self.c), focusPanelBindings(self.c), own)
}

// Help is the Files panel's keys, then those of every panel.
func (self *FilesPanelController) Help(labels presentation.KeyLabels) []HelpSection {
	tr := self.c.Tr
	rows := []presentation.HelpRow{
		{Keys: sideMoveKeys(labels), Words: tr.HelpFilesMove},
		{Keys: labels.Of("previous panel", "next panel"), Words: tr.HelpPanels},
		{Keys: labels.Of("open"), Words: tr.HelpOpenFile},
		{Keys: labels.Of("toggle fold"), Words: tr.HelpFoldTree},
	}
	return reviewScreenHelp(self.c, presentation.PanelNames(tr)[context.FilesPanel], rows, labels)
}

// Hints are what the bar shows of the Files panel's keys.
func (self *FilesPanelController) Hints(s context.ContextTree, labels presentation.KeyLabels) [][2]string {
	return reviewScreenHints(self.c, s, labels, func() [][2]string {
		tr := self.c.Tr
		return [][2]string{{labels.Of("open"), tr.Diff}, {labels.Of("toggle fold"), tr.Fold}, {labels.Of("next pull", "previous pull"), tr.PullRequest}, panelsHint(self.c, labels)}
	})
}
