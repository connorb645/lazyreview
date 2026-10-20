package gui

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

	"github.com/connorb645/lazyreview/internal/diagram"
	"github.com/connorb645/lazyreview/internal/gui/context"
	"github.com/connorb645/lazyreview/internal/gui/presentation"
	draw "github.com/connorb645/lazyreview/internal/gui/presentation/diagram"
	"github.com/connorb645/lazyreview/internal/gui/types"
	"github.com/connorb645/lazyreview/internal/gui/views"
	"github.com/connorb645/lazyreview/internal/models"
)

// filterStandIn is a made-up stack of #21 and #22: Order.swift changed by
// both, Rules.swift by #22 alone and Book.swift by #21 alone, with an
// unchanged Clock.swift for context. Core links to Clock three times, and
// Ledger to Core once.
func filterStandIn() types.ReviewData {
	box := func(x, y float64) models.Box { return models.Box{X: x, Y: y, H: 60} }
	file := func(path, component, pkg string, adds, dels int, byPull map[int][2]int) models.FileEntry {
		var pulls []int
		for pull := range byPull {
			pulls = append(pulls, pull)
		}
		slices.Sort(pulls)
		return models.FileEntry{Path: path, Status: "M", Adds: adds, Dels: dels, Pulls: pulls, ByPull: byPull, Component: component, Package: pkg}
	}
	node := func(id, path, component, pkg string, pulls ...int) models.Node {
		state := "changed"
		if len(pulls) == 0 {
			state = "same"
		}
		return models.Node{ID: id, Name: id, Kind: "class", File: path, Line: 1, Component: component, Package: pkg, State: state, Pulls: pulls}
	}
	nodes := []models.Node{
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
		State: models.State{Kind: "stack", Members: []models.Member{{Pull: 21, Title: "Add orders", Color: 1}, {Pull: 22, Title: "Price the orders", Color: 2}}},
		Page: models.Page{
			Stack: &models.PageStack{Members: []models.StackMember{{Pull: 21}, {Pull: 22}}},
			Model: models.Model{
				Components: []models.Component{
					{ID: "core", Title: "Core", Color: "teal", Packages: []models.Package{{ID: "model", Title: "Model"}, {ID: "rules", Title: "Rules"}}},
					{ID: "ledger", Title: "Ledger", Color: "blue", Packages: []models.Package{{ID: "books", Title: "Books", Bare: true}}},
					{ID: "context", Title: "Unchanged code", Color: "grey", Context: true, Packages: []models.Package{{ID: "clocks", Title: "Clocks", Bare: true}}},
				},
				Files: []models.FileEntry{
					file("Sources/Core/Order.swift", "core", "model", 3, 1, map[int][2]int{21: {2, 0}, 22: {1, 1}}),
					file("Sources/Core/Rules/Rules.swift", "core", "rules", 4, 0, map[int][2]int{22: {4, 0}}),
					file("Sources/Ledger/Book.swift", "ledger", "books", 5, 2, map[int][2]int{21: {5, 2}}),
				},
				ContextFiles: []models.FileEntry{{Path: "Sources/Clock.swift", Component: "context", Lines: 12}},
				Nodes:        nodes,
				Groups:       []models.Group{{ID: "kinds", Title: "Refunds", Members: []string{"Refund", "Return"}}},
				Edges: []models.Edge{
					{Source: "Order", Target: "group:kinds", Kind: "has", Properties: []string{"refunds"}},
					{Source: "Order", Target: "Pricer", Kind: "has", Properties: []string{"pricer"}},
				},
			},
			Layout: models.Layout{
				Components: map[string]models.Box{"core": box(0, 0), "ledger": box(300, 0), "context": box(600, 0)},
				Boxes: map[string]models.Box{"pkg:core:model": box(0, 0), "pkg:core:rules": box(300, 0), "pkg:context:clocks": box(600, 0),
					"Order": box(0, 0), "Pricer": box(200, 0), "group:kinds": box(400, 0), "Clock": box(0, 0)},
				Summaries: []models.ComponentArrow{
					{Source: "core", Target: "context", Lines: []string{"→ 3 links"}, Ways: []models.ArrowWay{{Arrow: "→", Links: [][2]string{{"Order", "Clock"}, {"group:kinds", "Clock"}, {"Pricer", "Clock"}}}}},
					{Source: "ledger", Target: "core", Lines: []string{"→ books · 1 link"}, Ways: []models.ArrowWay{{Arrow: "→", Label: "books", Links: [][2]string{{"Book", "Order"}}}}},
				},
				PkgArrows: []models.PackageArrow{{Component: "core", From: "model", To: "rules", Words: "prices", CountText: "1 link", Links: [][2]string{{"Order", "Rule"}}}},
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
		{Kind: "line", Path: "Sources/Core/Order.swift", Line: &models.Row{Kind: "add", New: 3, Text: "let total = 0", Owner: 21}},
		{Kind: "line", Path: "Sources/Core/Order.swift", Line: &models.Row{Kind: "add", New: 4, Text: "let tax = 0", Owner: 22}},
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

// A line the filter leaves out keeps its green or red, in a dimmer shade,
// sign and all, in the unified and the split view, so an added line and a
// removed one still read apart, and neither reads as unchanged code.
func TestALineTheFilterLeavesOutKeepsADimmerGreenOrRed(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })
	ink := func(colour string) string { return termenv.CSI + termenv.TrueColor.Color(colour).Sequence(false) + "m" }
	d := filterStandIn()
	held := &models.Row{Kind: "add", New: 3, Text: "let total = 0", Owner: 21}
	added := &models.Row{Kind: "add", New: 4, Text: "let tax = 0", Owner: 22}
	removed := &models.Row{Kind: "del", Old: 4, Text: "let fee = 1", Owner: 22}
	unified := presentation.DiffView{Width: 60, Rows: []presentation.DiffRow{
		{Kind: "line", Path: "Sources/Core/Order.swift", Line: held},
		{Kind: "line", Path: "Sources/Core/Order.swift", Line: removed},
		{Kind: "line", Path: "Sources/Core/Order.swift", Line: added},
	}}
	split := presentation.DiffView{Width: 80, Split: true, Rows: []presentation.DiffRow{{Kind: "line", Path: "Sources/Core/Order.swift", Left: removed, Right: added}}}
	for _, c := range []struct {
		theme                                       string
		added, dimAdded, dimRemoved, outPullRequest string
	}{
		{"dark", "#7ce2a7", "#4d9a74", "#b0647a", "#d95926"},
		{"light", "#146b3a", "#5c9a78", "#c8768a", "#eb6834"},
	} {
		cmn := commonFor(c.theme == "dark")
		lines := presentation.DiffLines(cmn.Tr, cmn.Theme, presentation.KeyLabels{}, d, unified, 3, types.PullFilter{21}, false)
		lines = append(lines, presentation.DiffLines(cmn.Tr, cmn.Theme, presentation.KeyLabels{}, d, split, 1, types.PullFilter{21}, false)...)
		for i, want := range [][]string{
			{ink(c.added) + "+ let total = 0"},
			{ink(c.outPullRequest) + "·", ink(c.dimRemoved) + "- let fee = 1"},
			{ink(c.outPullRequest) + "·", ink(c.dimAdded) + "+ let tax = 0"},
			{ink(c.outPullRequest) + "·", ink(c.dimRemoved) + "- let fee = 1", ink(c.dimAdded) + "+ let tax = 0"},
		} {
			for _, part := range want {
				if !strings.Contains(lines[i], part) {
					t.Errorf("on a %s terminal, line %d has no %q: %q", c.theme, i, part, lines[i])
				}
			}
		}
	}
}

// In the Review view, a pull request the filter leaves out has its counts
// of lines in the dimmer green and red, as its number and title are faint.
func TestTheReviewViewDimsTheCountsOfAPullRequestLeftOut(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })
	ink := func(colour string) string { return termenv.CSI + termenv.TrueColor.Color(colour).Sequence(false) + "m" }
	a, _ := opened(t, stack, 160, 30)
	a = press(t, a, "j", "space", "v")
	counts := map[string]string{}
	for _, line := range strings.Split(a.View(), "\n") {
		for _, pull := range []string{"#21", "#22"} {
			if at := strings.Index(line, "by sam"); at >= 0 && strings.Contains(ansi.Strip(line[:at]), "  "+pull+"  ") {
				counts[pull] = line[at:]
			}
		}
	}
	if !strings.Contains(counts["#21"], ink("#7ce2a7")+"+") || !strings.Contains(counts["#21"], ink("#ff9fb6")+"−") {
		t.Errorf("#21, which the filter holds, counts its lines as %q", counts["#21"])
	}
	if !strings.Contains(counts["#22"], ink("#4d9a74")+"+") || !strings.Contains(counts["#22"], ink("#b0647a")+"−") {
		t.Errorf("#22, which the filter leaves out, counts its lines as %q", counts["#22"])
	}
}

// A block of a rendered Markdown file changed only by pull requests the
// filter leaves out keeps its bar in the dimmer green or red, and its text
// fades, its own styles and all. With no colour, a dot in place of its bar
// marks it.
func TestABlockTheFilterLeavesOutFadesWithADimmerBar(t *testing.T) {
	d := filterStandIn()
	v := presentation.DiffView{Width: 60, Rows: []presentation.DiffRow{
		{Kind: "block", Path: "docs/guide.md", Text: "Every item.", Block: &models.Block{State: "add", Side: "RIGHT", StartLine: 3, Line: 3, Pulls: []int{21}}},
		{Kind: "block", Path: "docs/guide.md", Text: "\x1b[1mSome\x1b[0m items.", Block: &models.Block{State: "del", Side: "LEFT", StartLine: 3, Line: 3, Pulls: []int{22}}},
		{Kind: "block", Path: "docs/guide.md", Text: "A basket.", Block: &models.Block{State: "add", Side: "RIGHT", StartLine: 5, Line: 5, Pulls: []int{21, 22}}},
	}}
	plain := testCommon()
	if lines := presentation.DiffLines(plain.Tr, plain.Theme, presentation.KeyLabels{}, d, v, 3, types.PullFilter{21}, false); !strings.Contains(lines[0], "▌│") || !strings.Contains(lines[1], "·│") || !strings.Contains(lines[2], "▌│") {
		t.Fatalf("with no colour, the bars read %q", lines)
	}
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })
	ink := func(colour string) string { return termenv.CSI + termenv.TrueColor.Color(colour).Sequence(false) + "m" }
	for _, c := range []struct{ theme, added, dimRemoved, faint string }{
		{"dark", "#7ce2a7", "#b0647a", "#6d6a8a"},
		{"light", "#146b3a", "#c8768a", "#9b98b5"},
	} {
		cmn := commonFor(c.theme == "dark")
		lines := presentation.DiffLines(cmn.Tr, cmn.Theme, presentation.KeyLabels{}, d, v, 3, types.PullFilter{21}, false)
		if !strings.Contains(lines[1], ink(c.dimRemoved)+"▌") || !strings.Contains(lines[1], ink(c.faint)+"Some items.") {
			t.Errorf("on a %s terminal, the block the filter leaves out reads %q", c.theme, lines[1])
		}
		for _, held := range []string{lines[0], lines[2]} {
			if !strings.Contains(held, ink(c.added)+"▌") || strings.Contains(held, ink(c.faint)) {
				t.Errorf("on a %s terminal, a block the filter holds reads %q", c.theme, held)
			}
		}
	}
}

// In the split view, each side of a row has its pull request's mark at its
// left edge: the left side's where the row starts, the right side's in
// place of the line between them. So with no colour, a side the filter
// leaves out has a dot beside a side it holds.
func TestEachSideOfASplitRowIsMarkedForItsPullRequest(t *testing.T) {
	d := filterStandIn()
	cmn := testCommon()
	removed := func(owner int) *models.Row {
		return &models.Row{Kind: "del", Old: 4, Text: "let fee = 1", Owner: owner}
	}
	added := func(owner int) *models.Row {
		return &models.Row{Kind: "add", New: 4, Text: "let tax = 0", Owner: owner}
	}
	same := &models.Row{Kind: "same", Old: 5, New: 5, Text: "}"}
	v := presentation.DiffView{Width: 60, Split: true, Cursor: -1, Rows: []presentation.DiffRow{
		{Kind: "line", Path: "Sources/Core/Order.swift", Left: removed(22), Right: added(21)},
		{Kind: "line", Path: "Sources/Core/Order.swift", Left: removed(21), Right: added(22)},
		{Kind: "line", Path: "Sources/Core/Order.swift", Right: added(21)},
		{Kind: "line", Path: "Sources/Core/Order.swift", Left: same, Right: same},
	}}
	lines := presentation.DiffLines(cmn.Tr, cmn.Theme, presentation.KeyLabels{}, d, v, 4, types.PullFilter{21}, false)
	half := (v.Width - 3) / 2
	marks := func(line string) string {
		cells := []rune(line)
		return string(cells[1]) + string(cells[2+half])
	}
	for i, want := range []string{"·▎", "▎·", " ▎", " │"} {
		if got := marks(lines[i]); got != want {
			t.Errorf("row %d is marked %q, not %q: %q", i, got, want, lines[i])
		}
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
	model := diagram.Types(tr, d.Page, "core", "model")
	out := map[string]bool{}
	for _, l := range presentation.FilteredLinks(tr, d, model.Links, types.PullFilter{22}) {
		out[l.From+">"+l.To] = l.Out
	}
	if !out["Order>Pricer"] || out["Order>group:kinds"] {
		t.Errorf("under #22 the links between types left out are %v", out)
	}
	dottedLinks := func(filter types.PullFilter) int {
		scene := draw.Scene{Level: model, Grid: diagram.Arrange(tr, model.Cards, 4), Focus: -1}
		c, _, _ := draw.Draw(tr, d, scene, filter)
		n := 0
		for _, row := range c.Cells {
			for _, cell := range row {
				if cell.Dirs != 0 && cell.R == '·' {
					n++
				}
			}
		}
		return n
	}
	if dottedLinks(nil) != 0 || dottedLinks(types.PullFilter{22}) == 0 {
		t.Errorf("a link the filter leaves out is not dotted: %d dots with no filter, %d under #22", dottedLinks(nil), dottedLinks(types.PullFilter{22}))
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
