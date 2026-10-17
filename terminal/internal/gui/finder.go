package gui

import (
	"fmt"
	"path"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
)

func (a App) openFinder() App {
	tr := a.c.Tr
	if a.review == nil {
		return a.say(tr.NoReviewOpen, true)
	}
	page := a.review.Page
	pn := menu{Menu: presentation.Menu{Title: tr.FinderTitle, Finding: true}}
	for _, c := range page.Model.Components {
		at := types.DiagramPlace{Level: "components", Box: c.ID}
		pn.all = append(pn.all, presentation.MenuRow{Left: tr.ComponentNoun, Right: c.Title, Open: &presentation.Item{Kind: "box", Box: &at}})
	}
	for _, n := range page.Model.Nodes {
		if n.Kind == "artifact" {
			continue
		}
		at := boxOf(n)
		pn.all = append(pn.all, presentation.MenuRow{Left: tr.TypeNoun, Right: fmt.Sprintf("%s · %s", n.Name, path.Base(n.File)), Open: &presentation.Item{Kind: "box", Box: &at}})
	}
	for _, f := range append(append([]server.FileEntry{}, page.Model.Files...), page.Model.ContextFiles...) {
		pn.all = append(pn.all, presentation.MenuRow{Left: tr.FileNoun, Right: f.Path, Open: &presentation.Item{Kind: "file", Place: &server.Lines{Path: f.Path, Side: "file:"}}})
	}
	pn = pn.narrowed("")
	a.menu = &pn
	return a
}

func (a App) finderKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	pn := *a.menu
	switch msg.Type {
	case tea.KeyEsc:
		a.menu = nil
		return a, nil
	case tea.KeyEnter:
		return a.openRow()
	case tea.KeyCtrlN, tea.KeyDown:
		pn = pn.move(1, a.menuRows())
	case tea.KeyCtrlP, tea.KeyUp:
		pn = pn.move(-1, a.menuRows())
	case tea.KeyBackspace:
		if typed := []rune(pn.Query); len(typed) > 0 {
			pn = pn.narrowed(string(typed[:len(typed)-1]))
		}
	case tea.KeyCtrlU:
		pn = pn.narrowed("")
	case tea.KeyRunes, tea.KeySpace:
		pn = pn.narrowed(pn.Query + string(msg.Runes))
	}
	a.menu = &pn
	return a, nil
}

func (a App) openRow() (tea.Model, tea.Cmd) {
	pn := a.menu
	if pn == nil || pn.Cursor >= len(pn.Rows) || pn.Rows[pn.Cursor].Open == nil {
		return a, nil
	}
	it := *pn.Rows[pn.Cursor].Open
	a.menu = nil
	if it.Kind == "choose" && a.choosing != nil {
		e := *a.choosing
		e.pull, a.choosing = it.Pull, nil
		return a, checkWhere(a.opts.Server, a.review.ID, a.review.Page.Build, e)
	}
	a = a.leave()
	switch {
	case it.Box != nil:
		if a.focus != diagramPanel {
			a.focus, a.driving = mainPanel, diagramPanel
		}
		return a.showBox(*it.Box).mirrored(), nil
	case it.Place != nil:
		a.focus, a.driving = mainPanel, filesPanel
		next, cmd := a.openPlace(a.componentOf(it.Place.Path), it.Place)
		return next.(App).mirrored(), cmd
	}
	return a, nil
}
