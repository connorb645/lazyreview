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
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
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

func (a App) showBox(at types.DiagramPlace) App {
	v := newDiagramView(a.c.Tr, levelAt(a.c.Tr, a.review.Page, at), at.Box)
	v.All, v.query = a.dg.All, a.dg.query
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
			i := v.Grid.Nearest(v.Focus, dir, rowLines)
			if i < 0 {
				break
			}
			v.Focus = i
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
		a.dg = v.moved(map[string]int{"top": -len(v.Level.Cards), "bottom": len(v.Level.Cards)}[action])
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
		v.All = !v.All
		a.dg = v
		return a.diagramShown(), nil
	case "search", "search back":
		a.searchBack = action == "search back"
		return a.ask(map[bool]string{false: "/", true: "?"}[a.searchBack]), nil
	case "next match", "previous match":
		return a.findBox((action == "previous match") != a.searchBack)
	case "labels":
		order := v.Grid.Order()
		v.Labels, v.Typed = map[string]int{}, ""
		for i, at := range order {
			v.Labels[label(i, len(order))] = at
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
	typed := v.Typed + name
	v.Labels, v.Typed = nil, ""
	if name == "esc" || len([]rune(name)) != 1 {
		a.dg = v
		return a
	}
	labels := a.dg.Labels
	if at, ok := labels[typed]; ok {
		v.Focus = at
		a.dg = v
		return a.diagramShown()
	}
	for l := range labels {
		if strings.HasPrefix(l, typed) {
			v.Labels, v.Typed = labels, typed
			a.dg = v
			return a
		}
	}
	a.dg = v
	return a.say(utils.ResolvePlaceholderString(a.c.Tr.NoBoxHasLabel, map[string]string{"label": typed}), true)
}

func (a App) stepIn() (tea.Model, tea.Cmd) {
	card, ok := a.dg.Card()
	if !ok {
		return a, nil
	}
	switch card.Kind {
	case "component":
		packages, ok := diagram.Packages(a.c.Tr, a.review.Page, card.ID)
		switch {
		case ok:
			return a.showBox(types.DiagramPlace{Level: "packages", Component: card.ID}), nil
		case len(packages.Cards) == 1:
			return a.showBox(types.DiagramPlace{Level: "types", Component: card.ID, Package: packages.Cards[0].ID}), nil
		}
		return a.say(utils.ResolvePlaceholderString(a.c.Tr.NoTypesToDraw, map[string]string{"title": card.Title}), true), nil
	case "package":
		return a.showBox(types.DiagramPlace{Level: "types", Component: a.dg.Level.Component, Package: card.ID}), nil
	case "group":
		return a.membersPanel(card), nil
	}
	return a.boxDiff()
}

func (a App) stepOut() App {
	l := a.dg.Level
	switch l.Kind {
	case "types":
		if _, ok := diagram.Packages(a.c.Tr, a.review.Page, l.Component); ok {
			return a.showBox(types.DiagramPlace{Level: "packages", Component: l.Component, Box: l.Package})
		}
		return a.showBox(types.DiagramPlace{Level: "components", Box: l.Component})
	case "packages":
		return a.showBox(types.DiagramPlace{Level: "components", Box: l.Component})
	}
	a = a.leave()
	a.screen = screenReview
	return a.relaid()
}

func (a App) boxDiff() (tea.Model, tea.Cmd) {
	card, ok := a.dg.Card()
	if !ok {
		return a, nil
	}
	switch card.Kind {
	case "component":
		a = a.leave()
		return a.openPlace(card.ID, nil)
	case "package":
		for _, f := range presentation.DiffFiles(*a.review, a.dg.Level.Component, nil) {
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
	n, ok := presentation.NodeOf(a.review.Page, id)
	if !ok {
		return a, nil
	}
	a = a.leave()
	return a.openPlace(presentation.FileEntry(*a.review, n.File).Component, nodePlace(n))
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

func boxOf(n server.Node) types.DiagramPlace {
	box := n.ID
	if n.Group != "" {
		box = "group:" + n.Group
	}
	return types.DiagramPlace{Level: "types", Component: n.Component, Package: n.Package, Box: box}
}

func (a App) linkRows(out, in bool) []presentation.MenuRow {
	card, ok := a.dg.Card()
	if !ok {
		return nil
	}
	l := a.dg.Level
	var outs, ins []presentation.MenuRow
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
		at.Box = other
		name, place := a.boxName(other), ""
		if link.Outside != "" {
			n, _ := presentation.NodeOf(a.review.Page, other)
			at = boxOf(n)
			place = " · " + utils.ResolvePlaceholderString(a.c.Tr.InPlace, map[string]string{"place": link.OutsidePlace})
		}
		from, to := a.boxName(card.ID), name
		if way == "in" {
			from, to = name, a.boxName(card.ID)
		}
		row := presentation.MenuRow{Left: map[string]string{"out": a.c.Tr.LinkOut, "in": a.c.Tr.LinkIn}[way], Right: from + " ─ " + link.Words + " ─▶ " + to + place, Open: &presentation.Item{Kind: "box", Box: &at}}
		if way == "out" {
			outs = append(outs, row)
		} else {
			ins = append(ins, row)
		}
	}
	return append(outs, ins...)
}

func (a App) boxName(id string) string {
	if n, ok := presentation.NodeOf(a.review.Page, id); ok {
		return n.Name
	}
	if i := a.dg.Level.Index(id); i >= 0 {
		return a.dg.Level.Cards[i].Title
	}
	return id
}

func (a App) linksPanel(out, in bool, none string) App {
	card, ok := a.dg.Card()
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
	a.menu = &menu{Menu: presentation.Menu{Title: title, Rows: rows}}
	return a
}

func (a App) follow() (tea.Model, tea.Cmd) {
	card, ok := a.dg.Card()
	if !ok {
		return a, nil
	}
	rows := a.linkRows(true, false)
	switch len(rows) {
	case 0:
		return a.say(utils.ResolvePlaceholderString(a.c.Tr.LinksToNothing, map[string]string{"title": card.Title}), true), nil
	case 1:
		a = a.leave()
		return a.showBox(*rows[0].Open.Box), nil
	}
	a.menu = &menu{Menu: presentation.Menu{Title: utils.ResolvePlaceholderString(a.c.Tr.LinksTo, map[string]string{"name": a.boxName(card.ID)}), Rows: rows}}
	return a, nil
}

func (a App) membersPanel(card diagram.Card) App {
	pn := menu{Menu: presentation.Menu{Title: card.Title + " · " + utils.Plural(len(card.Members), a.c.Tr.MemberNoun)}}
	for _, id := range card.Members {
		n, ok := presentation.NodeOf(a.review.Page, id)
		if !ok {
			continue
		}
		pn.Rows = append(pn.Rows, presentation.MenuRow{Left: n.Name, Right: path.Base(n.File) + "  " + diagram.StateWord(a.c.Tr, n.State), Open: &presentation.Item{Kind: "line", Place: nodePlace(n)}})
	}
	a.menu = &pn
	return a
}

func (a App) propertiesPanel() App {
	card, ok := a.dg.Card()
	if !ok {
		return a
	}
	tr := a.c.Tr
	n, ok := presentation.NodeOf(a.review.Page, card.ID)
	if !ok || card.Kind != "type" {
		return a.say(tr.OnlyTypeHasProperties, true)
	}
	if len(n.Properties) == 0 {
		return a.say(utils.ResolvePlaceholderString(tr.NoStoredProperties, map[string]string{"name": n.Name}), true)
	}
	holds := map[string]string{}
	for _, e := range a.review.Page.Model.Edges {
		if e.Source == n.ID && e.Kind == "has" {
			target, _ := presentation.NodeOf(a.review.Page, e.Target)
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
	pn := menu{Menu: presentation.Menu{Title: n.Name + " · " + utils.Plural(len(n.Properties), tr.PropertyNoun)}}
	for _, p := range n.Properties {
		inferred := ""
		if p.Inferred {
			inferred = tr.Inferred
		}
		right := fmt.Sprintf("%-*s  %-8s  %-*s  %s", typeWidth, p.Type, inferred, lineWidth, lineAt(p.Line), holds[p.Name])
		pn.Rows = append(pn.Rows, presentation.MenuRow{Left: p.Name, Right: strings.TrimRight(right, " "), Open: &presentation.Item{Kind: "line", Place: &server.Lines{Path: n.File, Side: "RIGHT", Line: p.Line}}})
	}
	a.menu = &pn
	return a
}

func (a App) findBox(back bool) (tea.Model, tea.Cmd) {
	query := strings.ToLower(a.dg.query)
	if query == "" {
		return a, nil
	}
	nodes := a.review.Page.Model.Nodes
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
		if at := boxOf(n); here.Level == "types" && at == here {
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
	if v.Cursor >= len(v.Rows) {
		return a, nil
	}
	r := v.Rows[v.Cursor]
	var types []server.Node
	for _, n := range a.review.Page.Model.Nodes {
		if n.File == r.Path {
			types = append(types, n)
		}
	}
	tr := a.c.Tr
	if len(types) == 0 {
		return a.say(utils.ResolvePlaceholderString(tr.NoTypeInFile, map[string]string{"file": path.Base(r.Path)}), true), nil
	}
	sort.SliceStable(types, func(i, j int) bool { return types[i].Line < types[j].Line })
	if r.Kind == "file" {
		pn := menu{Menu: presentation.Menu{Title: utils.ResolvePlaceholderString(tr.TypesIn, map[string]string{"file": path.Base(r.Path)})}}
		for _, n := range types {
			at := boxOf(n)
			pn.Rows = append(pn.Rows, presentation.MenuRow{Left: n.Name, Right: n.Kind + " · " + utils.ResolvePlaceholderString(tr.LineAt, map[string]string{"line": utils.Itoa(n.Line)}), Open: &presentation.Item{Kind: "box", Box: &at}})
		}
		a.menu = &pn
		return a, nil
	}
	_, line, ok := r.At()
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

func (a App) diagramHints() [][2]string {
	tr := a.c.Tr
	card, _ := a.dg.Card()
	switch card.Kind {
	case "type", "file":
		return [][2]string{{"Enter", tr.Diff}, {"gd", tr.Follow}, {"Space l", tr.Links}, {"Space p", tr.Properties}, {"-", tr.Out}}
	case "group":
		return [][2]string{{"Enter", tr.Members}, {"gd", tr.Follow}, {"Space l", tr.Links}, {"-", tr.Out}}
	}
	out := tr.Out
	if a.dg.Level.Kind == "components" {
		out = tr.ReviewView
	}
	return [][2]string{{"h j k l", tr.Move}, {"Enter", tr.In}, {"Space d", tr.Diff}, {"/", tr.Find}, {"f", tr.Labels}, {"-", out}}
}
