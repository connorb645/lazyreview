package review

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
)

// standInPage is a page of the stand-in data home the terminal app's
// recorded answers come from: #21 and #22 as a stack, or #30 alone.
func standInPage(t *testing.T, pull string) *models.PageRecord {
	t.Helper()
	text, err := os.ReadFile(filepath.Join("..", "..", "testdata", "home", "reviews", "github.com", "octo", "widgets", pull, "page.json"))
	if err != nil {
		t.Fatal(err)
	}
	var page models.PageRecord
	if err := json.Unmarshal(text, &page); err != nil {
		t.Fatal(err)
	}
	return &page
}

func TestAFileOfAStackIsDrawnRowByRowEachRowWithThePullRequestThatChangedIt(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	page := standInPage(t, "21")
	basket, ok := FileView(tr, page, "Sources/Basket/Basket.swift")
	if !ok {
		t.Fatal("Basket.swift is not drawn")
	}
	find := func(kind string, line int) models.Row {
		for _, h := range basket.Hunks {
			for _, r := range h.Rows {
				if r.Kind == kind && (r.New == line || kind == "del") {
					return r
				}
			}
		}
		t.Fatalf("no %s row at %d", kind, line)
		return models.Row{}
	}
	if r := find("add", 3); r.Text != "    let search: Search" || r.Owner != 22 || r.Words != "Changed in #22. A comment here goes on #22, line 3." {
		t.Errorf("the added line reads %+v", r)
	}
	if r := find("same", 1); r.Owner != 0 || r.Words != "Unchanged. A comment here goes on #22, line 1." {
		t.Errorf("the unchanged line reads %+v", r)
	}
	if r := find("del", 0); r.Owner != 22 {
		t.Errorf("the removed line is owned by %d", r.Owner)
	}
	if len(basket.Whole) != 13 {
		t.Errorf("the whole text has %d lines", len(basket.Whole))
	}
	catalog, _ := FileView(tr, page, "Sources/Catalog/MemoryCatalog.swift")
	if len(catalog.Gaps) != 1 || catalog.Gaps[0] != (models.Gap{Before: 0, From: 1, To: 3, Delta: 0}) {
		t.Errorf("the unchanged lines before the first hunk are %+v", catalog.Gaps)
	}
	if search, _ := FileView(tr, page, "Sources/Catalog/Search.swift"); len(search.Gaps) != 0 {
		t.Errorf("an added file has runs between hunks: %+v", search.Gaps)
	}
	if _, ok := FileView(tr, page, "nowhere.swift"); ok {
		t.Errorf("a path the page does not draw is drawn")
	}
	view := PageView("abc", page)
	if view.Build != "abc" || view.Stack == nil || len(view.Stack.Members) != 2 {
		t.Errorf("the page view is %+v", view.Stack)
	}
}

func TestAFileOfASinglePullRequestHasRowsWithNoOwnerAndAMarkdownFileItsRenderedView(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	guide, _ := FileView(tr, standInPage(t, "30"), "docs/guide.md")
	for _, h := range guide.Hunks {
		for _, r := range h.Rows {
			if r.Owner != 0 || r.Words != "" {
				t.Fatalf("a row of a single pull request has an owner: %+v", r)
			}
		}
	}
	if guide.Rendered == nil {
		t.Fatal("a Markdown file has no Rendered view")
	}
	var changes []string
	for _, b := range guide.Rendered.Blocks {
		if b.State != "same" {
			changes = append(changes, b.State+" "+b.Side+" "+b.Markdown)
		}
	}
	if len(changes) < 2 || changes[0] != "del LEFT The catalog lists items." || changes[1] != "add RIGHT The catalog lists every item." {
		t.Fatalf("the Rendered view's changes are %v", changes)
	}
}
