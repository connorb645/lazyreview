package gui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

func panelNames(tr *i18n.TranslationSet) [sideCount + 1]string {
	return [sideCount + 1]string{tr.MainTitle, tr.ReviewTitle, tr.FilesTitle, tr.NotesTitle, tr.DiagramTitle}
}

func (a App) panelName() string {
	tr := a.c.Tr
	if a.focus != mainPanel {
		return panelNames(tr)[a.focus]
	}
	return map[screen]string{screenReview: tr.ReviewTitle, screenDiff: tr.DiffTitle, screenDiagram: tr.DiagramTitle}[a.screen]
}

func (a App) focusOn(n int) (tea.Model, tea.Cmd) {
	if a.review == nil {
		return a, nil
	}
	if n == mainPanel {
		a.focus = mainPanel
		return a.relaid(), nil
	}
	a.focus, a.driving = n, n
	a = a.relaid()
	return a.followed()
}

func (a App) sidesBuilt() App {
	d, tr, th := *a.review, a.c.Tr, a.c.Theme
	built := [sideCount + 1][]presentation.SideRow{
		reviewPanel:  presentation.ReviewRows(th, d, a.filter),
		filesPanel:   presentation.FileRows(tr, th, d, a.tree, a.filter),
		notesPanel:   presentation.NoteListRows(tr, th, d, a.filter),
		diagramPanel: presentation.BoxRows(tr, th, d, a.dg.Level, a.dg.Grid, a.filter),
	}
	for n := reviewPanel; n <= sideCount; n++ {
		a.sides[n] = a.sides[n].reset(built[n])
	}
	for n := reviewPanel; n <= sideCount; n++ {
		a.sides[n] = a.sides[n].shown(a.sideHeight(n))
	}
	return a
}

func (a App) mirrored() App {
	if a.review == nil || a.width == 0 {
		return a
	}
	pick := func(n int, key string) {
		if (n == a.focus && n != diagramPanel) || key == "" {
			return
		}
		s := a.sides[n]
		if i := s.index(key); i >= 0 {
			a.sides[n] = s.to(i, a.sideHeight(n))
		}
	}
	switch a.screen {
	case screenReview:
		if v := a.rv.view; v.cursor < len(v.items) && v.items[v.cursor].Pull != 0 {
			pick(reviewPanel, presentation.PullKey(v.items[v.cursor].Pull))
		}
	case screenDiff:
		if path := a.diff.file(); path != "" {
			pick(filesPanel, a.shownFile(path))
		}
		if v := a.diff; v.Cursor < len(v.Rows) && (v.Rows[v.Cursor].Kind == "notehead" || v.Rows[v.Cursor].Kind == "notetext") {
			pick(notesPanel, v.Rows[v.Cursor].Key)
		}
	case screenDiagram:
		a = a.sidesBuilt()
		if card, ok := a.dg.Card(); ok {
			pick(diagramPanel, presentation.BoxKey(card.ID))
		}
	}
	return a
}

func (a App) shownFile(path string) string {
	if a.sides[filesPanel].index(presentation.FileKey(path)) >= 0 {
		return presentation.FileKey(path)
	}
	component := a.componentOf(path)
	if dir := presentation.DirKey(component, path); a.sides[filesPanel].index(dir) >= 0 {
		return dir
	}
	return presentation.ComponentKey(component)
}

func (a App) followed() (tea.Model, tea.Cmd) {
	row, ok := a.sides[a.focus].selected()
	if !ok {
		return a, nil
	}
	switch a.focus {
	case reviewPanel:
		a.screen = screenReview
		return a.grown(row).relaid(), nil
	case filesPanel:
		place := row.Item.Place
		if row.Item.Kind == "component" {
			place = nil
		}
		next, cmd := a.openPlace(row.Item.Component, place)
		if place != nil && cmd == nil {
			after := next.(App)
			after.diff = after.diff.find(*place).atTop(after.mainHeight())
			return after.mirrored(), nil
		}
		return next.(App).mirrored(), cmd
	case notesPanel:
		return a.toNote(row)
	case diagramPanel:
		if i := a.dg.Level.Index(row.Item.Box.Box); i >= 0 {
			a.dg.Focus = i
		}
		a.screen = screenDiagram
		return a.diagramShown().mirrored(), nil
	}
	return a, nil
}

func (a App) toNote(row presentation.SideRow) (tea.Model, tea.Cmd) {
	place := row.Item.Place
	component := a.componentOf(place.Path)
	if component == "" {
		return a.say(utils.ResolvePlaceholderString(a.c.Tr.NotInDiff, map[string]string{"path": place.Path}), true), nil
	}
	next, cmd := a.openPlace(component, place)
	if cmd != nil {
		return next, cmd
	}
	a = next.(App)
	a.diff = a.diff.copied()
	a.diff.Notes[row.Key] = true
	a.diff = a.diff.rebuilt(a.c.Tr, a.md, a.opts.Location, *a.review, a.filter, a.mainHeight())
	if i := a.diff.first(func(r presentation.DiffRow) bool { return r.Kind == "notehead" && r.Key == row.Key }); i >= 0 {
		a.diff = a.diff.to(i, a.mainHeight()).placed(a.mainHeight())
	}
	return a.mirrored(), nil
}

func (a App) sideAction(action string, count int) (tea.Model, tea.Cmd) {
	s, h := a.sides[a.focus], a.sideHeight(a.focus)
	row, selected := s.selected()
	if a.focus == diagramPanel {
		return a.boxAction(action, count)
	}
	switch action {
	case "down":
		a.sides[a.focus] = s.move(count, h)
		return a.followed()
	case "up":
		a.sides[a.focus] = s.move(-count, h)
		return a.followed()
	case "top":
		a.sides[a.focus] = s.to(0, h)
		return a.followed()
	case "bottom":
		a.sides[a.focus] = s.to(s.landing(len(s.rows)-1, -1), h)
		return a.followed()
	case "open":
		if !selected {
			return a, nil
		}
		if a.focus == filesPanel && (row.Item.Kind == "component" || row.Item.Kind == "dir") {
			return a.treeFolded(row)
		}
		a = a.leave()
		a.focus = mainPanel
		return a.relaid(), nil
	case "next panel":
		return a.focusOn(reviewPanel + around(a.focus-reviewPanel+1, sideCount))
	case "previous panel":
		return a.focusOn(reviewPanel + around(a.focus-reviewPanel-1, sideCount))
	case "toggle fold":
		if selected && (row.Item.Kind == "component" || row.Item.Kind == "dir") {
			return a.treeFolded(row)
		}
	case "post":
		if selected {
			return a.openPosting(row.Item.Pull), nil
		}
	case "highlight":
		if selected {
			return a.highlight(row), nil
		}
	case "edit":
		return a.editQueued()
	case "drop":
		return a.dropQueued(), nil
	}
	return a, nil
}

func (a App) treeFolded(row presentation.SideRow) (tea.Model, tea.Cmd) {
	a.tree = copyBools(a.tree)
	a.tree[row.Key] = !a.tree[row.Key]
	return a.relaid(), nil
}

func (a App) boxAction(action string, count int) (tea.Model, tea.Cmd) {
	switch action {
	case "down":
		action = "next box"
	case "up":
		action = "previous box"
	}
	a.screen = screenDiagram
	next, cmd := a.diagramAction(action, count)
	after := next.(App)
	if after.screen != screenDiagram {
		after.focus = mainPanel
	}
	return after.relaid(), cmd
}
