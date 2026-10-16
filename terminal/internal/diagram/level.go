package diagram

import (
	"fmt"
	"path"
	"strings"

	"review-desk/terminal/internal/server"
)

var kindWords = map[string]string{"has": "holds", "realize": "conforms to", "inherit": "inherits from", "extend": "extends", "use": "uses", "test": "tests"}

// Level is the cards of one container of the diagram and the links between
// them: the components, the packages of Component, or the types of Package.
type Level struct {
	Kind      string
	Component string
	Package   string
	Cards     []Card
	Links     []Link
}

// Index is the position of the card with id, or -1.
func (l Level) Index(id string) int {
	for i, c := range l.Cards {
		if c.ID == id {
			return i
		}
	}
	return -1
}

// Components is the first level: every component, with the arrows between them.
func Components(page server.Page) Level {
	l := Level{Kind: "components"}
	for _, c := range page.Model.Components {
		l.Cards = append(l.Cards, Card{ID: c.ID, Kind: "component", Title: c.Title, Detail: componentDetail(page, c), Context: c.Context, Area: c.Area, Box: page.Layout.Components[c.ID]})
	}
	for _, s := range page.Layout.Summaries {
		words := strings.Join(s.Lines, " · ")
		l.Links = append(l.Links, Link{From: s.Source, To: s.Target, Kind: "summary", Words: words, AtRest: true})
		if s.Both {
			l.Links = append(l.Links, Link{From: s.Target, To: s.Source, Kind: "summary", AtRest: true})
		}
	}
	return l
}

// Packages is the second level: the packages of a component that hold
// types, with the arrows between them. A component of one package has no
// such level, and ok is false.
func Packages(page server.Page, component string) (Level, bool) {
	l := Level{Kind: "packages", Component: component}
	for _, c := range page.Model.Components {
		if c.ID != component {
			continue
		}
		for _, p := range c.Packages {
			box, drawn := page.Layout.Boxes["pkg:"+component+":"+p.ID]
			if !drawn {
				continue
			}
			title := p.Title
			if title == "" {
				title = p.Path
			}
			l.Cards = append(l.Cards, Card{ID: p.ID, Kind: "package", Title: title, Detail: packageDetail(page, component, p), Context: c.Context, Box: box})
		}
	}
	for _, a := range page.Layout.PkgArrows {
		if a.Component == component {
			words := a.CountText
			if a.Words != "" {
				words = a.Words + " · " + a.CountText
			}
			l.Links = append(l.Links, Link{From: a.From, To: a.To, Kind: "package", Words: words, AtRest: true})
		}
	}
	return l, len(l.Cards) > 1
}

// Types is the third level: the types of one package, a group of look-alike
// types as one card, and every link of each, those to a type elsewhere
// ending outside the level.
func Types(page server.Page, component, pkg string) Level {
	l := Level{Kind: "types", Component: component, Package: pkg}
	cardOf := map[string]string{}
	groups := map[string]server.Group{}
	for _, g := range page.Model.Groups {
		groups[g.ID] = g
	}
	for _, n := range page.Model.Nodes {
		if n.Component != component || n.Package != pkg {
			continue
		}
		if n.Group != "" {
			id := "group:" + n.Group
			cardOf[n.ID] = id
			if l.Index(id) < 0 {
				g := groups[n.Group]
				l.Cards = append(l.Cards, Card{ID: id, Kind: "group", Title: g.Title, Detail: fmt.Sprintf("a group of %d", len(g.Members)), State: n.State, Members: g.Members, Box: page.Layout.Boxes[id]})
			}
			continue
		}
		cardOf[n.ID] = n.ID
		l.Cards = append(l.Cards, nodeCard(n, page.Layout.Boxes[n.ID]))
	}
	nodes := map[string]server.Node{}
	for _, n := range page.Model.Nodes {
		nodes[n.ID] = n
	}
	seen := map[string]bool{}
	for _, e := range page.Model.Edges {
		from, fromIn := cardOf[e.Source]
		to, toIn := cardOf[e.Target]
		if !fromIn && !toIn || from == to && fromIn {
			continue
		}
		link := Link{From: from, To: to, Kind: e.Kind, Words: Words(e), AtRest: atRest(e)}
		switch {
		case !fromIn:
			link.From, link.Outside, link.AtRest = e.Source, outside(page, nodes[e.Source]), false
		case !toIn:
			link.To, link.Outside, link.AtRest = e.Target, outside(page, nodes[e.Target]), false
		}
		key := link.From + ">" + link.To + ">" + link.Words
		if !seen[key] {
			seen[key] = true
			l.Links = append(l.Links, link)
		}
	}
	return l
}

// Words are a link's words: its config's label, or its kind's, a has link
// naming the properties that hold its target.
func Words(e server.Edge) string {
	words := e.Label
	if words == "" {
		words = kindWords[e.Kind]
	}
	if e.Kind == "has" && len(e.Properties) > 0 {
		words += " " + strings.Join(e.Properties, ", ")
	}
	return words
}

func atRest(e server.Edge) bool {
	switch e.Kind {
	case "realize", "inherit", "has", "extend":
		return true
	case "use":
		return e.Key
	}
	return false
}

func nodeCard(n server.Node, box server.Box) Card {
	title := strings.TrimSpace(n.Stereotype + " " + n.Name)
	kind := "type"
	if n.Kind == "artifact" {
		kind, title = "file", n.Name
	}
	return Card{ID: n.ID, Kind: kind, Title: title, Detail: path.Base(n.File), State: n.State, Pulls: n.Pulls, Box: box}
}

func outside(page server.Page, n server.Node) string {
	place := n.Component
	for _, c := range page.Model.Components {
		if c.ID != n.Component {
			continue
		}
		place = c.Title
		for _, p := range c.Packages {
			if p.ID == n.Package && !p.Bare && len(c.Packages) > 1 {
				place += " › " + p.Title
			}
		}
	}
	name := n.Name
	if name == "" {
		name = n.ID
	}
	return name + " · in " + place
}

func componentDetail(page server.Page, c server.Component) string {
	if c.Context {
		n := 0
		for _, f := range page.Model.ContextFiles {
			if f.Component == c.ID {
				n++
			}
		}
		return count(n, "file") + " · unchanged"
	}
	n, adds, dels := 0, 0, 0
	for _, f := range page.Model.Files {
		if f.Component == c.ID {
			n, adds, dels = n+1, adds+f.Adds, dels+f.Dels
		}
	}
	return fmt.Sprintf("%s +%d −%d", count(n, "file"), adds, dels)
}

func packageDetail(page server.Page, component string, p server.Package) string {
	types, files := 0, 0
	for _, n := range page.Model.Nodes {
		if n.Component == component && n.Package == p.ID {
			if n.Kind == "artifact" {
				files++
			} else {
				types++
			}
		}
	}
	if types == 0 {
		return count(files, "file")
	}
	return count(types, "type")
}

func count(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}
