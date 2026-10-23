package loader

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/store"
	"github.com/connorb645/lazyreview/internal/testsupport"
)

// stackBench is a repository with three branches, each built on the one
// below: stack/search on main, stack/basket on it and stack/total on that.
func stackBench(t *testing.T) (*testsupport.ThrowawayRepo, *store.ReviewStore, *BranchLoader) {
	t.Helper()
	tr := i18n.EnglishTranslationSet()
	r := testsupport.NewThrowawayRepo(t)
	r.Commit("main", map[string]string{"src/catalog.ts": "export const items: string[] = [];\n"}, "base")
	r.Commit("stack/search", map[string]string{"src/search.ts": "export function find(text: string) {\n  return text;\n}\n"}, "Add search")
	r.Commit("stack/basket", map[string]string{"src/basket.ts": "export const basket: string[] = [];\n"}, "Add a basket")
	r.Commit("stack/total", map[string]string{"src/total.ts": "export function total() {\n  return 0;\n}\n"}, "Add a total")
	r.Checkout("main")
	reviews := store.NewReviewStore(tr, store.DataHome{Root: t.TempDir()}, store.NewLocks(tr))
	tick := 0
	now := func() string {
		at := time.Date(2026, 10, 23, 9, 0, tick, 0, time.UTC).Format("2006-01-02T15:04:05.000Z")
		tick++
		return at
	}
	return r, reviews, NewBranchLoader(tr, reviews, now)
}

func stackLoad(t *testing.T, branches *BranchLoader, r *testsupport.ThrowawayRepo, heads ...string) Loaded {
	t.Helper()
	loaded, err := branches.Load(BranchRequest{Repo: r.Work, Base: "main", Heads: heads})
	if err != nil {
		t.Fatalf("loading %v: %v", heads, err)
	}
	return loaded
}

func branchesOf(rev *models.Review) string {
	var out []string
	for _, b := range rev.Branches {
		out = append(out, b.Branch+":"+strconv.Itoa(b.Color))
	}
	return strings.Join(out, " ")
}

func headingOf(page *models.PageRecord) string {
	var words strings.Builder
	for _, p := range page.Model.Heading.Context {
		words.WriteString(p.Text)
	}
	return words.String()
}

func TestABranchGrowsIntoAStackOfTheBranchesAboveItAtTheSameAddress(t *testing.T) {
	r, reviews, branches := stackBench(t)
	one := stackLoad(t, branches, r, "stack/search")
	two := stackLoad(t, branches, r, "stack/search", "stack/basket")
	three := stackLoad(t, branches, r, "stack/search", "stack/basket", "stack/total")
	if one.Ref.ID() != two.Ref.ID() || two.Ref.ID() != three.Ref.ID() || !strings.HasSuffix(one.Ref.ID(), "/stack~search") {
		t.Fatalf("loaded at %s, then %s, then %s", one.Ref.ID(), two.Ref.ID(), three.Ref.ID())
	}
	if ids := reviewIDs(t, reviews); len(ids) != 1 {
		t.Errorf("the data home holds %v", ids)
	}
	rev, err := reviews.Read(three.Ref)
	if err != nil {
		t.Fatal(err)
	}
	if got := branchesOf(rev); got != "stack/search:1 stack/basket:2 stack/total:3" {
		t.Errorf("the stack's branches are %s", got)
	}
	if rev.Kind != "branch" || rev.Title != "Stack stack/search to stack/total" || rev.GivenTitle != nil {
		t.Errorf("a stack is named for its bottom and top unless given a title: %+v", rev)
	}
	main, search, basket, total := r.Git("rev-parse", "main"), r.Git("rev-parse", "stack/search"), r.Git("rev-parse", "stack/basket"), r.Git("rev-parse", "stack/total")
	if rev.Base != (models.Ref{Ref: "main", Sha: main}) || rev.Head != (models.Ref{Ref: "stack/total", Sha: total}) || rev.Start != main {
		t.Errorf("the stack reads from %+v at %s to %+v", rev.Base, rev.Start, rev.Head)
	}
	for i, want := range []models.BranchMember{{Branch: "stack/search", Head: search, Start: main, Color: 1}, {Branch: "stack/basket", Head: basket, Start: search, Color: 2}, {Branch: "stack/total", Head: total, Start: basket, Color: 3}} {
		if rev.Branches[i] != want {
			t.Errorf("branch %d is %+v, want %+v", i, rev.Branches[i], want)
		}
	}
	drawn, err := reviews.ReadPage(three.Ref)
	if err != nil {
		t.Fatal(err)
	}
	stack := drawn.Page.Stack
	if stack == nil || len(stack.Members) != 3 || len(stack.Warnings) != 0 {
		t.Fatalf("the page's stack is %+v", stack)
	}
	for i, m := range stack.Members {
		if m.Pull != i+1 || m.Branch != rev.Branches[i].Branch || m.Head != rev.Branches[i].Head || m.Files != 1 {
			t.Errorf("member %d of the page is %+v", i, m)
		}
	}
	for path, want := range map[string][]int{"src/search.ts": {1}, "src/basket.ts": {2}, "src/total.ts": {3}} {
		for _, f := range drawn.Page.Model.Files {
			if f.Path == path && (len(f.Pulls) != 1 || f.Pulls[0] != want[0]) {
				t.Errorf("%s is changed by %v, want %v", path, f.Pulls, want)
			}
		}
	}
	if !regexp.MustCompile(`^Stack stack/search to stack/total on main: head [0-9a-f]{8}, merge base [0-9a-f]{8}, 3 commits$`).MatchString(headingOf(drawn.Page)) {
		t.Errorf("the heading reads %q", headingOf(drawn.Page))
	}
	if three.Counts.Files != 3 || len(three.Notes) != 0 {
		t.Errorf("drew %+v, noting %v", three.Counts, three.Notes)
	}
}

func TestOneBranchReadsAndIsWrittenAsItAlwaysWasEvenAfterAStack(t *testing.T) {
	r, reviews, branches := stackBench(t)
	stackLoad(t, branches, r, "stack/search", "stack/basket")
	alone := stackLoad(t, branches, r, "stack/search")
	text, err := os.ReadFile(reviews.Home().Review(alone.Ref).Record())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(text), `"branches"`) {
		t.Errorf("one branch's record names branches:\n%s", text)
	}
	rev, err := reviews.Read(alone.Ref)
	if err != nil {
		t.Fatal(err)
	}
	if rev.Title != "stack/search" || rev.Head.Ref != "stack/search" || len(rev.Branches) != 0 {
		t.Errorf("loading the bottom branch alone again reviews it alone: %+v", rev)
	}
	drawn, err := reviews.ReadPage(alone.Ref)
	if err != nil {
		t.Fatal(err)
	}
	if drawn.Page.Stack != nil || !strings.HasPrefix(headingOf(drawn.Page), "Branch stack/search on main") {
		t.Errorf("its page is a branch's: %q, stack %+v", headingOf(drawn.Page), drawn.Page.Stack)
	}
}

func TestAStackOfBranchesRefreshesAsItsBranchesNowStandAndKeepsItsTitle(t *testing.T) {
	r, reviews, branches := stackBench(t)
	title := "Search, a basket and a total"
	first, err := branches.Load(BranchRequest{Repo: r.Work, Base: "main", Heads: []string{"stack/search", "stack/basket", "stack/total"}, Title: &title})
	if err != nil {
		t.Fatal(err)
	}
	r.Commit("stack/basket", map[string]string{"src/basket.ts": "export const basket: string[] = [\"tea\"];\n"}, "Fill the basket")
	r.Checkout("main")
	ref, err := models.ParseBranchRef(i18n.EnglishTranslationSet(), first.Ref.ID())
	if err != nil {
		t.Fatal(err)
	}
	refreshed, err := branches.Refresh(ref)
	if err != nil {
		t.Fatal(err)
	}
	rev := refreshed.Review
	if rev.Branches[1].Head != r.Git("rev-parse", "stack/basket") || rev.Title != title || branchesOf(&rev) != "stack/search:1 stack/basket:2 stack/total:3" {
		t.Errorf("a refresh reads each branch as it now stands, keeping the title and colours: %+v", rev)
	}
	want := "stack/total is not built on stack/basket's head: stack/basket has commits stack/total does not hold. Until stack/total is rebased, its own diff also shows the older stack/basket it holds."
	if len(refreshed.Notes) != 1 || refreshed.Notes[0] != want {
		t.Errorf("a branch left behind by the one below it is noted: %v", refreshed.Notes)
	}
	drawn, err := reviews.ReadPage(ref)
	if err != nil {
		t.Fatal(err)
	}
	if len(drawn.Page.Stack.Warnings) != 1 || drawn.Page.Stack.Warnings[0] != want {
		t.Errorf("and warned of on its page: %v", drawn.Page.Stack.Warnings)
	}
}

func TestDroppingABranchFromTheMiddleKeepsTheAddressAndReorderingMovesIt(t *testing.T) {
	r, reviews, branches := stackBench(t)
	whole := stackLoad(t, branches, r, "stack/search", "stack/basket", "stack/total")
	dropped := stackLoad(t, branches, r, "stack/search", "stack/total")
	if dropped.Ref.ID() != whole.Ref.ID() {
		t.Fatalf("dropping the middle branch moved the review from %s to %s", whole.Ref.ID(), dropped.Ref.ID())
	}
	rev, err := reviews.Read(dropped.Ref)
	if err != nil {
		t.Fatal(err)
	}
	if got := branchesOf(rev); got != "stack/search:1 stack/total:3" {
		t.Errorf("each branch left keeps its colour: %s", got)
	}
	if rev.Branches[1].Start != r.Git("rev-parse", "stack/search") || len(dropped.Notes) != 0 {
		t.Errorf("the top's own diff now starts at the branch below it: %+v, noting %v", rev.Branches[1], dropped.Notes)
	}
	drawn, err := reviews.ReadPage(dropped.Ref)
	if err != nil {
		t.Fatal(err)
	}
	if top := drawn.Page.Stack.Members[1]; top.Branch != "stack/total" || top.Files != 2 {
		t.Errorf("and holds the dropped branch's change: %+v", top)
	}

	reordered := stackLoad(t, branches, r, "stack/basket", "stack/search", "stack/total")
	if reordered.Ref.ID() == whole.Ref.ID() || !strings.HasSuffix(reordered.Ref.ID(), "/stack~basket") {
		t.Errorf("a stack lives with its bottom branch, so reordering it loads %s", reordered.Ref.ID())
	}
	if len(reordered.Notes) != 1 || !strings.HasPrefix(reordered.Notes[0], "stack/search is not built on stack/basket's head") {
		t.Errorf("a branch below the one it is built on is noted: %v", reordered.Notes)
	}
	if ids := reviewIDs(t, reviews); len(ids) != 2 {
		t.Errorf("the review of the old order stays as it was last loaded, beside the new one: %v", ids)
	}
}

func TestAStackOfBranchesNamesWhatWasWrongAndWritesNothing(t *testing.T) {
	r, reviews, branches := stackBench(t)
	r.Checkout("stack/basket")
	for _, c := range []struct {
		heads []string
		want  string
	}{
		{[]string{"stack/search", "stack/basket", "HEAD"}, `^stack/basket is named twice; name each branch of a stack once, bottom first$`},
		{[]string{"stack/search", r.Git("rev-parse", "stack/basket")}, `is not a branch of .*; give a branch, so loading it again updates the same review$`},
		{[]string{"stack/search", "stack/nothing"}, `^"stack/nothing" is not a branch of `},
		{nil, `^give the branch to load, or every branch of a stack, bottom first$`},
	} {
		if _, err := branches.Load(BranchRequest{Repo: r.Work, Base: "main", Heads: c.heads}); err == nil || !regexp.MustCompile(c.want).MatchString(err.Error()) {
			t.Errorf("%v loads with the error %v", c.heads, err)
		}
	}
	if ids := reviewIDs(t, reviews); len(ids) != 0 {
		t.Errorf("the data home holds %v", ids)
	}
}
