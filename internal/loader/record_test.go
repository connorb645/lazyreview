package loader

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/connorb645/lazyreview/internal/commands/git"
	"github.com/connorb645/lazyreview/internal/commands/github"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/store"
	"github.com/connorb645/lazyreview/internal/testsupport"
)

const standInOrigin = "/stand-in/origin.git"

type recordedLoad struct {
	Name    string            `json:"name"`
	Script  []json.RawMessage `json:"script"`
	Routes  json.RawMessage   `json:"routes"`
	Request struct {
		Links   []string        `json:"links"`
		Context *bool           `json:"context"`
		Config  json.RawMessage `json:"config"`
		Account *string         `json:"account"`
		Refresh string          `json:"refresh"`
	} `json:"request"`
	Answer struct {
		Ref    string   `json:"ref"`
		Counts any      `json:"counts"`
		Notes  []string `json:"notes"`
		Error  *string  `json:"error"`
	} `json:"answer"`
	Files map[string]string `json:"files"`
}

func TestEveryRecordedLoadLoadsAsNodeLoadedIt(t *testing.T) {
	text, err := os.ReadFile(filepath.Join("..", "..", "testdata", "loads.json"))
	if err != nil {
		t.Fatal(err)
	}
	var record struct {
		Steps []recordedLoad `json:"steps"`
	}
	if err := json.Unmarshal(text, &record); err != nil {
		t.Fatal(err)
	}
	if len(record.Steps) < 11 {
		t.Fatalf("only %d recorded loads", len(record.Steps))
	}
	tr := i18n.EnglishTranslationSet()
	repo := testsupport.NewThrowawayRepo(t)
	root := t.TempDir()
	reviews := store.NewReviewStore(tr, store.DataHome{Root: root}, store.NewLocks(tr))
	fake := &testsupport.FakeGh{}
	tick := 0
	now := func() string {
		at := time.Date(2026, 10, 15, 12, 0, tick, 0, time.UTC).Format("2006-01-02T15:04:05.000Z")
		tick++
		return at
	}
	gh := func(host, account string) *github.GitHub { return github.New(tr, host, account, fake.Run) }
	loads := NewLoader(tr, reviews, store.NewLocks(tr), git.NewClones(tr, nil), gh, now)
	for _, step := range record.Steps {
		for _, op := range step.Script {
			var write struct {
				Write []string `json:"write"`
			}
			if json.Unmarshal(op, &write) == nil && write.Write != nil {
				if err := os.WriteFile(filepath.Join(root, write.Write[0]), []byte(write.Write[1]), 0o644); err != nil {
					t.Fatal(err)
				}
				continue
			}
			var played testsupport.ScriptStep
			if err := json.Unmarshal(op, &played); err != nil {
				t.Fatal(err)
			}
			repo.Play([]testsupport.ScriptStep{played})
		}
		origin := repo.Publish(map[int]string{21: "feature/search", 22: "feature/basket", 30: "fix/wording"})
		if err := json.Unmarshal([]byte(strings.ReplaceAll(string(step.Routes), standInOrigin, origin)), &fake.Routes); err != nil {
			t.Fatal(err)
		}
		var loaded Loaded
		var err error
		if step.Request.Refresh != "" {
			ref, perr := models.ParsePullRef(tr, step.Request.Refresh)
			if perr != nil {
				t.Fatal(perr)
			}
			loaded, err = loads.Refresh(context.Background(), ref)
		} else {
			r := PullsRequest{Links: step.Request.Links, Context: step.Request.Context, Account: step.Request.Account}
			if len(step.Request.Config) > 0 {
				r.Config = step.Request.Config
			}
			loaded, err = loads.Load(context.Background(), r)
		}
		t.Run(step.Name, func(t *testing.T) {
			switch {
			case step.Answer.Error != nil:
				if err == nil || err.Error() != *step.Answer.Error {
					t.Errorf("got the error %v, want %q", err, *step.Answer.Error)
				}
			case err != nil:
				t.Fatal(err)
			default:
				if loaded.Ref.ID() != step.Answer.Ref || !reflect.DeepEqual(normal(t, loaded.Counts), normal(t, step.Answer.Counts)) || !reflect.DeepEqual(normal(t, loaded.Notes), normal(t, step.Answer.Notes)) {
					t.Errorf("answered %s %+v %q, want %s %v %q", loaded.Ref.ID(), loaded.Counts, loaded.Notes, step.Answer.Ref, step.Answer.Counts, step.Answer.Notes)
				}
			}
			got := filesUnder(t, root)
			for path := range got {
				if _, held := step.Files[path]; !held {
					t.Errorf("Go wrote %s, which Node did not", path)
				}
			}
			for path, want := range step.Files {
				have, held := got[path]
				if !held {
					t.Errorf("Go did not write %s", path)
					continue
				}
				if same, why := sameFile(t, path, have, want); !same {
					t.Errorf("%s differs from Node's: %s", path, why)
				}
			}
		})
	}
}

func filesUnder(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	reviews := filepath.Join(root, "reviews")
	if _, err := os.Stat(reviews); err != nil {
		return out
	}
	err := filepath.WalkDir(reviews, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if d != nil && d.IsDir() && strings.HasSuffix(path, ".lock") {
				return filepath.SkipDir
			}
			return err
		}
		text, err := os.ReadFile(path)
		rel, _ := filepath.Rel(root, path)
		out[filepath.ToSlash(rel)] = string(text)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func sameFile(t *testing.T, path, have, want string) (bool, string) {
	t.Helper()
	switch {
	case strings.HasSuffix(path, "page.json"):
		var got, node models.PageRecord
		if err := json.Unmarshal([]byte(have), &got); err != nil {
			return false, err.Error()
		}
		if err := json.Unmarshal([]byte(want), &node); err != nil {
			t.Fatal(err)
		}
		got.Layout = models.Layout{}
		g, w := normal(t, got), normal(t, node)
		if !reflect.DeepEqual(g, w) {
			gt, _ := json.Marshal(g)
			wt, _ := json.Marshal(w)
			return false, fmt.Sprintf("\n got %s\nwant %s", gt, wt)
		}
		return true, ""
	case strings.HasSuffix(path, ".json"):
		var g, w any
		if err := json.Unmarshal([]byte(have), &g); err != nil {
			return false, err.Error()
		}
		if err := json.Unmarshal([]byte(want), &w); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(g, w) {
			return false, fmt.Sprintf("\n got %s\nwant %s", have, want)
		}
		return true, ""
	}
	return have == want, fmt.Sprintf("\n got %q\nwant %q", have, want)
}

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
