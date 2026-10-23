package gui

import (
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/internal/gui/context"
	"github.com/connorb645/lazyreview/internal/models"
)

// onBranchStack is the recorded stack of #21 and #22 as a stack of the
// branches they head, feature/search on main and feature/basket on it,
// which has no pull request, no queue and nothing on GitHub.
func onBranchStack(t *testing.T, width, height int) App {
	t.Helper()
	a, _ := opened(t, stack, width, height)
	d := *a.Review
	names := map[int]string{21: "feature/search", 22: "feature/basket"}
	stacked := *d.Page.Stack
	stacked.Members = slices.Clone(stacked.Members)
	for i := range stacked.Members {
		stacked.Members[i].Branch = names[stacked.Members[i].Pull]
	}
	d.Page.Stack = &stacked
	st := d.State
	members := make([]models.Member, len(st.Members))
	for i, m := range st.Members {
		base := "main"
		if i > 0 {
			base = names[st.Members[i-1].Pull]
		}
		members[i] = models.Member{Pull: m.Pull, Branch: names[m.Pull], Head: m.Head, Base: base, Color: m.Color}
	}
	st.Kind, st.Repo, st.Title, st.Members, st.Earlier, st.Queue, st.GitHub, st.Written = "branch", "widgets", "Stack feature/search to feature/basket", members, nil, nil, nil, nil
	st.Host, st.Me, st.Account, st.Moved = "", "", nil, nil
	st.Base, st.Head = models.Ref{Ref: "main", Sha: st.Base.Sha}, models.Ref{Ref: "feature/basket", Sha: members[1].Head}
	d.State = st
	a.Review = &d
	return relaid(a)
}

func viewWords(a App) string {
	return ansi.Strip(a.View())
}

func TestAStackOfBranchesShowsItsBranchesWhereAStackShowsItsPullRequests(t *testing.T) {
	a := onBranchStack(t, 100, 30)
	screenIs(t, a, "branch-stack")
	if got := strings.Join(rowKeys(a, context.ReviewPanel), " "); got != "all pull:21 pull:22" {
		t.Fatalf("the Review panel's rows are %s", got)
	}
	a = press(t, a, "0")
	screenIs(t, a, "branch-stack-review-view")
	words := viewWords(a)
	for _, want := range []string{"A stack of 2 branches in widgets", "feature/search  on main", "on feature/search · head", "read only"} {
		if !strings.Contains(words, want) {
			t.Errorf("the Review view does not say %q:\n%s", want, words)
		}
	}
	if strings.Contains(words, "#21") || strings.Contains(words, "#22") || strings.Contains(words, "Queued") {
		t.Errorf("the Review view of a stack of branches names a pull request or a queue:\n%s", words)
	}
}

func TestAStackOfBranchesFiltersByBranchAsAStackFiltersByPullRequest(t *testing.T) {
	a := onBranchStack(t, 100, 30)
	a = press(t, a, "j")
	filterIs(t, a, "[21]")
	screenIs(t, a, "branch-stack-filtered")
	if words := viewWords(a); !strings.Contains(words, "Showing feature/search only.") {
		t.Errorf("the filter is not named by its branch:\n%s", words)
	}
	a = press(t, a, "]", "p")
	filterIs(t, a, "[22]")
	a = press(t, a, ":")
	a = typed(t, a, "pull feature/search")
	filterIs(t, a, "[21]")
	a = press(t, a, ":")
	a = typed(t, a, "pull feature/nothing")
	filterIs(t, a, "[21]")
	if a.Notice != "no branch feature/nothing in this stack" {
		t.Errorf(":pull with a branch the stack does not hold says %q", a.Notice)
	}
	a = press(t, a, ":")
	a = typed(t, a, "pull all")
	filterIs(t, a, "[]")
}

func TestAStackOfBranchesNamesTheBranchesThatChangedAFileInItsDiff(t *testing.T) {
	a := onBranchStack(t, 120, 30)
	a = press(t, a, "2", "j", "j")
	a = press(t, a, "enter")
	screenIs(t, a, "branch-stack-diff")
	if words := viewWords(a); !strings.Contains(words, "feature/basket") || strings.Contains(words, "#22") {
		t.Errorf("the diff does not name the branch that changed the file:\n%s", words)
	}
	a = press(t, a, "g", "x")
	if a.Notice != "this has no pull request on GitHub" {
		t.Errorf("gx on a stack of branches says %q", a.Notice)
	}
}

func TestAPullRequestTheStackDoesNotHoldIsNamedOnce(t *testing.T) {
	a, _ := opened(t, stack, 100, 30)
	a = press(t, a, ":")
	a = typed(t, a, "pull #99")
	if a.Notice != "no pull request #99 in this stack" {
		t.Errorf(":pull #99 says %q", a.Notice)
	}
}
