package helpers

import (
	"fmt"
	"path"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// MenuHelper opens the finder, and the place a row of a menu stands for.
type MenuHelper struct {
	c *HelperCommon
}

// OpenFinder is the state with the finder open on every component, type
// and file of the review.
func (self *MenuHelper) OpenFinder(s context.ContextTree) context.ContextTree {
	tr := self.c.Tr
	if s.Review == nil {
		return s.Said(utils.ResolvePlaceholderString(tr.NoReviewOpen, map[string]string{"open": self.c.KeyLabels(context.HomeKey).Of("open")}), true)
	}
	page := s.Review.Page
	pn := context.MenuContext{Menu: presentation.Menu{Title: tr.FinderTitle, Finding: true}}
	for _, c := range page.Model.Components {
		at := types.DiagramPlace{Level: "components", Box: c.ID}
		pn.All = append(pn.All, presentation.MenuRow{Left: tr.ComponentNoun, Right: c.Title, Open: &presentation.Item{Kind: "box", Box: &at}})
	}
	for _, n := range page.Model.Nodes {
		if n.Kind == "artifact" {
			continue
		}
		at := BoxOf(n)
		pn.All = append(pn.All, presentation.MenuRow{Left: tr.TypeNoun, Right: fmt.Sprintf("%s · %s", n.Name, path.Base(n.File)), Open: &presentation.Item{Kind: "box", Box: &at}})
	}
	for _, f := range append(append([]server.FileEntry{}, page.Model.Files...), page.Model.ContextFiles...) {
		pn.All = append(pn.All, presentation.MenuRow{Left: tr.FileNoun, Right: f.Path, Open: &presentation.Item{Kind: "file", Place: &server.Lines{Path: f.Path, Side: "file:"}}})
	}
	return s.WithMenu(pn.Narrowed(""))
}

// OpenRow is the state with the place the menu's selected row stands for
// open, the menu closed: a box, a file, or the pull request a comment
// goes to.
func (self *MenuHelper) OpenRow(s context.ContextTree) (context.ContextTree, tea.Cmd) {
	pn, h := s.Menu, self.c.Helpers
	if !s.MenuOpen() || pn.Cursor >= len(pn.Rows) || pn.Rows[pn.Cursor].Open == nil {
		return s, nil
	}
	it := *pn.Rows[pn.Cursor].Open
	s = s.WithoutMenu()
	if it.Kind == "choose" && s.Choosing != nil {
		e := *s.Choosing
		e.Pull, s.Choosing = it.Pull, nil
		return s, h.Editing.CheckWhere(s, e)
	}
	s = h.Jump.Leave(s)
	switch {
	case it.Box != nil:
		if s.Focus() != context.DiagramPanel {
			s = s.Focused(context.MainPanel, context.DiagramPanel)
		}
		return h.Refresh.Mirrored(h.Diagram.ShowBox(s, *it.Box)), nil
	case it.Place != nil:
		s = s.Focused(context.MainPanel, context.FilesPanel)
		next, cmd := h.Diff.OpenFile(s, it.Place.Path, it.Place)
		return h.Refresh.Mirrored(next), cmd
	}
	return s, nil
}

// Moved is the state with the menu's cursor moved by rows.
func (self *MenuHelper) Moved(s context.ContextTree, by int) context.ContextTree {
	s.Menu = s.Menu.Move(by, self.c.Helpers.Window.MenuRows(s))
	return s
}

// MovedTo is the state with the menu's cursor on its nth row that is not a
// heading, counting from 1, or on its last when it has fewer.
func (self *MenuHelper) MovedTo(s context.ContextTree, n int) context.ContextTree {
	s.Menu = s.Menu.Nth(n, self.c.Helpers.Window.MenuRows(s))
	return s
}
