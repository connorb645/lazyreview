package loader

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/internal/commands/git"
	"github.com/connorb645/lazyreview/internal/commands/github"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/store"
	"github.com/connorb645/lazyreview/internal/testsupport"
)

func TestAHeadThatCannotBeFetchedIsNamedShortBeforeGitsReason(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	repo := testsupport.NewThrowawayRepo(t)
	base := repo.Commit("main", map[string]string{"a.txt": "a\n"}, "base")
	repo.Commit("feature/a", map[string]string{"a.txt": "b\n"}, "a")
	origin := repo.Publish(map[int]string{7: "feature/a"})
	gone := strings.Repeat("f", 40)
	fake := &testsupport.FakeGh{Routes: map[string]json.RawMessage{}}
	fake.Route(t, "GET repos/octo/widgets", map[string]any{"full_name": "octo/widgets", "clone_url": origin})
	fake.Route(t, "GET user", map[string]any{"login": "reviewer"})
	fake.Route(t, "GET repos/octo/widgets/pulls/7", map[string]any{
		"number": 7, "title": "A", "html_url": "https://github.com/octo/widgets/pull/7", "state": "open", "user": map[string]any{"login": "sam"},
		"base": map[string]any{"ref": "main", "sha": base, "repo": map[string]any{"full_name": "octo/widgets"}},
		"head": map[string]any{"ref": "feature/a", "sha": gone, "repo": map[string]any{"full_name": "octo/widgets"}},
	})
	fake.Route(t, "POST graphql octo/widgets#7", map[string]any{"data": map[string]any{"repository": map[string]any{"pullRequest": map[string]any{"reviewThreads": map[string]any{"pageInfo": map[string]any{"hasNextPage": false}, "nodes": []any{}}}}}})
	reviews := store.NewReviewStore(tr, store.DataHome{Root: t.TempDir()}, store.NewLocks(tr))
	gh := func(host, account string) *github.GitHub { return github.New(tr, host, account, fake.Run) }
	loads := NewLoader(tr, reviews, store.NewLocks(tr), git.NewClones(tr, nil), gh, func() string { return "2026-10-15T12:00:00.000Z" })
	_, err := loads.Load(context.Background(), PullsRequest{Links: []string{"https://github.com/octo/widgets/pull/7"}})
	want := "could not fetch #7's head ffffffff from octo/widgets: git fetch failed: "
	if err == nil || !strings.HasPrefix(err.Error(), want) || !strings.Contains(err.Error(), gone) {
		t.Fatalf("the load was refused with %v, not %q and git's reason", err, want)
	}
}
