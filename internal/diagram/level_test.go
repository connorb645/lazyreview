package diagram

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
)

func recordedPage(t *testing.T) models.Page {
	t.Helper()
	text, err := os.ReadFile(filepath.Join("..", "..", "testdata", "record.json"))
	if err != nil {
		t.Fatal(err)
	}
	var answers map[string]struct{ Body json.RawMessage }
	if err := json.Unmarshal(text, &answers); err != nil {
		t.Fatal(err)
	}
	var page models.Page
	if err := json.Unmarshal(answers["GET /api/reviews/github.com/octo/widgets/21/page"].Body, &page); err != nil {
		t.Fatal(err)
	}
	return page
}

func TestTheThreeLevelsComeFromTheLayout(t *testing.T) {
	page, tr := recordedPage(t), i18n.EnglishTranslationSet()
	top := Components(tr, page)
	if len(top.Cards) != 2 || top.Cards[0].Title != "widgets" || top.Cards[1].Detail != "1 file · unchanged" || !top.Cards[1].Context {
		t.Fatalf("components: %+v", top.Cards)
	}
	if len(top.Links) != 1 || top.Links[0].Words != "→ 5 links" {
		t.Fatalf("the arrow between components: %+v", top.Links)
	}
	pkgs, ok := Packages(tr, page, "root")
	if !ok || len(pkgs.Cards) != 2 || pkgs.Cards[0].Title != "Sources" || pkgs.Cards[0].Detail != "3 types" {
		t.Fatalf("packages of root: %v %+v", ok, pkgs.Cards)
	}
	if _, ok := Packages(tr, page, "context"); ok {
		t.Fatal("a component of one package has no level of packages")
	}
	types := Types(tr, page, "root", "sources")
	var titles []string
	for _, c := range types.Cards {
		titles = append(titles, c.Title+"|"+c.Detail+"|"+c.State)
	}
	want := []string{"Basket|Basket.swift|changed", "MemoryCatalog|MemoryCatalog.swift|changed", "«struct» Search|Search.swift|new"}
	if len(titles) != 3 || titles[0] != want[0] || titles[1] != want[1] || titles[2] != want[2] {
		t.Fatalf("types: %v", titles)
	}
	links := map[string]Link{}
	for _, l := range types.Links {
		links[l.From+">"+l.To] = l
	}
	if l := links["Basket>Search"]; l.Words != "holds search" || !l.AtRest || l.Outside != "" {
		t.Fatalf("Basket holds Search, inside the level: %+v", l)
	}
	if l := links["Basket>Catalog"]; l.Outside != "Catalog · in Unchanged code" || l.AtRest {
		t.Fatalf("a link to a type elsewhere ends outside the level: %+v", l)
	}
	if l := links["Search>Item"]; l.Words != "uses" || l.AtRest {
		t.Fatalf("a use is no structure: %+v", l)
	}
}

func TestRowsAndColumnsInReadingOrder(t *testing.T) {
	cards := []Card{
		{ID: "a", Title: "Alpha", Box: models.Box{X: 0, Y: 0, H: 60}},
		{ID: "b", Title: "Beta", Box: models.Box{X: 200, Y: 10, H: 60}},
		{ID: "c", Title: "Gamma", Box: models.Box{X: 0, Y: 200, H: 60}},
		{ID: "d", Title: "Delta", Box: models.Box{X: 400, Y: 200, H: 60}},
		{ID: "e", Title: "Epsilon", Box: models.Box{X: 900, Y: 0, H: 60}},
	}
	g := Arrange(i18n.EnglishTranslationSet(), cards, 10)
	if len(g.Rows) != 2 || len(g.Rows[0]) != 3 || g.Rows[1][0] != 2 || g.Rows[1][1] != 3 {
		t.Fatalf("rows: %v", g.Rows)
	}
	if g.Spots[1].Col != 2+14+10 {
		t.Fatalf("Beta sits after Alpha and a gap, at %d", g.Spots[1].Col)
	}
	if order := g.Order(); len(order) != 5 || order[3] != 2 {
		t.Fatalf("reading order: %v", order)
	}
}

// table is a grid of rows as long as `lengths`.
func table(lengths ...int) Grid {
	var cards []Card
	for r, n := range lengths {
		for k := range n {
			title := fmt.Sprintf("Card %d.%d", r, k)
			cards = append(cards, Card{ID: title, Title: title, Box: models.Box{X: float64(k * 200), Y: float64(r * 200), H: 60}})
		}
	}
	return Arrange(i18n.EnglishTranslationSet(), cards, 16)
}

func TestACardMovesAlongItsRowAndToTheSameItemOfTheNext(t *testing.T) {
	g := table(10, 3, 10)
	at := func(row, item int) int { return g.Rows[row][item] }
	if g.Item(at(0, 9)) != 9 || g.Item(at(1, 0)) != 0 || g.Item(at(2, 4)) != 4 {
		t.Fatalf("each card's item in its row: %d %d %d", g.Item(at(0, 9)), g.Item(at(1, 0)), g.Item(at(2, 4)))
	}
	along := []struct{ from, by, want int }{
		{at(0, 4), 1, at(0, 5)}, {at(0, 4), -1, at(0, 3)}, {at(0, 4), 3, at(0, 7)},
		{at(0, 9), 1, at(0, 9)}, {at(0, 0), -1, at(0, 0)}, {at(1, 1), 5, at(1, 2)}, {at(2, 8), -20, at(2, 0)},
	}
	for _, c := range along {
		if got := g.Along(c.from, c.by); got != c.want {
			t.Errorf("%d along from row %d item %d: got item %d of row %d", c.by, g.Spots[c.from].Row, g.Item(c.from), g.Item(got), g.Spots[got].Row)
		}
	}
	down := []struct{ from, by, item, want int }{
		{at(0, 9), 1, 9, at(1, 2)}, {at(1, 2), 1, 9, at(2, 9)}, {at(2, 9), -1, 9, at(1, 2)}, {at(1, 2), -1, 9, at(0, 9)},
		{at(0, 1), 1, 1, at(1, 1)}, {at(0, 9), 2, 9, at(2, 9)}, {at(0, 9), 5, 9, at(2, 9)}, {at(2, 3), -9, 3, at(0, 3)},
		{at(0, 9), -1, 9, at(0, 9)}, {at(2, 9), 1, 9, at(2, 9)}, {at(1, 2), 1, 2, at(2, 2)},
	}
	for _, c := range down {
		if got := g.Down(c.from, c.by, c.item); got != c.want {
			t.Errorf("%d down from row %d item %d keeping item %d: got item %d of row %d", c.by, g.Spots[c.from].Row, g.Item(c.from), c.item, g.Item(got), g.Spots[got].Row)
		}
	}
}
