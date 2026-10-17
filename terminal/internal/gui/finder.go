package gui

import (
	"fmt"
	"path"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
)

func (a App) openFinder() App {
	tr := a.c.Tr
	if a.Review == nil {
		return a.say(tr.NoReviewOpen, true)
	}
	page := a.Review.Page
	pn := context.MenuContext{Menu: presentation.Menu{Title: tr.FinderTitle, Finding: true}}
	for _, c := range page.Model.Components {
		at := types.DiagramPlace{Level: "components", Box: c.ID}
		pn.All = append(pn.All, presentation.MenuRow{Left: tr.ComponentNoun, Right: c.Title, Open: &presentation.Item{Kind: "box", Box: &at}})
	}
	for _, n := range page.Model.Nodes {
		if n.Kind == "artifact" {
			continue
		}
		at := boxOf(n)
		pn.All = append(pn.All, presentation.MenuRow{Left: tr.TypeNoun, Right: fmt.Sprintf("%s · %s", n.Name, path.Base(n.File)), Open: &presentation.Item{Kind: "box", Box: &at}})
	}
	for _, f := range append(append([]server.FileEntry{}, page.Model.Files...), page.Model.ContextFiles...) {
		pn.All = append(pn.All, presentation.MenuRow{Left: tr.FileNoun, Right: f.Path, Open: &presentation.Item{Kind: "file", Place: &server.Lines{Path: f.Path, Side: "file:"}}})
	}
	pn = pn.Narrowed("")
	a = a.withMenu(pn)
	return a
}

func (a App) finderKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	pn := a.Menu
	switch msg.Type {
	case tea.KeyEsc:
		a = a.withoutMenu()
		return a, nil
	case tea.KeyEnter:
		return a.openRow()
	case tea.KeyCtrlN, tea.KeyDown:
		pn = pn.Move(1, a.menuRows())
	case tea.KeyCtrlP, tea.KeyUp:
		pn = pn.Move(-1, a.menuRows())
	case tea.KeyBackspace:
		if typed := []rune(pn.Query); len(typed) > 0 {
			pn = pn.Narrowed(string(typed[:len(typed)-1]))
		}
	case tea.KeyCtrlU:
		pn = pn.Narrowed("")
	case tea.KeyRunes, tea.KeySpace:
		pn = pn.Narrowed(pn.Query + string(msg.Runes))
	}
	a = a.withMenu(pn)
	return a, nil
}

func (a App) openRow() (tea.Model, tea.Cmd) {
	pn := a.Menu
	if !a.menuOpen() || pn.Cursor >= len(pn.Rows) || pn.Rows[pn.Cursor].Open == nil {
		return a, nil
	}
	it := *pn.Rows[pn.Cursor].Open
	a = a.withoutMenu()
	if it.Kind == "choose" && a.Choosing != nil {
		e := *a.Choosing
		e.Pull, a.Choosing = it.Pull, nil
		return a, checkWhere(a.opts.Server, a.Review.ID, a.Review.Page.Build, e)
	}
	a = a.leave()
	switch {
	case it.Box != nil:
		if a.Focus() != context.DiagramPanel {
			a.ContextTree = a.Focused(context.MainPanel, context.DiagramPanel)
		}
		return a.showBox(*it.Box).mirrored(), nil
	case it.Place != nil:
		a.ContextTree = a.Focused(context.MainPanel, context.FilesPanel)
		next, cmd := a.openPlace(a.componentOf(it.Place.Path), it.Place)
		return next.(App).mirrored(), cmd
	}
	return a, nil
}
