package gui

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestTheDiffOfAReviewThatTakesNoCommentsOffersNoCommentKey(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/30", 100, 30)
	a = press(t, a, "2", "j", "j", "enter")
	if bar := lastLine(a); !strings.Contains(bar, "Comment: c") {
		t.Fatalf("a pull request's diff does not offer to comment: %q", bar)
	}
	state := a.Review.State
	state.Kind, state.Members, state.Queue, state.GitHub = "branch", nil, nil, nil
	a = relaid(a.with(a.helpers.Refresh.WithState(a.ContextTree, state)))
	if bar := lastLine(a); strings.Contains(bar, "Comment") || !strings.Contains(bar, "Ask Claude: a") {
		t.Errorf("a branch's diff offers %q", bar)
	}
	a = press(t, a, "j", "j", "V")
	if bar := lastLine(a); strings.Contains(bar, "Comment") || !strings.Contains(bar, "Ask Claude: a") {
		t.Errorf("a selection in a branch's diff offers %q", bar)
	}
}

func lastLine(a App) string {
	lines := strings.Split(ansi.Strip(a.View()), "\n")
	return lines[len(lines)-1]
}

func TestALoadWithNotesSaysHowManyInANoticeWholeAtEightyColumns(t *testing.T) {
	notes := `["#22 is not built on #21's head: #21 has commits #22 does not hold. Until #22 is rebased, its own diff also shows the older #21 it holds; a comment on a line #21's head still holds goes to #21.", "whole text left out for 1 large file, which show their diff only: big.json"]`
	a, _, changed := openedWith(t, "github.com/octo/widgets/21", 80, 24)
	changed.set("POST /api/reviews/github.com/octo/widgets/21/refresh {}", answer{Status: 200, Body: json.RawMessage(`{"id": "github.com/octo/widgets/21", "notes": ` + notes + `}`)})
	a = press(t, a, "R")
	if want := "Loaded again, with 2 notes, shown in the Review view."; a.Notice != want || !strings.Contains(lastLine(a), want) {
		t.Errorf("after R the bar reads %q, its notice %q", lastLine(a), a.Notice)
	}

	h, _, home := homeOn(t, "", 80, 24)
	home.set(`POST /api/load {"account":"reviewer","links":["https://github.com/octo/widgets/pull/30"]}`, answer{Status: 200, Body: json.RawMessage(`{"id": "github.com/octo/widgets/30", "notes": ` + notes + `}`)})
	h = press(t, h, "2", "j", "j")
	next, cmd := h.Update(keyMsg("enter"))
	h = drive(t, next.(App), cmd)
	if want := "Loaded, with 2 notes, shown in the Review view."; h.Notice != want || !strings.Contains(lastLine(h), want) {
		t.Errorf("after a load from home the bar reads %q, its notice %q", lastLine(h), h.Notice)
	}
}

func TestTheNotesPanelOfAReviewThatTakesNoCommentsSaysNothingIsAskedInFull(t *testing.T) {
	for _, c := range []struct {
		name string
		open func(t *testing.T) App
		keys []string
		want string
	}{
		{"a branch", func(t *testing.T) App {
			a, _ := opened(t, "github.com/octo/widgets/30", 80, 24)
			state := a.Review.State
			state.Kind, state.Members, state.Queue, state.GitHub, state.Threads = "branch", nil, nil, nil, nil
			return relaid(a.with(a.helpers.Refresh.WithState(a.ContextTree, state)))
		}, nil, "│ Nothing asked yet "},
		{"a stack of branches", func(t *testing.T) App {
			a := onBranchStackNamed(t, 80, 24, "catalogue-search/part-01", "catalogue-search/part-02")
			d := *a.Review
			d.State.Threads = nil
			a.Review = &d
			return relaid(a)
		}, nil, "│ Nothing asked yet "},
		{"a stack of branches on one branch", func(t *testing.T) App {
			a := onBranchStackNamed(t, 80, 24, "catalogue-search/part-01", "catalogue-search/part-02")
			d := *a.Review
			d.State.Threads = nil
			a.Review = &d
			return relaid(a)
		}, []string{"j"}, "│ Nothing asked here yet "},
	} {
		a := press(t, c.open(t), c.keys...)
		if shown := ansi.Strip(a.View()); !strings.Contains(shown, c.want) || strings.Contains(shown, "queued") {
			t.Errorf("the Notes panel of %s does not say %q:\n%s", c.name, c.want, shown)
		}
	}
}

// keysListed are the keys the keys popup lists where the keys are, each
// with what it says it does.
func keysListed(t *testing.T, a App) map[string]string {
	t.Helper()
	listed := map[string]string{}
	for _, r := range press(t, a, "space", "?").Menu.Rows {
		if r.Heading == "" {
			listed[r.Left] = r.Right
		}
	}
	return listed
}

func onOneBranch(t *testing.T) App {
	t.Helper()
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	state := a.Review.State
	state.Kind, state.Members, state.Earlier, state.Queue, state.GitHub, state.Written = "branch", nil, nil, nil, nil, nil
	d := *a.Review
	d.Page.Stack = nil
	a.Review = &d
	return relaid(a.with(a.helpers.Refresh.WithState(a.ContextTree, state)))
}

func TestAReadOnlyReviewsKeysListsOfferOnlyKeysThatActThere(t *testing.T) {
	deletes := "deletes the conversation with Claude at the cursor, after y confirms"
	for _, c := range []struct {
		name    string
		at      App
		stacked bool
	}{{"one branch", onOneBranch(t), false}, {"a stack of branches", onBranchStack(t, 100, 30), true}} {
		for place, keys := range map[string][]string{
			"the Review panel": {"1"}, "Notes": {"3"}, "the Review view": {"space", "v"}, "the diff": {"2", "j", "j", "enter"},
		} {
			listed := keysListed(t, press(t, c.at, keys...))
			for _, key := range []string{"e", "P", "c", "gx", "gX"} {
				if words, ok := listed[key]; ok {
					t.Errorf("on %s, %s's keys list offers %s: %q", c.name, place, key, words)
				}
			}
			if words, ok := listed["dd"]; ok && words != deletes {
				t.Errorf("on %s, %s's keys list says dd %q", c.name, place, words)
			}
			for key, words := range listed {
				if strings.Contains(words, "queued") || strings.Contains(words, "thread") {
					t.Errorf("on %s, %s's keys list says %s %q", c.name, place, key, words)
				}
			}
			if _, ok := listed["]p [p"]; ok != c.stacked {
				t.Errorf("on %s, %s's keys list offers ]p [p: %v", c.name, place, ok)
			}
			if _, ok := listed["v"]; place == "the Review panel" && ok != c.stacked {
				t.Errorf("on %s, the Review panel's keys list offers v: %v", c.name, ok)
			}
		}
	}
	a := onOneBranch(t)
	if a = press(t, a, "2", "j", "j", "enter", "d", "d"); a.Notice != "dd deletes the conversation with Claude at the cursor" {
		t.Errorf("off a conversation on a branch, dd says %q", a.Notice)
	}
	pull, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	listed := keysListed(t, press(t, pull, "3"))
	if listed["e"] == "" || listed["P"] == "" || listed["gx"] == "" || !strings.HasPrefix(listed["dd"], "drops the queued comment") {
		t.Errorf("a pull request's Notes keys list is %v", listed)
	}
}
