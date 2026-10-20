package git

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/testsupport"
)

func TestPrepareFetchesTheBaseAndEachPullRequestsHeadAsTheAccountGiven(t *testing.T) {
	repo := testsupport.NewThrowawayRepo(t)
	base := repo.Commit("main", map[string]string{"a.ts": "1\n"}, "base")
	head := repo.Commit("feature/a", map[string]string{"a.ts": "2\n"}, "a")
	url := repo.Publish(map[int]string{7: "feature/a"})
	var fetched [][]string
	run := func(ctx context.Context, dir string, args, env []string, input []byte) Ran {
		if slices.Contains(args, "fetch") {
			fetched = append(fetched, env)
		}
		return RunGit(ctx, dir, args, env, input)
	}
	clone := filepath.Join(t.TempDir(), "clones", "github.com", "octo", "widgets.git")
	clones := NewClones(i18n.EnglishTranslationSet(), run)
	made, err := clones.Prepare(context.Background(), clone, url, "main", []int{7}, []string{base, head}, []string{"GH_TOKEN=token-sam"})
	if err != nil || made != clone {
		t.Fatalf("the clone is %q, %v", made, err)
	}
	if got := testsupport.RunGit(t, clone, "rev-parse", "refs/heads/main", "refs/pull/7/head"); got != base+"\n"+head {
		t.Errorf("the clone's refs are %q", got)
	}
	if len(fetched) != 1 || !slices.Contains(fetched[0], "GH_TOKEN=token-sam") {
		t.Errorf("the fetches ran with %v", fetched)
	}

	moved := repo.Commit("feature/a", map[string]string{"a.ts": "3\n"}, "again")
	testsupport.RunGit(t, repo.Work, "push", "-q", "--force", url, "--all")
	testsupport.RunGit(t, url, "update-ref", "refs/pull/7/head", moved)
	if _, err := clones.Prepare(context.Background(), clone, url, "main", []int{7}, []string{base, moved}, nil); err != nil {
		t.Fatal(err)
	}
	if got := testsupport.RunGit(t, clone, "rev-parse", "refs/pull/7/head"); got != moved {
		t.Errorf("a second fetch leaves the head at %s", got)
	}
	if got := testsupport.RunGit(t, clone, "remote", "get-url", "origin"); got != url {
		t.Errorf("the clone fetches from %s", got)
	}
}

func TestPrepareFetchesAMissingCommitByItsID(t *testing.T) {
	repo := testsupport.NewThrowawayRepo(t)
	base := repo.Commit("main", map[string]string{"a.ts": "1\n"}, "base")
	kept := repo.Commit("kept", map[string]string{"a.ts": "kept\n"}, "kept")
	repo.Checkout("main")
	url := repo.Publish(nil)
	testsupport.RunGit(t, url, "update-ref", "refs/keep/x", kept)
	testsupport.RunGit(t, url, "update-ref", "-d", "refs/heads/kept")
	clone := filepath.Join(t.TempDir(), "widgets.git")
	if _, err := NewClones(i18n.EnglishTranslationSet(), nil).Prepare(context.Background(), clone, url, "main", nil, []string{base, kept}, nil); err != nil {
		t.Fatal(err)
	}
	if got := testsupport.RunGit(t, clone, "cat-file", "-t", kept); got != "commit" {
		t.Errorf("the missing commit is %q in the clone", got)
	}
}

func TestPrepareSaysWhichGitCommandFailed(t *testing.T) {
	clone := filepath.Join(t.TempDir(), "widgets.git")
	_, err := NewClones(i18n.EnglishTranslationSet(), nil).Prepare(context.Background(), clone, filepath.Join(t.TempDir(), "none.git"), "main", []int{7}, nil, nil)
	if err == nil || !strings.HasPrefix(err.Error(), "git fetch failed: ") {
		t.Errorf("a failed fetch says %v", err)
	}
}
