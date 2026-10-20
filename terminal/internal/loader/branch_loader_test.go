package loader

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/connorb645/lazyreview/terminal/internal/build/page"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/store"
	"github.com/connorb645/lazyreview/terminal/internal/testsupport"
)

var branchID = regexp.MustCompile(`^local/work-[0-9a-f]{8}/feature~search$`)

func branchBench(t *testing.T) (*testsupport.ThrowawayRepo, *store.ReviewStore, *BranchLoader) {
	t.Helper()
	tr := i18n.EnglishTranslationSet()
	r := testsupport.NewThrowawayRepo(t)
	r.Commit("main", map[string]string{"src/catalog.ts": "export const items: string[] = [];\n"}, "base")
	r.Commit("feature/search", map[string]string{"src/search.ts": "export function find(text: string) {\n  return text;\n}\n"}, "Add search")
	r.Checkout("main")
	reviews := store.NewReviewStore(tr, store.DataHome{Root: t.TempDir()}, store.NewLocks(tr))
	tick := 0
	now := func() string {
		at := time.Date(2026, 10, 16, 9, 0, tick, 0, time.UTC).Format("2006-01-02T15:04:05.000Z")
		tick++
		return at
	}
	return r, reviews, NewBranchLoader(tr, reviews, now)
}

func reviewIDs(t *testing.T, reviews *store.ReviewStore) []string {
	t.Helper()
	var ids []string
	root := reviews.Home().Reviews()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Name() == "review.json" {
			rel, _ := filepath.Rel(root, filepath.Dir(path))
			ids = append(ids, filepath.ToSlash(rel))
		}
		if os.IsNotExist(err) {
			return nil
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return ids
}

func TestABranchLoadsIntoAReadOnlyReviewKeptByRepositoryAndBranch(t *testing.T) {
	r, reviews, branches := branchBench(t)
	loaded, err := branches.Load(BranchRequest{Repo: r.Work, Base: "main", Head: "feature/search"})
	if err != nil {
		t.Fatal(err)
	}
	if !branchID.MatchString(loaded.Ref.ID()) {
		t.Errorf("the review's id is %s", loaded.Ref.ID())
	}
	if loaded.Counts != (page.Counts{Files: 1, Adds: 3, Components: 1}) {
		t.Errorf("drew %+v", loaded.Counts)
	}
	review, err := reviews.Read(loaded.Ref)
	if err != nil || review == nil {
		t.Fatalf("read %v, %v", review, err)
	}
	if review.Kind != "branch" || review.Title != "feature/search" || review.Path != r.Work || review.GivenTitle != nil {
		t.Errorf("a branch is named for itself unless given a title: %+v", review)
	}
	if review.Base != (models.Ref{Ref: "main", Sha: r.Git("rev-parse", "main")}) || review.Head != (models.Ref{Ref: "feature/search", Sha: r.Git("rev-parse", "feature/search")}) {
		t.Errorf("read between %+v and %+v", review.Base, review.Head)
	}
	if review.Start != r.Git("rev-parse", "main") || review.LoadedAt != "2026-10-16T09:00:00.000Z" || len(review.Members) != 0 {
		t.Errorf("starts at %s, loaded at %s, with members %v", review.Start, review.LoadedAt, review.Members)
	}
	drawn, err := reviews.ReadPage(loaded.Ref)
	if err != nil || drawn == nil {
		t.Fatalf("read the page %v, %v", drawn, err)
	}
	if len(drawn.Page.Diff) != 1 || drawn.Page.Diff["src/search.ts"] == "" || drawn.Page.Stack != nil {
		t.Errorf("the page holds %v", drawn.Page.Diff)
	}
	var heading strings.Builder
	for _, p := range drawn.Page.Model.Heading.Context {
		heading.WriteString(p.Text)
	}
	if !regexp.MustCompile(`^Branch feature/search on main: head [0-9a-f]{8}, merge base [0-9a-f]{8}, 1 commit$`).MatchString(heading.String()) {
		t.Errorf("the heading reads %q", heading.String())
	}
	if ids := reviewIDs(t, reviews); len(ids) != 1 || ids[0] != loaded.Ref.ID() {
		t.Errorf("the data home holds %v", ids)
	}
}

func TestLoadingABranchAgainFromAnyWorktreeUpdatesItsReviewInPlace(t *testing.T) {
	r, reviews, branches := branchBench(t)
	outside, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	second := filepath.Join(outside, "second")
	title, withContext := "Search", true
	first, err := branches.Load(BranchRequest{Repo: r.Work, Base: "main", Head: "feature/search", Title: &title, Context: &withContext,
		Config: []byte(`{"components": [{"id": "web", "title": "Web", "paths": ["src/"]}]}`)})
	if err != nil {
		t.Fatal(err)
	}
	r.Git("worktree", "add", "-q", second, "feature/search")
	t.Cleanup(func() { r.Git("worktree", "remove", "--force", second) })
	testsupport.RunGit(t, second, "commit", "-q", "--allow-empty", "-m", "Nothing")
	again, err := branches.Load(BranchRequest{Repo: second, Base: "main", Head: "HEAD"})
	if err != nil {
		t.Fatal(err)
	}
	if again.Ref.ID() != first.Ref.ID() {
		t.Fatalf("loaded %s, then %s", first.Ref.ID(), again.Ref.ID())
	}
	review, err := reviews.Read(again.Ref)
	if err != nil {
		t.Fatal(err)
	}
	if review.Head.Sha != r.Git("rev-parse", "feature/search") || review.Path != second {
		t.Errorf("the head is the branch as it now stands, read from where it was loaded: %+v at %s", review.Head, review.Path)
	}
	if review.Title != "Search" || review.GivenTitle == nil || *review.GivenTitle != "Search" || !review.Context {
		t.Errorf("the title and the choice of context given once are kept: %+v", review)
	}
	drawn, err := reviews.ReadPage(again.Ref)
	if err != nil {
		t.Fatal(err)
	}
	if components := drawn.Page.Model.Components; len(components) != 1 || components[0].Title != "Web" {
		t.Errorf("and so is the config: %+v", components)
	}
	if ids := reviewIDs(t, reviews); len(ids) != 1 {
		t.Errorf("the data home holds %v", ids)
	}

	testsupport.RunGit(t, second, "commit", "-q", "--allow-empty", "-m", "Again")
	ref, err := models.ParseBranchRef(i18n.EnglishTranslationSet(), first.Ref.ID())
	if err != nil {
		t.Fatal(err)
	}
	refreshed, err := branches.Refresh(ref)
	if err != nil {
		t.Fatal(err)
	}
	if refreshed.Review.Head.Sha != r.Git("rev-parse", "feature/search") || refreshed.Review.LoadedAt != "2026-10-16T09:00:02.000Z" || refreshed.Review.Path != second || refreshed.Review.Title != "Search" {
		t.Errorf("a refresh loads from the folder and base it was last loaded with: %+v", refreshed.Review)
	}
}

func TestTwoRepositoriesOfTheSameNameKeepTheirBranchesApart(t *testing.T) {
	one, _, oneLoader := branchBench(t)
	two, _, twoLoader := branchBench(t)
	a, err := oneLoader.Load(BranchRequest{Repo: one.Work, Base: "main", Head: "feature/search"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := twoLoader.Load(BranchRequest{Repo: two.Work, Base: "main", Head: "feature/search"})
	if err != nil {
		t.Fatal(err)
	}
	refA, refB := a.Ref.(models.BranchRef), b.Ref.(models.BranchRef)
	if refA.ID() == refB.ID() || strings.Split(refA.RepoKey, "-")[0] != strings.Split(refB.RepoKey, "-")[0] {
		t.Errorf("loaded %s and %s", refA.ID(), refB.ID())
	}
}

func TestABranchLoadNamesWhatWasWrongAndWritesNothing(t *testing.T) {
	r, reviews, branches := branchBench(t)
	empty := t.TempDir()
	for _, c := range []struct {
		request BranchRequest
		want    string
	}{
		{BranchRequest{Repo: r.Work, Base: "main", Head: r.Git("rev-parse", "feature/search")}, `is not a branch of .*; give a branch, so loading it again updates the same review$`},
		{BranchRequest{Repo: r.Work, Base: "main", Head: "feature/nothing"}, `^"feature/nothing" is not a branch of `},
		{BranchRequest{Repo: r.Work, Base: "nowhere", Head: "feature/search"}, `^"nowhere" is not a commit in this repository$`},
		{BranchRequest{Repo: empty, Base: "main", Head: "main"}, `not a git repository`},
		{BranchRequest{Repo: r.Work, Base: "main", Head: "main"}, `^there is no difference between main and main$`},
	} {
		if _, err := branches.Load(c.request); err == nil || !regexp.MustCompile(c.want).MatchString(err.Error()) {
			t.Errorf("%+v loads with the error %v", c.request, err)
		}
	}
	if ids := reviewIDs(t, reviews); len(ids) != 0 {
		t.Errorf("the data home holds %v", ids)
	}
}
