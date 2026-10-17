package diagram

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

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
func Components(tr *i18n.TranslationSet, page server.Page) Level {
	l := Level{Kind: "components"}
	for _, c := range page.Model.Components {
		var pulls [][]int
		for _, f := range page.Model.Files {
			if f.Component == c.ID {
				pulls = append(pulls, f.Pulls)
			}
		}
		l.Cards = append(l.Cards, Card{ID: c.ID, Kind: "component", Title: c.Title, Detail: ComponentDetail(tr, page, c, everything, wholeLines), Pulls: union(pulls...), Context: c.Context, Area: c.Area, Box: page.Layout.Components[c.ID]})
	}
	for _, s := range page.Layout.Summaries {
		words := strings.Join(s.Lines, " · ")
		l.Links = append(l.Links, Link{From: s.Source, To: s.Target, Kind: "summary", Words: words, AtRest: true, Ways: s.Ways})
		if s.Both {
			l.Links = append(l.Links, Link{From: s.Target, To: s.Source, Kind: "summary", AtRest: true, Ways: s.Ways})
		}
	}
	return l
}

// Packages is the second level: the packages of a component that hold
// types, with the arrows between them. A component of one package has no
// such level, and ok is false.
func Packages(tr *i18n.TranslationSet, page server.Page, component string) (Level, bool) {
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
			var pulls [][]int
			for _, n := range page.Model.Nodes {
				if n.Component == component && n.Package == p.ID {
					pulls = append(pulls, n.Pulls)
				}
			}
			l.Cards = append(l.Cards, Card{ID: p.ID, Kind: "package", Title: title, Detail: PackageDetail(tr, page, c, p, everything), Pulls: union(pulls...), Context: c.Context, Box: box})
		}
	}
	for _, a := range page.Layout.PkgArrows {
		if a.Component == component {
			words := a.CountText
			if a.Words != "" {
				words = a.Words + " · " + a.CountText
			}
			link := Link{From: a.From, To: a.To, Kind: "package", Words: words, AtRest: true}
			if a.Links != nil {
				link.Ways = []server.ArrowWay{{Label: a.Words, Links: a.Links}}
			}
			l.Links = append(l.Links, link)
		}
	}
	return l, len(l.Cards) > 1
}

// Types is the third level: the types of one package, a group of look-alike
// types as one card, and every link of each, those to a type elsewhere
// ending outside the level.
func Types(tr *i18n.TranslationSet, page server.Page, component, pkg string) Level {
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
				detail := utils.ResolvePlaceholderString(tr.GroupOf, map[string]string{"count": utils.Itoa(len(g.Members))})
				l.Cards = append(l.Cards, Card{ID: id, Kind: "group", Title: g.Title, Detail: detail, State: n.State, Members: g.Members, Box: page.Layout.Boxes[id]})
			}
			card := &l.Cards[l.Index(id)]
			card.Pulls = union(card.Pulls, n.Pulls)
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
		link := Link{From: from, To: to, Kind: e.Kind, Words: Words(tr, e), AtRest: atRest(e)}
		switch {
		case !fromIn:
			link.From, link.AtRest = e.Source, false
			link.Outside, link.OutsideName, link.OutsidePlace = outside(tr, page, nodes[e.Source])
		case !toIn:
			link.To, link.AtRest = e.Target, false
			link.Outside, link.OutsideName, link.OutsidePlace = outside(tr, page, nodes[e.Target])
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
func Words(tr *i18n.TranslationSet, e server.Edge) string {
	words := e.Label
	if words == "" {
		words = kindWords(tr)[e.Kind]
	}
	if e.Kind == "has" && len(e.Properties) > 0 {
		words += " " + strings.Join(e.Properties, ", ")
	}
	return words
}

func kindWords(tr *i18n.TranslationSet) map[string]string {
	return map[string]string{"has": tr.LinkHolds, "realize": tr.LinkConformsTo, "inherit": tr.LinkInheritsFrom, "extend": tr.LinkExtends, "use": tr.LinkUses, "test": tr.LinkTests}
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

func outside(tr *i18n.TranslationSet, page server.Page, n server.Node) (string, string, string) {
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
	in := utils.ResolvePlaceholderString(tr.InPlace, map[string]string{"place": place})
	return name + " · " + in, name, place
}

// ComponentDetail is a component card's second line: an unchanged
// component's count of files, or a changed one's count of the files shows
// takes and the lines that lines counts of them.
func ComponentDetail(tr *i18n.TranslationSet, page server.Page, c server.Component, shows func(pulls []int) bool, lines func(f server.FileEntry) (int, int)) string {
	if c.Context {
		n := 0
		for _, f := range page.Model.ContextFiles {
			if f.Component == c.ID {
				n++
			}
		}
		return count(n, tr.FileNoun) + " · " + tr.Unchanged
	}
	n, adds, dels := 0, 0, 0
	for _, f := range page.Model.Files {
		if f.Component == c.ID && shows(f.Pulls) {
			a, d := lines(f)
			n, adds, dels = n+1, adds+a, dels+d
		}
	}
	if n == 0 {
		return count(n, tr.FileNoun)
	}
	return fmt.Sprintf("%s +%d −%d", count(n, tr.FileNoun), adds, dels)
}

// PackageDetail is a package card's second line: the count of its types
// that shows takes, or of its files when it holds no types; every one in an
// unchanged component.
func PackageDetail(tr *i18n.TranslationSet, page server.Page, c server.Component, p server.Package, shows func(pulls []int) bool) string {
	types, files, holdsTypes := 0, 0, false
	for _, n := range page.Model.Nodes {
		if n.Component != c.ID || n.Package != p.ID {
			continue
		}
		holdsTypes = holdsTypes || n.Kind != "artifact"
		if !c.Context && !shows(n.Pulls) {
			continue
		}
		if n.Kind == "artifact" {
			files++
		} else {
			types++
		}
	}
	if !holdsTypes {
		return count(files, tr.FileNoun)
	}
	return count(types, tr.TypeNoun)
}

func everything([]int) bool { return true }

func wholeLines(f server.FileEntry) (int, int) { return f.Adds, f.Dels }

func union(lists ...[]int) []int {
	var out []int
	for _, l := range lists {
		out = append(out, l...)
	}
	slices.Sort(out)
	return slices.Compact(out)
}

func count(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}
