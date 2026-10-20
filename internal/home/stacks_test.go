package home

import (
	"reflect"
	"testing"

	"github.com/connorb645/lazyreview/internal/models"
)

func pull(number int, base, head, repo string) models.GitHubPull {
	return models.GitHubPull{Number: number, Base: models.GitHubPullBranch{Ref: base}, Head: models.GitHubPullBranch{Ref: head, Repo: &models.GitHubRepo{FullName: repo}}}
}

func numbers(found map[int][]models.GitHubPull) map[int][]int {
	out := map[int][]int{}
	for n, group := range found {
		for _, p := range group {
			out[n] = append(out[n], p.Number)
		}
	}
	return out
}

func TestEachPullRequestOfAChainIsReviewedWithTheWholeChainBottomFirst(t *testing.T) {
	pulls := []models.GitHubPull{pull(23, "basket", "checkout", "octo/widgets"), pull(21, "main", "search", "octo/widgets"), pull(22, "search", "basket", "octo/widgets"), pull(30, "main", "wording", "octo/widgets")}
	if got := numbers(Stacks(pulls, "octo/widgets")); !reflect.DeepEqual(got, map[int][]int{21: {21, 22, 23}, 22: {21, 22, 23}, 23: {21, 22, 23}, 30: {30}}) {
		t.Errorf("the chain is %v", got)
	}
}

func TestAChainThatForksIsNoStackAndABranchFromAnotherRepositoryChainsNothing(t *testing.T) {
	forked := []models.GitHubPull{pull(21, "main", "search", "octo/widgets"), pull(22, "search", "basket", "octo/widgets"), pull(24, "search", "filters", "octo/widgets")}
	if got := numbers(Stacks(forked, "octo/widgets")); !reflect.DeepEqual(got, map[int][]int{21: {21}, 22: {22}, 24: {24}}) {
		t.Errorf("a fork gives %v", got)
	}
	elsewhere := []models.GitHubPull{pull(21, "main", "search", "sam/widgets"), pull(22, "search", "basket", "octo/widgets")}
	if got := numbers(Stacks(elsewhere, "octo/widgets")); !reflect.DeepEqual(got, map[int][]int{21: {21}, 22: {22}}) {
		t.Errorf("a branch from another repository gives %v", got)
	}
}

func TestAnEntryHidesTheRepositoryItNamesAndOwnerStarEveryRepositoryOfTheOwner(t *testing.T) {
	entries := []string{"octo/widgets", "reviewer-home/*"}
	cases := []struct {
		entries []string
		name    string
		want    []string
	}{
		{entries, "octo/widgets", []string{"octo/widgets"}}, {entries, "octo/gadgets", []string{}}, {entries, "reviewer-home/notes", []string{"reviewer-home/*"}},
		{entries, "reviewer/notes", []string{}}, {[]string{"octo/widgets"}, "octo/widgets-two", []string{}}, {nil, "octo/widgets", []string{}},
		{[]string{"Octo/*", "octo/Widgets", "octo/gadgets"}, "octo/widgets", []string{"Octo/*", "octo/Widgets"}}, {[]string{"octo/*"}, "OCTO/Widgets", []string{"octo/*"}},
	}
	for _, c := range cases {
		if got := HiddenBy(c.entries, c.name); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%v hide %s as %v", c.entries, c.name, got)
		}
	}
}

func TestRepositoriesAreSortedAsPeoplesTextIsPunctuationFirst(t *testing.T) {
	names := []string{"ab", "a/b", "abc", "A.b", "a-b", "a0", "a_b", "ab-c", "a"}
	sorted := append([]string{}, names...)
	for i := range sorted {
		for j := i + 1; j < len(sorted); j++ {
			if beforeByName(sorted[j], sorted[i]) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	if want := []string{"a", "a_b", "a-b", "A.b", "a/b", "a0", "ab", "ab-c", "abc"}; !reflect.DeepEqual(sorted, want) {
		t.Errorf("the names sort as %v", sorted)
	}
}
