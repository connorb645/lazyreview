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
// branches they head, feature/search on main and feature/cart on it,
// which has no pull request, no queue and nothing on GitHub.
func onBranchStack(t *testing.T, width, height int) App {
	t.Helper()
	return onBranchStackNamed(t, width, height, "feature/search", "feature/cart")
}

// onBranchStackNamed is onBranchStack with its two branches named bottom
// and top.
func onBranchStackNamed(t *testing.T, width, height int, bottom, top string) App {
	t.Helper()
	a, _ := opened(t, stack, width, height)
	d := *a.Review
	names := map[int]string{21: bottom, 22: top}
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
	st.Kind, st.Repo, st.Title, st.Members, st.Earlier, st.Queue, st.GitHub, st.Written = "branch", "widgets", "Stack "+bottom+" to "+top, members, nil, nil, nil, nil
	st.Host, st.Me, st.Account, st.Moved = "", "", nil, nil
	st.Base, st.Head = models.Ref{Ref: "main", Sha: st.Base.Sha}, models.Ref{Ref: top, Sha: members[1].Head}
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
	for _, want := range []string{"A stack of 2 branches in widgets", "feature/search  head ", "feature/cart    head ", "read only"} {
		if !strings.Contains(words, want) {
			t.Errorf("the Review view does not say %q:\n%s", want, words)
		}
	}
	if strings.Contains(words, "#21") || strings.Contains(words, "#22") || strings.Contains(words, "Queued") || strings.Contains(words, "on main") {
		t.Errorf("the Review view of a stack of branches names a pull request, a queue or the branch below:\n%s", words)
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
	a = press(t, a, "2")
	if bar := lastLine(a); !strings.Contains(bar, "Branch: ]p [p") {
		t.Errorf("the Files panel's bar offers %q", bar)
	}
	a = press(t, a, "j", "j")
	a = press(t, a, "enter")
	screenIs(t, a, "branch-stack-diff")
	if words := viewWords(a); !strings.Contains(words, "feature/cart") || strings.Contains(words, "#22") {
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

func TestAStackOfLongBranchNamesWrapsEachBranchUnderItsName(t *testing.T) {
	a := onBranchStackNamed(t, 80, 24, "catalogue-search/part-01", "catalogue-search/part-02")
	a = press(t, a, "0")
	screenIs(t, a, "branch-stack-long-names")
	words := viewWords(a)
	if !strings.Contains(words, "…/part-01") || !strings.Contains(words, "…/part-02") {
		t.Errorf("the Review panel does not tell the branches apart:\n%s", words)
	}
	a = press(t, a, "esc", "2", "j", "j", "enter")
	screenIs(t, a, "branch-stack-long-names-diff")
	if words := viewWords(a); !strings.Contains(words, "Basket.swift  changed  …/part-02  +3 −1") {
		t.Errorf("the diff's header does not keep the file and its counts:\n%s", words)
	}
}

func TestAStackOfBranchesPutsEveryBranchOnOneLineOrWrapsThemAll(t *testing.T) {
	for width := 70; width <= 130; width++ {
		a := press(t, onBranchStackNamed(t, width, 30, "catalogue-search/part-01", "catalogue-search/part-02"), "0")
		alone, whole := 0, 0
		for _, line := range strings.Split(viewWords(a), "\n") {
			switch {
			case strings.Contains(line, "  catalogue-search/part-0") && strings.Contains(line, "head "):
				whole++
				if end := []rune(strings.TrimRight(line, " ")); end[len(end)-2] != ' ' {
					t.Errorf("at %d columns a branch's line reaches the border: %q", width, line)
				}
			case strings.Contains(line, "  catalogue-search/part-0"):
				alone++
			}
		}
		if alone+whole != 2 || alone != 0 && whole != 0 {
			t.Errorf("at %d columns, %d branches wrap and %d do not:\n%s", width, alone, whole, viewWords(a))
		}
	}
}

func TestEveryFileHeaderOfADiffNamesTheBranchesAlike(t *testing.T) {
	for width := 70; width <= 161; width++ {
		a := press(t, onBranchStackNamed(t, width, 60, "catalogue-search/part-01", "catalogue-search/part-02"), "2")
		full, cut := 0, 0
		for _, line := range strings.Split(viewWords(a), "\n") {
			if !strings.Contains(line, " changed ") && !strings.Contains(line, " new ") {
				continue
			}
			if strings.Contains(line, "catalogue-search/part-0") {
				full++
			}
			if strings.Contains(line, "  …/part-0") {
				cut++
			}
		}
		if full+cut < 3 || full != 0 && cut != 0 {
			t.Errorf("at %d columns %d file headers name a branch in full and %d by its last part:\n%s", width, full, cut, viewWords(a))
		}
	}
}
