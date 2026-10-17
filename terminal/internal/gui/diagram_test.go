package gui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
)

func onDiagram(t *testing.T) App {
	t.Helper()
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	return press(t, a, "space", "m", "enter", "enter")
}

func focusedOn(t *testing.T, a App, want string) {
	t.Helper()
	if card, _ := a.dg.card(); card.ID != want {
		t.Fatalf("the focus is on %q, not %q", card.ID, want)
	}
}

func TestTheDiagramStepsInAndOutOfItsLevels(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	a = press(t, a, "space", "m")
	screenIs(t, a, "diagram-components")
	a = press(t, a, "enter")
	screenIs(t, a, "diagram-packages")
	a = press(t, a, "enter")
	screenIs(t, a, "diagram-types")
	a = press(t, a, "-")
	focusedOn(t, a, "sources")
	a = press(t, a, "-")
	focusedOn(t, a, "root")
	a = press(t, a, "j", "enter")
	if a.dg.level.Kind != "types" || a.dg.level.Package != "top" {
		t.Fatalf("a component of one package steps straight to its types: %+v", a.dg.level)
	}
	a = press(t, a, "-", "-")
	if a.screen != screenReview {
		t.Fatalf("out of the components is the Review view: %v", a.screen)
	}
}

func TestTheFocusMovesToTheNearestBox(t *testing.T) {
	a := onDiagram(t)
	focusedOn(t, a, "MemoryCatalog")
	a = press(t, a, "j")
	focusedOn(t, a, "Basket")
	a = press(t, a, "j")
	focusedOn(t, a, "Search")
	a = press(t, a, "j")
	focusedOn(t, a, "Search")
	a = press(t, a, "g", "g")
	focusedOn(t, a, "MemoryCatalog")
	a = press(t, a, "G")
	focusedOn(t, a, "Search")
	a = press(t, a, "b")
	focusedOn(t, a, "Basket")
	a = press(t, a, "2", "k")
	focusedOn(t, a, "MemoryCatalog")
	screenIs(t, a, "diagram-focus")
}

func TestTheFocusedBoxListsItsLinksAndProperties(t *testing.T) {
	a := press(t, onDiagram(t), "j")
	a = press(t, a, "space", "l")
	screenIs(t, a, "diagram-links")
	a = press(t, a, "j", "enter")
	focusedOn(t, a, "Search")
	if a.menu != nil {
		t.Fatal("going to a link's other end closes the list")
	}
	a = press(t, a, "k", "space", "p")
	screenIs(t, a, "diagram-properties")
	a = press(t, a, "enter")
	if a.screen != screenDiff || cursorAt(a) != "Sources/Basket/Basket.swift RIGHT 2" {
		t.Fatalf("a property goes to its line: %v %s", a.screen, cursorAt(a))
	}
}

func TestGdFollowsALinkAndGrListsWhatLinksIn(t *testing.T) {
	a := onDiagram(t)
	a = typed(t, press(t, a, "/"), "Catalog")
	focusedOn(t, a, "Catalog")
	if a.dg.level.Component != "context" {
		t.Fatalf("a match in another component shows its level: %+v", a.dg.level)
	}
	a = press(t, a, "g", "d")
	focusedOn(t, a, "Item")
	a = press(t, a, "g", "r")
	screenIs(t, a, "diagram-references")
	a = press(t, a, "enter")
	focusedOn(t, a, "MemoryCatalog")
	a = press(t, a, "ctrl+o")
	focusedOn(t, a, "Item")
	a = press(t, a, "ctrl+o")
	focusedOn(t, a, "Catalog")
	a = press(t, a, "N")
	focusedOn(t, a, "MemoryCatalog")
}

func TestEnterOpensATypesDiffAndDashComesBack(t *testing.T) {
	a := press(t, onDiagram(t), "j", "enter")
	if a.screen != screenDiff || cursorAt(a) != "Sources/Basket/Basket.swift RIGHT 1" {
		t.Fatalf("Enter on a type opens its line: %v %s", a.screen, cursorAt(a))
	}
	a = press(t, a, "-")
	if a.screen != screenDiagram {
		t.Fatalf("- goes back to the diagram: %v", a.screen)
	}
	focusedOn(t, a, "Basket")
	a = press(t, a, "g", "d")
	if a.screen != screenDiagram || a.menu == nil || len(a.menu.rows) != 2 {
		t.Fatalf("Basket links to two types, so gd lists them: %+v", a.menu)
	}
}

func TestGdInTheDiffShowsTheTypeAtTheCursor(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	a = typed(t, press(t, a, "space", "f"), "Catalog/Search")
	if a.screen != screenDiff || cursorAt(a) != "Sources/Catalog/Search.swift file" {
		t.Fatalf("the finder opens a file at its header: %v %s", a.screen, cursorAt(a))
	}
	a = press(t, a, "g", "d")
	if a.menu == nil || a.menu.title != "Types in Search.swift" {
		t.Fatalf("gd on a file's header lists its types: %+v", a.menu)
	}
	a = press(t, a, "esc", "]", "c", "g", "d")
	if a.screen != screenDiagram {
		t.Fatalf("gd on a line shows its type: %v %q", a.screen, a.notice)
	}
	focusedOn(t, a, "Search")
}

func TestTheFinderNarrowsAsItIsTyped(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	a = press(t, a, "space", "f")
	for _, r := range "bask" {
		a = send(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	screenIs(t, a, "finder")
	a = send(t, a, tea.KeyMsg{Type: tea.KeyBackspace})
	if a.menu.query != "bas" {
		t.Fatalf("Backspace takes back a letter: %q", a.menu.query)
	}
	a = send(t, a, tea.KeyMsg{Type: tea.KeyEnter})
	if a.screen != screenDiagram {
		t.Fatalf("Enter on a type shows it: %v", a.screen)
	}
	focusedOn(t, a, "Basket")
}

func TestLabelsJumpToABox(t *testing.T) {
	a := press(t, onDiagram(t), "f")
	screenIs(t, a, "diagram-labels")
	a = press(t, a, "d")
	focusedOn(t, a, "Search")
	if a.dg.labels != nil {
		t.Fatal("the labels go once one is typed")
	}
	a = press(t, a, "f", "x")
	focusedOn(t, a, "Search")
	if !a.bad || a.dg.labels != nil {
		t.Fatalf("a label no box has stops the labels: %q", a.notice)
	}
}

func TestAClickFocusesABox(t *testing.T) {
	a := onDiagram(t)
	x, y := drawnAt(t, a, "Basket.swift")
	a = send(t, a, tea.MouseMsg{X: x, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	focusedOn(t, a, "Basket")
}

func drawnAt(t *testing.T, a App, text string) (int, int) {
	t.Helper()
	main := a.layout().main.inner()
	for y, line := range strings.Split(a.View(), "\n") {
		inside := ansi.TruncateLeft(line, main.x, "")
		if x := strings.Index(inside, text); x >= 0 && y >= main.y && y < main.y+main.h {
			return main.x + ansi.StringWidth(inside[:x]), y
		}
	}
	t.Fatalf("%q is not drawn in the main panel", text)
	return 0, 0
}

func TestThePullRequestFilterDimsTheTypesItLeavesOut(t *testing.T) {
	a := press(t, onDiagram(t), "]", "p")
	if got := fmt.Sprint(a.filter); got != "[21]" {
		t.Fatalf("the filter: %s", got)
	}
	c, tops, _ := drawDiagram(*a.review, a.dg, a.filter)
	titleStyle := func(id string) string {
		s := a.dg.grid.Spots[a.dg.level.Index(id)]
		return c.cells[tops[s.Row]+1][s.Col+2].style
	}
	if titleStyle("Basket") != "dim" || titleStyle("Search") != "title" {
		t.Fatalf("#21 changed Search and not Basket: %q %q", titleStyle("Basket"), titleStyle("Search"))
	}
}

func standIn() reviewData {
	box := func(x, y float64) server.Box { return server.Box{X: x, Y: y, H: 60} }
	node := func(id, file string, line int) server.Node {
		return server.Node{ID: id, Name: id, Kind: "class", File: "Sources/Core/" + file, Line: line, Component: "core", Package: "model", State: "changed"}
	}
	nodes := []server.Node{node("Order", "Order.swift", 1), node("Ledger", "Ledger.swift", 1), node("Clock", "Clock.swift", 1),
		node("Pricer", "Pricer.swift", 1), node("Invoice", "Invoice.swift", 1), node("Receipt", "Receipt.swift", 1),
		node("Refund", "Refund.swift", 1), node("Return", "Refund.swift", 9)}
	nodes[6].Group, nodes[7].Group = "kinds", "kinds"
	nodes = append(nodes, server.Node{ID: "Store", Name: "Store", Kind: "protocol", File: "Sources/Shop/Store.swift", Line: 1, Component: "shop", Package: "top", State: "same"})
	return reviewData{id: "stand-in", page: server.Page{
		Model: server.Model{
			Components: []server.Component{
				{ID: "core", Title: "Core", Color: "teal", Area: "money", Packages: []server.Package{{ID: "model", Title: "Model"}}},
				{ID: "ledger", Title: "Ledger", Color: "blue", Area: "money"},
				{ID: "shop", Title: "Shop", Color: "purple", Packages: []server.Package{{ID: "top", Title: "Shop", Bare: true}}},
				{ID: "context", Title: "Unchanged code", Color: "grey", Context: true},
			},
			Areas:  []server.Area{{ID: "money", Title: "Money"}},
			Nodes:  nodes,
			Groups: []server.Group{{ID: "kinds", Title: "Refunds", Members: []string{"Refund", "Return"}}},
			Edges: []server.Edge{
				{Source: "Order", Target: "Ledger", Kind: "has", Properties: []string{"ledger"}},
				{Source: "Ledger", Target: "Order", Kind: "use"},
				{Source: "Order", Target: "Invoice", Kind: "realize"},
				{Source: "Receipt", Target: "Clock", Kind: "inherit"},
				{Source: "Clock", Target: "Pricer", Kind: "use", Key: true, Label: "ticks"},
				{Source: "Pricer", Target: "Refund", Kind: "has", Properties: []string{"refunds"}},
				{Source: "Order", Target: "Store", Kind: "use"},
			},
		},
		Layout: server.Layout{
			Components: map[string]server.Box{"core": box(0, 0), "ledger": box(300, 0), "shop": box(0, 300), "context": box(300, 300)},
			Boxes: map[string]server.Box{"pkg:core:model": box(0, 0), "pkg:shop:top": box(0, 0),
				"Order": box(0, 0), "Ledger": box(200, 0), "Clock": box(400, 0), "Pricer": box(200, 200),
				"group:kinds": box(400, 200), "Invoice": box(0, 400), "Receipt": box(400, 400), "Store": box(0, 0)},
			Summaries: []server.ComponentArrow{
				{Source: "core", Target: "ledger", Lines: []string{"→ 2 links"}},
				{Source: "shop", Target: "core", Lines: []string{"→ 1 link"}},
				{Source: "core", Target: "context", Lines: []string{"→ 4 links"}, Both: true},
			},
		},
	}}
}

func standInApp(width, height int) App {
	a := New(Options{Dark: true})
	d := standIn()
	a.review, a.loading, a.width, a.height, a.focus = &d, false, width, height, mainPanel
	return a
}

func TestLinksRunAroundTheCards(t *testing.T) {
	a := standInApp(110, 40).showBox(diagramPlace{level: "components", box: "core"})
	screenIs(t, a, "stand-in-components")
	a = a.showBox(diagramPlace{level: "types", component: "core", pkg: "model", box: "Order"})
	screenIs(t, a, "stand-in-order")
	a = a.showBox(diagramPlace{level: "types", component: "core", pkg: "model", box: "Invoice"})
	screenIs(t, a, "stand-in-at-rest")
	a = press(t, a, "space", "a")
	screenIs(t, a, "stand-in-every-link")
	a = press(t, a, "space", "a", "g", "g", "l", "l", "j")
	focusedOn(t, a, "group:kinds")
	a = press(t, a, "enter")
	screenIs(t, a, "stand-in-group")
}

func pathsStandIn() reviewData {
	box := func(x, y float64) server.Box { return server.Box{X: x, Y: y, H: 60} }
	pkg := func(id, title string) server.Package { return server.Package{ID: id, Title: title} }
	file := func(path, pkg string) server.FileEntry {
		return server.FileEntry{Path: path, Status: "A", Adds: 3, Component: "shop", Package: pkg}
	}
	flag := "Sources/Store/Web/BasketFlow/Pricing/RemoteOverrides/Experiments/ExperimentFlag.swift"
	return reviewData{id: "paths", state: server.State{Kind: "branch", Base: server.Ref{Ref: "main"}, Head: server.Ref{Ref: "feature/pricing"}}, page: server.Page{
		Model: server.Model{
			Components: []server.Component{{ID: "shop", Title: "Shop", Color: "blue", Packages: []server.Package{
				pkg("checkout", "Checkout"), pkg("experiments", "Pricing/Remote Overrides/Experiments"),
				pkg("local", "docs/store/help/local"), pkg("pricing", "Pricing rules for every basket kind"),
			}}},
			Files: []server.FileEntry{file("Sources/Store/Web/BasketFlow/Checkout/CheckoutFlow.swift", "checkout"), file(flag, "experiments")},
			Nodes: []server.Node{{ID: "ExperimentFlag", Name: "ExperimentFlag", Kind: "struct", File: flag, Line: 1, Component: "shop", Package: "experiments", State: "new"}},
		},
		Layout: server.Layout{
			Components: map[string]server.Box{"shop": box(0, 0)},
			Boxes: map[string]server.Box{"pkg:shop:checkout": box(0, 0), "pkg:shop:experiments": box(400, 0),
				"pkg:shop:local": box(0, 200), "pkg:shop:pricing": box(400, 200), "ExperimentFlag": box(0, 0)},
		},
	}}
}

func TestAPathTitleLosesItsFirstFoldersWhereverItIsCut(t *testing.T) {
	d := pathsStandIn()
	a := New(Options{Dark: true})
	a.review, a.loading, a.width, a.height, a.focus, a.driving = &d, false, 80, 24, diagramPanel, diagramPanel
	a = a.showBox(diagramPlace{level: "packages", component: "shop", box: "experiments"}).relaid()
	screenIs(t, a, "paths-packages")
	shown := a.View()
	for _, want := range []string{"┃ …/Remote Overrides/Experiments ", "│ docs/store/help/local ", "▌…/Experiments ", "│ Pricing rules for ever…│",
		"─…/Remote Overrides/Experiments · package · 1 type─╯"} {
		if !strings.Contains(shown, want) {
			t.Errorf("the packages show no %q", want)
		}
	}
	a = a.showBox(diagramPlace{level: "types", component: "shop", pkg: "experiments", box: "ExperimentFlag"}).relaid()
	screenIs(t, a, "paths-types")
	shown = a.View()
	for _, want := range []string{"[0]─Diagram · Shop › …/Experiments─", "│ Shop › …/Experiments ", "─ExperimentFlag · struct · …erimentFlag.swift · new─╯"} {
		if !strings.Contains(shown, want) {
			t.Errorf("the types show no %q", want)
		}
	}
	a.focus = filesPanel
	if shown := a.relaid().View(); !strings.Contains(shown, "│   ▾ …/Experiments ") || !strings.Contains(shown, "│   ▾ …/Checkout ") {
		t.Errorf("the Files panel's folders keep their ends:\n%s", shown)
	}
	found := press(t, a, "space", "f").View()
	for _, want := range []string{" Sources/Store/Web/BasketFlow/Checkout/CheckoutFlow.swift ", " …/Pricing/RemoteOverrides/Experiments/ExperimentFlag.swift "} {
		if !strings.Contains(found, want) {
			t.Errorf("the finder shows no %q:\n%s", want, found)
		}
	}
}

func TestTheDiffsTitleAndFileHeadersKeepTheFilesName(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 80, 24)
	a = press(t, a, "2", "j", "j", "j", "j")
	shown := strings.Split(a.View(), "\n")
	if !strings.Contains(shown[0], "[0]─Diff · widgets › …/Catalog/MemoryCatalog.swift─") {
		t.Fatalf("the diff's title cuts its path at the start: %s", shown[0])
	}
	if !strings.Contains(shown[1], "│  ▾ …/MemoryCatalog.swift  changed  #21  +") {
		t.Fatalf("a file's header cuts its path at the start: %s", shown[1])
	}
}

func TestTheDiagramScrollsToTheFocus(t *testing.T) {
	a := standInApp(40, 14).showBox(diagramPlace{level: "types", component: "core", pkg: "model", box: "Order"})
	a = press(t, a, "G")
	focusedOn(t, a, "Receipt")
	screenIs(t, a, "stand-in-scrolled")
}
