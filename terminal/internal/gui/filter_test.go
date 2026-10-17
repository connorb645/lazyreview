package gui

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/diagram"
	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	draw "github.com/connorb645/lazyreview/terminal/internal/gui/presentation/diagram"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/gui/views"
)

// filterStandIn is a made-up stack of #21 and #22: Order.swift changed by
// both, Rules.swift by #22 alone and Book.swift by #21 alone, with an
// unchanged Clock.swift for context. Core links to Clock three times, and
// Ledger to Core once.
func filterStandIn() types.ReviewData {
	box := func(x, y float64) server.Box { return server.Box{X: x, Y: y, H: 60} }
	file := func(path, component, pkg string, adds, dels int, byPull map[int][2]int) server.FileEntry {
		var pulls []int
		for pull := range byPull {
			pulls = append(pulls, pull)
		}
		slices.Sort(pulls)
		return server.FileEntry{Path: path, Status: "M", Adds: adds, Dels: dels, Pulls: pulls, ByPull: byPull, Component: component, Package: pkg}
	}
	node := func(id, path, component, pkg string, pulls ...int) server.Node {
		state := "changed"
		if len(pulls) == 0 {
			state = "same"
		}
		return server.Node{ID: id, Name: id, Kind: "class", File: path, Line: 1, Component: component, Package: pkg, State: state, Pulls: pulls}
	}
	nodes := []server.Node{
		node("Order", "Sources/Core/Order.swift", "core", "model", 21, 22),
		node("Pricer", "Sources/Core/Order.swift", "core", "model"),
		node("Refund", "Sources/Core/Order.swift", "core", "model", 22),
		node("Return", "Sources/Core/Order.swift", "core", "model", 22),
		node("Rule", "Sources/Core/Rules/Rules.swift", "core", "rules", 22),
		node("Book", "Sources/Ledger/Book.swift", "ledger", "books", 21),
		{ID: "Clock", Name: "Clock", Kind: "class", File: "Sources/Clock.swift", Line: 1, Component: "context", Package: "clocks", State: "context"},
	}
	nodes[2].Group, nodes[3].Group = "kinds", "kinds"
	return types.ReviewData{ID: "filter-stand-in",
		State: server.State{Kind: "stack", Members: []server.Member{{Pull: 21, Title: "Add orders", Color: 1}, {Pull: 22, Title: "Price the orders", Color: 2}}},
		Page: server.Page{
			Stack: &server.PageStack{Members: []server.StackMember{{Pull: 21}, {Pull: 22}}},
			Model: server.Model{
				Components: []server.Component{
					{ID: "core", Title: "Core", Color: "teal", Packages: []server.Package{{ID: "model", Title: "Model"}, {ID: "rules", Title: "Rules"}}},
					{ID: "ledger", Title: "Ledger", Color: "blue", Packages: []server.Package{{ID: "books", Title: "Books", Bare: true}}},
					{ID: "context", Title: "Unchanged code", Color: "grey", Context: true, Packages: []server.Package{{ID: "clocks", Title: "Clocks", Bare: true}}},
				},
				Files: []server.FileEntry{
					file("Sources/Core/Order.swift", "core", "model", 3, 1, map[int][2]int{21: {2, 0}, 22: {1, 1}}),
					file("Sources/Core/Rules/Rules.swift", "core", "rules", 4, 0, map[int][2]int{22: {4, 0}}),
					file("Sources/Ledger/Book.swift", "ledger", "books", 5, 2, map[int][2]int{21: {5, 2}}),
				},
				ContextFiles: []server.FileEntry{{Path: "Sources/Clock.swift", Component: "context", Lines: 12}},
				Nodes:        nodes,
				Groups:       []server.Group{{ID: "kinds", Title: "Refunds", Members: []string{"Refund", "Return"}}},
			},
			Layout: server.Layout{
				Components: map[string]server.Box{"core": box(0, 0), "ledger": box(300, 0), "context": box(600, 0)},
				Boxes: map[string]server.Box{"pkg:core:model": box(0, 0), "pkg:core:rules": box(300, 0), "pkg:context:clocks": box(600, 0),
					"Order": box(0, 0), "Pricer": box(200, 0), "group:kinds": box(400, 0), "Clock": box(0, 0)},
				Summaries: []server.ComponentArrow{
					{Source: "core", Target: "context", Lines: []string{"→ 3 links"}, Ways: []server.ArrowWay{{Arrow: "→", Links: [][2]string{{"Order", "Clock"}, {"group:kinds", "Clock"}, {"Pricer", "Clock"}}}}},
					{Source: "ledger", Target: "core", Lines: []string{"→ books · 1 link"}, Ways: []server.ArrowWay{{Arrow: "→", Label: "books", Links: [][2]string{{"Book", "Order"}}}}},
				},
				PkgArrows: []server.PackageArrow{{Component: "core", From: "model", To: "rules", Words: "prices", CountText: "1 link", Links: [][2]string{{"Order", "Rule"}}}},
			},
		}}
}

func cardDetails(d types.ReviewData, l diagram.Level, filter types.PullFilter) map[string]string {
	tr := testCommon().Tr
	out := map[string]string{}
	for _, card := range l.Cards {
		out[card.ID] = presentation.CardDetail(tr, d, l, card, filter)
	}
	return out
}

func dimmedCards(l diagram.Level, filter types.PullFilter) string {
	var out []string
	for _, card := range l.Cards {
		if presentation.Dimmed(card, filter) {
			out = append(out, card.ID)
		}
	}
	return strings.Join(out, " ")
}

func TestAComponentCardCountsOnlyWhatTheFilterShows(t *testing.T) {
	d := filterStandIn()
	l := diagram.Components(testCommon().Tr, d.Page)
	for filter, want := range map[string]map[string]string{
		"":   {"core": "2 files +7 −1", "ledger": "1 file +5 −2", "context": "1 file · unchanged"},
		"21": {"core": "1 file +2 −0", "ledger": "1 file +5 −2", "context": "1 file · unchanged"},
		"22": {"core": "2 files +5 −1", "ledger": "0 files", "context": "1 file · unchanged"},
	} {
		f := filterOf(filter)
		if got := cardDetails(d, l, f); !maps.Equal(got, want) {
			t.Errorf("under %v the component cards read %v, not %v", f, got, want)
		}
	}
	if got := dimmedCards(l, types.PullFilter{22}); got != "ledger" {
		t.Errorf("under #22 the cards dimmed are %q", got)
	}
	if got := dimmedCards(l, nil); got != "" {
		t.Errorf("with no filter the cards dimmed are %q", got)
	}
}

func TestAPackageCardCountsOnlyWhatTheFilterShows(t *testing.T) {
	d := filterStandIn()
	l, _ := diagram.Packages(testCommon().Tr, d.Page, "core")
	for filter, want := range map[string]map[string]string{
		"":   {"model": "4 types", "rules": "1 type"},
		"21": {"model": "1 type", "rules": "0 types"},
		"22": {"model": "3 types", "rules": "1 type"},
	} {
		f := filterOf(filter)
		if got := cardDetails(d, l, f); !maps.Equal(got, want) {
			t.Errorf("under %v the package cards read %v, not %v", f, got, want)
		}
	}
	if got := dimmedCards(l, types.PullFilter{21}); got != "rules" {
		t.Errorf("under #21 the package cards dimmed are %q", got)
	}
}

func TestAGroupCardDimsOnlyWhenTheFilterShowsNoneOfItsMembers(t *testing.T) {
	l := diagram.Types(testCommon().Tr, filterStandIn().Page, "core", "model")
	if got := dimmedCards(l, types.PullFilter{22}); got != "Pricer" {
		t.Errorf("under #22 the cards dimmed are %q", got)
	}
	if got := dimmedCards(l, types.PullFilter{21}); got != "Pricer group:kinds" {
		t.Errorf("under #21 the cards dimmed are %q", got)
	}
}

func TestTheFilesPanelTheReviewViewAndTheDiffCountOnlyWhatTheFilterShows(t *testing.T) {
	d := filterStandIn()
	tr, th := testCommon().Tr, testCommon().Theme
	rows := presentation.FileRows(tr, th, d, map[string]bool{}, types.PullFilter{22})
	if rows[0].Key != presentation.ComponentKey("core") || rows[0].Right != "+5 −1" {
		t.Errorf("under #22 the Files panel's first row is %s %q", rows[0].Key, rows[0].Right)
	}
	order := presentation.FileEntry(d, "Sources/Core/Order.swift")
	if adds, dels := types.PullFilter(nil).Lines(order); adds != 3 || dels != 1 {
		t.Errorf("with no filter Order.swift counts +%d −%d", adds, dels)
	}
	if adds, dels := (types.PullFilter{22}).Lines(order); adds != 1 || dels != 1 {
		t.Errorf("under #22 Order.swift counts +%d −%d", adds, dels)
	}
	v := presentation.DiffView{Width: 80, Rows: []presentation.DiffRow{{Kind: "file", Path: "Sources/Core/Order.swift"}}, Folded: map[string]bool{}}
	header := strings.Join(presentation.DiffLines(tr, th, presentation.KeyLabels{}, d, v, 1, types.PullFilter{22}, false), "")
	if !strings.HasSuffix(strings.TrimRight(header, " "), "+1 −1") {
		t.Errorf("under #22 Order.swift's header reads %q", header)
	}
	l := diagram.Components(tr, d.Page)
	if got := presentation.BoxWords(tr, d, l, l.Cards[0], types.PullFilter{22}, 80); got != "Core · component · 2 files +5 −1" {
		t.Errorf("under #22 the foot names Core as %q", got)
	}
}

func TestARunOfEveryPullRequestIsTheWholeStack(t *testing.T) {
	whole, _ := opened(t, stack, 100, 40)
	a := press(t, whole, "j", "v", "j")
	filterIs(t, a, "[21 22]")
	if len(a.ShownPulls()) != 0 {
		t.Fatalf("a run of every pull request shows %v", a.ShownPulls())
	}
	if strings.Contains(a.View(), "only") {
		t.Fatalf("a run of every pull request says only:\n%s", a.View())
	}
	if got := litKeys(a); got != "all pull:21 pull:22" {
		t.Fatalf("a run of every pull request lights %q", got)
	}
	for _, n := range []int{context.FilesPanel, context.NotesPanel, context.DiagramPanel} {
		if got, want := strings.Join(rowKeys(a, n), " "), strings.Join(rowKeys(whole, n), " "); got != want {
			t.Errorf("panel %d shows %s, not the whole stack's %s", n, got, want)
		}
	}
}

func TestWithNoColourWhatTheFilterLeavesOutIsMarked(t *testing.T) {
	a, _ := opened(t, stack, 100, 30)
	a = press(t, a, "j")
	if view := a.View(); !strings.Contains(view, "│·All ") || !strings.Contains(view, "│▌#21 ") || !strings.Contains(view, "│·#22 Add a basket") {
		t.Fatalf("with no colour the Review panel does not mark what #21 leaves out:\n%s", view)
	}
	d := filterStandIn()
	tr, th := testCommon().Tr, testCommon().Theme
	v := presentation.DiffView{Width: 60, Rows: []presentation.DiffRow{
		{Kind: "line", Path: "Sources/Core/Order.swift", Line: &server.Row{Kind: "add", New: 3, Text: "let total = 0", Owner: 21}},
		{Kind: "line", Path: "Sources/Core/Order.swift", Line: &server.Row{Kind: "add", New: 4, Text: "let tax = 0", Owner: 22}},
	}}
	lines := presentation.DiffLines(tr, th, presentation.KeyLabels{}, d, v, 2, types.PullFilter{21}, false)
	if !strings.HasPrefix(lines[0], " ▎") || !strings.HasPrefix(lines[1], " ·") {
		t.Fatalf("with no colour a line #22 changed reads like #21's: %q", lines)
	}
	l := diagram.Types(tr, d.Page, "core", "model")
	scene := draw.Scene{Level: l, Grid: diagram.Arrange(tr, l.Cards, 4), Focus: -1}
	c, tops, _ := draw.Draw(tr, d, scene, types.PullFilter{21})
	for i, card := range l.Cards {
		s := scene.Grid.Spots[i]
		corner := c.Cells[tops[s.Row]][s.Col].R
		if dim := presentation.Dimmed(card, types.PullFilter{21}); dim != (corner == '·') {
			t.Errorf("%s, dimmed %v, has the corner %q", card.ID, dim, corner)
		}
	}
	plain := th
	plain.Plain = true
	out, _ := presentation.SideContent(tr, plain, []presentation.SideRow{{Key: "a", Text: "Order"}, {Key: "b", Text: "Pricer", Out: true}}, 0, 0, views.Rect{W: 20, H: 2}, false)
	if !strings.HasPrefix(out[1], "·Pricer") {
		t.Fatalf("with no colour a row the filter leaves out reads %q", out[1])
	}
}

func TestUnchangedCodeIsNeverLeftOutByAFilter(t *testing.T) {
	d := filterStandIn()
	tr := testCommon().Tr
	l := diagram.Components(tr, d.Page)
	if got := dimmedCards(l, types.PullFilter{22}); got != "ledger" {
		t.Errorf("under #22 the component cards dimmed are %q", got)
	}
	clocks := diagram.Types(tr, d.Page, "context", "clocks")
	if got := dimmedCards(clocks, types.PullFilter{22}); got != "" {
		t.Errorf("under #22 the unchanged types dimmed are %q", got)
	}
	for _, row := range presentation.BoxRows(tr, testCommon().Theme, d, l, diagram.Arrange(tr, l.Cards, 4), types.PullFilter{22}) {
		if row.Out && row.Key == presentation.BoxKey("context") {
			t.Errorf("under #22 the Diagram panel marks Unchanged code as left out")
		}
	}
}

func TestAnArrowCountsOnlyTheLinksTheFilterShows(t *testing.T) {
	d := filterStandIn()
	tr := testCommon().Tr
	words := func(links []diagram.Link) string {
		var out []string
		for _, l := range links {
			mark := ""
			if l.Out {
				mark = " (out)"
			}
			out = append(out, l.Words+mark)
		}
		return strings.Join(out, " | ")
	}
	components := diagram.Components(tr, d.Page)
	for filter, want := range map[string]string{
		"":   "→ 3 links | → books · 1 link",
		"21": "→ 1 link | → books · 1 link",
		"22": "→ 2 links | → books · 0 links (out)",
	} {
		if got := words(presentation.FilteredLinks(tr, d, components.Links, filterOf(filter))); got != want {
			t.Errorf("under %q the arrows between components read %q, not %q", filter, got, want)
		}
	}
	packages, _ := diagram.Packages(tr, d.Page, "core")
	if got := words(presentation.FilteredLinks(tr, d, packages.Links, types.PullFilter{21})); got != "prices · 0 links (out)" {
		t.Errorf("under #21 the arrow between packages reads %q", got)
	}
	scene := draw.Scene{Level: components, Grid: diagram.Arrange(tr, components.Cards, 4), Focus: -1}
	c, _, _ := draw.Draw(tr, d, scene, types.PullFilter{22})
	if drawn := strings.Join(c.Lines(0, 0, c.W, c.H, draw.Styles(testCommon().Theme)), "\n"); !strings.Contains(drawn, "→ 2 links") || strings.Contains(drawn, "→ 3 links") {
		t.Errorf("under #22 the diagram draws:\n%s", drawn)
	}
}

func filterOf(words string) types.PullFilter {
	switch words {
	case "21":
		return types.PullFilter{21}
	case "22":
		return types.PullFilter{22}
	}
	return nil
}
