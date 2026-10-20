package stack

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/connorb645/lazyreview/internal/build/diff"
	"github.com/connorb645/lazyreview/internal/commands/git"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/testsupport"
)

type recordedStack struct {
	Name    string `json:"name"`
	Base    string `json:"base"`
	Top     string `json:"top"`
	Members []struct {
		Pull int    `json:"pull"`
		Head string `json:"head"`
	} `json:"members"`
	Script []testsupport.ScriptStep `json:"script"`
	Error  *string                  `json:"error"`
}

func TestEveryRecordedStackReadsAsNodeReadIt(t *testing.T) {
	text, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "stacks.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cases []recordedStack
	var answers []map[string]any
	if err := json.Unmarshal(text, &cases); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(text, &answers); err != nil {
		t.Fatal(err)
	}
	if len(cases) < 8 {
		t.Fatalf("only %d recorded stacks", len(cases))
	}
	tr := i18n.EnglishTranslationSet()
	for i, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			repo := rebuilt(t, c)
			got, err := answered(tr, repo, c)
			want := answers[i]
			if c.Error != nil {
				if err == nil || err.Error() != *c.Error {
					t.Fatalf("got the error %v, want %q", err, *c.Error)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			for _, key := range []string{"heads", "starts", "warnings", "counts", "lines", "files"} {
				if !reflect.DeepEqual(got[key], want[key]) {
					g, _ := json.Marshal(got[key])
					w, _ := json.Marshal(want[key])
					t.Errorf("%s differs from Node's:\n got %s\nwant %s", key, g, w)
				}
			}
		})
	}
}

func rebuilt(t *testing.T, c recordedStack) *git.Repository {
	t.Helper()
	r := testsupport.NewThrowawayRepo(t)
	r.Play(c.Script)
	for _, m := range c.Members {
		if c.Error == nil && r.Git("cat-file", "-t", m.Head+"^{commit}") != "commit" {
			t.Fatalf("the Go throwaway repository does not make Node's commit %s", m.Head)
		}
	}
	if top := r.Git("rev-parse", c.Top+"^{commit}"); top != c.Top {
		t.Fatalf("the Go throwaway repository does not make Node's top commit %s", c.Top)
	}
	return git.NewRepository(i18n.EnglishTranslationSet(), r.Work, nil)
}

func answered(tr *i18n.TranslationSet, repo *git.Repository, c recordedStack) (map[string]any, error) {
	baseCommit, err := repo.Resolve(c.Base)
	if err != nil {
		return nil, err
	}
	combinedBase, err := repo.MergeBase(baseCommit, c.Top)
	if err != nil {
		return nil, err
	}
	text, err := repo.Diff(combinedBase, c.Top)
	if err != nil {
		return nil, err
	}
	var members []Member
	var pulls []int
	for _, m := range c.Members {
		members = append(members, Member{Pull: m.Pull, Head: m.Head})
		pulls = append(pulls, m.Pull)
	}
	read, err := ReadSteps(tr, repo, combinedBase, baseCommit, c.Top, members)
	if err != nil {
		return nil, err
	}
	mapped := MapLines(tr, read.Steps, diff.SplitDiff(text), pulls)
	starts := []string{}
	for _, s := range read.Steps {
		starts = append(starts, s.Start)
	}
	counts := []map[string]int{}
	for _, n := range read.Counts {
		counts = append(counts, map[string]int{"pull": n.Pull, "files": n.Files, "commits": n.Commits, "adds": n.Adds, "dels": n.Dels})
	}
	files := map[string]any{}
	for path, f := range mapped.Files {
		files[path] = map[string]any{"pulls": f.Pulls, "byPull": f.ByPull}
	}
	shown, err := json.Marshal(map[string]any{"heads": read.Heads, "starts": starts, "warnings": read.Warnings, "counts": counts, "lines": mapped.Lines, "files": files})
	if err != nil {
		return nil, err
	}
	var out map[string]any
	return out, json.Unmarshal(shown, &out)
}
