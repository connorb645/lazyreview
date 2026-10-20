package layout

import (
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/connorb645/lazyreview/terminal/internal/diagram"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
)

func typeNode(id, component, pkg, kind string) models.Node {
	return models.Node{ID: id, Name: id, Kind: kind, File: "Sources/" + pkg + "/" + id + ".swift", Line: 1, Component: component, Package: pkg, State: "new"}
}

// shop is a made-up change: a protocol and the types that conform to it, a
// holder and what it holds, a group, unlinked types, test code, an area,
// and an unchanged component.
func shop() models.Model {
	nodes := []models.Node{
		typeNode("Store", "app", "model", "protocol"), typeNode("MemoryStore", "app", "model", "class"), typeNode("DiskStore", "app", "model", "class"),
		typeNode("Cache", "app", "model", "class"), typeNode("Entry", "app", "model", "struct"), typeNode("Clock", "app", "model", "protocol"),
		typeNode("Apple", "app", "model", "struct"), typeNode("Pear", "app", "model", "struct"), typeNode("Fruit", "app", "model", "protocol"),
		typeNode("Price", "app", "model", "struct"), typeNode("Coupon", "app", "model", "struct"), typeNode("Receipt", "app", "model", "struct"),
		typeNode("Home", "app", "ui", "class"), typeNode("Detail", "app", "ui", "class"),
		typeNode("StoreTests", "app", "tests", "class"),
		typeNode("Client", "sdk", "core", "class"), typeNode("Token", "sdk", "core", "struct"),
		typeNode("Logger", "context", "top", "protocol"),
	}
	nodes[6].Group, nodes[7].Group = "fruit", "fruit"
	nodes[14].Test = true
	nodes[17].State = "context"
	return models.Model{
		Components: []models.Component{
			{ID: "app", Title: "App", Tag: "App/", Packages: []models.Package{{ID: "model", Title: "Model", Path: "App/model/"}, {ID: "ui", Title: "Screens", Path: "App/ui/", Description: "What people see"}, {ID: "tests", Title: "Tests", Path: "Tests/"}}},
			{ID: "sdk", Title: "SDK", Tag: "Sources/", Area: "lib", Packages: []models.Package{{ID: "core", Title: "Core", Path: "Sources/core/"}}},
			{ID: "context", Title: "Unchanged code", Tag: "Base/", Context: true, Packages: []models.Package{{ID: "top", Title: "Unchanged code", Path: "Base/", Bare: true}}},
		},
		Areas:  []models.Area{{ID: "lib", Title: "The library", Components: []string{"sdk"}}},
		Files:  []models.FileEntry{{Path: "App/model/Store.swift", Adds: 1500, Dels: 20, Component: "app", Package: "model"}},
		Nodes:  nodes,
		Groups: []models.Group{{ID: "fruit", Title: "2 types conforming to Fruit", Description: "Fruit", Members: []string{"Apple", "Pear"}}},
		Edges: []models.Edge{
			{Source: "MemoryStore", Target: "Store", Kind: "realize"},
			{Source: "DiskStore", Target: "Store", Kind: "realize"},
			{Source: "MemoryStore", Target: "Cache", Kind: "has"},
			{Source: "Cache", Target: "Entry", Kind: "has"},
			{Source: "Cache", Target: "Clock", Kind: "use", Key: true, Label: "ticks"},
			{Source: "group:fruit", Target: "Fruit", Kind: "realize"},
			{Source: "Home", Target: "Detail", Kind: "has"},
			{Source: "Detail", Target: "Store", Kind: "use", Key: true, Label: "reads"},
			{Source: "StoreTests", Target: "Store", Kind: "test"},
			{Source: "Detail", Target: "Client", Kind: "use"},
			{Source: "Client", Target: "Token", Kind: "has"},
			{Source: "Cache", Target: "Logger", Kind: "use"},
		},
		Summary: []models.ComponentLinks{
			{Source: "app", Target: "context", Count: 1, Links: [][2]string{{"Cache", "Logger"}}},
			{Source: "app", Target: "sdk", Count: 1, Label: "calls", Links: [][2]string{{"Detail", "Client"}}},
		},
	}
}

func overlaps(a, b box) bool {
	return a.x < b.x+b.w && b.x < a.x+a.w && a.y < b.y+b.h && b.y < a.y+a.h
}

func inside(b, frame box) bool {
	return b.x >= frame.x && b.y >= frame.y && b.x+b.w <= frame.x+frame.w && b.y+b.h <= frame.y+frame.h
}

func checkPlaces(t *testing.T, m models.Model, d drawn) {
	t.Helper()
	var components []box
	for _, c := range m.Components {
		cb, ok := d.components[c.ID]
		if !ok {
			t.Fatalf("no place for component %s", c.ID)
		}
		for _, other := range components {
			if overlaps(cb, other) {
				t.Errorf("component %s overlaps another", c.ID)
			}
		}
		components = append(components, cb)
		var frames []box
		for _, p := range c.Packages {
			frame, ok := d.boxes["pkg:"+c.ID+":"+p.ID]
			if !ok {
				continue
			}
			if !inside(frame, cb) {
				t.Errorf("package %s sits outside component %s", p.ID, c.ID)
			}
			for _, other := range frames {
				if overlaps(frame, other) {
					t.Errorf("package %s overlaps another", p.ID)
				}
			}
			frames = append(frames, frame)
			var cards []box
			for _, n := range m.Nodes {
				if n.Component != c.ID || n.Package != p.ID {
					continue
				}
				b, ok := d.boxes[n.ID]
				if !ok {
					t.Fatalf("no place for type %s", n.ID)
				}
				if !inside(b, frame) {
					t.Errorf("type %s sits outside its package frame", n.ID)
				}
				if n.Group != "" {
					if !inside(b, d.boxes["group:"+n.Group]) {
						t.Errorf("type %s sits outside its group", n.ID)
					}
					continue
				}
				cards = append(cards, b)
			}
			for _, g := range m.Groups {
				if b, ok := d.boxes["group:"+g.ID]; ok && inside(b, frame) {
					cards = append(cards, b)
				}
			}
			for i := range cards {
				for j := i + 1; j < len(cards); j++ {
					if overlaps(cards[i], cards[j]) {
						t.Errorf("two boxes overlap in package %s: %+v %+v", p.ID, cards[i], cards[j])
					}
				}
			}
		}
	}
}

func TestEveryPartHasAPlaceInsideItsFrameAndNothingOverlaps(t *testing.T) {
	m := shop()
	checkPlaces(t, m, place(i18n.EnglishTranslationSet(), m))
}

func TestSupertypesSitAboveTheirSubtypesAndHoldersAboveWhatTheyHold(t *testing.T) {
	d := place(i18n.EnglishTranslationSet(), shop())
	above := func(top, under string) {
		t.Helper()
		if a, b := d.boxes[top], d.boxes[under]; a.y+a.h > b.y {
			t.Errorf("%s does not sit above %s: %+v %+v", top, under, a, b)
		}
	}
	above("Store", "MemoryStore")
	above("Store", "DiskStore")
	above("MemoryStore", "Cache")
	above("Cache", "Entry")
	above("Cache", "Clock")
	above("Fruit", "group:fruit")
	above("Home", "Detail")
	above("Client", "Token")
	if d.boxes["MemoryStore"].y != d.boxes["DiskStore"].y {
		t.Error("the types that conform to Store share a row")
	}
}

func TestBoxesNothingLinksSitInTidyRows(t *testing.T) {
	var nodes []models.Node
	for i := range 10 {
		nodes = append(nodes, typeNode(fmt.Sprintf("Type%d", i), "app", "model", "struct"))
	}
	m := models.Model{Components: []models.Component{{ID: "app", Title: "App", Packages: []models.Package{{ID: "model", Title: "Model"}}}}, Nodes: nodes}
	d := place(i18n.EnglishTranslationSet(), m)
	var tops []float64
	for _, n := range nodes {
		if y := d.boxes[n.ID].y; !slices.Contains(tops, y) {
			tops = append(tops, y)
		}
	}
	if len(tops) < 2 || len(tops) > 5 {
		t.Fatalf("ten boxes sit in %d rows", len(tops))
	}
	for i := 1; i < len(nodes); i++ {
		a, b := d.boxes[nodes[i-1].ID], d.boxes[nodes[i].ID]
		if a.y == b.y && a.x >= b.x || a.y > b.y {
			t.Errorf("%s does not follow %s in reading order", nodes[i].ID, nodes[i-1].ID)
		}
	}
	checkPlaces(t, m, d)
}

func TestTheSameModelAlwaysGetsTheSamePlaces(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	first := Of(tr, shop())
	for range 20 {
		if again := Of(tr, shop()); !reflect.DeepEqual(again, first) {
			t.Fatal("the layout changed between two runs")
		}
	}
}

// The terminal app reads a level's rows from the places: one row per
// layer, supertypes first, and every box of a layer in the same row.
func TestTheTerminalAppReadsOneRowPerLayer(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	m := shop()
	page := models.Page{Model: m, Layout: Of(tr, m)}
	level := diagram.Types(tr, page, "app", "model")
	grid := diagram.Arrange(tr, level.Cards, 2)
	rowOf := func(id string) int { return grid.Spots[level.Index(id)].Row }
	if rowOf("Store") >= rowOf("MemoryStore") || rowOf("MemoryStore") != rowOf("DiskStore") || rowOf("MemoryStore") >= rowOf("Cache") || rowOf("Cache") >= rowOf("Entry") {
		t.Errorf("the rows do not follow the layers: %v", grid.Rows)
	}
	if rowOf("Fruit") >= rowOf("group:fruit") {
		t.Errorf("a group sits below what it conforms to: %v", grid.Rows)
	}
}

func TestALargeGraphIsLaidOutQuickly(t *testing.T) {
	var packages []models.Package
	var nodes []models.Node
	var edges []models.Edge
	seed := uint32(11)
	next := func(n int) int {
		seed = seed*1664525 + 1013904223
		return int(seed>>8) % n
	}
	for p := range 12 {
		id := fmt.Sprintf("p%d", p)
		packages = append(packages, models.Package{ID: id, Title: id})
		count := 10
		if p == 0 {
			count = 380
		}
		for k := range count {
			nodes = append(nodes, typeNode(fmt.Sprintf("%s.T%03d", id, k), "whole", id, "class"))
		}
	}
	kinds := []string{"has", "realize", "inherit", "use"}
	for range 460 {
		a, b := nodes[next(len(nodes))], nodes[next(len(nodes))]
		edges = append(edges, models.Edge{Source: a.ID, Target: b.ID, Kind: kinds[next(len(kinds))], Key: true})
	}
	m := models.Model{Components: []models.Component{{ID: "whole", Title: "whole", Packages: packages}}, Nodes: nodes, Edges: edges}
	start := time.Now()
	d := place(i18n.EnglishTranslationSet(), m)
	if took := time.Since(start); took > time.Second {
		t.Errorf("laying out %d types took %v", len(nodes), took)
	}
	checkPlaces(t, m, d)
}
