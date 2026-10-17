package gui

import tea "github.com/charmbracelet/bubbletea"

var panelNames = [sideCount + 1]string{"Main", "Review", "Files", "Notes", "Diagram"}

func (a App) panelName() string {
	if a.focus != mainPanel {
		return panelNames[a.focus]
	}
	return map[screen]string{screenReview: "Review", screenDiff: "Diff", screenDiagram: "Diagram"}[a.screen]
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
	d := *a.review
	built := [sideCount + 1][]sideRow{
		reviewPanel:  reviewRows(d, a.filter, a.p),
		filesPanel:   fileRows(d, a.tree, a.filter, a.p),
		notesPanel:   noteListRows(d, a.filter, a.p),
		diagramPanel: boxRows(d, a.dg, a.filter, a.p),
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
		if v := a.rv.view; v.cursor < len(v.items) && v.items[v.cursor].pull != 0 {
			pick(reviewPanel, pullKey(v.items[v.cursor].pull))
		}
	case screenDiff:
		if path := a.diff.file(); path != "" {
			pick(filesPanel, a.shownFile(path))
		}
		if v := a.diff; v.cursor < len(v.rows) && (v.rows[v.cursor].kind == "notehead" || v.rows[v.cursor].kind == "notetext") {
			pick(notesPanel, v.rows[v.cursor].key)
		}
	case screenDiagram:
		a = a.sidesBuilt()
		if card, ok := a.dg.card(); ok {
			pick(diagramPanel, boxKey(card.ID))
		}
	}
	return a
}

func (a App) shownFile(path string) string {
	if a.sides[filesPanel].index(fileKey(path)) >= 0 {
		return fileKey(path)
	}
	component := a.componentOf(path)
	if dir := dirKey(component, path); a.sides[filesPanel].index(dir) >= 0 {
		return dir
	}
	return componentKey(component)
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
		place := row.it.place
		if row.it.kind == "component" {
			place = nil
		}
		next, cmd := a.openPlace(row.it.component, place)
		if place != nil && cmd == nil {
			after := next.(App)
			after.diff = after.diff.find(*place).atTop(after.mainHeight())
			return after.mirrored(), nil
		}
		return next.(App).mirrored(), cmd
	case notesPanel:
		return a.toNote(row)
	case diagramPanel:
		if i := a.dg.level.Index(row.it.box.box); i >= 0 {
			a.dg.focus = i
		}
		a.screen = screenDiagram
		return a.diagramShown().mirrored(), nil
	}
	return a, nil
}

func (a App) toNote(row sideRow) (tea.Model, tea.Cmd) {
	place := row.it.place
	component := a.componentOf(place.Path)
	if component == "" {
		return a.say(place.Path+" is not in this change's diff", true), nil
	}
	next, cmd := a.openPlace(component, place)
	if cmd != nil {
		return next, cmd
	}
	a = next.(App)
	a.diff = a.diff.copied()
	a.diff.notes[row.key] = true
	a.diff = a.diff.rebuilt(*a.review, a.filter, a.mainHeight())
	if i := a.diff.first(func(r diffRow) bool { return r.kind == "notehead" && r.key == row.key }); i >= 0 {
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
		if a.focus == filesPanel && (row.it.kind == "component" || row.it.kind == "dir") {
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
		if selected && (row.it.kind == "component" || row.it.kind == "dir") {
			return a.treeFolded(row)
		}
	case "post":
		if selected {
			return a.openPosting(row.it.pull), nil
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

func (a App) treeFolded(row sideRow) (tea.Model, tea.Cmd) {
	a.tree = copyBools(a.tree)
	a.tree[row.key] = !a.tree[row.key]
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
