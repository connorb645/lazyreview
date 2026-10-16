package diagram

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"review-desk/terminal/internal/server"
)

func recordedPage(t *testing.T) server.Page {
	t.Helper()
	text, err := os.ReadFile(filepath.Join("..", "..", "testdata", "record.json"))
	if err != nil {
		t.Fatal(err)
	}
	var answers map[string]json.RawMessage
	if err := json.Unmarshal(text, &answers); err != nil {
		t.Fatal(err)
	}
	var page server.Page
	if err := json.Unmarshal(answers["GET /api/reviews/github.com/octo/widgets/21/page"], &page); err != nil {
		t.Fatal(err)
	}
	return page
}

func TestTheThreeLevelsComeFromTheLayout(t *testing.T) {
	page := recordedPage(t)
	top := Components(page)
	if len(top.Cards) != 2 || top.Cards[0].Title != "widgets" || top.Cards[1].Detail != "1 file · unchanged" || !top.Cards[1].Context {
		t.Fatalf("components: %+v", top.Cards)
	}
	if len(top.Links) != 1 || top.Links[0].Words != "→ 5 links" {
		t.Fatalf("the arrow between components: %+v", top.Links)
	}
	pkgs, ok := Packages(page, "root")
	if !ok || len(pkgs.Cards) != 2 || pkgs.Cards[0].Title != "Sources" || pkgs.Cards[0].Detail != "3 types" {
		t.Fatalf("packages of root: %v %+v", ok, pkgs.Cards)
	}
	if _, ok := Packages(page, "context"); ok {
		t.Fatal("a component of one package has no level of packages")
	}
	types := Types(page, "root", "sources")
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

func TestRowsAndColumnsAndTheNearestCard(t *testing.T) {
	cards := []Card{
		{ID: "a", Title: "Alpha", Box: server.Box{X: 0, Y: 0, H: 60}},
		{ID: "b", Title: "Beta", Box: server.Box{X: 200, Y: 10, H: 60}},
		{ID: "c", Title: "Gamma", Box: server.Box{X: 0, Y: 200, H: 60}},
		{ID: "d", Title: "Delta", Box: server.Box{X: 400, Y: 200, H: 60}},
		{ID: "e", Title: "Epsilon", Box: server.Box{X: 900, Y: 0, H: 60}},
	}
	g := Arrange(cards, 10)
	if len(g.Rows) != 2 || len(g.Rows[0]) != 3 || g.Rows[1][0] != 2 || g.Rows[1][1] != 3 {
		t.Fatalf("rows: %v", g.Rows)
	}
	if g.Spots[1].Col != 2+14+10 {
		t.Fatalf("Beta sits after Alpha and a gap, at %d", g.Spots[1].Col)
	}
	cases := []struct {
		from int
		dir  string
		want int
	}{{0, "l", 1}, {1, "l", 4}, {0, "j", 2}, {1, "j", 3}, {3, "k", 1}, {2, "l", 3}, {0, "h", -1}, {0, "k", -1}}
	for _, c := range cases {
		if got := g.Nearest(c.from, c.dir, 7); got != c.want {
			t.Errorf("from %s going %s: got %d, want %d", cards[c.from].ID, c.dir, got, c.want)
		}
	}
	if order := g.Order(); len(order) != 5 || order[3] != 2 {
		t.Fatalf("reading order: %v", order)
	}
}
