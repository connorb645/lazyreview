package gui

import (
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/diagram"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

const labelLetters = "asdfghjklqwertyuiopzxcvbnm"

func (a App) openDiagram() App {
	if a.review == nil {
		return a.say(a.c.Tr.NoReviewOpen, true)
	}
	a.focus, a.driving = mainPanel, diagramPanel
	if a.screen == screenDiagram {
		return a
	}
	a = a.leave()
	a.screen = screenDiagram
	return a.diagramShown()
}

func (a App) showBox(at diagramPlace) App {
	v := newDiagramView(a.c.Tr, levelAt(a.c.Tr, a.review.page, at), at.box)
	v.all, v.query = a.dg.all, a.dg.query
	a.dg, a.screen = v, screenDiagram
	return a.diagramShown()
}

func (a App) diagramShown() App {
	a.dg = a.dg.shownIn(a.c.Tr, *a.review, a.mainWidth(), a.mainHeight(), a.filter)
	return a
}

func (a App) diagramAction(action string, count int) (tea.Model, tea.Cmd) {
	if a.review == nil {
		return a, nil
	}
	v := a.dg
	switch action {
	case "left", "down", "up", "right":
		dir := map[string]string{"left": "h", "down": "j", "up": "k", "right": "l"}[action]
		for ; count > 0; count-- {
			i := v.grid.Nearest(v.focus, dir, rowLines)
			if i < 0 {
				break
			}
			v.focus = i
		}
		a.dg = v
		return a.diagramShown(), nil
	case "next box", "previous box":
		if action == "previous box" {
			count = -count
		}
		a.dg = v.moved(count)
		return a.diagramShown(), nil
	case "top", "bottom":
		a.dg = v.moved(map[string]int{"top": -len(v.level.Cards), "bottom": len(v.level.Cards)}[action])
		return a.diagramShown(), nil
	case "open":
		return a.stepIn()
	case "back":
		return a.stepOut(), nil
	case "follow":
		return a.follow()
	case "references":
		return a.linksPanel(false, true, a.c.Tr.NothingLinksTo), nil
	case "links":
		return a.linksPanel(true, true, a.c.Tr.HasNoLinks), nil
	case "properties":
		return a.propertiesPanel(), nil
	case "box diff":
		return a.boxDiff()
	case "all links":
		v.all = !v.all
		a.dg = v
		return a.diagramShown(), nil
	case "search", "search back":
		a.searchBack = action == "search back"
		return a.ask(map[bool]string{false: "/", true: "?"}[a.searchBack]), nil
	case "next match", "previous match":
		return a.findBox((action == "previous match") != a.searchBack)
	case "labels":
		order := v.grid.Order()
		v.labels, v.typed = map[string]int{}, ""
		for i, at := range order {
			v.labels[label(i, len(order))] = at
		}
		a.dg = v
	}
	return a, nil
}

func label(i, n int) string {
	if n <= len(labelLetters) {
		return labelLetters[i : i+1]
	}
	return labelLetters[i/len(labelLetters)%len(labelLetters):][:1] + labelLetters[i%len(labelLetters):][:1]
}

func (a App) labelKey(name string) App {
	v := a.dg
	typed := v.typed + name
	v.labels, v.typed = nil, ""
	if name == "esc" || len([]rune(name)) != 1 {
		a.dg = v
		return a
	}
	labels := a.dg.labels
	if at, ok := labels[typed]; ok {
		v.focus = at
		a.dg = v
		return a.diagramShown()
	}
	for l := range labels {
		if strings.HasPrefix(l, typed) {
			v.labels, v.typed = labels, typed
			a.dg = v
			return a
		}
	}
	a.dg = v
	return a.say(utils.ResolvePlaceholderString(a.c.Tr.NoBoxHasLabel, map[string]string{"label": typed}), true)
}

func (a App) stepIn() (tea.Model, tea.Cmd) {
	card, ok := a.dg.card()
	if !ok {
		return a, nil
	}
	switch card.Kind {
	case "component":
		packages, ok := diagram.Packages(a.c.Tr, a.review.page, card.ID)
		switch {
		case ok:
			return a.showBox(diagramPlace{level: "packages", component: card.ID}), nil
		case len(packages.Cards) == 1:
			return a.showBox(diagramPlace{level: "types", component: card.ID, pkg: packages.Cards[0].ID}), nil
		}
		return a.say(utils.ResolvePlaceholderString(a.c.Tr.NoTypesToDraw, map[string]string{"title": card.Title}), true), nil
	case "package":
		return a.showBox(diagramPlace{level: "types", component: a.dg.level.Component, pkg: card.ID}), nil
	case "group":
		return a.membersPanel(card), nil
	}
	return a.boxDiff()
}

func (a App) stepOut() App {
	l := a.dg.level
	switch l.Kind {
	case "types":
		if _, ok := diagram.Packages(a.c.Tr, a.review.page, l.Component); ok {
			return a.showBox(diagramPlace{level: "packages", component: l.Component, box: l.Package})
		}
		return a.showBox(diagramPlace{level: "components", box: l.Component})
	case "packages":
		return a.showBox(diagramPlace{level: "components", box: l.Component})
	}
	a = a.leave()
	a.screen = screenReview
	return a.relaid()
}

func (a App) boxDiff() (tea.Model, tea.Cmd) {
	card, ok := a.dg.card()
	if !ok {
		return a, nil
	}
	switch card.Kind {
	case "component":
		a = a.leave()
		return a.openPlace(card.ID, nil)
	case "package":
		for _, f := range diffFiles(*a.review, a.dg.level.Component, nil) {
			if f.Package == card.ID {
				a = a.leave()
				return a.openPlace(f.Component, &server.Lines{Path: f.Path, Side: "file:"})
			}
		}
		return a.say(utils.ResolvePlaceholderString(a.c.Tr.NoFilesInChange, map[string]string{"title": card.Title}), true), nil
	case "group":
		return a.openNode(card.Members[0])
	}
	return a.openNode(card.ID)
}

func (a App) openNode(id string) (tea.Model, tea.Cmd) {
	n, ok := nodeOf(a.review.page, id)
	if !ok {
		return a, nil
	}
	a = a.leave()
	return a.openPlace(fileEntry(*a.review, n.File).Component, nodePlace(n))
}

func nodePlace(n server.Node) *server.Lines {
	switch {
	case n.Kind == "artifact":
		return &server.Lines{Path: n.File, Side: "file:"}
	case n.State == "removed":
		return &server.Lines{Path: n.File, Side: "LEFT", Line: n.Line}
	}
	return &server.Lines{Path: n.File, Side: "RIGHT", Line: n.Line}
}

func nodeOf(page server.Page, id string) (server.Node, bool) {
	for _, n := range page.Model.Nodes {
		if n.ID == id {
			return n, true
		}
	}
	return server.Node{}, false
}

func boxOf(n server.Node) diagramPlace {
	box := n.ID
	if n.Group != "" {
		box = "group:" + n.Group
	}
	return diagramPlace{level: "types", component: n.Component, pkg: n.Package, box: box}
}

func (a App) linkRows(out, in bool) []menuRow {
	card, ok := a.dg.card()
	if !ok {
		return nil
	}
	l := a.dg.level
	var outs, ins []menuRow
	for _, link := range l.Links {
		other, way := link.To, "out"
		switch {
		case out && link.From == card.ID:
		case in && link.To == card.ID:
			other, way = link.From, "in"
		default:
			continue
		}
		at := a.dg.place()
		at.box = other
		name, place := a.boxName(other), ""
		if link.Outside != "" {
			n, _ := nodeOf(a.review.page, other)
			at = boxOf(n)
			place = " · " + utils.ResolvePlaceholderString(a.c.Tr.InPlace, map[string]string{"place": link.OutsidePlace})
		}
		from, to := a.boxName(card.ID), name
		if way == "in" {
			from, to = name, a.boxName(card.ID)
		}
		row := menuRow{left: map[string]string{"out": a.c.Tr.LinkOut, "in": a.c.Tr.LinkIn}[way], right: from + " ─ " + link.Words + " ─▶ " + to + place, open: &item{kind: "box", box: &at}}
		if way == "out" {
			outs = append(outs, row)
		} else {
			ins = append(ins, row)
		}
	}
	return append(outs, ins...)
}

func (a App) boxName(id string) string {
	if n, ok := nodeOf(a.review.page, id); ok {
		return n.Name
	}
	if i := a.dg.level.Index(id); i >= 0 {
		return a.dg.level.Cards[i].Title
	}
	return id
}

func (a App) linksPanel(out, in bool, none string) App {
	card, ok := a.dg.card()
	if !ok {
		return a
	}
	rows := a.linkRows(out, in)
	if len(rows) == 0 {
		return a.say(utils.ResolvePlaceholderString(none, map[string]string{"title": card.Title}), true)
	}
	title := a.boxName(card.ID) + " · " + utils.Plural(len(rows), a.c.Tr.LinkNoun)
	if !out {
		title = utils.ResolvePlaceholderString(a.c.Tr.WhatLinksTo, map[string]string{"name": a.boxName(card.ID)})
	}
	a.menu = &menu{title: title, rows: rows}
	return a
}

func (a App) follow() (tea.Model, tea.Cmd) {
	card, ok := a.dg.card()
	if !ok {
		return a, nil
	}
	rows := a.linkRows(true, false)
	switch len(rows) {
	case 0:
		return a.say(utils.ResolvePlaceholderString(a.c.Tr.LinksToNothing, map[string]string{"title": card.Title}), true), nil
	case 1:
		a = a.leave()
		return a.showBox(*rows[0].open.box), nil
	}
	a.menu = &menu{title: utils.ResolvePlaceholderString(a.c.Tr.LinksTo, map[string]string{"name": a.boxName(card.ID)}), rows: rows}
	return a, nil
}

func (a App) membersPanel(card diagram.Card) App {
	pn := menu{title: card.Title + " · " + utils.Plural(len(card.Members), a.c.Tr.MemberNoun)}
	for _, id := range card.Members {
		n, ok := nodeOf(a.review.page, id)
		if !ok {
			continue
		}
		pn.rows = append(pn.rows, menuRow{left: n.Name, right: path.Base(n.File) + "  " + diagram.StateWord(a.c.Tr, n.State), open: &item{kind: "line", place: nodePlace(n)}})
	}
	a.menu = &pn
	return a
}

func (a App) propertiesPanel() App {
	card, ok := a.dg.card()
	if !ok {
		return a
	}
	tr := a.c.Tr
	n, ok := nodeOf(a.review.page, card.ID)
	if !ok || card.Kind != "type" {
		return a.say(tr.OnlyTypeHasProperties, true)
	}
	if len(n.Properties) == 0 {
		return a.say(utils.ResolvePlaceholderString(tr.NoStoredProperties, map[string]string{"name": n.Name}), true)
	}
	holds := map[string]string{}
	for _, e := range a.review.page.Model.Edges {
		if e.Source == n.ID && e.Kind == "has" {
			target, _ := nodeOf(a.review.page, e.Target)
			for _, p := range e.Properties {
				holds[p] = tr.LinkHolds + " " + target.Name
			}
		}
	}
	lineAt := func(n int) string {
		return utils.ResolvePlaceholderString(tr.LineAt, map[string]string{"line": utils.Itoa(n)})
	}
	typeWidth, lineWidth := 0, 0
	for _, p := range n.Properties {
		typeWidth = max(typeWidth, len([]rune(p.Type)))
		lineWidth = max(lineWidth, len(lineAt(p.Line)))
	}
	pn := menu{title: n.Name + " · " + utils.Plural(len(n.Properties), tr.PropertyNoun)}
	for _, p := range n.Properties {
		inferred := ""
		if p.Inferred {
			inferred = tr.Inferred
		}
		right := fmt.Sprintf("%-*s  %-8s  %-*s  %s", typeWidth, p.Type, inferred, lineWidth, lineAt(p.Line), holds[p.Name])
		pn.rows = append(pn.rows, menuRow{left: p.Name, right: strings.TrimRight(right, " "), open: &item{kind: "line", place: &server.Lines{Path: n.File, Side: "RIGHT", Line: p.Line}}})
	}
	a.menu = &pn
	return a
}

func (a App) findBox(back bool) (tea.Model, tea.Cmd) {
	query := strings.ToLower(a.dg.query)
	if query == "" {
		return a, nil
	}
	nodes := a.review.page.Model.Nodes
	var hits []int
	for i, n := range nodes {
		if strings.Contains(strings.ToLower(n.Name), query) || strings.Contains(strings.ToLower(path.Base(n.File)), query) {
			hits = append(hits, i)
		}
	}
	if len(hits) == 0 {
		return a.say(utils.ResolvePlaceholderString(a.c.Tr.NothingIsCalled, map[string]string{"query": strconv.Quote(a.dg.query)}), true), nil
	}
	here := a.dg.place()
	from, to := -1, len(nodes)
	for i, n := range nodes {
		if at := boxOf(n); here.level == "types" && at == here {
			from, to = max(from, i), min(to, i)
		}
	}
	pick := 0
	if back {
		pick = len(hits) - 1
		for k := len(hits) - 1; k >= 0; k-- {
			if hits[k] < to {
				pick = k
				break
			}
		}
	} else {
		for k, i := range hits {
			if i > from {
				pick = k
				break
			}
		}
	}
	a = a.leave()
	a = a.showBox(boxOf(nodes[hits[pick]]))
	return a.say(utils.ResolvePlaceholderString(a.c.Tr.MatchOfFor, map[string]string{"at": utils.Itoa(pick + 1), "total": utils.Itoa(len(hits)), "query": strconv.Quote(a.dg.query)}), false), nil
}

func (a App) showType() (tea.Model, tea.Cmd) {
	v := a.diff
	if v.cursor >= len(v.rows) {
		return a, nil
	}
	r := v.rows[v.cursor]
	var types []server.Node
	for _, n := range a.review.page.Model.Nodes {
		if n.File == r.path {
			types = append(types, n)
		}
	}
	tr := a.c.Tr
	if len(types) == 0 {
		return a.say(utils.ResolvePlaceholderString(tr.NoTypeInFile, map[string]string{"file": path.Base(r.path)}), true), nil
	}
	sort.SliceStable(types, func(i, j int) bool { return types[i].Line < types[j].Line })
	if r.kind == "file" {
		pn := menu{title: utils.ResolvePlaceholderString(tr.TypesIn, map[string]string{"file": path.Base(r.path)})}
		for _, n := range types {
			at := boxOf(n)
			pn.rows = append(pn.rows, menuRow{left: n.Name, right: n.Kind + " · " + utils.ResolvePlaceholderString(tr.LineAt, map[string]string{"line": utils.Itoa(n.Line)}), open: &item{kind: "box", box: &at}})
		}
		a.menu = &pn
		return a, nil
	}
	_, line, ok := r.at()
	if !ok {
		return a.say(tr.ShowTypeHow, true), nil
	}
	at := types[0]
	for _, n := range types {
		if n.Line <= line {
			at = n
		}
	}
	a = a.leave()
	return a.showBox(boxOf(at)), nil
}

func (a App) boxWords(width int) string {
	tr := a.c.Tr
	card, ok := a.dg.card()
	if !ok {
		return tr.NothingToDrawHere
	}
	switch card.Kind {
	case "component":
		return card.Title + " · " + tr.ComponentNoun + " · " + card.Detail
	case "package":
		return utils.FitPathIn("", card.Title, " · "+tr.PackageNoun+" · "+card.Detail, width)
	case "group":
		return card.Title + " · " + card.Detail
	}
	n, _ := nodeOf(a.review.page, card.ID)
	kind := n.Kind
	if kind == "artifact" {
		kind = tr.FileNoun
	}
	tail := ""
	if state := diagram.StateWord(tr, n.State); state != "" {
		tail = " · " + state
		if len(n.Pulls) > 0 && a.review.page.Stack != nil {
			pulls := make([]string, len(n.Pulls))
			for i, p := range n.Pulls {
				pulls[i] = fmt.Sprintf("#%d", p)
			}
			tail += " " + utils.ResolvePlaceholderString(tr.InPulls, map[string]string{"pulls": strings.Join(pulls, ", ")})
		}
	}
	return utils.FitPathIn(n.Name+" · "+kind+" · ", n.File, tail, width)
}

func (a App) diagramTitle() (string, string) {
	l := a.dg.level
	out := a.c.Tr.DiagramTitle
	if l.Component != "" {
		out += " · " + componentTitle(*a.review, l.Component)
	}
	if l.Kind == "types" {
		return out + " › ", packageTitle(a.review.page, l.Component, l.Package)
	}
	return out, ""
}

func packageTitle(page server.Page, component, id string) string {
	for _, c := range page.Model.Components {
		for _, pkg := range c.Packages {
			if c.ID == component && pkg.ID == id {
				if pkg.Title != "" {
					return pkg.Title
				}
				return pkg.Path
			}
		}
	}
	return id
}

func (a App) diagramHints() [][2]string {
	tr := a.c.Tr
	card, _ := a.dg.card()
	switch card.Kind {
	case "type", "file":
		return [][2]string{{"Enter", tr.Diff}, {"gd", tr.Follow}, {"Space l", tr.Links}, {"Space p", tr.Properties}, {"-", tr.Out}}
	case "group":
		return [][2]string{{"Enter", tr.Members}, {"gd", tr.Follow}, {"Space l", tr.Links}, {"-", tr.Out}}
	}
	out := tr.Out
	if a.dg.level.Kind == "components" {
		out = tr.ReviewView
	}
	return [][2]string{{"h j k l", tr.Move}, {"Enter", tr.In}, {"Space d", tr.Diff}, {"/", tr.Find}, {"f", tr.Labels}, {"-", out}}
}
