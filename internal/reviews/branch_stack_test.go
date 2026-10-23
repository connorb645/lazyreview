package reviews

import (
	"reflect"
	"testing"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/loader"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/store"
	"github.com/connorb645/lazyreview/internal/testsupport"
)

// A stack of branches gives its branches as its members, with no queue,
// and loads again as its branches now stand, as R in the app does.
func TestAStackOfBranchesGivesItsBranchesAsMembersAndRefreshesFromItsRepository(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	repo := testsupport.NewThrowawayRepo(t)
	repo.Commit("main", map[string]string{"src/catalog.ts": "export const items: string[] = [];\n"}, "base")
	repo.Commit("stack/search", map[string]string{"src/search.ts": "export const find = 1;\n"}, "Add search")
	repo.Commit("stack/basket", map[string]string{"src/basket.ts": "export const basket = 2;\n"}, "Add a basket")
	repo.Checkout("main")
	kept := store.NewReviewStore(tr, store.DataHome{Root: t.TempDir()}, store.NewLocks(tr))
	branches := loader.NewBranchLoader(tr, kept, func() string { return "2026-10-23T09:00:00.000Z" })
	loaded, err := branches.Load(loader.BranchRequest{Repo: repo.Work, Base: "main", Heads: []string{"stack/search", "stack/basket"}})
	if err != nil {
		t.Fatal(err)
	}
	r := New(tr, kept, nil, nil, nil, nil, nil, branches)
	state, err := r.State(loaded.Ref.ID())
	if err != nil {
		t.Fatal(err)
	}
	want := []models.Member{
		{Pull: 1, Branch: "stack/search", Head: repo.Git("rev-parse", "stack/search"), Base: "main", Color: 1},
		{Pull: 2, Branch: "stack/basket", Head: repo.Git("rev-parse", "stack/basket"), Base: "stack/search", Color: 2},
	}
	if state.Kind != "branch" || state.Queue != nil || !reflect.DeepEqual(state.Members, want) {
		t.Errorf("the stack's state is %+v", state)
	}
	if state.Head.Ref != "stack/basket" || state.Title != "Stack stack/search to stack/basket" {
		t.Errorf("the stack reads to its top, %+v, as %q", state.Head, state.Title)
	}

	repo.Commit("stack/basket", map[string]string{"src/basket.ts": "export const basket = 3;\n"}, "Fill the basket")
	repo.Checkout("main")
	if _, err := r.Refresh(loaded.Ref.ID()); err != nil {
		t.Fatal(err)
	}
	again, err := r.State(loaded.Ref.ID())
	if err != nil {
		t.Fatal(err)
	}
	if again.Members[1].Head != repo.Git("rev-parse", "stack/basket") || again.Version == state.Version {
		t.Errorf("a refresh reads the moved branch: %+v", again.Members)
	}
	page, err := r.Page(loaded.Ref.ID())
	if err != nil {
		t.Fatal(err)
	}
	if page.Stack == nil || len(page.Stack.Members) != 2 || page.Stack.Members[1].Branch != "stack/basket" {
		t.Errorf("the page's stack is %+v", page.Stack)
	}
}
