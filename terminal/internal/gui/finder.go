package gui

import (
	"fmt"
	"path"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
)

func (a App) openFinder() App {
	if a.review == nil {
		return a.say("no review is open; Enter on one opens it", true)
	}
	page := a.review.page
	pn := menu{title: "Find a file, type or component", finding: true}
	for _, c := range page.Model.Components {
		at := diagramPlace{level: "components", box: c.ID}
		pn.all = append(pn.all, menuRow{left: "component", right: c.Title, open: &item{kind: "box", box: &at}})
	}
	for _, n := range page.Model.Nodes {
		if n.Kind == "artifact" {
			continue
		}
		at := boxOf(n)
		pn.all = append(pn.all, menuRow{left: "type", right: fmt.Sprintf("%s · %s", n.Name, path.Base(n.File)), open: &item{kind: "box", box: &at}})
	}
	for _, f := range append(append([]server.FileEntry{}, page.Model.Files...), page.Model.ContextFiles...) {
		pn.all = append(pn.all, menuRow{left: "file", right: f.Path, open: &item{kind: "file", place: &server.Lines{Path: f.Path, Side: "file:"}}})
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
		if typed := []rune(pn.query); len(typed) > 0 {
			pn = pn.narrowed(string(typed[:len(typed)-1]))
		}
	case tea.KeyCtrlU:
		pn = pn.narrowed("")
	case tea.KeyRunes, tea.KeySpace:
		pn = pn.narrowed(pn.query + string(msg.Runes))
	}
	a.menu = &pn
	return a, nil
}

func (a App) openRow() (tea.Model, tea.Cmd) {
	pn := a.menu
	if pn == nil || pn.cursor >= len(pn.rows) || pn.rows[pn.cursor].open == nil {
		return a, nil
	}
	it := *pn.rows[pn.cursor].open
	a.menu = nil
	if it.kind == "choose" && a.choosing != nil {
		e := *a.choosing
		e.pull, a.choosing = it.pull, nil
		return a, checkWhere(a.opts.Server, a.review.id, a.review.page.Build, e)
	}
	a = a.leave()
	switch {
	case it.box != nil:
		if a.focus != diagramPanel {
			a.focus, a.driving = mainPanel, diagramPanel
		}
		return a.showBox(*it.box).mirrored(), nil
	case it.place != nil:
		a.focus, a.driving = mainPanel, filesPanel
		next, cmd := a.openPlace(a.componentOf(it.place.Path), it.place)
		return next.(App).mirrored(), cmd
	}
	return a, nil
}
