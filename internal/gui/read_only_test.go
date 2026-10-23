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
