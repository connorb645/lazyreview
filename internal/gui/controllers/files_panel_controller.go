package controllers

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/internal/gui/context"
	"github.com/connorb645/lazyreview/internal/gui/controllers/helpers"
	"github.com/connorb645/lazyreview/internal/gui/presentation"
	"github.com/connorb645/lazyreview/internal/utils"
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
// left, and the Files panel's own: folding the tree and selecting rows.
func (self *FilesPanelController) Bindings() []Binding {
	h, kb := self.c.Helpers, self.c.Config.Keybinding
	own := withReviewAll([]Binding{
		{Action: "toggle fold", Keys: kb.FilesPanel.ToggleFold, Handler: selectedRow(func(s context.ContextTree, row presentation.SideRow) (context.ContextTree, tea.Cmd) {
			if !folds(row) {
				return s, nil
			}
			return h.Panels.TreeFolded(s, row)
		}), Description: self.c.Tr.Fold},
		{Action: "select", Keys: kb.FilesPanel.Select, Handler: uncounted(h.Files.Select), Description: self.c.Tr.Select},
	})
	return join(reviewScreenBindings(self.c), sidePanelBindings(self.c), focusPanelBindings(self.c), own)
}

// Help is the Files panel's keys, then those of every panel.
func (self *FilesPanelController) Help(labels presentation.KeyLabels) []HelpSection {
	tr := self.c.Tr
	rows := []presentation.HelpRow{
		{Keys: sideMoveKeys(labels), Words: tr.HelpFilesMove},
		{Keys: labels.Of("half down", "half up"), Words: tr.HelpHalfPages},
		{Keys: labels.Of("page down", "page up"), Words: tr.HelpPages},
		{Keys: labels.Of("previous panel", "next panel"), Words: tr.HelpPanels},
		{Keys: labels.Of("open"), Words: tr.HelpOpenFile},
		{Keys: labels.Of("select"), Words: utils.ResolvePlaceholderString(tr.HelpSelectFiles, map[string]string{"move": labels.Of("down", "up"), "select": labels.Of("select"), "stop": labels.Nth("close", 1)})},
		{Keys: labels.Nth("close", 1), Words: tr.HelpStopSelection},
		{Keys: labels.Of("toggle fold"), Words: tr.HelpFoldTree},
	}
	return reviewScreenHelp(self.c, presentation.PanelNames(tr)[context.FilesPanel], rows, labels)
}

// Hints are what the bar shows of the Files panel's keys: growing the
// selection while it grows, else opening, selecting, the stack's filter on
// a stack, the panels and folding, which falls off a narrow bar first.
func (self *FilesPanelController) Hints(s context.ContextTree, labels presentation.KeyLabels) [][2]string {
	return reviewScreenHints(self.c, s, labels, func() [][2]string {
		tr := self.c.Tr
		panels := panelsHint(self.c, labels)
		if s.FilesPanel.GrowFrom != "" {
			return [][2]string{{labels.Of("down", "up"), tr.GrowShrink}, {labels.Nth("close", 1), tr.Stop}, panels}
		}
		hints := [][2]string{{labels.Of("open"), tr.Diff}, {labels.Of("select"), tr.Select}}
		switch {
		case s.Review.Page.Stack != nil && s.Review.State.Kind == "branch":
			hints = append(hints, [2]string{labels.Of("next pull", "previous pull"), tr.Branch})
		case s.Review.Page.Stack != nil:
			hints = append(hints, [2]string{labels.Of("next pull", "previous pull"), tr.PullRequest})
		}
		return append(hints, panels, [2]string{labels.Of("toggle fold"), tr.Fold})
	})
}
