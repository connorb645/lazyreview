package gui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation/diagram"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
)

func onDiagram(t *testing.T) App {
	t.Helper()
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	return press(t, a, "space", "m", "enter", "enter")
}

func focusedOn(t *testing.T, a App, want string) {
	t.Helper()
	if card, _ := a.Diagram.Card(); card.ID != want {
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
	if a.Diagram.Level.Kind != "types" || a.Diagram.Level.Package != "top" {
		t.Fatalf("a component of one package steps straight to its types: %+v", a.Diagram.Level)
	}
	a = press(t, a, "-", "-")
	if a.Screen() != context.ScreenReview {
		t.Fatalf("out of the components is the Review view: %v", a.Screen())
	}
}

func TestTheFocusMovesBetweenTheBoxes(t *testing.T) {
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
	if a.MenuOpen() {
		t.Fatal("going to a link's other end closes the list")
	}
	a = press(t, a, "k", "space", "p")
	screenIs(t, a, "diagram-properties")
	a = press(t, a, "enter")
	if a.Screen() != context.ScreenDiff || cursorAt(a) != "Sources/Basket/Basket.swift RIGHT 2" {
		t.Fatalf("a property goes to its line: %v %s", a.Screen(), cursorAt(a))
	}
}

func TestGdFollowsALinkAndGrListsWhatLinksIn(t *testing.T) {
	a := onDiagram(t)
	a = typed(t, press(t, a, "/"), "Catalog")
	focusedOn(t, a, "Catalog")
	if a.Diagram.Level.Component != "context" {
		t.Fatalf("a match in another component shows its level: %+v", a.Diagram.Level)
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
	if a.Screen() != context.ScreenDiff || cursorAt(a) != "Sources/Basket/Basket.swift RIGHT 1" {
		t.Fatalf("Enter on a type opens its line: %v %s", a.Screen(), cursorAt(a))
	}
	a = press(t, a, "-")
	if a.Screen() != context.ScreenDiagram {
		t.Fatalf("- goes back to the diagram: %v", a.Screen())
	}
	focusedOn(t, a, "Basket")
	a = press(t, a, "g", "d")
	if a.Screen() != context.ScreenDiagram || !a.MenuOpen() || len(a.Menu.Rows) != 2 {
		t.Fatalf("Basket links to two types, so gd lists them: %+v", a.Menu)
	}
}

func TestGdInTheDiffShowsTheTypeAtTheCursor(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	a = typed(t, press(t, a, "space", "f"), "Catalog/Search")
	if a.Screen() != context.ScreenDiff || cursorAt(a) != "Sources/Catalog/Search.swift file" {
		t.Fatalf("the finder opens a file at its header: %v %s", a.Screen(), cursorAt(a))
	}
	a = press(t, a, "g", "d")
	if !a.MenuOpen() || a.Menu.Title != "Types in Search.swift" {
		t.Fatalf("gd on a file's header lists its types: %+v", a.Menu)
	}
	a = press(t, a, "esc", "]", "c", "g", "d")
	if a.Screen() != context.ScreenDiagram {
		t.Fatalf("gd on a line shows its type: %v %q", a.Screen(), a.Notice)
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
	if a.Menu.Query != "bas" {
		t.Fatalf("Backspace takes back a letter: %q", a.Menu.Query)
	}
	a = send(t, a, tea.KeyMsg{Type: tea.KeyEnter})
	if a.Screen() != context.ScreenDiagram {
		t.Fatalf("Enter on a type shows it: %v", a.Screen())
	}
	focusedOn(t, a, "Basket")
}

func TestLabelsJumpToABox(t *testing.T) {
	a := press(t, onDiagram(t), "f")
	screenIs(t, a, "diagram-labels")
	a = press(t, a, "d")
	focusedOn(t, a, "Search")
	if a.Diagram.Labels != nil {
		t.Fatal("the labels go once one is typed")
	}
	a = press(t, a, "f", "x")
	focusedOn(t, a, "Search")
	if !a.Bad || a.Diagram.Labels != nil {
		t.Fatalf("a label no box has stops the labels: %q", a.Notice)
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
	main := a.helpers.Window.Layout(a.ContextTree).Main.Inner()
	for y, line := range strings.Split(a.View(), "\n") {
		inside := ansi.TruncateLeft(line, main.X, "")
		if x := strings.Index(inside, text); x >= 0 && y >= main.Y && y < main.Y+main.H {
			return main.X + ansi.StringWidth(inside[:x]), y
		}
	}
	t.Fatalf("%q is not drawn in the main panel", text)
	return 0, 0
}

func TestThePullRequestFilterDimsTheTypesItLeavesOut(t *testing.T) {
	a := press(t, onDiagram(t), "]", "p")
	if got := fmt.Sprint(a.Filter); got != "[21]" {
		t.Fatalf("the filter: %s", got)
	}
	c, tops, _ := diagram.Draw(a.c.Tr, *a.Review, a.Diagram.Scene, a.Filter)
	titleStyle := func(id string) string {
		s := a.Diagram.Grid.Spots[a.Diagram.Level.Index(id)]
		return c.Cells[tops[s.Row]+1][s.Col+2].Style
	}
	if titleStyle("Basket") != "dim" || titleStyle("Search") != "title" {
		t.Fatalf("#21 changed Search and not Basket: %q %q", titleStyle("Basket"), titleStyle("Search"))
	}
}

func standIn() types.ReviewData {
	box := func(x, y float64) server.Box { return server.Box{X: x, Y: y, H: 60} }
	node := func(id, file string, line int) server.Node {
		return server.Node{ID: id, Name: id, Kind: "class", File: "Sources/Core/" + file, Line: line, Component: "core", Package: "model", State: "changed"}
	}
	nodes := []server.Node{node("Order", "Order.swift", 1), node("Ledger", "Ledger.swift", 1), node("Clock", "Clock.swift", 1),
		node("Pricer", "Pricer.swift", 1), node("Invoice", "Invoice.swift", 1), node("Receipt", "Receipt.swift", 1),
		node("Refund", "Refund.swift", 1), node("Return", "Refund.swift", 9)}
	nodes[6].Group, nodes[7].Group = "kinds", "kinds"
	nodes = append(nodes, server.Node{ID: "Store", Name: "Store", Kind: "protocol", File: "Sources/Shop/Store.swift", Line: 1, Component: "shop", Package: "top", State: "same"})
	return types.ReviewData{ID: "stand-in", Page: server.Page{
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
	a := New(testCommon(), Options{})
	d := standIn()
	a.Review, a.Loading, a.Width, a.Height = &d, false, width, height
	a.ContextTree = a.Focused(context.MainPanel, a.Driving())
	return a
}

func TestLinksRunAroundTheCards(t *testing.T) {
	a := showBox(standInApp(110, 40), types.DiagramPlace{Level: "components", Box: "core"})
	screenIs(t, a, "stand-in-components")
	a = showBox(a, types.DiagramPlace{Level: "types", Component: "core", Package: "model", Box: "Order"})
	screenIs(t, a, "stand-in-order")
	a = showBox(a, types.DiagramPlace{Level: "types", Component: "core", Package: "model", Box: "Invoice"})
	screenIs(t, a, "stand-in-at-rest")
	a = press(t, a, "space", "a")
	screenIs(t, a, "stand-in-every-link")
	a = press(t, a, "space", "a", "g", "g", "l", "l", "j")
	focusedOn(t, a, "group:kinds")
	a = press(t, a, "enter")
	screenIs(t, a, "stand-in-group")
}

func pathsStandIn() types.ReviewData {
	box := func(x, y float64) server.Box { return server.Box{X: x, Y: y, H: 60} }
	pkg := func(id, title string) server.Package { return server.Package{ID: id, Title: title} }
	file := func(path, pkg string) server.FileEntry {
		return server.FileEntry{Path: path, Status: "A", Adds: 3, Component: "shop", Package: pkg}
	}
	flag := "Sources/Store/Web/BasketFlow/Pricing/RemoteOverrides/Experiments/ExperimentFlag.swift"
	return types.ReviewData{ID: "paths", State: server.State{Kind: "branch", Base: server.Ref{Ref: "main"}, Head: server.Ref{Ref: "feature/pricing"}}, Page: server.Page{
		Model: server.Model{
			Components: []server.Component{{ID: "shop", Title: "Shop", Color: "blue", Packages: []server.Package{
				pkg("checkout", "Checkout"), pkg("experiments", "Pricing/RemoteOverrides/Experiments"),
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

func pathsStandInApp(width, height int) App {
	a := New(testCommon(), Options{})
	d := pathsStandIn()
	a.Review, a.Loading, a.Width, a.Height = &d, false, width, height
	a.ContextTree = a.Focused(context.DiagramPanel, context.DiagramPanel)
	return a
}

func TestAPathTitleLosesItsFirstFoldersWhereverItIsCut(t *testing.T) {
	a := relaid(showBox(pathsStandInApp(80, 24), types.DiagramPlace{Level: "packages", Component: "shop", Box: "experiments"}))
	screenIs(t, a, "paths-packages")
	shown := a.View()
	for _, want := range []string{"┃ …/RemoteOverrides/Experiments ", "│ docs/store/help/local ", "▌…/Experiments ", "│ Pricing rules for ever…│",
		"─…/RemoteOverrides/Experiments · package · 1 type─╯"} {
		if !strings.Contains(shown, want) {
			t.Errorf("the packages show no %q", want)
		}
	}
	a = relaid(showBox(a, types.DiagramPlace{Level: "types", Component: "shop", Package: "experiments", Box: "ExperimentFlag"}))
	screenIs(t, a, "paths-types")
	shown = a.View()
	for _, want := range []string{"[0]─Diagram · Shop › …/RemoteOverrides/Experiments─", "│ Shop › …/Experiments ", "─ExperimentFlag · struct · …erimentFlag.swift · new─╯"} {
		if !strings.Contains(shown, want) {
			t.Errorf("the types show no %q", want)
		}
	}
	a.ContextTree = a.Focused(context.FilesPanel, a.Driving())
	if shown := relaid(a).View(); !strings.Contains(shown, "│   ▾ …/Experiments ") || !strings.Contains(shown, "│   ▾ …/Checkout ") {
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
	a := showBox(standInApp(40, 14), types.DiagramPlace{Level: "types", Component: "core", Package: "model", Box: "Order"})
	a = press(t, a, "G")
	focusedOn(t, a, "Receipt")
	screenIs(t, a, "stand-in-scrolled")
}

// tableStandIn is a package of types in three rows: ten, three and ten.
func tableStandIn() types.ReviewData {
	rows := [][]string{
		{"Watered", "Pruned", "Sunlit", "Fed", "Labelled", "Staked", "Mulched", "Netted", "Weeded", "Logged"},
		{"Bed", "Border", "Planter"},
		{"Tomato", "Bean", "Pea", "Leek", "Kale", "Chard", "Onion", "Garlic", "Carrot", "Radish"},
	}
	var nodes []server.Node
	boxes := map[string]server.Box{"pkg:garden:beds": {}}
	for r, row := range rows {
		for k, id := range row {
			nodes = append(nodes, server.Node{ID: id, Name: id, Kind: "class", File: "src/beds/" + id + ".ts", Line: 1, Component: "garden", Package: "beds", State: "new"})
			boxes[id] = server.Box{X: float64(k * 200), Y: float64(r * 200), H: 60}
		}
	}
	return types.ReviewData{ID: "table", Page: server.Page{
		Model:  server.Model{Components: []server.Component{{ID: "garden", Title: "Garden", Color: "teal", Packages: []server.Package{{ID: "beds", Title: "Beds"}}}}, Nodes: nodes},
		Layout: server.Layout{Components: map[string]server.Box{"garden": {}}, Boxes: boxes},
	}}
}

func onTable(width, height int) App {
	a := New(testCommon(), Options{})
	d := tableStandIn()
	a.Review, a.Loading, a.Width, a.Height = &d, false, width, height
	a.ContextTree = a.Focused(context.MainPanel, a.Driving())
	return showBox(a, types.DiagramPlace{Level: "types", Component: "garden", Package: "beds"})
}

func TestJAndKKeepTheItemThroughAShorterRow(t *testing.T) {
	a := press(t, onTable(80, 24), "9", "l")
	focusedOn(t, a, "Logged")
	a = press(t, a, "j")
	focusedOn(t, a, "Planter")
	a = press(t, a, "j")
	focusedOn(t, a, "Radish")
	screenIs(t, a, "table-last-row")
	a = press(t, a, "k")
	focusedOn(t, a, "Planter")
	a = press(t, a, "k")
	focusedOn(t, a, "Logged")
	a = press(t, a, "2", "j")
	focusedOn(t, a, "Radish")
	a = press(t, a, "2", "h", "k")
	focusedOn(t, a, "Planter")
	a = press(t, a, "k")
	focusedOn(t, a, "Netted")
}

func TestTheMovesStopAtTheEndsOfARowAndAtTheTopAndBottomRows(t *testing.T) {
	a := onTable(80, 24)
	for _, step := range []struct{ keys, want string }{
		{"h", "Watered"}, {"k", "Watered"}, {"20l", "Logged"}, {"l", "Logged"}, {"5j", "Radish"},
		{"j", "Radish"}, {"30h", "Tomato"}, {"h", "Tomato"}, {"9k", "Watered"},
	} {
		var keys []string
		for _, r := range step.keys {
			keys = append(keys, string(r))
		}
		a = press(t, a, keys...)
		focusedOn(t, a, step.want)
	}
}

func TestEveryOtherMoveSetsTheItemThatJAndKKeep(t *testing.T) {
	twice := func(a App, key, picked, through, to string) App {
		t.Helper()
		focusedOn(t, a, picked)
		a = press(t, a, key)
		focusedOn(t, a, through)
		a = press(t, a, key)
		focusedOn(t, a, to)
		return a
	}
	a := twice(onTable(161, 48), "j", "Watered", "Bed", "Tomato")
	a = twice(press(t, a, "G", "b"), "k", "Carrot", "Planter", "Weeded")
	a = twice(press(t, a, "w"), "j", "Logged", "Planter", "Radish")
	a = twice(press(t, a, "f", "h"), "j", "Staked", "Planter", "Chard")
	a = twice(press(t, a, "4", "G"), "j", "Fed", "Planter", "Leek")
	a = twice(typed(t, press(t, a, "space", "f"), "Kale"), "k", "Kale", "Planter", "Labelled")
	a = press(t, a, "4", "l")
	focusedOn(t, a, "Weeded")
	x, y := drawnAt(t, a, "Netted.ts")
	a = send(t, a, tea.MouseMsg{X: x, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	twice(a, "j", "Netted", "Planter", "Garlic")
}

func TestAReloadKeepsTheItemThatJAndKKeep(t *testing.T) {
	a := press(t, onTable(80, 24), "9", "l", "j")
	focusedOn(t, a, "Planter")
	a = send(t, a, types.ReviewReloaded{Data: *a.Review})
	focusedOn(t, a, "Planter")
	a = press(t, a, "j")
	focusedOn(t, a, "Radish")
}

func TestTheKeysListSaysHowTheBoxesMove(t *testing.T) {
	tr := testCommon().Tr
	help := map[string]string{}
	for _, r := range press(t, onTable(80, 24), "space", "?").Menu.Rows {
		help[r.Left] = r.Right
	}
	if help["h l"] != tr.HelpBoxesAlong || help["j k"] != tr.HelpBoxesDown {
		t.Fatalf("Space ? lists h l as %q and j k as %q", help["h l"], help["j k"])
	}
}
