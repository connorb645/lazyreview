package gui

import (
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/internal/gui/context"
	"github.com/connorb645/lazyreview/internal/models"
)

func TestPostingShowsWhatItSendsThenSendsIt(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/30", 100, 30)
	a = press(t, a, "P")
	screenIs(t, a, "posting")
	a = press(t, a, "j", "j", "k")
	if a.Posting.Choice != 1 {
		t.Fatalf("j and k move the choice: %d", a.Posting.Choice)
	}
	a = press(t, a, "enter")
	screenIs(t, a, "posting-confirm")
	a = press(t, a, "n")
	if !a.Has(context.PostingKey) || a.Has(context.PostingConfirmationKey) {
		t.Fatal("n goes back to the choice")
	}
	a = press(t, a, "2", "enter", "y")
	if a.Has(context.PostingKey) || a.Bad || !a.MenuOpen() || a.Menu.Rows[0].Right != "Approved with 1 comment: https://github.com/octo/widgets/pull/30#pullrequestreview-8" {
		t.Fatalf("the answer: %q %+v", a.Notice, a.Menu)
	}
	screenIs(t, a, "posting-sent")
	if m, _ := a.helpers.Posting.QueueMember(a.ContextTree, 30); len(m.Comments) != 0 || len(m.Posted) != 1 {
		t.Fatalf("the queue after posting: %+v", m)
	}
}

func TestARefusedPostSaysWhy(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/30", 100, 30)
	state := a.Review.State
	queue := *state.Queue
	queue.Members = append([]models.QueueMember{}, queue.Members...)
	sets := map[string]models.Set{}
	for k, v := range queue.Members[0].Sets {
		sets[k] = v
	}
	sets["comment"] = models.Set{Set: "stale0000000", Words: sets["comment"].Words}
	queue.Members[0].Sets = sets
	state.Queue = &queue
	a = a.with(a.helpers.Refresh.WithState(a.ContextTree, state))
	a = press(t, a, "P", "1", "enter", "y")
	if !a.Bad || !a.MenuOpen() || a.Menu.Title != "Post to octo/widgets#30" || a.Menu.Rows[0].Right != `The page showed set stale0000000, but "comment" now gives set 43028caa56d9; nothing was posted` {
		t.Fatalf("the answer: %q %+v", a.Notice, a.Menu)
	}
}

func TestThePostingBarOffersOnlyTheChoicesThereAre(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/30", 100, 30)
	words := func(a App) string {
		var out []string
		for _, h := range a.controllers.Hints(a.ContextTree) {
			out = append(out, h[1]+": "+h[0])
		}
		return strings.Join(out, " | ")
	}
	if got := words(press(t, a, "P")); got != "Choose: j k 1 2 3 | Show what it sends: Enter | Cancel: Esc" {
		t.Fatalf("with three actions the bar says %q", got)
	}
	state := a.Review.State
	queue := *state.Queue
	queue.Members = append([]models.QueueMember{}, queue.Members...)
	queue.Members[0].Sets = map[string]models.Set{"approve": queue.Members[0].Sets["approve"]}
	state.Queue = &queue
	a = a.with(a.helpers.Refresh.WithState(a.ContextTree, state))
	if got := words(press(t, a, "P")); got != "Show what it sends: Enter | Cancel: Esc" {
		t.Fatalf("with one action the bar says %q", got)
	}
}

func TestEscClosesThePostingPanel(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/30", 100, 30)
	a = press(t, a, "P", "esc")
	if a.Has(context.PostingKey) {
		t.Fatal("Esc closes the panel")
	}
}
