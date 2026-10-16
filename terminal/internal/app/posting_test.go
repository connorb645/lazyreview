package app

import (
	"testing"

	"review-desk/terminal/internal/server"
)

func TestPostingShowsWhatItSendsThenSendsIt(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/30", 100, 30)
	a = press(t, a, "P")
	screenIs(t, a, "posting")
	a = press(t, a, "j", "j", "k")
	if a.posting.choice != 1 {
		t.Fatalf("j and k move the choice: %d", a.posting.choice)
	}
	a = press(t, a, "enter")
	screenIs(t, a, "posting-confirm")
	a = press(t, a, "n")
	if a.posting == nil || a.posting.confirming {
		t.Fatal("n goes back to the choice")
	}
	a = press(t, a, "2", "enter", "y")
	if a.posting != nil || a.bad || a.menu == nil || a.menu.rows[0].right != "Approved with 1 comment: https://github.com/octo/widgets/pull/30#pullrequestreview-8" {
		t.Fatalf("the answer: %q %+v", a.notice, a.menu)
	}
	screenIs(t, a, "posting-sent")
	if m, _ := a.queueMember(30); len(m.Comments) != 0 || len(m.Posted) != 1 {
		t.Fatalf("the queue after posting: %+v", m)
	}
}

func TestAPostTheServerRefusesSaysWhy(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/30", 100, 30)
	state := a.review.state
	queue := *state.Queue
	queue.Members = append([]server.QueueMember{}, queue.Members...)
	sets := map[string]server.Set{}
	for k, v := range queue.Members[0].Sets {
		sets[k] = v
	}
	sets["comment"] = server.Set{Set: "stale0000000", Words: sets["comment"].Words}
	queue.Members[0].Sets = sets
	state.Queue = &queue
	a = a.withState(state)
	a = press(t, a, "P", "1", "enter", "y")
	if !a.bad || a.menu == nil || a.menu.title != "Nothing was posted" || a.menu.rows[0].right != `The page showed set stale0000000, but "comment" now gives set 43028caa56d9; nothing was posted` {
		t.Fatalf("the answer: %q %+v", a.notice, a.menu)
	}
}

func TestEscClosesThePostingPanel(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/30", 100, 30)
	a = press(t, a, "P", "esc")
	if a.posting != nil {
		t.Fatal("Esc closes the panel")
	}
}
