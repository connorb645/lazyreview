package page

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/connorb645/lazyreview/internal/build/components"
	"github.com/connorb645/lazyreview/internal/commands/git"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/testsupport"
)

type recordedPage struct {
	Name    string                   `json:"name"`
	Script  []testsupport.ScriptStep `json:"script"`
	Request struct {
		Base    string `json:"base"`
		Members []struct {
			Pull int    `json:"pull"`
			Head string `json:"head"`
		} `json:"members"`
		BaseBranch string `json:"baseBranch"`
		Title      string `json:"title"`
		Subject    struct {
			Pulls  []int  `json:"pulls"`
			Branch string `json:"branch"`
		} `json:"subject"`
		Config    json.RawMessage `json:"config"`
		Context   bool            `json:"context"`
		RootTitle string          `json:"rootTitle"`
	} `json:"request"`
	Built *struct {
		Page   json.RawMessage `json:"page"`
		Starts []string        `json:"starts"`
		Counts Counts          `json:"counts"`
		Notes  []string        `json:"notes"`
	} `json:"built"`
	Error *string `json:"error"`
}

func TestEveryRecordedPageBuildsAsNodeBuiltIt(t *testing.T) {
	text, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "pages.json"))
	if err != nil {
		t.Fatal(err)
	}
	var record struct {
		Pages []recordedPage `json:"pages"`
	}
	if err := json.Unmarshal(text, &record); err != nil {
		t.Fatal(err)
	}
	if len(record.Pages) < 10 {
		t.Fatalf("only %d recorded pages", len(record.Pages))
	}
	tr := i18n.EnglishTranslationSet()
	for _, c := range record.Pages {
		t.Run(c.Name, func(t *testing.T) {
			r := testsupport.NewThrowawayRepo(t)
			r.Play(c.Script)
			built, err := Build(tr, git.NewRepository(tr, r.Work, nil), request(t, tr, c))
			if c.Error != nil {
				if err == nil || err.Error() != *c.Error {
					t.Fatalf("got the error %v, want %q", err, *c.Error)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var want models.PageRecord
			if err := json.Unmarshal(c.Built.Page, &want); err != nil {
				t.Fatal(err)
			}
			got := built.Page
			got.Layout = models.Layout{}
			for _, part := range []struct {
				name      string
				got, want any
			}{
				{"heading", got.Model.Heading, want.Model.Heading},
				{"components", got.Model.Components, want.Model.Components},
				{"areas", got.Model.Areas, want.Model.Areas},
				{"files", got.Model.Files, want.Model.Files},
				{"context files", got.Model.ContextFiles, want.Model.ContextFiles},
				{"nodes", got.Model.Nodes, want.Model.Nodes},
				{"edges", got.Model.Edges, want.Model.Edges},
				{"groups", got.Model.Groups, want.Model.Groups},
				{"summary", got.Model.Summary, want.Model.Summary},
				{"diff", got.Diff, want.Diff},
				{"sources", got.Sources, want.Sources},
				{"context", got.Context, want.Context},
				{"stack", got.Stack, want.Stack},
				{"starts", built.Starts, c.Built.Starts},
				{"counts", built.Counts, c.Built.Counts},
				{"notes", built.Notes, c.Built.Notes},
			} {
				if g, w := normal(t, part.got), normal(t, part.want); !reflect.DeepEqual(g, w) {
					gt, _ := json.Marshal(g)
					wt, _ := json.Marshal(w)
					t.Errorf("%s differ from Node's:\n got %s\nwant %s", part.name, gt, wt)
				}
			}
		})
	}
}

func request(t *testing.T, tr *i18n.TranslationSet, c recordedPage) Request {
	t.Helper()
	var raw []byte
	if len(c.Request.Config) > 0 && string(c.Request.Config) != "null" {
		raw = c.Request.Config
	}
	config, err := components.Read(tr, raw)
	if err != nil {
		t.Fatal(err)
	}
	r := Request{Base: c.Request.Base, BaseBranch: c.Request.BaseBranch, Title: c.Request.Title, Config: config, Context: c.Request.Context, RootTitle: c.Request.RootTitle}
	for _, m := range c.Request.Members {
		r.Members = append(r.Members, Member{Pull: m.Pull, Head: m.Head})
	}
	if c.Request.Subject.Branch != "" {
		r.Subject = BranchSubject(tr, []string{c.Request.Subject.Branch})
	} else {
		r.Subject = PullsSubject(tr, c.Request.Subject.Pulls)
	}
	return r
}

// normal is a value as JSON reads it back, so an empty list and none, or a
// list and its copy, compare alike.
func normal(t *testing.T, v any) any {
	t.Helper()
	text, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(text, &out); err != nil {
		t.Fatal(err)
	}
	return emptied(out)
}

func emptied(v any) any {
	switch x := v.(type) {
	case []any:
		if len(x) == 0 {
			return nil
		}
		for i := range x {
			x[i] = emptied(x[i])
		}
	case map[string]any:
		for k := range x {
			x[k] = emptied(x[k])
		}
	}
	return v
}
