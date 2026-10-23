package gui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/connorb645/lazyreview/internal/gui/context"
	"github.com/connorb645/lazyreview/internal/gui/types"
	"github.com/connorb645/lazyreview/internal/models"
)

const (
	stepInBudget = time.Second
	moveBudget   = 250 * time.Millisecond
	walks        = 3
)

// largeStandIn is a review the size of a whole app loaded as one branch
// from an empty tree: 588 types and file boxes and 458 links, all in one
// component of 12 packages, the largest holding 378 boxes, laid out in rows
// as wide as a layout of 6022 by 2234 sets them.
func largeStandIn() types.ReviewData {
	sizes := []struct {
		id    string
		count int
	}{
		{"top", 5}, {"schema", 1}, {"bin", 2}, {"testdata", 2}, {"images", 2}, {"docs", 4},
		{"terminal", 3}, {"app", 38}, {"test", 56}, {"scripts", 2}, {"src", 95}, {"internal", 378},
	}
	var packages []models.Package
	var nodes []models.Node
	byPackage := map[string][]string{}
	boxes := map[string]models.Box{}
	for p, size := range sizes {
		packages = append(packages, models.Package{ID: size.id, Title: size.id})
		boxes["pkg:whole:"+size.id] = models.Box{X: float64(p%4) * 1500, Y: float64(p/4) * 700, H: 600}
		for k := 0; k < size.count; k++ {
			id := fmt.Sprintf("%s/file%03d", size.id, k)
			nodes = append(nodes, models.Node{ID: id, Name: fmt.Sprintf("file%03d.go", k), Kind: "artifact", File: id + ".go", Line: 1,
				Component: "whole", Package: size.id, State: "new"})
			boxes[id] = models.Box{X: float64(k%26) * 230, Y: float64(k/26) * 130, H: 60}
			byPackage[size.id] = append(byPackage[size.id], id)
		}
	}
	var edges []models.Edge
	seed := uint32(7)
	pick := func(list []string) string {
		seed = seed*1664525 + 1013904223
		return list[int(seed>>8)%len(list)]
	}
	for _, links := range []struct {
		from, to string
		count    int
	}{{"src", "src", 231}, {"test", "src", 115}, {"app", "app", 74}, {"test", "test", 28}, {"test", "app", 7}, {"scripts", "scripts", 1}, {"test", "scripts", 1}, {"scripts", "src", 1}} {
		for i := 0; i < links.count; i++ {
			kind := "use"
			if links.from == "test" {
				kind = "test"
			}
			edges = append(edges, models.Edge{Source: pick(byPackage[links.from]), Target: pick(byPackage[links.to]), Kind: kind})
		}
	}
	return types.ReviewData{ID: "large", Page: models.Page{
		Model: models.Model{
			Components: []models.Component{{ID: "whole", Title: "whole", Color: "blue", Packages: packages}},
			Nodes:      nodes,
			Edges:      edges,
		},
		Layout: models.Layout{Components: map[string]models.Box{"whole": {}}, Boxes: boxes},
	}}
}

// The graph of a review that size opens within a second of Enter at every
// level, drawn in full colour, and each move about it takes a fraction of
// that. A step can overrun when every package's tests share the machine, so
// a walk that overran walks again, and only one that overruns every time
// fails.
func TestALargeGraphOpensQuicklyAndMovesSmoothly(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(termenv.Ascii)
	d := largeStandIn()
	if len(d.Page.Model.Nodes) != 588 || len(d.Page.Model.Edges) != 458 {
		t.Fatalf("the stand-in has %d boxes and %d links", len(d.Page.Model.Nodes), len(d.Page.Model.Edges))
	}
	for _, size := range [][2]int{{161, 48}, {80, 24}} {
		var slow []string
		for walk := 1; walk <= walks; walk++ {
			if slow = walkLargeGraph(t, d, size); len(slow) == 0 {
				break
			}
			t.Logf("at %dx%d, walk %d overran: %s", size[0], size[1], walk, strings.Join(slow, "; "))
		}
		for _, overran := range slow {
			t.Errorf("at %dx%d, %s, on each of %d walks", size[0], size[1], overran, walks)
		}
	}
}

func walkLargeGraph(t *testing.T, d types.ReviewData, size [2]int) []string {
	t.Helper()
	a := New(testCommon(), Options{})
	a.Review, a.Loading, a.Width, a.Height = &d, false, size[0], size[1]
	a.ContextTree = a.Focused(context.MainPanel, a.Driving())
	a = showBox(a, types.DiagramPlace{Level: "components"})
	var slow []string
	step := func(name string, budget time.Duration, keys ...string) {
		t.Helper()
		start := time.Now()
		a = press(t, a, keys...)
		_ = a.View()
		took := time.Since(start)
		t.Logf("%dx%d %-22s %v", size[0], size[1], name, took)
		if took > budget {
			slow = append(slow, fmt.Sprintf("%s took %v, more than %v", name, took, budget))
		}
	}
	step("Enter into the component", stepInBudget, "enter")
	focusedOn(t, a, "top")
	step("G to the last package", moveBudget, "G")
	focusedOn(t, a, "internal")
	step("Enter into 378 boxes", stepInBudget, "enter")
	for _, key := range []string{"j", "l", "w", "G", "g"} {
		if key == "g" {
			step("gg", moveBudget, "g", "g")
			continue
		}
		step(key, moveBudget, key)
	}
	step("Space a, every link", moveBudget, "space", "a")
	step("f, every label", moveBudget, "f")
	step("Esc", moveBudget, "esc")
	step("- back out", stepInBudget, "-")
	step("h to the box left", moveBudget, "h")
	focusedOn(t, a, "src")
	step("Enter into src", stepInBudget, "enter")
	step("Space a, every link", moveBudget, "space", "a")
	step("gr, what links in", moveBudget, "g", "r")
	step("Esc", moveBudget, "esc")
	step("- back out", stepInBudget, "-")
	return slow
}
