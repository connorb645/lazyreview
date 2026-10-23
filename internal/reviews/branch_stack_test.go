package reviews

import (
	"reflect"
	"testing"

	"github.com/connorb645/lazyreview/internal/ask"
	"github.com/connorb645/lazyreview/internal/commands/claude"
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

// A conversation asked on one branch goes on in the stack it grows into:
// on the same code, though a branch above added lines over it, with Claude
// answering a follow-up, and after R.
func TestAConversationAskedOnABranchGoesOnInTheStackItGrowsInto(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	repo := testsupport.NewThrowawayRepo(t)
	repo.Commit("main", map[string]string{"src/catalog.ts": "export const items: string[] = [];\n"}, "base")
	repo.Commit("stack/search", map[string]string{"src/search.ts": "export function find(text: string) {\n  return text;\n}\n"}, "Add search")
	repo.Commit("stack/basket", map[string]string{"src/search.ts": "// Search the catalog by name.\n\nexport function find(text: string) {\n  return text;\n}\n"}, "Say what search does")
	repo.Checkout("main")
	locks := store.NewLocks(tr)
	kept := store.NewReviewStore(tr, store.DataHome{Root: t.TempDir()}, locks)
	now := func() string { return "2026-10-23T09:00:00.000Z" }
	fake := testsupport.NewFakeClaude(t)
	branches := loader.NewBranchLoader(tr, kept, now)
	r := New(tr, kept, nil, ask.NewService(tr, kept, locks, claude.New(tr, fake.Command, ""), madeCopies{}, nil, nil, nil, now), nil, nil, nil, branches)
	loaded, err := branches.Load(loader.BranchRequest{Repo: repo.Work, Base: "main", Heads: []string{"stack/search"}})
	if err != nil {
		t.Fatal(err)
	}
	id := loaded.Ref.ID()
	fake.Answer(testsupport.FakeAnswer{Text: "It hands the text back."})
	first, err := r.Ask(id, models.Lines{Path: "src/search.ts", Side: "RIGHT", Line: 2}, "What does this return?", pageBuild(t, kept, loaded.Ref))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := branches.Load(loader.BranchRequest{Repo: repo.Work, Base: "main", Heads: []string{"stack/search", "stack/basket"}}); err != nil {
		t.Fatal(err)
	}
	state, err := r.State(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Threads) != 1 || state.Threads[0].Line != 4 || len(state.Threads[0].Messages) != 2 {
		t.Fatalf("in the stack, the conversation is %+v", state.Threads)
	}
	file, err := r.File(id, "src/search.ts", pageBuild(t, kept, loaded.Ref))
	if err != nil || file.Whole[3] != "  return text;" {
		t.Fatalf("its line reads %q: %v", file.Whole, err)
	}
	fake.Knows(first.Session)
	fake.Answer(testsupport.FakeAnswer{Text: "Nothing else: the caller trims it."})
	reply, err := r.Reply(id, first.ID, "And nothing else?", pageBuild(t, kept, loaded.Ref))
	if err != nil {
		t.Fatal(err)
	}
	if len(reply.Messages) != 4 || reply.Messages[3].Text != "Nothing else: the caller trims it." || reply.Line != 4 {
		t.Errorf("the follow-up in the stack is %+v", reply)
	}
	if _, err := r.Refresh(id); err != nil {
		t.Fatal(err)
	}
	if again, err := r.State(id); err != nil || len(again.Threads) != 1 || again.Threads[0].Line != 4 || len(again.Threads[0].Messages) != 4 {
		t.Errorf("after R the conversation is %+v: %v", again.Threads, err)
	}
}
