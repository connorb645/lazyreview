package github_test

import (
	"encoding/json"
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/commands/github"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/testsupport"
)

func threadsPage(next string, ids ...string) map[string]any {
	var nodes []map[string]any
	for _, id := range ids {
		nodes = append(nodes, map[string]any{"id": id, "path": "a.ts", "line": 3, "diffSide": "RIGHT", "subjectType": "LINE",
			"comments": map[string]any{"totalCount": 1, "nodes": []map[string]any{{"databaseId": 1, "author": map[string]any{"login": "alex", "__typename": "User"}}}}})
	}
	var cursor any
	if next != "" {
		cursor = next
	}
	return map[string]any{"data": map[string]any{"repository": map[string]any{"pullRequest": map[string]any{"reviewThreads": map[string]any{
		"pageInfo": map[string]any{"hasNextPage": next != "", "endCursor": cursor}, "nodes": nodes,
	}}}}}
}

func TestReviewThreadsReadsEveryPageOfThreads(t *testing.T) {
	gh := &testsupport.FakeGh{Routes: map[string]json.RawMessage{}}
	gh.Route(t, "POST graphql octo/widgets#30", threadsPage("c1", "T1", "T2"))
	gh.Route(t, "POST graphql octo/widgets#30 after c1", threadsPage("", "T3"))
	threads, err := github.New(i18n.EnglishTranslationSet(), "github.com", "", gh.Run).ReviewThreads("octo/widgets", 30)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, th := range threads {
		ids = append(ids, th.ID)
	}
	if len(ids) != 3 || ids[2] != "T3" || threads[0].Comments.Nodes[0].Author.Login != "alex" || *threads[0].Line != 3 {
		t.Errorf("the threads read as %+v", threads)
	}
	var sent struct {
		Variables map[string]any `json:"variables"`
	}
	if err := json.Unmarshal(gh.Calls[0].Body, &sent); err != nil || sent.Variables["after"] != nil || sent.Variables["number"] != float64(30) {
		t.Errorf("the first page is asked for as %s", gh.Calls[0].Body)
	}
}

func TestReviewThreadsSaysWhyGitHubGaveNone(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	gh := &testsupport.FakeGh{Routes: map[string]json.RawMessage{}}
	gh.Route(t, "POST graphql octo/widgets#30", map[string]any{"errors": []map[string]any{{"message": "Something went wrong"}, {"message": "Try again"}}})
	gh.Route(t, "POST graphql octo/widgets#31", map[string]any{"data": map[string]any{"repository": nil}})
	client := github.New(tr, "github.com", "", gh.Run)
	if _, err := client.ReviewThreads("octo/widgets", 30); err == nil || err.Error() != "GitHub refused the review threads of octo/widgets#30: Something went wrong; Try again" {
		t.Errorf("errors read as %v", err)
	}
	if _, err := client.ReviewThreads("octo/widgets", 31); err == nil || err.Error() != "GitHub gave no review threads for octo/widgets#31" {
		t.Errorf("no threads read as %v", err)
	}
}

func TestRepositoryGivesTheAddressACloneFetchesFrom(t *testing.T) {
	gh := &testsupport.FakeGh{Routes: map[string]json.RawMessage{}}
	gh.Route(t, "GET repos/octo/widgets", map[string]any{"full_name": "octo/widgets", "clone_url": "https://github.com/octo/widgets.git"})
	repo, err := github.New(i18n.EnglishTranslationSet(), "github.com", "", gh.Run).Repository("octo/widgets")
	if err != nil || repo.CloneURL != "https://github.com/octo/widgets.git" || repo.FullName != "octo/widgets" {
		t.Errorf("the repository reads as %+v, %v", repo, err)
	}
}
