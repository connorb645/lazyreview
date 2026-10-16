package app

import (
	"fmt"
	"path"

	tea "github.com/charmbracelet/bubbletea"

	"review-desk/terminal/internal/server"
)

// openFinder lists every component, type and file of the review, for
// typing to narrow.
func (a App) openFinder() App {
	if a.review == nil {
		return a.say("no review is open; Enter on one opens it", true)
	}
	page := a.review.page
	pn := panel{title: "Find a file, type or component", finding: true}
	for _, c := range page.Model.Components {
		at := diagramPlace{level: "components", box: c.ID}
		pn.all = append(pn.all, panelRow{left: "component", right: c.Title, open: &item{kind: "box", box: &at}})
	}
	for _, n := range page.Model.Nodes {
		if n.Kind == "artifact" {
			continue
		}
		at := boxOf(n)
		pn.all = append(pn.all, panelRow{left: "type", right: fmt.Sprintf("%s · %s", n.Name, path.Base(n.File)), open: &item{kind: "box", box: &at}})
	}
	for _, f := range append(append([]server.FileEntry{}, page.Model.Files...), page.Model.ContextFiles...) {
		pn.all = append(pn.all, panelRow{left: "file", right: f.Path, open: &item{kind: "file", place: &server.Lines{Path: f.Path, Side: "file:"}}})
	}
	pn = pn.narrowed("")
	a.panel = &pn
	return a
}

// finderKey takes a key while the finder is open: typing narrows it.
func (a App) finderKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	pn := *a.panel
	switch msg.Type {
	case tea.KeyEsc:
		a.panel = nil
		return a, nil
	case tea.KeyEnter:
		return a.openRow()
	case tea.KeyCtrlN, tea.KeyDown:
		pn = pn.move(1, a.panelRows())
	case tea.KeyCtrlP, tea.KeyUp:
		pn = pn.move(-1, a.panelRows())
	case tea.KeyBackspace:
		if typed := []rune(pn.query); len(typed) > 0 {
			pn = pn.narrowed(string(typed[:len(typed)-1]))
		}
	case tea.KeyCtrlU:
		pn = pn.narrowed("")
	case tea.KeyRunes, tea.KeySpace:
		pn = pn.narrowed(pn.query + string(msg.Runes))
	}
	a.panel = &pn
	return a, nil
}

// openRow goes where the panel's row at the cursor leads, and closes it.
func (a App) openRow() (tea.Model, tea.Cmd) {
	pn := a.panel
	if pn == nil || pn.cursor >= len(pn.rows) || pn.rows[pn.cursor].open == nil {
		return a, nil
	}
	it := *pn.rows[pn.cursor].open
	a.panel = nil
	if it.kind == "choose" && a.choosing != nil {
		e := *a.choosing
		e.pull, a.choosing = it.pull, nil
		return a, checkWhere(a.opts.Server, a.review.id, a.review.page.Build, e)
	}
	a = a.leave()
	switch {
	case it.box != nil:
		return a.showBox(*it.box), nil
	case it.place != nil:
		return a.openPlace(a.componentOf(it.place.Path), it.place)
	}
	return a, nil
}
