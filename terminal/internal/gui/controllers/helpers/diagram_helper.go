package helpers

import (
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/diagram"
	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

const labelLetters = "asdfghjklqwertyuiopzxcvbnm"

// DiagramHelper moves through the diagram's levels and boxes, and lists a
// box's links, members and properties, for the diagram in the main panel
// and the Diagram panel alike.
type DiagramHelper struct {
	c *HelperCommon
}

// OpenDiagram is the state with the diagram in the main panel, the keys
// there, driven by the Diagram panel.
func (self *DiagramHelper) OpenDiagram(s context.ContextTree) context.ContextTree {
	if s.Review == nil {
		return s.Said(utils.ResolvePlaceholderString(self.c.Tr.NoReviewOpen, map[string]string{"open": self.c.KeyLabels(context.HomeKey).Of("open")}), true)
	}
	s = s.Focused(context.MainPanel, context.DiagramPanel)
	if s.Screen() == context.ScreenDiagram {
		return s
	}
	s = self.c.Helpers.Jump.Leave(s)
	return self.Shown(s.Showing(context.ScreenDiagram))
}

// ShowBox is the state with the diagram on the level of a place, focused
// on its box.
func (self *DiagramHelper) ShowBox(s context.ContextTree, at types.DiagramPlace) context.ContextTree {
	v := context.NewDiagramContext(self.c.Tr, context.LevelAt(self.c.Tr, s.Review.Page, at), at.Box)
	v.All, v.Query = s.Diagram.All, s.Diagram.Query
	s.Diagram = v
	s = s.Showing(context.ScreenDiagram).Closed(context.LabelsKey)
	return self.Shown(s)
}

// Shown is the state with the diagram scrolled to show the box in focus.
func (self *DiagramHelper) Shown(s context.ContextTree) context.ContextTree {
	window := self.c.Helpers.Window
	s.Diagram = s.Diagram.ShownIn(self.c.Tr, *s.Review, window.MainWidth(s), window.MainHeight(s), s.ShownPulls())
	return s
}

// Nearest is the state with the focus on the nearest box in a direction,
// one of h, j, k and l, count times over.
func (self *DiagramHelper) Nearest(s context.ContextTree, dir string, count int) context.ContextTree {
	v := s.Diagram
	for ; count > 0; count-- {
		i := v.Grid.Nearest(v.Focus, dir, context.RowLines)
		if i < 0 {
			break
		}
		v.Focus = i
	}
	s.Diagram = v
	return self.Shown(s)
}

// MovedBy is the state with the focus moved by boxes in reading order.
func (self *DiagramHelper) MovedBy(s context.ContextTree, by int) context.ContextTree {
	s.Diagram = s.Diagram.Moved(by)
	return self.Shown(s)
}

// AllLinks is the state with every link drawn faintly, or no longer.
func (self *DiagramHelper) AllLinks(s context.ContextTree) context.ContextTree {
	s.Diagram.All = !s.Diagram.All
	return self.Shown(s)
}

// Labelled is the state with a label on each box, a key to jump to it.
func (self *DiagramHelper) Labelled(s context.ContextTree) context.ContextTree {
	v := s.Diagram
	order := v.Grid.Order()
	v.Labels, v.Typed = map[string]int{}, ""
	for i, at := range order {
		v.Labels[label(i, len(order))] = at
	}
	s = s.Opened(context.LabelsKey)
	s.Diagram = v
	return s
}

// TypeLabel is the state with a key typed toward a box's label: the box
// in focus once a label is whole, the labels kept while one begins with
// what is typed, or a notice that none does. Esc, or any key named by more
// than one character, puts the labels away.
func (self *DiagramHelper) TypeLabel(s context.ContextTree, name string) context.ContextTree {
	v := s.Diagram
	typed := v.Typed + name
	v.Labels, v.Typed = nil, ""
	s = s.Closed(context.LabelsKey)
	if name == "esc" || len([]rune(name)) != 1 {
		s.Diagram = v
		return s
	}
	labels := s.Diagram.Labels
	if at, ok := labels[typed]; ok {
		v.Focus = at
		s.Diagram = v
		return self.Shown(s)
	}
	for l := range labels {
		if strings.HasPrefix(l, typed) {
			v.Labels, v.Typed = labels, typed
			s = s.Opened(context.LabelsKey)
			s.Diagram = v
			return s
		}
	}
	s.Diagram = v
	return s.Said(utils.ResolvePlaceholderString(self.c.Tr.NoBoxHasLabel, map[string]string{"label": typed}), true)
}

// StepIn is the state with the diagram a level into the box in focus, or
// its members listed, or a type's diff open.
func (self *DiagramHelper) StepIn(s context.ContextTree) (context.ContextTree, tea.Cmd) {
	card, ok := s.Diagram.Card()
	if !ok {
		return s, nil
	}
	switch card.Kind {
	case "component":
		packages, ok := diagram.Packages(self.c.Tr, s.Review.Page, card.ID)
		switch {
		case ok:
			return self.ShowBox(s, types.DiagramPlace{Level: "packages", Component: card.ID}), nil
		case len(packages.Cards) == 1:
			return self.ShowBox(s, types.DiagramPlace{Level: "types", Component: card.ID, Package: packages.Cards[0].ID}), nil
		}
		return s.Said(utils.ResolvePlaceholderString(self.c.Tr.NoTypesToDraw, map[string]string{"title": card.Title, "boxDiff": self.c.KeyLabels(s.Place()).Of("box diff")}), true), nil
	case "package":
		return self.ShowBox(s, types.DiagramPlace{Level: "types", Component: s.Diagram.Level.Component, Package: card.ID}), nil
	case "group":
		return self.membersPanel(s, card), nil
	}
	return self.BoxDiff(s)
}

// StepOut is the state with the diagram a level out, or the Review view
// from the first level.
func (self *DiagramHelper) StepOut(s context.ContextTree) context.ContextTree {
	l := s.Diagram.Level
	switch l.Kind {
	case "types":
		if _, ok := diagram.Packages(self.c.Tr, s.Review.Page, l.Component); ok {
			return self.ShowBox(s, types.DiagramPlace{Level: "packages", Component: l.Component, Box: l.Package})
		}
		return self.ShowBox(s, types.DiagramPlace{Level: "components", Box: l.Component})
	case "packages":
		return self.ShowBox(s, types.DiagramPlace{Level: "components", Box: l.Component})
	}
	s = self.c.Helpers.Jump.Leave(s)
	s = s.Showing(context.ScreenReview)
	return self.c.Helpers.Refresh.Relaid(s)
}

// BoxDiff is the state with the diff of the box in focus open.
func (self *DiagramHelper) BoxDiff(s context.ContextTree) (context.ContextTree, tea.Cmd) {
	h := self.c.Helpers
	card, ok := s.Diagram.Card()
	if !ok {
		return s, nil
	}
	switch card.Kind {
	case "component":
		s = h.Jump.Leave(s)
		return h.Diff.OpenPlace(s, card.ID, nil)
	case "package":
		for _, f := range presentation.DiffFiles(*s.Review, s.Diagram.Level.Component, nil) {
			if f.Package == card.ID {
				s = h.Jump.Leave(s)
				return h.Diff.OpenPlace(s, f.Component, &server.Lines{Path: f.Path, Side: "file:"})
			}
		}
		return s.Said(utils.ResolvePlaceholderString(self.c.Tr.NoFilesInChange, map[string]string{"title": card.Title}), true), nil
	case "group":
		return self.openNode(s, card.Members[0])
	}
	return self.openNode(s, card.ID)
}

// Links is the state with the links of the box in focus listed, out then
// in, or a notice that it has none.
func (self *DiagramHelper) Links(s context.ContextTree) context.ContextTree {
	return self.linksPanel(s, true, true, self.c.Tr.HasNoLinks)
}

// References is the state with the boxes that link to the one in focus
// listed, or a notice that none does.
func (self *DiagramHelper) References(s context.ContextTree) context.ContextTree {
	return self.linksPanel(s, false, true, self.c.Tr.NothingLinksTo)
}

// Follow is the state with the diagram on the box the one in focus links
// to, or the boxes listed when it links to more than one.
func (self *DiagramHelper) Follow(s context.ContextTree) (context.ContextTree, tea.Cmd) {
	card, ok := s.Diagram.Card()
	if !ok {
		return s, nil
	}
	rows := self.linkRows(s, true, false)
	switch len(rows) {
	case 0:
		return s.Said(utils.ResolvePlaceholderString(self.c.Tr.LinksToNothing, map[string]string{"title": card.Title}), true), nil
	case 1:
		s = self.c.Helpers.Jump.Leave(s)
		return self.ShowBox(s, *rows[0].Open.Box), nil
	}
	title := utils.ResolvePlaceholderString(self.c.Tr.LinksTo, map[string]string{"name": self.boxName(s, card.ID)})
	return s.WithMenu(context.MenuContext{Menu: presentation.Menu{Title: title, Rows: rows}}), nil
}

// Properties is the state with the stored properties of the type in focus
// listed, or a notice of why not.
func (self *DiagramHelper) Properties(s context.ContextTree) context.ContextTree {
	card, ok := s.Diagram.Card()
	if !ok {
		return s
	}
	tr := self.c.Tr
	n, ok := presentation.NodeOf(s.Review.Page, card.ID)
	if !ok || card.Kind != "type" {
		return s.Said(tr.OnlyTypeHasProperties, true)
	}
	if len(n.Properties) == 0 {
		return s.Said(utils.ResolvePlaceholderString(tr.NoStoredProperties, map[string]string{"name": n.Name}), true)
	}
	holds := map[string]string{}
	for _, e := range s.Review.Page.Model.Edges {
		if e.Source == n.ID && e.Kind == "has" {
			target, _ := presentation.NodeOf(s.Review.Page, e.Target)
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
	pn := context.MenuContext{Menu: presentation.Menu{Title: n.Name + " · " + utils.Plural(len(n.Properties), tr.PropertyNoun)}}
	for _, p := range n.Properties {
		inferred := ""
		if p.Inferred {
			inferred = tr.Inferred
		}
		right := fmt.Sprintf("%-*s  %-8s  %-*s  %s", typeWidth, p.Type, inferred, lineWidth, lineAt(p.Line), holds[p.Name])
		pn.Rows = append(pn.Rows, presentation.MenuRow{Left: p.Name, Right: strings.TrimRight(right, " "), Open: &presentation.Item{Kind: "line", Place: &server.Lines{Path: n.File, Side: "RIGHT", Line: p.Line}}})
	}
	return s.WithMenu(pn)
}

// FindBox is the state with the diagram on the next type or file whose
// name holds what was searched for, or the previous.
func (self *DiagramHelper) FindBox(s context.ContextTree, back bool) (context.ContextTree, tea.Cmd) {
	query := strings.ToLower(s.Diagram.Query)
	if query == "" {
		return s, nil
	}
	nodes := s.Review.Page.Model.Nodes
	var hits []int
	for i, n := range nodes {
		if strings.Contains(strings.ToLower(n.Name), query) || strings.Contains(strings.ToLower(path.Base(n.File)), query) {
			hits = append(hits, i)
		}
	}
	if len(hits) == 0 {
		return s.Said(utils.ResolvePlaceholderString(self.c.Tr.NothingIsCalled, map[string]string{"query": strconv.Quote(s.Diagram.Query)}), true), nil
	}
	here := s.Diagram.Place()
	from, to := -1, len(nodes)
	for i, n := range nodes {
		if at := BoxOf(n); here.Level == "types" && at == here {
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
	s = self.c.Helpers.Jump.Leave(s)
	s = self.ShowBox(s, BoxOf(nodes[hits[pick]]))
	return s.Said(utils.ResolvePlaceholderString(self.c.Tr.MatchOfFor, map[string]string{"at": utils.Itoa(pick + 1), "total": utils.Itoa(len(hits)), "query": strconv.Quote(s.Diagram.Query)}), false), nil
}

// ShowType is the state with the diagram on the type at the diff's cursor,
// or a file's types listed from its header.
func (self *DiagramHelper) ShowType(s context.ContextTree) (context.ContextTree, tea.Cmd) {
	v := s.Diff
	if v.Cursor >= len(v.Rows) {
		return s, nil
	}
	r := v.Rows[v.Cursor]
	var nodes []server.Node
	for _, n := range s.Review.Page.Model.Nodes {
		if n.File == r.Path {
			nodes = append(nodes, n)
		}
	}
	tr := self.c.Tr
	if len(nodes) == 0 {
		return s.Said(utils.ResolvePlaceholderString(tr.NoTypeInFile, map[string]string{"file": path.Base(r.Path)}), true), nil
	}
	sort.SliceStable(nodes, func(i, j int) bool { return nodes[i].Line < nodes[j].Line })
	if r.Kind == "file" {
		pn := context.MenuContext{Menu: presentation.Menu{Title: utils.ResolvePlaceholderString(tr.TypesIn, map[string]string{"file": path.Base(r.Path)})}}
		for _, n := range nodes {
			at := BoxOf(n)
			pn.Rows = append(pn.Rows, presentation.MenuRow{Left: n.Name, Right: n.Kind + " · " + utils.ResolvePlaceholderString(tr.LineAt, map[string]string{"line": utils.Itoa(n.Line)}), Open: &presentation.Item{Kind: "box", Box: &at}})
		}
		return s.WithMenu(pn), nil
	}
	_, line, ok := r.At()
	if !ok {
		return s.Said(utils.ResolvePlaceholderString(tr.ShowTypeHow, map[string]string{"showType": self.c.KeyLabels(s.Place()).Of("show type")}), true), nil
	}
	at := nodes[0]
	for _, n := range nodes {
		if n.Line <= line {
			at = n
		}
	}
	s = self.c.Helpers.Jump.Leave(s)
	return self.ShowBox(s, BoxOf(at)), nil
}

// BoxOf is the box of the diagram a type is drawn in: its own, or its
// group's.
func BoxOf(n server.Node) types.DiagramPlace {
	box := n.ID
	if n.Group != "" {
		box = "group:" + n.Group
	}
	return types.DiagramPlace{Level: "types", Component: n.Component, Package: n.Package, Box: box}
}

func label(i, n int) string {
	if n <= len(labelLetters) {
		return labelLetters[i : i+1]
	}
	return labelLetters[i/len(labelLetters)%len(labelLetters):][:1] + labelLetters[i%len(labelLetters):][:1]
}

func (self *DiagramHelper) openNode(s context.ContextTree, id string) (context.ContextTree, tea.Cmd) {
	n, ok := presentation.NodeOf(s.Review.Page, id)
	if !ok {
		return s, nil
	}
	s = self.c.Helpers.Jump.Leave(s)
	return self.c.Helpers.Diff.OpenPlace(s, presentation.FileEntry(*s.Review, n.File).Component, nodePlace(n))
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

func (self *DiagramHelper) linkRows(s context.ContextTree, out, in bool) []presentation.MenuRow {
	card, ok := s.Diagram.Card()
	if !ok {
		return nil
	}
	l := s.Diagram.Level
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
		at := s.Diagram.Place()
		at.Box = other
		name, place := self.boxName(s, other), ""
		if link.Outside != "" {
			n, _ := presentation.NodeOf(s.Review.Page, other)
			at = BoxOf(n)
			place = " · " + utils.ResolvePlaceholderString(self.c.Tr.InPlace, map[string]string{"place": link.OutsidePlace})
		}
		from, to := self.boxName(s, card.ID), name
		if way == "in" {
			from, to = name, self.boxName(s, card.ID)
		}
		row := presentation.MenuRow{Left: map[string]string{"out": self.c.Tr.LinkOut, "in": self.c.Tr.LinkIn}[way], Right: from + " ─ " + link.Words + " ─▶ " + to + place, Open: &presentation.Item{Kind: "box", Box: &at}}
		if way == "out" {
			outs = append(outs, row)
		} else {
			ins = append(ins, row)
		}
	}
	return append(outs, ins...)
}

func (self *DiagramHelper) boxName(s context.ContextTree, id string) string {
	if n, ok := presentation.NodeOf(s.Review.Page, id); ok {
		return n.Name
	}
	if i := s.Diagram.Level.Index(id); i >= 0 {
		return s.Diagram.Level.Cards[i].Title
	}
	return id
}

func (self *DiagramHelper) linksPanel(s context.ContextTree, out, in bool, none string) context.ContextTree {
	card, ok := s.Diagram.Card()
	if !ok {
		return s
	}
	rows := self.linkRows(s, out, in)
	if len(rows) == 0 {
		return s.Said(utils.ResolvePlaceholderString(none, map[string]string{"title": card.Title}), true)
	}
	title := self.boxName(s, card.ID) + " · " + utils.Plural(len(rows), self.c.Tr.LinkNoun)
	if !out {
		title = utils.ResolvePlaceholderString(self.c.Tr.WhatLinksTo, map[string]string{"name": self.boxName(s, card.ID)})
	}
	return s.WithMenu(context.MenuContext{Menu: presentation.Menu{Title: title, Rows: rows}})
}

func (self *DiagramHelper) membersPanel(s context.ContextTree, card diagram.Card) context.ContextTree {
	pn := context.MenuContext{Menu: presentation.Menu{Title: card.Title + " · " + utils.Plural(len(card.Members), self.c.Tr.MemberNoun)}}
	for _, id := range card.Members {
		n, ok := presentation.NodeOf(s.Review.Page, id)
		if !ok {
			continue
		}
		pn.Rows = append(pn.Rows, presentation.MenuRow{Left: n.Name, Right: path.Base(n.File) + "  " + diagram.StateWord(self.c.Tr, n.State), Open: &presentation.Item{Kind: "line", Place: nodePlace(n)}})
	}
	return s.WithMenu(pn)
}
