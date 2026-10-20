package components

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
)

func TestEveryRecordedConfigReadsAsNodeReadIt(t *testing.T) {
	text, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "pages.json"))
	if err != nil {
		t.Fatal(err)
	}
	var record struct {
		Configs []struct {
			Given  json.RawMessage `json:"given"`
			Config json.RawMessage `json:"config"`
			Error  *string         `json:"error"`
		} `json:"configs"`
	}
	if err := json.Unmarshal(text, &record); err != nil {
		t.Fatal(err)
	}
	if len(record.Configs) < 7 {
		t.Fatalf("only %d recorded configs", len(record.Configs))
	}
	tr := i18n.EnglishTranslationSet()
	for i, c := range record.Configs {
		var raw []byte
		if string(c.Given) != "null" {
			raw = c.Given
		}
		got, err := Read(tr, raw)
		if c.Error != nil {
			if err == nil || err.Error() != *c.Error {
				t.Errorf("config %d: got the error %v, want:\n%s", i, err, *c.Error)
			}
			continue
		}
		if err != nil {
			t.Errorf("config %d: %v", i, err)
			continue
		}
		var want any
		if err := json.Unmarshal(c.Config, &want); err != nil {
			t.Fatal(err)
		}
		if shown := asNodeShows(t, got); !reflect.DeepEqual(shown, want) {
			g, _ := json.Marshal(shown)
			t.Errorf("config %d reads as\n%s\nwant\n%s", i, g, c.Config)
		}
	}
}

func TestNoConfigGroupsByFolder(t *testing.T) {
	got, err := Read(i18n.EnglishTranslationSet(), nil)
	if err != nil || len(got.Components) != 0 || got.ContextTypes.Max != defaultMax {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func asNodeShows(t *testing.T, c Config) any {
	t.Helper()
	color := func(name string) any {
		if name == "" {
			return nil
		}
		return name
	}
	strings := func(list []string) []string {
		if list == nil {
			return []string{}
		}
		return list
	}
	areas := []any{}
	for _, a := range c.Areas {
		areas = append(areas, map[string]any{"id": a.ID, "title": a.Title, "description": a.Description, "components": strings(a.Components)})
	}
	components := []any{}
	for _, x := range c.Components {
		packages := []any{}
		for _, p := range x.Packages {
			packages = append(packages, map[string]any{"id": p.ID, "title": p.Title, "description": p.Description, "paths": strings(p.Paths)})
		}
		components = append(components, map[string]any{
			"id": x.ID, "title": x.Title, "description": x.Description, "tag": x.Tag, "color": color(x.Color),
			"group": x.Group, "context": x.Context, "paths": strings(x.Paths), "packages": packages,
		})
	}
	links := []any{}
	for _, l := range c.Links {
		links = append(links, map[string]any{"from": l.From, "to": l.To, "label": l.Label, "kind": l.Kind})
	}
	summaries := []any{}
	for _, s := range c.Summaries {
		summaries = append(summaries, map[string]any{"from": s.From, "to": s.To, "label": s.Label})
	}
	groups := []any{}
	for _, g := range c.Groups {
		groups = append(groups, map[string]any{"id": g.ID, "of": g.Of, "title": g.Title, "description": g.Description})
	}
	shown := map[string]any{
		"title": c.Title, "summary": c.Summary, "areas": areas, "components": components, "links": links, "summaries": summaries, "groups": groups,
		"contextTypes": map[string]any{"include": strings(c.ContextTypes.Include), "exclude": strings(c.ContextTypes.Exclude), "max": c.ContextTypes.Max},
	}
	text, err := json.Marshal(shown)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(text, &out); err != nil {
		t.Fatal(err)
	}
	return out
}
