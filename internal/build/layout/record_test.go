package layout

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
)

type recordedLayout struct {
	Name       string                  `json:"name"`
	Model      models.Model            `json:"model"`
	Components []string                `json:"components"`
	Boxes      []string                `json:"boxes"`
	PkgArrows  []models.PackageArrow   `json:"pkgArrows"`
	Summaries  []models.ComponentArrow `json:"summaries"`
}

func readJSON(t *testing.T, path string, into any) {
	t.Helper()
	text, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(text, into); err != nil {
		t.Fatal(err)
	}
}

func keys(m map[string]models.Box) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

func sameArrows(t *testing.T, name string, got models.Layout, components, boxes []string, pkgArrows []models.PackageArrow, summaries []models.ComponentArrow) {
	t.Helper()
	if !reflect.DeepEqual(keys(got.Components), components) {
		t.Errorf("%s: placed components %v, not %v", name, keys(got.Components), components)
	}
	if !reflect.DeepEqual(keys(got.Boxes), boxes) {
		t.Errorf("%s: placed boxes %v, not %v", name, keys(got.Boxes), boxes)
	}
	if !reflect.DeepEqual(got.PkgArrows, pkgArrows) {
		t.Errorf("%s: arrows between packages\n%+v\nnot\n%+v", name, got.PkgArrows, pkgArrows)
	}
	if !reflect.DeepEqual(got.Summaries, summaries) {
		t.Errorf("%s: arrows between components\n%+v\nnot\n%+v", name, got.Summaries, summaries)
	}
}

// The arrows between packages and between components carry the words, the
// counts and the links the Node layout gave them, for made-up models.
func TestTheArrowsCarryTheWordsNodeGaveThem(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	var record []recordedLayout
	readJSON(t, filepath.Join("..", "..", "..", "testdata", "layouts.json"), &record)
	if len(record) != 3 {
		t.Fatalf("the record holds %d models", len(record))
	}
	for _, r := range record {
		sameArrows(t, r.Name, Of(tr, r.Model), r.Components, r.Boxes, r.PkgArrows, r.Summaries)
	}
}

// A recorded review's pages, laid out again in Go, keep every part and the
// arrows Node gave them; only the places differ.
func TestARecordedPageKeepsItsPartsAndArrows(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	for _, pull := range []string{"21", "30"} {
		var page models.PageRecord
		readJSON(t, filepath.Join("..", "..", "..", "testdata", "home", "reviews", "github.com", "octo", "widgets", pull, "page.json"), &page)
		sameArrows(t, "#"+pull, Of(tr, page.Model), keys(page.Layout.Components), keys(page.Layout.Boxes), page.Layout.PkgArrows, page.Layout.Summaries)
	}
}
